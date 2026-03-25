//go:build !windows

package printer

import (
	"testing"
)

func TestNew_ReturnsValidPrinter(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if p == nil {
		t.Fatal("New() returned nil printer")
	}
	defer p.Close()
}

func TestListPrinters_ReturnsStubs(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer p.Close()

	printers, err := p.ListPrinters()
	if err != nil {
		t.Fatalf("ListPrinters() returned error: %v", err)
	}

	if len(printers) == 0 {
		t.Fatal("ListPrinters() returned empty list, expected stub printers")
	}

	expected := map[string]bool{
		"Zebra_ZD421_stub": false,
		"HP_LaserJet_stub": false,
	}
	for _, name := range printers {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected stub printer %q not found in list: %v", name, printers)
		}
	}
}

func TestPrint_StubNoError(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer p.Close()

	if err := p.Print("/tmp/test.pdf", "Zebra_ZD421_stub"); err != nil {
		t.Errorf("Print() returned unexpected error: %v", err)
	}
}

func TestClose_NoError(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}
