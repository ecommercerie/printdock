//go:build !windows

package printer

import "fmt"

type StubPrinter struct{}

func New() (Printer, error) {
	return &StubPrinter{}, nil
}

func (p *StubPrinter) Print(filePath, printerName string) error {
	fmt.Printf("[STUB] Would print %s to %s\n", filePath, printerName)
	return nil
}

func (p *StubPrinter) ListPrinters() ([]string, error) {
	return []string{"Zebra_ZD421_stub", "HP_LaserJet_stub"}, nil
}

func (p *StubPrinter) IsOnline(printerName string) bool { return true }

func (p *StubPrinter) TestPrint(printerName string) error {
	fmt.Printf("[STUB] Test print on %s\n", printerName)
	return nil
}

func (p *StubPrinter) IsSumatraInstalled() bool { return true }

func (p *StubPrinter) DownloadSumatra() error {
	fmt.Println("[STUB] Would download SumatraPDF")
	return nil
}

func (p *StubPrinter) Close() error { return nil }
