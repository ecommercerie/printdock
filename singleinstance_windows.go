//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const mutexName = "PrintDock_SingleInstance_Mutex"
const wmShowPrintDock = 0xC000 + 100 // Custom registered message

var (
	pCreateMutex      = kernel32.NewProc("CreateMutexW")
	pFindWindow       = user32.NewProc("FindWindowW")
	pPostMessage      = user32.NewProc("PostMessageW")
	pRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")
)

// acquireSingleInstance tries to acquire a named mutex.
// Returns true if this is the first instance. If another instance is running,
// it signals it to show its window and returns false.
func acquireSingleInstance() bool {
	name, _ := syscall.UTF16PtrFromString(mutexName)
	_, _, err := pCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))

	const errorAlreadyExists = 183
	if errno, ok := err.(syscall.Errno); ok && errno == errorAlreadyExists {
		// Another instance is running — find its tray window and signal it
		className, _ := syscall.UTF16PtrFromString("PrintDockTray")
		hwnd, _, _ := pFindWindow.Call(uintptr(unsafe.Pointer(className)), 0)
		if hwnd != 0 {
			// Send custom message to show the window
			msgName, _ := syscall.UTF16PtrFromString("PrintDock_ShowWindow")
			wmShow, _, _ := pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(msgName)))
			if wmShow != 0 {
				pPostMessage.Call(hwnd, wmShow, 0, 0)
			}
		}
		return false
	}
	return true
}

// registerShowMessage registers the custom window message and returns its ID.
func registerShowMessage() uint32 {
	msgName, _ := syscall.UTF16PtrFromString("PrintDock_ShowWindow")
	ret, _, _ := pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(msgName)))
	return uint32(ret)
}

// patchTrayWndProc is not needed — we'll handle the message in the existing tray wndproc.
// We need to store the registered message ID.
var showWindowMsg uint32

func init() {
	msgName, _ := syscall.UTF16PtrFromString("PrintDock_ShowWindow")
	ret, _, _ := pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(msgName)))
	showWindowMsg = uint32(ret)
}

func isShowWindowMessage(umsg uint32) bool {
	return showWindowMsg != 0 && umsg == showWindowMsg
}

// printSingleInstanceError prints a message (for debugging).
func printSingleInstanceError() {
	fmt.Println("PrintDock is already running. Bringing existing window to front.")
}
