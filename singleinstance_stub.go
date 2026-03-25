//go:build !windows

package main

// acquireSingleInstance always returns true on non-Windows (no mutex).
func acquireSingleInstance() bool { return true }

// isShowWindowMessage always returns false on non-Windows.
func isShowWindowMessage(umsg uint32) bool { return false }
