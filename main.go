package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend
var assets embed.FS

func ensureSingleInstance() func() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")
	name, _ := syscall.UTF16PtrFromString("Global\\" + appTitle + "-SingleInstance")
	handle, _, err := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 || err == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		fmt.Fprintln(os.Stderr, "already running")
		os.Exit(0)
	}
	return func() { syscall.CloseHandle(syscall.Handle(handle)) }
}

func main() {
	release := ensureSingleInstance()
	defer release()
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     appTitle,
		Width:     420,
		Height:    680,
		MinWidth:  420,
		MinHeight: 680,
		MaxWidth:  420,
		MaxHeight:     680,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 28, G: 28, B: 30, A: 255},
		StartHidden:      true,
		Frameless:        true,
		HideWindowOnClose: true,
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.quitting {
				return false
			}
			app.HideWindow()
			return true
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
