package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed icon.png
var iconPNG []byte

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	procFindWindow           = user32.NewProc("FindWindowW")
	procGetWindowLong        = user32.NewProc("GetWindowLongW")
	procSetWindowLong        = user32.NewProc("SetWindowLongW")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	initOnce                 sync.Once
	appHwnd                  uintptr
)

type winRECT struct {
	Left, Top, Right, Bottom int32
}

const (
	wsExToolWindow = 0x00000080
	wsExAppWindow  = 0x00040000
	wsThickFrame   = 0x00040000
	gwlStyleVal    = ^uintptr(15) // GWL_STYLE = -16
	gwlExStyleVal  = ^uintptr(19) // GWL_EXSTYLE = -20
)

func findAppHwnd() uintptr {
	if appHwnd != 0 {
		return appHwnd
	}
	titlePtr, _ := syscall.UTF16PtrFromString("CJ-BapAlimi")
	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd != 0 {
		appHwnd = hwnd
	}
	return hwnd
}

func initAppWindow() {
	hwnd := findAppHwnd()
	if hwnd == 0 {
		return
	}

	exStyle, _, _ := procGetWindowLong.Call(hwnd, gwlExStyleVal)
	exStyle = exStyle &^ wsExAppWindow
	exStyle = exStyle | wsExToolWindow
	procSetWindowLong.Call(hwnd, gwlExStyleVal, exStyle)

	style, _, _ := procGetWindowLong.Call(hwnd, gwlStyleVal)
	procSetWindowLong.Call(hwnd, gwlStyleVal, style&^wsThickFrame)
}

func (a *App) setupTray() {
	go systray.Run(a.onTrayReady, a.onTrayExit)
}

func (a *App) onTrayReady() {
	systray.SetIcon(createTrayIcon())
	systray.SetTooltip("CJ-밥알리미")

	systray.SetOnClick(func(menu systray.IMenu) {
		a.toggleWindow()
	})

	mShow := systray.AddMenuItem("앱 표시", "앱 표시")
	mSettings := systray.AddMenuItem("설정", "설정")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("종료", "종료")

	mShow.Click(func() { a.showWindowNearTray() })
	mSettings.Click(func() {
		a.showWindowNearTray()
		wailsRuntime.EventsEmit(a.ctx, "show:settings")
	})
	mQuit.Click(func() { a.QuitApp() })
}

func (a *App) QuitApp() {
	a.quitting = true
	systray.Quit()
	wailsRuntime.Quit(a.ctx)
}

func (a *App) onTrayExit() {}

func (a *App) toggleWindow() {
	a.mu.Lock()
	if time.Since(a.lastToggle) < 300*time.Millisecond {
		a.mu.Unlock()
		return
	}
	a.lastToggle = time.Now()
	visible := a.windowVisible
	a.mu.Unlock()

	if visible {
		wailsRuntime.WindowHide(a.ctx)
		a.mu.Lock()
		a.windowVisible = false
		a.mu.Unlock()
	} else {
		a.showWindowNearTray()
	}
}

func getPrimaryWorkArea() (left, top, right, bottom int) {
	var rect winRECT
	procSystemParametersInfo.Call(0x0030, 0, uintptr(unsafe.Pointer(&rect)), 0)
	return int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)
}

func (a *App) showWindowNearTray() {
	left, _, right, bottom := getPrimaryWorkArea()
	x := right - 420
	y := bottom - 680
	if x < left {
		x = left
	}
	if y < 0 {
		y = 0
	}

	wailsRuntime.WindowShow(a.ctx)
	initOnce.Do(initAppWindow)

	hwnd := findAppHwnd()
	if hwnd != 0 {
		const swpNoSize = 0x0001
		const swpShowWindow = 0x0040
		procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpShowWindow)
	}

	wailsRuntime.EventsEmit(a.ctx, "window:showing")
	a.mu.Lock()
	a.windowVisible = true
	a.mu.Unlock()
}

func (a *App) SetAlwaysOnTop(onTop bool) {
	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, onTop)
}

func (a *App) HideWindow() {
	wailsRuntime.WindowHide(a.ctx)
	a.mu.Lock()
	a.windowVisible = false
	a.mu.Unlock()
}

func createTrayIcon() []byte {
	src, _, err := image.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return iconPNG
	}

	size := 32
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, dst)
	pngData := pngBuf.Bytes()

	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	icoDir := []byte{
		byte(size), byte(size), 0, 0,
		1, 0,
		32, 0,
	}
	dataLen := uint32(len(pngData))
	icoDir = append(icoDir, byte(dataLen), byte(dataLen>>8), byte(dataLen>>16), byte(dataLen>>24))
	icoDir = append(icoDir, 22, 0, 0, 0)
	ico.Write(icoDir)
	ico.Write(pngData)

	return ico.Bytes()
}
