//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"printdock/internal/updater"
)

// applyUpdate downloads the new exe, writes a batch script that waits for
// this process to exit, replaces the exe, and restarts it.
// The script is launched elevated (UAC) since Program Files requires admin.
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
	// 2. Replaces the exe (needs admin for Program Files)
	// 3. Restarts PrintDock (as normal user)
	// 4. Cleans up the batch file and temp exe
	batPath := filepath.Join(os.TempDir(), "printdock-update.bat")
	batContent := fmt.Sprintf(`@echo off
timeout /t 3 /nobreak >nul
copy /y "%s" "%s" >nul
del "%s" >nul
start "" "%s"
del "%%~f0"
`, tmpExe, currentExe, tmpExe, currentExe)

	if err := os.WriteFile(batPath, []byte(batContent), 0644); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	// Launch the batch script elevated via ShellExecuteW("runas")
	verb, _ := syscall.UTF16PtrFromString("runas")
	exe, _ := syscall.UTF16PtrFromString("cmd")
	args, _ := syscall.UTF16PtrFromString("/C \"" + batPath + "\"")

	shellExecute := shell32.NewProc("ShellExecuteW")
	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exe)),
		uintptr(unsafe.Pointer(args)),
		0,
		0, // SW_HIDE
	)
	if ret <= 32 {
		return fmt.Errorf("failed to launch elevated update script (code %d)", ret)
	}

	return nil
}
