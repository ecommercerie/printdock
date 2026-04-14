//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"printdock/internal/updater"
)

// applyUpdate downloads the new exe, writes a batch script that waits for
// this process to exit, replaces the exe, and restarts it.
func applyUpdate(status updater.UpdateStatus) error {
	if status.DownloadURL == "" {
		return fmt.Errorf("no download URL available")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable: %w", err)
	}

	// Download new exe to temp location
	tmpExe := filepath.Join(os.TempDir(), "printdock-update.exe")
	if err := updater.DownloadTo(status.DownloadURL, tmpExe); err != nil {
		return err
	}

	// Write a batch script that:
	// 1. Waits for the current process to exit
	// 2. Replaces the exe
	// 3. Restarts PrintDock
	// 4. Cleans up the batch file
	batPath := filepath.Join(os.TempDir(), "printdock-update.bat")
	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
copy /y "%s" "%s" >nul
start "" "%s"
del "%%~f0"
`, tmpExe, currentExe, currentExe)

	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	// Launch the batch script hidden
	cmd := exec.Command("cmd", "/C", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update script: %w", err)
	}

	return nil
}
