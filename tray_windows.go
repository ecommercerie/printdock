//go:build windows

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	shell32              = syscall.NewLazyDLL("shell32.dll")
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	pShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
	pCreateWindowEx      = user32.NewProc("CreateWindowExW")
	pDefWindowProc       = user32.NewProc("DefWindowProcW")
	pRegisterClassEx     = user32.NewProc("RegisterClassExW")
	pGetMessage          = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessage     = user32.NewProc("DispatchMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pLoadImage           = user32.NewProc("LoadImageW")
	pDestroyIcon         = user32.NewProc("DestroyIcon")
	pCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	pAppendMenu          = user32.NewProc("AppendMenuW")
	pTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	pDestroyMenu         = user32.NewProc("DestroyMenu")
	pGetCursorPos        = user32.NewProc("GetCursorPos")
	pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	pGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
)

const (
	nimAdd          = 0x00000000
	nimModify       = 0x00000001
	nimDelete       = 0x00000002
	nifMessage      = 0x00000001
	nifIcon         = 0x00000002
	nifTip          = 0x00000004
	imageIcon       = 1
	lrLoadFromFile  = 0x00000010
	lrDefaultSize   = 0x00000040
	wmApp           = 0x8000
	wmTrayicon      = wmApp + 1
	wmLbuttonup     = 0x0202
	wmRbuttonup     = 0x0205
	wmLbuttondblclk = 0x0203
	wmCommand       = 0x0111
	wmDestroy       = 0x0002
	mfString        = 0x00000000
	mfSeparator     = 0x00000800
	tpmBottomalign  = 0x0020
	tpmLeftalign    = 0x0000
	idOpen          = 1001
	idQuit          = 1002
	csDbclks        = 0x0008
	idcArrow        = 32512
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSm     syscall.Handle
}

type point struct {
	x, y int32
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type notifyIconData struct {
	cbSize           uint32
	hwnd             syscall.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            syscall.Handle
	szTip            [128]uint16
}

var trayApp *App
var trayHwnd syscall.Handle
var trayIconHandle syscall.Handle

func trayWndProc(hwnd syscall.Handle, umsg uint32, wParam, lParam uintptr) uintptr {
	// Handle custom "show window" message from second instance
	if isShowWindowMessage(umsg) {
		if trayApp != nil {
			trayApp.ShowWindow()
		}
		return 0
	}

	switch umsg {
	case wmTrayicon:
		switch lParam {
		case wmLbuttondblclk:
			if trayApp != nil {
				trayApp.ShowWindow()
			}
		case wmRbuttonup:
			showTrayMenu(hwnd)
		}
		return 0
	case wmCommand:
		switch wParam {
		case idOpen:
			if trayApp != nil {
				trayApp.ShowWindow()
			}
		case idQuit:
			if trayApp != nil {
				trayApp.QuitApp()
			}
		}
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProc.Call(uintptr(hwnd), uintptr(umsg), wParam, lParam)
	return ret
}

func showTrayMenu(hwnd syscall.Handle) {
	hMenu, _, _ := pCreatePopupMenu.Call()
	openText, _ := syscall.UTF16PtrFromString("Ouvrir PrintDock")
	quitText, _ := syscall.UTF16PtrFromString("Quitter")
	pAppendMenu.Call(hMenu, mfString, idOpen, uintptr(unsafe.Pointer(openText)))
	pAppendMenu.Call(hMenu, mfSeparator, 0, 0)
	pAppendMenu.Call(hMenu, mfString, idQuit, uintptr(unsafe.Pointer(quitText)))

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWindow.Call(uintptr(hwnd))
	pTrackPopupMenu.Call(hMenu, tpmBottomalign|tpmLeftalign, uintptr(pt.x), uintptr(pt.y), 0, uintptr(hwnd), 0)
	pDestroyMenu.Call(hMenu)
}

func (a *App) initTray() {
	trayApp = a
	go func() {
		hInst, _, _ := pGetModuleHandle.Call(0)
		className, _ := syscall.UTF16PtrFromString("PrintDockTray")

		wc := wndClassEx{
			style:     csDbclks,
			wndProc:   syscall.NewCallback(trayWndProc),
			instance:  syscall.Handle(hInst),
			className: className,
		}
		wc.size = uint32(unsafe.Sizeof(wc))

		// Load default arrow cursor
		pLoadCursor := user32.NewProc("LoadCursorW")
		cursor, _, _ := pLoadCursor.Call(0, uintptr(idcArrow))
		wc.cursor = syscall.Handle(cursor)

		pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

		hwnd, _, _ := pCreateWindowEx.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(className)),
			0, 0, 0, 0, 0, 0, 0, hInst, 0,
		)
		trayHwnd = syscall.Handle(hwnd)

		// Create a small icon from resource or default
		hIcon := loadTrayIcon()

		nid := notifyIconData{
			hwnd:             trayHwnd,
			uID:              1,
			uFlags:           nifMessage | nifIcon | nifTip,
			uCallbackMessage: wmTrayicon,
			hIcon:            hIcon,
		}
		nid.cbSize = uint32(unsafe.Sizeof(nid))
		tip, _ := syscall.UTF16FromString("PrintDock")
		copy(nid.szTip[:], tip)

		pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))

		// Message loop
		var m msg
		for {
			ret, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if ret == 0 {
				break
			}
			pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}

		// Cleanup
		pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	}()
}

func (a *App) cleanupTray() {
	if trayHwnd != 0 {
		nid := notifyIconData{
			hwnd: trayHwnd,
			uID:  1,
		}
		nid.cbSize = uint32(unsafe.Sizeof(nid))
		pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	}

	if trayIconHandle != 0 {
		pDestroyIcon.Call(uintptr(trayIconHandle))
		trayIconHandle = 0
	}
}

func loadTrayIcon() syscall.Handle {
	iconDir := filepath.Join(os.TempDir(), "PrintDock")
	iconPath, err := prepareTrayIconFile(iconDir, trayIconBytes)
	if err == nil {
		iconPathPtr, pathErr := syscall.UTF16PtrFromString(iconPath)
		if pathErr == nil {
			icon, _, _ := pLoadImage.Call(
				0,
				uintptr(unsafe.Pointer(iconPathPtr)),
				uintptr(imageIcon),
				0,
				0,
				uintptr(lrLoadFromFile|lrDefaultSize),
			)
			if icon != 0 {
				trayIconHandle = syscall.Handle(icon)
				return trayIconHandle
			}
		}
	}

	pLoadIcon := user32.NewProc("LoadIconW")
	icon, _, _ := pLoadIcon.Call(0, uintptr(32512)) // IDI_APPLICATION
	return syscall.Handle(icon)
}

// QuitApp fully closes the application.
func (a *App) QuitApp() {
	a.cleanupTray()
	runtime.Quit(a.ctx)
}
