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
			if event.Op&fsnotify.Create == fsnotify.Create {
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
	var lastSize int64 = -1
	for {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		size := info.Size()
		if size == lastSize {
			w.fileChan <- path
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
