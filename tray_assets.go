package main

import (
	_ "embed"
	"os"
	"path/filepath"
)

const trayIconFilename = "printdock-tray.ico"

//go:embed build/windows/tray.ico
var trayIconBytes []byte

func prepareTrayIconFile(dir string, iconData []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, trayIconFilename)
	if err := os.WriteFile(path, iconData, 0o644); err != nil {
		return "", err
	}

	return path, nil
}
