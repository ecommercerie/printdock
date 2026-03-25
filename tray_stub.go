//go:build !windows

package main

import "github.com/wailsapp/wails/v2/pkg/runtime"

// initTray is a no-op on non-Windows platforms.
func (a *App) initTray() {}

// cleanupTray is a no-op on non-Windows platforms.
func (a *App) cleanupTray() {}

// QuitApp fully closes the application.
func (a *App) QuitApp() {
	runtime.Quit(a.ctx)
}
