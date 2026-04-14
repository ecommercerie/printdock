//go:build !windows

package main

import "printdock/internal/updater"

func applyUpdate(status updater.UpdateStatus) error {
	return nil
}
