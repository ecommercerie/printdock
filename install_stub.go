//go:build !windows

package main

import "os"

func getDataDir() string {
	home, _ := os.UserHomeDir()
	return home + "/.printdock"
}

func needsInstall() bool        { return false }
func selfInstall() error        { return nil }
func relaunchInstalled()        {}
func runElevated() error        { return nil }
func showInstallDialog() bool   { return false }
func showInstallSuccess()       {}
func showInstallError(e string) {}
