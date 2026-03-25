package archiver_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"printdock/internal/archiver"
)

// createTempPDF creates a temporary PDF file with the given name in a temp dir.
func createTempPDF(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("%PDF-1.4 fake content"), 0644); err != nil {
		t.Fatalf("failed to create temp file %s: %v", path, err)
	}
	return path
}

func TestArchive_PrintedFile(t *testing.T) {
	srcDir := t.TempDir()
	baseDir := t.TempDir()

	src := createTempPDF(t, srcDir, "document.pdf")

	a := archiver.New(baseDir)
	dest, err := a.Archive(src, "printed", "invoices", "HP_LaserJet")
	if err != nil {
		t.Fatalf("Archive returned unexpected error: %v", err)
	}

	// Destination path must contain "printed"
	if !strings.Contains(dest, "printed") {
		t.Errorf("expected 'printed' in dest path, got: %s", dest)
	}

	// Destination path must contain the subdir
	if !strings.Contains(dest, "invoices") {
		t.Errorf("expected subdir 'invoices' in dest path, got: %s", dest)
	}

	// Destination path must contain the printer name
	if !strings.Contains(dest, "HP_LaserJet") {
		t.Errorf("expected printer 'HP_LaserJet' in dest path, got: %s", dest)
	}

	// Destination file must exist
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Errorf("destination file does not exist: %s", dest)
	}

	// Source file must be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file should be gone after archive, but still exists: %s", src)
	}
}

func TestArchive_FailedFile_NoprinterPlaceholder(t *testing.T) {
	srcDir := t.TempDir()
	baseDir := t.TempDir()

	src := createTempPDF(t, srcDir, "broken.pdf")

	a := archiver.New(baseDir)
	dest, err := a.Archive(src, "failed", "", "")
	if err != nil {
		t.Fatalf("Archive returned unexpected error: %v", err)
	}

	// Destination path must contain "failed"
	if !strings.Contains(dest, "failed") {
		t.Errorf("expected 'failed' in dest path, got: %s", dest)
	}

	// When printer is empty, filename must use "noprinter"
	base := filepath.Base(dest)
	if !strings.Contains(base, "noprinter") {
		t.Errorf("expected 'noprinter' in filename when printer is empty, got: %s", base)
	}

	// Destination file must exist
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Errorf("destination file does not exist: %s", dest)
	}

	// Source file must be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file should be gone after archive, but still exists: %s", src)
	}
}

func TestArchive_UnknownStatus(t *testing.T) {
	srcDir := t.TempDir()
	baseDir := t.TempDir()

	src := createTempPDF(t, srcDir, "mystery.pdf")

	a := archiver.New(baseDir)
	dest, err := a.Archive(src, "someweirdstatus", "", "printerX")
	if err != nil {
		t.Fatalf("Archive returned unexpected error: %v", err)
	}

	// Destination path must contain "unknown"
	if !strings.Contains(dest, "unknown") {
		t.Errorf("expected 'unknown' in dest path for unknown status, got: %s", dest)
	}

	// Source file must be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source file should be gone after archive, but still exists: %s", src)
	}
}

func TestArchive_PrintedEmptySubdir_UsesUnsorted(t *testing.T) {
	srcDir := t.TempDir()
	baseDir := t.TempDir()

	src := createTempPDF(t, srcDir, "nodoc.pdf")

	a := archiver.New(baseDir)
	dest, err := a.Archive(src, "printed", "", "myprinter")
	if err != nil {
		t.Fatalf("Archive returned unexpected error: %v", err)
	}

	// When subdir is empty for "printed", must default to "unsorted"
	if !strings.Contains(dest, "unsorted") {
		t.Errorf("expected 'unsorted' in dest path when subdir is empty, got: %s", dest)
	}
}

