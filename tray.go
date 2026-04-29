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
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procMonitorFromPoint     = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfo       = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	initOnce                 sync.Once
	appHwnd                  uintptr
)

type winRECT struct {
	Left, Top, Right, Bottom int32
}

type winPOINT struct {
	X, Y int32
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor winRECT
	RcWork    winRECT
	DwFlags   uint32
}

const (
	wsExToolWindow       = 0x00000080
	wsExAppWindow        = 0x00040000
	wsThickFrame         = 0x00040000
	gwlStyleVal          = ^uintptr(15) // GWL_STYLE = -16
	gwlExStyleVal        = ^uintptr(19) // GWL_EXSTYLE = -20
	monitorDefaultNearest = 0x00000002
)

func findAppHwnd() uintptr {
	if appHwnd != 0 {
		return appHwnd
	}
	titlePtr, _ := syscall.UTF16PtrFromString(appTitle)
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
	systray.SetTooltip("밥알리미 v" + appVersion)

	systray.SetOnClick(func(menu systray.IMenu) {
		a.toggleWindow()
	})

	mShow := systray.AddMenuItem("앱 표시", "앱 표시")
	mSettings := systray.AddMenuItem("설정", "설정")
	mUpdate := systray.AddMenuItem("업데이트 확인", "업데이트 확인")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("종료", "종료")

	mShow.Click(func() { a.showWindowNearTray() })
	mSettings.Click(func() {
		a.showWindowNearTray()
		wailsRuntime.EventsEmit(a.ctx, "show:settings")
	})
	mUpdate.Click(func() {
		a.showWindowNearTray()
		wailsRuntime.EventsEmit(a.ctx, "show:settings")
		wailsRuntime.EventsEmit(a.ctx, "check:update")
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
		a.saveWindowPosition()
		wailsRuntime.WindowHide(a.ctx)
		a.mu.Lock()
		a.windowVisible = false
		a.mu.Unlock()
	} else {
		a.showWindowNearTray()
	}
}

func getWorkAreaNearCursor() (left, top, right, bottom int) {
	var pt winPOINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	hMon, _, _ := procMonitorFromPoint.Call(uintptr(pt.X), uintptr(pt.Y), monitorDefaultNearest)
	if hMon != 0 {
		var mi monitorInfo
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		ret, _, _ := procGetMonitorInfo.Call(hMon, uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			return int(mi.RcWork.Left), int(mi.RcWork.Top), int(mi.RcWork.Right), int(mi.RcWork.Bottom)
		}
	}
	var rect winRECT
	procSystemParametersInfo.Call(0x0030, 0, uintptr(unsafe.Pointer(&rect)), 0)
	return int(rect.Left), int(rect.Top), int(rect.Right), int(rect.Bottom)
}

func (a *App) saveWindowPosition() {
	if a.config == nil {
		return
	}
	hwnd := findAppHwnd()
	if hwnd == 0 {
		return
	}
	var rect winRECT
	ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return
	}
	a.config.WindowX = int(rect.Left)
	a.config.WindowY = int(rect.Top)
	a.config.HasWindowPos = true
	a.saveConfig()
}

func (a *App) showWindowNearTray() {
	var x, y int

	if a.config != nil && a.config.HasWindowPos {
		x = a.config.WindowX
		y = a.config.WindowY
	} else {
		left, top, right, bottom := getWorkAreaNearCursor()
		x = right - 420
		y = bottom - 680
		if x < left {
			x = left
		}
		if y < top {
			y = top
		}
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
	a.saveWindowPosition()
	wailsRuntime.WindowHide(a.ctx)
	a.mu.Lock()
	a.windowVisible = false
	a.mu.Unlock()
}

func (a *App) ResetWindowPosition() {
	if a.config != nil {
		a.config.HasWindowPos = false
		a.config.WindowX = 0
		a.config.WindowY = 0
		a.saveConfig()
	}
	left, top, right, bottom := getWorkAreaNearCursor()
	x := right - 420
	y := bottom - 680
	if x < left {
		x = left
	}
	if y < top {
		y = top
	}
	hwnd := findAppHwnd()
	if hwnd != 0 {
		const swpNoSize = 0x0001
		const swpShowWindow = 0x0040
		procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpShowWindow)
	}
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
