package archiver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Archiver moves processed PDF files into a structured archive directory.
type Archiver struct {
	baseDir      string
	keepOriginal bool
}

// New creates an Archiver that stores archived files under baseDir.
func New(baseDir string) *Archiver {
	return &Archiver{baseDir: baseDir}
}

// SetKeepOriginal configures whether Archive copies (true) or moves (false) the source file.
func (a *Archiver) SetKeepOriginal(keep bool) {
	a.keepOriginal = keep
}

// Archive moves the file at srcPath into a subdirectory determined by status,
// subdir, and printer, renaming it with a timestamp prefix.
//
// Directory structure:
//   - "printed"  → baseDir/printed/<subdir>/YYYY/MM/DD/
//   - "failed"   → baseDir/failed/YYYY/MM/DD/
//   - anything   → baseDir/unknown/YYYY/MM/DD/
//
// If subdir is empty for "printed", it defaults to "unsorted".
// If printer is empty, it defaults to "noprinter".
// Files are renamed to YYYYMMDD_HHMMSS_<printer>_<original>.pdf.
// On name collision the suffix _1, _2, … is inserted before .pdf.
func (a *Archiver) Archive(srcPath, status, subdir, printer string) (string, error) {
	now := time.Now()
	datePath := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	if printer == "" {
		printer = "noprinter"
	}

	var destDir string
	switch status {
	case "printed":
		if subdir == "" {
			subdir = "unsorted"
		}
		destDir = filepath.Join(a.baseDir, "printed", subdir, datePath)
	case "failed":
		destDir = filepath.Join(a.baseDir, "failed", datePath)
	default:
		destDir = filepath.Join(a.baseDir, "unknown", datePath)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("archiver: create destination directory %s: %w", destDir, err)
	}

	original := filepath.Base(srcPath)
	ext := filepath.Ext(original)
	nameWithoutExt := strings.TrimSuffix(original, ext)

	timestamp := now.Format("20060102_150405")
	baseName := fmt.Sprintf("%s_%s_%s%s", timestamp, printer, nameWithoutExt, ext)

	destPath := filepath.Join(destDir, baseName)

	// Resolve collisions by appending _1, _2, … before the extension.
	if _, err := os.Stat(destPath); err == nil {
		nameNoExt := strings.TrimSuffix(baseName, ext)
		for i := 1; ; i++ {
			candidate := filepath.Join(destDir, fmt.Sprintf("%s_%d%s", nameNoExt, i, ext))
			if _, err := os.Stat(candidate); os.IsNotExist(err) {
				destPath = candidate
				break
			}
		}
	}

	if a.keepOriginal {
		if err := copyFile(srcPath, destPath); err != nil {
			return "", fmt.Errorf("archiver: copy %s → %s: %w", srcPath, destPath, err)
		}
	} else {
		if err := moveFile(srcPath, destPath); err != nil {
			return "", fmt.Errorf("archiver: move %s → %s: %w", srcPath, destPath, err)
		}
	}

	return destPath, nil
}

// copyFile copies src to dst without removing src.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// moveFile attempts os.Rename first; falls back to copy+delete for cross-device moves.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Cross-device fallback: copy then remove source.
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close destination: %w", err)
	}

	return os.Remove(src)
}
