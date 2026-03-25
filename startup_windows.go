//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	registryKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	registryAppName = "PrintDock"
)

var (
	advapi32         = syscall.NewLazyDLL("advapi32.dll")
	procRegSetValueExW = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW = advapi32.NewProc("RegDeleteValueW")
)

// IsStartupEnabled checks if PrintDock is registered in Windows startup.
func (a *App) IsStartupEnabled() bool {
	var hKey syscall.Handle
	keyPath, _ := syscall.UTF16PtrFromString(registryKeyPath)
	err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, keyPath, 0, syscall.KEY_READ, &hKey)
	if err != nil {
		return false
	}
	defer syscall.RegCloseKey(hKey)

	valueName, _ := syscall.UTF16PtrFromString(registryAppName)
	var dataType uint32
	var dataSize uint32
	err = syscall.RegQueryValueEx(hKey, valueName, nil, &dataType, nil, &dataSize)
	return err == nil && dataSize > 0
}

// SetStartupEnabled enables or disables PrintDock in Windows startup.
func (a *App) SetStartupEnabled(enabled bool) error {
	var hKey syscall.Handle
	keyPath, _ := syscall.UTF16PtrFromString(registryKeyPath)
	err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, keyPath, 0, syscall.KEY_SET_VALUE, &hKey)
	if err != nil {
		return fmt.Errorf("cannot open registry key: %w", err)
	}
	defer syscall.RegCloseKey(hKey)

	valueName, _ := syscall.UTF16PtrFromString(registryAppName)

	if !enabled {
		procRegDeleteValueW.Call(uintptr(hKey), uintptr(unsafe.Pointer(valueName)))
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	valueData, _ := syscall.UTF16FromString(exePath)
	dataSize := uint32(len(valueData) * 2)
	ret, _, callErr := procRegSetValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valueName)),
		0,
		uintptr(syscall.REG_SZ),
		uintptr(unsafe.Pointer(&valueData[0])),
		uintptr(dataSize),
	)
	if ret != 0 {
		return fmt.Errorf("RegSetValueExW failed: %w", callErr)
	}
	return nil
}
