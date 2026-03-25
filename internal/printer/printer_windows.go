//go:build windows

package printer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const (
	sumatraURL      = "https://www.sumatrapdfreader.org/dl/rel/3.5.2/SumatraPDF-3.5.2-64.exe"
	sumatraExeName  = "SumatraPDF.exe"
	downloadTimeout = 120 * time.Second
	printTimeout    = 120 * time.Second
)

type WindowsPrinter struct {
	dataDir    string
	sumatraExe string
}

func New() (Printer, error) {
	// Store SumatraPDF next to the executable
	dataDir := ""
	if exe, err := os.Executable(); err == nil {
		dataDir = filepath.Dir(exe)
	}
	if dataDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			dataDir = cwd
		}
	}

	p := &WindowsPrinter{
		dataDir:    dataDir,
		sumatraExe: filepath.Join(dataDir, sumatraExeName),
	}

	// Also check common install locations
	if !p.IsSumatraInstalled() {
		for _, candidate := range sumatraSearchPaths() {
			if _, err := os.Stat(candidate); err == nil {
				p.sumatraExe = candidate
				break
			}
		}
	}

	return p, nil
}

func sumatraSearchPaths() []string {
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LOCALAPPDATA")
	paths := []string{}
	for _, dir := range []string{programFiles, programFilesX86, localAppData} {
		if dir != "" {
			paths = append(paths, filepath.Join(dir, "SumatraPDF", sumatraExeName))
		}
	}
	return paths
}

func (p *WindowsPrinter) IsSumatraInstalled() bool {
	_, err := os.Stat(p.sumatraExe)
	return err == nil
}

func (p *WindowsPrinter) DownloadSumatra() error {
	destPath := filepath.Join(p.dataDir, sumatraExeName)

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumatraURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download SumatraPDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download SumatraPDF: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(p.dataDir, "sumatra-download-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write SumatraPDF: %w", err)
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("install SumatraPDF: %w", err)
	}

	p.sumatraExe = destPath
	return nil
}

func (p *WindowsPrinter) Print(filePath, printerName string) error {
	if !p.IsSumatraInstalled() {
		return fmt.Errorf("SumatraPDF not found at %s — please download it from the settings page", p.sumatraExe)
	}

	// SumatraPDF CLI:
	//   SumatraPDF.exe -print-to "Printer Name" -print-settings "fit" -silent file.pdf
	// -silent: no error dialogs
	// -print-settings: "fit" scales to page, "noscale" prints at original size
	ctx, cancel := context.WithTimeout(context.Background(), printTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.sumatraExe,
		"-print-to", printerName,
		"-print-settings", "fit",
		"-silent",
		filePath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("SumatraPDF print failed: %w, output: %s", err, out)
	}
	return nil
}

func (p *WindowsPrinter) ListPrinters() ([]string, error) {
	const PRINTER_ENUM_LOCAL = 0x00000002
	const PRINTER_ENUM_CONNECTIONS = 0x00000004

	winspool := syscall.NewLazyDLL("winspool.drv")
	enumPrinters := winspool.NewProc("EnumPrintersW")

	flags := uint32(PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS)
	var needed, returned uint32

	enumPrinters.Call(
		uintptr(flags),
		0,
		2,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)

	if needed == 0 {
		return nil, nil
	}

	buf := make([]byte, needed)
	ret, _, err := enumPrinters.Call(
		uintptr(flags),
		0,
		2,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&returned)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumPrintersW: %w", err)
	}

	const printerInfo2Size = 136
	var names []string
	for i := uint32(0); i < returned; i++ {
		offset := uintptr(i) * printerInfo2Size
		namePtr := *(**uint16)(unsafe.Pointer(&buf[offset+8]))
		if namePtr != nil {
			names = append(names, syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(namePtr))[:]))
		}
	}
	return names, nil
}

func (p *WindowsPrinter) IsOnline(printerName string) bool {
	winspool := syscall.NewLazyDLL("winspool.drv")
	openPrinter := winspool.NewProc("OpenPrinterW")
	closePrinter := winspool.NewProc("ClosePrinter")

	namePtr, err := syscall.UTF16PtrFromString(printerName)
	if err != nil {
		return false
	}

	var handle uintptr
	ret, _, _ := openPrinter.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&handle)),
		0,
	)
	if ret == 0 {
		return false
	}
	closePrinter.Call(handle)
	return true
}

func (p *WindowsPrinter) TestPrint(printerName string) error {
	tmpFile, err := os.CreateTemp("", "printdock-test-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(testPDFContent())
	tmpFile.Close()
	return p.Print(tmpFile.Name(), printerName)
}

func testPDFContent() []byte {
	return []byte(`%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/MediaBox[0 0 595 842]/Parent 2 0 R/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj
4 0 obj<</Length 44>>stream
BT /F1 24 Tf 50 750 Td (PrintDock - Test Page) Tj ET
endstream
endobj
5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj
xref
0 6
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000266 00000 n
0000000360 00000 n
trailer<</Size 6/Root 1 0 R>>
startxref
435
%%EOF`)
}

func (p *WindowsPrinter) Close() error {
	return nil
}
