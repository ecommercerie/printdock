//go:build !windows

package main

// IsStartupEnabled returns false on non-Windows platforms.
func (a *App) IsStartupEnabled() bool { return false }

// SetStartupEnabled is a no-op on non-Windows platforms.
func (a *App) SetStartupEnabled(enabled bool) error { return nil }
