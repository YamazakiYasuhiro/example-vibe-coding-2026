//go:build windows

package winui

import (
	"fmt"
	"image"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/axsh/tokotachi/features/clock/internal/face"
	"golang.org/x/sys/windows"
)

// Options configures the desktop clock window.
type Options struct {
	Size      int
	QuitAfter time.Duration
}

const (
	className      = "TokotachiClockFace"
	trayUID        = 1
	wmApp          = 0x8000
	wmTrayIcon     = wmApp + 1
	timerTickID    = 1
	timerQuitID    = 2
	cmdExit        = 1001
	wsPopup        = 0x80000000
	wsExLayered    = 0x00080000
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	hwndTopmost    = ^uintptr(0) // HWND_TOPMOST == (HWND)-1
	swpNoActivate  = 0x0010
	swpShowWindow  = 0x0040
	ulwAlpha       = 0x00000002
	acSrcOver      = 0x00
	acSrcAlpha     = 0x01
	nimAdd         = 0x00000000
	nimDelete      = 0x00000002
	nifMessage     = 0x00000001
	nifIcon        = 0x00000002
	nifTip         = 0x00000004
	tpmRightButton = 0x0002
	mfString       = 0x00000000
	idiApplication = 32512
	smCxScreen     = 0
	smCyScreen     = 1
	wmDestroy      = 0x0002
	wmCommand      = 0x0111
	wmTimer        = 0x0113
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmMouseMove    = 0x0200
	wmRButtonUp    = 0x0205
	wmContextMenu  = 0x007B
	swShow         = 5
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procSetCapture           = user32.NewProc("SetCapture")
	procReleaseCapture       = user32.NewProc("ReleaseCapture")
	procUpdateLayeredWindow  = user32.NewProc("UpdateLayeredWindow")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procSetTimer             = user32.NewProc("SetTimer")
	procKillTimer            = user32.NewProc("KillTimer")
	procLoadIconW            = user32.NewProc("LoadIconW")
	procCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	procAppendMenuW          = user32.NewProc("AppendMenuW")
	procTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	procDestroyMenu          = user32.NewProc("DestroyMenu")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procShowWindow           = user32.NewProc("ShowWindow")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

type wndClassExW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type point struct {
	X, Y int32
}

type sizeStruct struct {
	CX, CY int32
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

type notifyIconDataW struct {
	Size            uint32
	Wnd             windows.HWND
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            windows.Handle
	Tip             [128]uint16
}

type msg struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type appState struct {
	mu        sync.Mutex
	hwnd      windows.HWND
	size      int
	dragging  bool
	dragDX    int32
	dragDY    int32
	trayAdded bool
}

var (
	state     appState
	wndProcCb = windows.NewCallback(wndProc)
)

// Run blocks until Exit, QuitAfter, or an error.
func Run(opts Options) error {
	size := opts.Size
	if size <= 0 {
		size = face.DefaultSize
	}
	state = appState{size: size}

	instance, _, callErr := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW: %v", callErr)
	}

	classPtr, err := windows.UTF16PtrFromString(className)
	if err != nil {
		return err
	}

	wc := wndClassExW{
		Size:      uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:   wndProcCb,
		Instance:  windows.Handle(instance),
		ClassName: classPtr,
	}
	atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW: %v", callErr)
	}

	screenW, _, _ := procGetSystemMetrics.Call(smCxScreen)
	screenH, _, _ := procGetSystemMetrics.Call(smCyScreen)
	x := int32((int(screenW) - size) / 2)
	y := int32((int(screenH) - size) / 2)

	hwnd, _, callErr := procCreateWindowExW.Call(
		uintptr(wsExLayered|wsExTopmost|wsExToolWindow),
		uintptr(unsafe.Pointer(classPtr)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("Clock"))),
		uintptr(wsPopup),
		uintptr(x),
		uintptr(y),
		uintptr(size),
		uintptr(size),
		0,
		0,
		instance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW: %v", callErr)
	}
	state.hwnd = windows.HWND(hwnd)
	log.Printf("DEBUG: clock window created hwnd=%v size=%d", hwnd, size)

	procShowWindow.Call(hwnd, swShow)
	if err := blit(windows.HWND(hwnd), size); err != nil {
		return err
	}

	if err := addTrayIcon(windows.HWND(hwnd)); err != nil {
		return err
	}
	state.trayAdded = true
	log.Printf("INFO: clock started (tray Exit to quit)")

	procSetTimer.Call(hwnd, timerTickID, 1000, 0)
	if opts.QuitAfter > 0 {
		ms := opts.QuitAfter.Milliseconds()
		if ms < 1 {
			ms = 1
		}
		procSetTimer.Call(hwnd, timerQuitID, uintptr(ms), 0)
		log.Printf("DEBUG: QuitAfter timer set to %v", opts.QuitAfter)
	}

	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	cleanup()
	log.Printf("INFO: clock stopped")
	return nil
}

func cleanup() {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.hwnd != 0 {
		procKillTimer.Call(uintptr(state.hwnd), timerTickID)
		procKillTimer.Call(uintptr(state.hwnd), timerQuitID)
	}
	if state.trayAdded && state.hwnd != 0 {
		_ = deleteTrayIcon(state.hwnd)
		state.trayAdded = false
	}
	if state.hwnd != 0 {
		procDestroyWindow.Call(uintptr(state.hwnd))
		state.hwnd = 0
	}
}

func wndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmLButtonDown:
		var pt point
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		var rc rect
		procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
		state.mu.Lock()
		state.dragging = true
		state.dragDX = pt.X - rc.Left
		state.dragDY = pt.Y - rc.Top
		state.mu.Unlock()
		procSetCapture.Call(uintptr(hwnd))
		return 0

	case wmMouseMove:
		state.mu.Lock()
		dragging := state.dragging
		dx, dy := state.dragDX, state.dragDY
		size := state.size
		state.mu.Unlock()
		if dragging {
			var pt point
			procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			procSetWindowPos.Call(
				uintptr(hwnd),
				hwndTopmost,
				uintptr(pt.X-dx),
				uintptr(pt.Y-dy),
				uintptr(size),
				uintptr(size),
				swpNoActivate,
			)
		}
		return 0

	case wmLButtonUp:
		state.mu.Lock()
		state.dragging = false
		state.mu.Unlock()
		procReleaseCapture.Call()
		return 0

	case wmTimer:
		switch wParam {
		case timerTickID:
			state.mu.Lock()
			sz := state.size
			state.mu.Unlock()
			if err := blit(hwnd, sz); err != nil {
				log.Printf("ERROR: blit failed: %v", err)
			}
		case timerQuitID:
			log.Printf("DEBUG: QuitAfter fired")
			procPostQuitMessage.Call(0)
		}
		return 0

	case wmTrayIcon:
		// lParam low word is the mouse message
		mouseMsg := uint32(lParam & 0xFFFF)
		if mouseMsg == wmRButtonUp || mouseMsg == wmContextMenu {
			showTrayMenu(hwnd)
		}
		return 0

	case wmCommand:
		if uint16(wParam&0xFFFF) == cmdExit {
			log.Printf("DEBUG: tray Exit selected")
			procPostQuitMessage.Call(0)
		}
		return 0

	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func showTrayMenu(hwnd windows.HWND) {
	menu, _, err := procCreatePopupMenu.Call()
	if menu == 0 {
		log.Printf("ERROR: CreatePopupMenu: %v", err)
		return
	}
	defer procDestroyMenu.Call(menu)

	label, _ := windows.UTF16PtrFromString("Exit")
	procAppendMenuW.Call(menu, mfString, cmdExit, uintptr(unsafe.Pointer(label)))

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(hwnd))
	procTrackPopupMenu.Call(menu, tpmRightButton, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
}

func addTrayIcon(hwnd windows.HWND) error {
	icon, _, err := procLoadIconW.Call(0, uintptr(idiApplication))
	if icon == 0 {
		return fmt.Errorf("LoadIconW: %v", err)
	}

	var nid notifyIconDataW
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hwnd
	nid.ID = trayUID
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.CallbackMessage = wmTrayIcon
	nid.Icon = windows.Handle(icon)
	copy(nid.Tip[:], windows.StringToUTF16("Clock"))

	r, _, callErr := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_ADD: %v", callErr)
	}
	return nil
}

func deleteTrayIcon(hwnd windows.HWND) error {
	var nid notifyIconDataW
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hwnd
	nid.ID = trayUID
	r, _, callErr := procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_DELETE: %v", callErr)
	}
	return nil
}

func blit(hwnd windows.HWND, size int) error {
	img := face.Render(size, time.Now())
	hdcScreen, _, err := procGetDC.Call(0)
	if hdcScreen == 0 {
		return fmt.Errorf("GetDC: %v", err)
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, err := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return fmt.Errorf("CreateCompatibleDC: %v", err)
	}
	defer procDeleteDC.Call(hdcMem)

	var bi bitmapInfo
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	bi.Header.Width = int32(size)
	bi.Header.Height = -int32(size) // top-down
	bi.Header.Planes = 1
	bi.Header.BitCount = 32
	bi.Header.Compression = 0 // BI_RGB

	var bits unsafe.Pointer
	hbmp, _, err := procCreateDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if hbmp == 0 || bits == nil {
		return fmt.Errorf("CreateDIBSection: %v", err)
	}
	defer procDeleteObject.Call(hbmp)

	copyPremultipliedBGRA((*[1 << 30]byte)(bits)[:size*size*4], img)

	prev, _, _ := procSelectObject.Call(hdcMem, hbmp)
	defer procSelectObject.Call(hdcMem, prev)

	var wndRect rect
	procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wndRect)))
	dst := point{X: wndRect.Left, Y: wndRect.Top}
	sz := sizeStruct{CX: int32(size), CY: int32(size)}
	src := point{X: 0, Y: 0}
	blend := blendFunction{
		BlendOp:             acSrcOver,
		BlendFlags:          0,
		SourceConstantAlpha: 255,
		AlphaFormat:         acSrcAlpha,
	}

	r, _, callErr := procUpdateLayeredWindow.Call(
		uintptr(hwnd),
		hdcScreen,
		uintptr(unsafe.Pointer(&dst)),
		uintptr(unsafe.Pointer(&sz)),
		hdcMem,
		uintptr(unsafe.Pointer(&src)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		ulwAlpha,
	)
	if r == 0 {
		return fmt.Errorf("UpdateLayeredWindow: %v", callErr)
	}
	return nil
}

func copyPremultipliedBGRA(dst []byte, img *image.RGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := img.PixOffset(x, y)
			r := img.Pix[off+0]
			g := img.Pix[off+1]
			bl := img.Pix[off+2]
			a := img.Pix[off+3]
			dst[i+0] = uint8(uint16(bl) * uint16(a) / 255)
			dst[i+1] = uint8(uint16(g) * uint16(a) / 255)
			dst[i+2] = uint8(uint16(r) * uint16(a) / 255)
			dst[i+3] = a
			i += 4
		}
	}
}
