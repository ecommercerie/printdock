package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	dir            string
	fileChan       chan string
	stabilityDelay time.Duration
	fsWatcher      *fsnotify.Watcher
	stop           chan struct{}
	mu             sync.Mutex
	running        bool
	seen           sync.Map // tracks paths already sent to fileChan
}

func New(dir string, fileChan chan string, stabilityDelaySeconds int) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		dir:            dir,
		fileChan:       fileChan,
		stabilityDelay: time.Duration(stabilityDelaySeconds) * time.Second,
		fsWatcher:      fsw,
		stop:           make(chan struct{}),
	}, nil
}

func (w *Watcher) Start() error {
	if err := w.fsWatcher.Add(w.dir); err != nil {
		return err
	}
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}
			// Handle Create, Rename (move into dir), and Write (overwrite)
			if event.Op&(fsnotify.Create|fsnotify.Rename|fsnotify.Write) != 0 {
				if strings.ToLower(filepath.Ext(event.Name)) == ".pdf" {
					go w.waitForStability(event.Name)
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("[watcher] error: %v", err)
		case <-w.stop:
			return nil
		}
	}
}

func (w *Watcher) waitForStability(path string) {
	// Deduplicate: if another goroutine is already handling this path, skip
	if _, loaded := w.seen.LoadOrStore(path, true); loaded {
		return
	}
	var lastSize int64 = -1
	statErrors := 0
	for {
		info, err := os.Stat(path)
		if err != nil {
			statErrors++
			if statErrors >= 3 {
				log.Printf("[watcher] file inaccessible after %d retries, skipping: %s (%v)", statErrors, path, err)
				w.seen.Delete(path)
				return
			}
			time.Sleep(w.stabilityDelay)
			continue
		}
		statErrors = 0
		size := info.Size()
		if size == lastSize {
			w.fileChan <- path
			// Clean up seen entry so the same filename can be detected again later
			w.seen.Delete(path)
			return
		}
		lastSize = size
		time.Sleep(w.stabilityDelay)
	}
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stop)
		w.fsWatcher.Close()
		w.running = false
	}
}

func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// ScanExisting scans the watched directory for existing PDF files and sends them to fileChan.
func (w *Watcher) ScanExisting() (int, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != ".pdf" {
			continue
		}
		path := filepath.Join(w.dir, e.Name())
		if _, loaded := w.seen.LoadOrStore(path, true); loaded {
			continue
		}
		w.fileChan <- path
		w.seen.Delete(path)
		count++
	}
	return count, nil
}
