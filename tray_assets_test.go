package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareTrayIconFileWritesIconBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	iconBytes := []byte("test-icon")

	path, err := prepareTrayIconFile(dir, iconBytes)
	if err != nil {
		t.Fatalf("prepareTrayIconFile() error = %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Fatalf("prepareTrayIconFile() dir = %q, want %q", filepath.Dir(path), dir)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	if string(got) != string(iconBytes) {
		t.Fatalf("prepareTrayIconFile() wrote %q, want %q", string(got), string(iconBytes))
	}
}

func TestPrepareTrayIconFileReplacesExistingBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, trayIconFilename)
	if err := os.WriteFile(target, []byte("old-icon"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", target, err)
	}

	path, err := prepareTrayIconFile(dir, []byte("new-icon"))
	if err != nil {
		t.Fatalf("prepareTrayIconFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	if string(got) != "new-icon" {
		t.Fatalf("prepareTrayIconFile() wrote %q, want %q", string(got), "new-icon")
	}
}
