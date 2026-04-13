//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const installDir = `C:\Program Files\PrintDock`
const appDataSubdir = "PrintDock"

func getDataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("USERPROFILE")
	}
	return filepath.Join(appData, appDataSubdir)
}

// isInstalledLocation checks if the current exe is running from Program Files.
func isInstalledLocation() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.ToLower(exe), strings.ToLower(installDir))
}

// needsInstall returns true if the exe is not in the install directory.
func needsInstall() bool {
	return !isInstalledLocation()
}

// killRunningInstance finds and gracefully stops any running PrintDock instance.
func killRunningInstance() {
	className, _ := syscall.UTF16PtrFromString("PrintDockTray")
	hwnd, _, _ := user32.NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(className)), 0)
	if hwnd == 0 {
		return
	}

	// Send WM_CLOSE to the tray window
	const wmClose = 0x0010
	user32.NewProc("PostMessageW").Call(hwnd, wmClose, 0, 0)

	// Wait for the process to exit (check every 500ms, max 10s)
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		hwnd2, _, _ := user32.NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(className)), 0)
		if hwnd2 == 0 {
			return
		}
	}

	// Still running — try taskkill as last resort
	exec.Command("taskkill", "/F", "/IM", "printdock.exe").Run()
	time.Sleep(1 * time.Second)
}

// selfInstall copies the exe to Program Files, creates shortcuts, and relaunches.
// This requires admin elevation.
func selfInstall() error {
	srcExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// Kill any running instance first (so we can overwrite the exe)
	killRunningInstance()

	// Create install directory
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("cannot create install dir: %w", err)
	}

	// Copy exe (retry a few times in case the old process is still releasing the file)
	destExe := filepath.Join(installDir, "printdock.exe")
	var copyErr error
	for attempt := 0; attempt < 5; attempt++ {
		copyErr = copyFile(srcExe, destExe)
		if copyErr == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if copyErr != nil {
		return fmt.Errorf("cannot copy exe: %w", copyErr)
	}

	// Copy configuration files from source directory to %APPDATA%\PrintDock
	dataDir := getDataDir()
	os.MkdirAll(dataDir, 0755)
	srcDir := filepath.Dir(srcExe)
	for _, name := range []string{"config.yaml", "rules.yaml", "history.db"} {
		srcFile := filepath.Join(srcDir, name)
		if _, err := os.Stat(srcFile); err == nil {
			destFile := filepath.Join(dataDir, name)
			// Don't overwrite existing config on update, only copy if missing
			if _, err := os.Stat(destFile); os.IsNotExist(err) {
				copyFile(srcFile, destFile)
			}
		}
	}

	// Create Start Menu shortcut
	startMenuDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs")
	createShortcut(destExe, filepath.Join(startMenuDir, "PrintDock.lnk"))

	// Create Desktop shortcut
	desktopDir := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	createShortcut(destExe, filepath.Join(desktopDir, "PrintDock.lnk"))

	return nil
}

// relaunchInstalled starts the installed exe and exits the current process.
func relaunchInstalled() {
	destExe := filepath.Join(installDir, "printdock.exe")
	cmd := exec.Command(destExe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008, // DETACHED_PROCESS
	}
	cmd.Start()
}

// runElevated re-runs the current exe with admin privileges and --install flag.
func runElevated() error {
	exe, _ := os.Executable()
	verb, _ := syscall.UTF16PtrFromString("runas")
	exeW, _ := syscall.UTF16PtrFromString(exe)
	args, _ := syscall.UTF16PtrFromString("--install")

	shellExecute := shell32.NewProc("ShellExecuteW")

	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exeW)),
		uintptr(unsafe.Pointer(args)),
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed with code %d", ret)
	}
	return nil
}

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

// createShortcut creates a .lnk Windows shortcut via PowerShell.
func createShortcut(target, lnkPath string) error {
	script := fmt.Sprintf(
		`$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.WorkingDirectory = '%s'; $s.Description = 'PrintDock'; $s.Save()`,
		lnkPath, target, filepath.Dir(target),
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

// showInstallDialog shows a Windows message box asking the user to install or update.
// Returns true if user clicks Yes.
func showInstallDialog() bool {
	msgBox := user32.NewProc("MessageBoxW")

	title, _ := syscall.UTF16PtrFromString("PrintDock — Installation")

	// Check if already installed (update scenario)
	destExe := filepath.Join(installDir, "printdock.exe")
	var msg string
	if _, err := os.Stat(destExe); err == nil {
		msg = "Une version de PrintDock est déjà installée.\n\nVoulez-vous la mettre à jour ?\n\n(L'instance en cours sera arrêtée automatiquement)"
	} else {
		msg = "PrintDock n'est pas encore installé.\n\nVoulez-vous l'installer dans :\nC:\\Program Files\\PrintDock\n\n(Un raccourci sera créé sur le Bureau et dans le menu Démarrer)"
	}

	text, _ := syscall.UTF16PtrFromString(msg)

	const mbYesNo = 0x00000004
	const mbIconQuestion = 0x00000020
	const idYes = 6

	ret, _, _ := msgBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbYesNo|mbIconQuestion)
	return ret == idYes
}

// showInstallSuccess shows a success message.
func showInstallSuccess() {
	msgBox := user32.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("PrintDock — Installation")
	text, _ := syscall.UTF16PtrFromString("PrintDock a été installé avec succès !\n\nL'application va maintenant démarrer.")
	const mbOk = 0x00000000
	const mbIconInfo = 0x00000040
	msgBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbOk|mbIconInfo)
}

// showInstallError shows an error message.
func showInstallError(errMsg string) {
	msgBox := user32.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("PrintDock — Erreur")
	text, _ := syscall.UTF16PtrFromString("Erreur lors de l'installation :\n" + errMsg)
	const mbOk = 0x00000000
	const mbIconError = 0x00000010
	msgBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbOk|mbIconError)
}
