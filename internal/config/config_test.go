package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StabilityDelaySeconds != 2 {
		t.Errorf("expected default stability_delay=2, got %d", cfg.StabilityDelaySeconds)
	}
	if cfg.WorkerCount != 2 {
		t.Errorf("expected default worker_count=2, got %d", cfg.WorkerCount)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("watch_dir: /tmp/watch\narchive_dir: /tmp/archive\nstability_delay_seconds: 5\nworker_count: 4\n"), 0644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WatchDir != "/tmp/watch" {
		t.Errorf("expected watch_dir=/tmp/watch, got %s", cfg.WatchDir)
	}
	if cfg.StabilityDelaySeconds != 5 {
		t.Errorf("expected stability_delay=5, got %d", cfg.StabilityDelaySeconds)
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Config{WatchDir: "/tmp/test", ArchiveDir: "/tmp/archive", StabilityDelaySeconds: 3, WorkerCount: 1}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.WatchDir != cfg.WatchDir {
		t.Errorf("expected %s, got %s", cfg.WatchDir, loaded.WatchDir)
	}
}
