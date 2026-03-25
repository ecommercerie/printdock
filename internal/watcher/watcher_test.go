package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectsNewPDF(t *testing.T) {
	dir := t.TempDir()
	fileChan := make(chan string, 10)
	w, err := New(dir, fileChan, 1)
	if err != nil {
		t.Fatal(err)
	}
	go w.Start()
	defer w.Stop()

	time.Sleep(200 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "test.pdf"), []byte("pdf content"), 0644)

	select {
	case path := <-fileChan:
		if filepath.Base(path) != "test.pdf" {
			t.Errorf("expected test.pdf, got %s", filepath.Base(path))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for file detection")
	}
}

func TestIgnoresNonPDF(t *testing.T) {
	dir := t.TempDir()
	fileChan := make(chan string, 10)
	w, _ := New(dir, fileChan, 1)
	go w.Start()
	defer w.Stop()

	time.Sleep(200 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("not a pdf"), 0644)

	select {
	case <-fileChan:
		t.Fatal("should not detect non-PDF files")
	case <-time.After(3 * time.Second):
		// expected
	}
}