func TestArchive_Collision_PathsDiffer(t *testing.T) {
	srcDir := t.TempDir()
	baseDir := t.TempDir()

	src1 := createTempPDF(t, srcDir, "report.pdf")
	src2 := createTempPDF(t, srcDir, "report2.pdf")

	a := archiver.New(baseDir)

	dest1, err := a.Archive(src1, "printed", "reports", "Canon")
	if err != nil {
		t.Fatalf("first Archive call failed: %v", err)
	}

	// Rename src2 to the same original name so collision can occur
	sameNameSrc := filepath.Join(srcDir, "report_dup.pdf")
	if err := os.WriteFile(sameNameSrc, []byte("%PDF-1.4 content2"), 0644); err != nil {
		t.Fatalf("failed to create duplicate-named file: %v", err)
	}
	_ = src2 // unused, we made sameNameSrc instead

	// Archive a second file that will produce the same timestamp-based name.
	// To force collision we archive dest1 back to src (copy it) and re-archive.
	// Instead, we directly test that two files with the same original name
	// going to the same folder get distinct destinations.

	// Copy dest1 back to srcDir with same original name to force collision
	data, err := os.ReadFile(dest1)
	if err != nil {
		t.Fatalf("failed to read dest1: %v", err)
	}
	collidingSrc := filepath.Join(srcDir, "report_collide.pdf")
	if err := os.WriteFile(collidingSrc, data, 0644); err != nil {
		t.Fatalf("failed to write colliding source: %v", err)
	}

	// Place a file in the dest1 location with the exact same computed name
	// by using the Archiver's own naming. We already have dest1.
	// Now rename collidingSrc to the same original filename as the one used for dest1.
	// We'll extract the original filename from dest1: last segment after printer prefix.
	// Format: YYYYMMDD_HHMMSS_<printer>_<original>.pdf
	// We need to fake a collision: put dest1 back in place (it's already there),
	// and archive a new file that would produce the same destination name.
	// The simplest way: archive two files with the exact same original name in rapid succession.

	src3 := filepath.Join(srcDir, "same_name.pdf")
	src4 := filepath.Join(srcDir, "same_name_copy.pdf") // will be renamed
	if err := os.WriteFile(src3, []byte("%PDF-1.4 a"), 0644); err != nil {
		t.Fatalf("failed to write src3: %v", err)
	}
	if err := os.WriteFile(src4, []byte("%PDF-1.4 b"), 0644); err != nil {
		t.Fatalf("failed to write src4: %v", err)
	}

	dest3, err := a.Archive(src3, "printed", "col", "TestPrinter")
	if err != nil {
		t.Fatalf("third Archive call failed: %v", err)
	}

	// Rename src4 to same original name and place a file at dest3's name to force collision
	forcedSrc := filepath.Join(srcDir, "same_name.pdf")
	if err := os.WriteFile(forcedSrc, []byte("%PDF-1.4 c"), 0644); err != nil {
		t.Fatalf("failed to write forcedSrc: %v", err)
	}

	// Place a file at the path that dest3 was, so the next archive with same name must bump
	// (dest3 is already there from previous archive)
	dest4, err := a.Archive(forcedSrc, "printed", "col", "TestPrinter")
	if err != nil {
		t.Fatalf("fourth Archive call (collision) failed: %v", err)
	}

	if dest3 == dest4 {
		t.Errorf("collision not resolved: both archives returned same path %s", dest3)
	}

	// Both destination files must exist
	if _, err := os.Stat(dest3); os.IsNotExist(err) {
		t.Errorf("dest3 file does not exist: %s", dest3)
	}
	if _, err := os.Stat(dest4); os.IsNotExist(err) {
		t.Errorf("dest4 file does not exist: %s", dest4)
	}
}

func TestArchive_KeepOriginal(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "keep.pdf")
	os.WriteFile(srcFile, []byte("pdf content"), 0644)

	a := archiver.New(filepath.Join(dir, "archive"))
	a.SetKeepOriginal(true)
	_, err := a.Archive(srcFile, "printed", "test", "HP")
	if err != nil {
		t.Fatal(err)
	}
	// Source should still exist
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		t.Error("source file should still exist when keepOriginal is true")
	}
}

func TestArchive_CreatesDirectories(t *testing.T) {
	srcDir := t.TempDir()
	// Use a deeply nested baseDir that doesn't exist yet
	baseDir := filepath.Join(t.TempDir(), "deeply", "nested", "archive")

	src := createTempPDF(t, srcDir, "test.pdf")

	a := archiver.New(baseDir)
	dest, err := a.Archive(src, "printed", "sub", "printer1")
	if err != nil {
		t.Fatalf("Archive failed when directories don't exist: %v", err)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Errorf("destination file does not exist after creating dirs: %s", dest)
	}
}
