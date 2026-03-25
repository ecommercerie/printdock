package applog

import "testing"

func TestAddAndGet(t *testing.T) {
	l := New(100)
	l.Info("test message %d", 1)
	l.Warn("warning")
	l.Error("error")

	entries := l.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Level != "info" || entries[0].Message != "test message 1" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
}

func TestRingBuffer(t *testing.T) {
	l := New(3)
	l.Info("a")
	l.Info("b")
	l.Info("c")
	l.Info("d") // should evict "a"

	entries := l.GetEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "b" {
		t.Errorf("expected 'b' as first, got %s", entries[0].Message)
	}
}

func TestClear(t *testing.T) {
	l := New(100)
	l.Info("test")
	l.Clear()
	if len(l.GetEntries()) != 0 {
		t.Error("expected empty after clear")
	}
}
