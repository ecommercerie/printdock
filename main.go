package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	// Handle --install flag (running elevated to perform installation)
	if len(os.Args) > 1 && os.Args[1] == "--install" {
		if err := selfInstall(); err != nil {
			showInstallError(err.Error())
			os.Exit(1)
		}
		showInstallSuccess()
		relaunchInstalled()
		os.Exit(0)
	}

	// First launch: not installed yet — ask user
	if needsInstall() {
		if showInstallDialog() {
			if err := runElevated(); err != nil {
				showInstallError(err.Error())
			}
			os.Exit(0)
		}
		// User declined install — run from current location anyway
	}

	if !acquireSingleInstance() {
		// Another instance is running — it was signaled to show its window.
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "PrintDock",
		Width:            1024,
		Height:           700,
		HideWindowOnClose: true,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
