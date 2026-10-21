//go:build windows

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"

	"rsync-controller/core"
)

// Control IDs
const (
	// Navigation & Top Universal Search bar
	ID_BTN_BACK             = 91
	ID_BTN_FWD              = 92
	ID_BTN_UP               = 93
	ID_TXT_TARGET           = 101
	ID_BTN_GO               = 102
	ID_BTN_BROWSE           = 103
	ID_TXT_TOP_UNIV_SEARCH  = 104
	ID_BTN_TOP_UNIV_SEARCH  = 105
	ID_BTN_THEME_TOGGLE     = 106
	ID_BTN_RESCAN           = 107
	ID_BTN_SETTINGS         = 108
	ID_BTN_DRAWER_TOGGLE    = 109
	ID_BTN_JUMP             = 110

	// Operation Banner
	ID_LBL_OP_BANNER        = 120
	ID_BTN_OP_ACTION        = 121

	// Tabs
	ID_TAB_CONTROL          = 122

	// Tab 0: Local Explorer controls
	ID_TXT_FILTER           = 130
	ID_BTN_TOGGLE_VIEW      = 131
	ID_LISTVIEW             = 132
	ID_BTN_EXECUTE          = 133
	ID_BTN_DRYRUN           = 134
	ID_BTN_SELECTALL        = 135
	ID_BTN_CLEAR            = 136
	ID_TREEVIEW             = 137
	ID_BTN_TOGGLE_TREE      = 138
	ID_BTN_MODE             = 139

	// Tab 1: Universal Search controls
	ID_TXT_UNIV_SEARCH      = 201
	ID_CMB_UNIV_DRIVE       = 202
	ID_BTN_UNIV_SEARCH      = 203
	ID_LISTVIEW_UNIV        = 204
	ID_BTN_UNIV_STAGE       = 205
	ID_BTN_UNIV_OPENDIR     = 206
	ID_BTN_UNIV_CHECKALL    = 207
	ID_BTN_UNIV_CLEARALL    = 208

	// Tab 2: Remote Server Sync
	ID_CMB_REMOTE_SRV       = 301
	ID_TXT_REMOTE_DEST      = 302
	ID_BTN_REMOTE_PUSH      = 303
	ID_BTN_REMOTE_PULL      = 304

	// Tab 3: Server Health
	ID_BTN_PING_TEST        = 351
	ID_LBL_PING_RESULT      = 352

	// Tab 4: Transfer Log
	ID_EDIT_LOG             = 380
	ID_STATUSBAR            = 381
	ID_BTN_COPY_LOG         = 382
	ID_BTN_COPY_CMD         = 383

	// Tab 5: Settings controls
	ID_GRP_SHELL_CFG        = 401
	ID_CHK_ICONS            = 402
	ID_CHK_SEPARATORS       = 403
	ID_CHK_ACCELERATORS     = 404
	ID_CMB_POSITION         = 405
	ID_GRP_ENGINE_CFG       = 406
	ID_CMB_ENGINE           = 407
	ID_GRP_SERVICE_CFG      = 408
	ID_LBL_SERVICE          = 409
	ID_BTN_SERVICE          = 410
	ID_GRP_THEME_CFG        = 411
	ID_CMB_THEME            = 412
	ID_BTN_SAVESETTINGS     = 413
	ID_CMB_RADIUS           = 414
	ID_CMB_SHADOW           = 415
	ID_BTN_EDIT_THEME       = 416
	ID_BTN_RESET_THEME      = 417

	// Mini Right Summary Drawer controls
	ID_GRP_DRAWER           = 501
	ID_LBL_DRAWER_SUMMARY   = 502
	ID_LISTVIEW_STAGED      = 503
	ID_BTN_DRAWER_REMOVE    = 504
	ID_BTN_DRAWER_CLEAR     = 505
	ID_BTN_DRAWER_EXECUTE   = 506

	// Custom user messages
	WM_APP_SCAN_COMPLETE    = w32.WM_APP + 1
	WM_APP_UNIV_COMPLETE    = w32.WM_APP + 2
	WM_APP_RSYNC_LOG        = w32.WM_APP + 3
	WM_APP_PAYLOAD          = w32.WM_APP + 4
	WM_APP_TRIGGER_ACTION   = w32.WM_APP + 5
	WM_APP_TRANSFER_COMPLETE = w32.WM_APP + 6
)

type DisplayItem struct {
	Name     string
	FullPath string
	RelPath  string
	Size     int64
	IsDir    bool
	IsParent bool
}

// Global state
var (
	appCfg               *core.AppConfig
	engineMgr            *core.EngineManager
	currentScanIndex     *core.VolumeIndex
	stagedPath           string
	stagedMode           = "copy"
	stagedServer         = ""
	stagedDest           = ""
	initialAutoStage     = false
	initialPathsToStage  []string
	targetItemToSelect   string

	activeTab            = 0
	showAllRecursive     = false

	// Navigation history
	navHistory           []string
	navIndex             = -1

	// Subclassing
	origEditProc         uintptr

	// Window & Top Header controls
	hwndMain             w32.HWND
	hwndBtnBack          w32.HWND
	hwndBtnFwd           w32.HWND
	hwndBtnUp            w32.HWND
	hwndTxtTarget        w32.HWND
	hwndBtnGo            w32.HWND
	hwndBtnBrowse        w32.HWND
	hwndBtnJump          w32.HWND
	hwndTxtTopUnivSearch w32.HWND
	hwndBtnTopUnivSearch w32.HWND
	hwndBtnThemeToggle   w32.HWND
	hwndBtnRescan        w32.HWND
	hwndBtnSettings      w32.HWND
	hwndBtnDrawerToggle  w32.HWND

	// Operation Banner
	hwndLblOpBanner      w32.HWND
	hwndBtnOpAction      w32.HWND

	// Tabs
	hwndTabControl       w32.HWND

	// Tab 0: Local Explorer
	hwndLblFilter        w32.HWND
	hwndTxtFilter        w32.HWND
	hwndBtnToggleView    w32.HWND
	hwndListView         w32.HWND
	hwndBtnMode          w32.HWND
	hwndBtnExecute       w32.HWND
	hwndBtnDryRun        w32.HWND
	hwndBtnSelectAll     w32.HWND
	hwndBtnClear         w32.HWND

	// Tab 1: Universal Search
	hwndLblUniv          w32.HWND
	hwndTxtUnivSearch    w32.HWND
	hwndCmbUnivDrive     w32.HWND
	hwndBtnUnivSearch    w32.HWND
	hwndListViewUniv     w32.HWND
	hwndBtnUnivStage     w32.HWND
	hwndBtnUnivOpenDir   w32.HWND
	hwndBtnUnivCheckAll  w32.HWND
	hwndBtnUnivClearAll  w32.HWND

	// Tab 2: Remote Sync
	hwndGrpRemote        w32.HWND
	hwndLblRemoteSrv     w32.HWND
	hwndCmbRemoteSrv     w32.HWND
	hwndLblRemoteDest    w32.HWND
	hwndTxtRemoteDest    w32.HWND
	hwndBtnRemotePush    w32.HWND
	hwndBtnRemotePull    w32.HWND

	// Tab 3: Server Health
	hwndGrpHealth        w32.HWND
	hwndBtnPingTest      w32.HWND
	hwndLblPingResult    w32.HWND

	// Tab 4: Transfer Log
	hwndEditLog          w32.HWND
	hwndBtnCopyLog       w32.HWND
	hwndBtnCopyCmd       w32.HWND

	// Tab 5: Settings Tab Controls
	hwndGrpShell         w32.HWND
	hwndChkIcons         w32.HWND
	hwndChkSeps          w32.HWND
	hwndChkAccel         w32.HWND
	hwndLblPosition      w32.HWND
	hwndCmbPosition      w32.HWND

	hwndGrpEngine        w32.HWND
	hwndLblEngine        w32.HWND
	hwndCmbEngine        w32.HWND

	hwndGrpService       w32.HWND
	hwndLblService       w32.HWND
	hwndBtnService       w32.HWND

	hwndGrpTheme         w32.HWND
	hwndLblTheme         w32.HWND
	hwndCmbTheme         w32.HWND
	hwndLblRadius        w32.HWND
	hwndCmbRadius        w32.HWND
	hwndLblShadow        w32.HWND
	hwndCmbShadow        w32.HWND
	hwndBtnEditTheme     w32.HWND
	hwndBtnResetTheme    w32.HWND
	hwndLblThemeHint     w32.HWND

	hwndBtnSaveCfg       w32.HWND

	// Mini Right Summary Drawer
	hwndGrpDrawer        w32.HWND
	hwndLblDrawerSummary w32.HWND
	hwndListViewStaged   w32.HWND
	hwndBtnDrawerRemove  w32.HWND
	hwndBtnDrawerClear   w32.HWND
	hwndBtnDrawerExecute w32.HWND

	// Status Bar
	hwndStatusBar        w32.HWND

	// Concurrency & Items
	scanLock             sync.Mutex
	isScanning           bool
	currentListItems     []DisplayItem
	univResults          []*core.SearchResult

	// Persistent Staged Registry
	stagedRegistry       = make(map[string]DisplayItem)
	stagedTotalBytes     int64
	showStagedDrawer     = true
	stagedListPaths      []string
	isPopulatingListView bool

	payloadMutex         sync.Mutex
	queuedPayloads       []core.LaunchPayload
)

type ThemeMeta struct {
	ID       string
	Name     string
	BtnLabel string
}

var allThemeMetas = []ThemeMeta{
	{"win95-classic", "Windows 95 Classic (Retro Bevel)", "WIN95"},
	{"dark-obsidian", "Dark Obsidian (Cyber Night)", "OBSIDIAN"},
	{"y2k-candy", "Y2K Cyber Candy (Retro Gel)", "CANDY"},
	{"vaporwave-cyber", "Cyber Vaporwave (Neon 80s)", "VAPOR"},
	{"monochrome-mac", "Monochrome 1-Bit (Retro Classic)", "1-BIT"},
	{"aqua-win32", "Y2K Aqua (Win32 Teal & Cerulean)", "AQUA"},
	{"warm-handdrawn", "Warm Handdrawn Retro PC (Parchment & Amber)", "WARM"},
	{"pastel-grid", "Pastel Retro Grid (Pink & Lavender)", "PASTEL"},
	{"nostalgic-media", "Nostalgic Media Popup UI (Cobalt & Ivory)", "MEDIA"},
	{"y2k-pink", "Pink & Yellow Y2K Pop (Raspberry & Lemon)", "POP PINK"},
	{"retrowave-pc", "Retrowave Vintage PC (Hot Magenta & Neon)", "RETRO PC"},
	{"win95-figma", "Classic Win95/98 Teal Desktop", "WIN TEAL"},
	{"y2k-blue", "Y2K Classic Desktop Blue Folders", "Y2K BLUE"},
	{"pixel-pc", "Vintage Pixel PC Screens (Ruby Crimson)", "PIXEL PC"},
	{"music-chat", "Y2K Music Chat Desktop (Coral Peach)", "CHAT"},
	{"player-clock", "Y2K Desktop Music Player & Clock (Emerald)", "CLOCK"},
	{"dialog-green", "Y2K System Dialogs (Sky Blue & Soft Mint)", "DIALOG G"},
	{"dialog-sunset", "Y2K System Dialogs (Sunset Magenta & Rose)", "SUNSET"},
	{"custom", "Custom Theme (User Palette)", "CUSTOM"},
}

func cleanArgParam(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, `" --`); idx != -1 {
		s = s[:idx]
	}
	for len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		s = s[1 : len(s)-1]
		s = strings.TrimSpace(s)
	}
	s = strings.TrimRight(s, `\"'`)
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	if upper == "%1" || upper == "%V" || upper == "%W" || upper == "%L" || upper == `"%1"` || upper == `"%V"` || upper == `"%W"` || upper == `"%L"` {
		return ""
	}
	if len(s) == 2 && s[1] == ':' {
		s += `\`
	}
	return s
}

func queueIncomingPayload(p core.LaunchPayload) {
	payloadMutex.Lock()
	queuedPayloads = append(queuedPayloads, p)
	payloadMutex.Unlock()
	if hwndMain != 0 {
		w32.PostMessage(hwndMain, WM_APP_PAYLOAD, 0, 0)
	}
}

var (
	actionTriggerTimer *time.Timer
	actionTriggerLock  sync.Mutex
)

func scheduleActionTrigger() {
	actionTriggerLock.Lock()
	defer actionTriggerLock.Unlock()
	if actionTriggerTimer != nil {
		actionTriggerTimer.Stop()
	}
	actionTriggerTimer = time.AfterFunc(350*time.Millisecond, func() {
		if hwndMain != 0 {
			w32.PostMessage(hwndMain, WM_APP_TRIGGER_ACTION, 0, 0)
		}
	})
}

func main() {
	exePath, errExe := os.Executable()
	if errExe == nil {
		_ = os.Chdir(filepath.Dir(exePath))
	}

	regShell := false
	unregShell := false
	rawSrc := ""
	rawDest := ""
	bgPath := ""
	modeExplicit := false

	for _, arg := range os.Args[1:] {
		switch {
		case arg == "--register-shell":
			regShell = true
		case arg == "--unregister-shell":
			unregShell = true
		case strings.HasPrefix(arg, "--mode="):
			stagedMode = strings.ToLower(cleanArgParam(strings.TrimPrefix(arg, "--mode=")))
			modeExplicit = true
		case strings.HasPrefix(arg, "--server="):
			stagedServer = cleanArgParam(strings.TrimPrefix(arg, "--server="))
		case strings.HasPrefix(arg, "--src="):
			rawSrc = cleanArgParam(strings.TrimPrefix(arg, "--src="))
		case strings.HasPrefix(arg, "--dest="):
			rawDest = cleanArgParam(strings.TrimPrefix(arg, "--dest="))
		case strings.HasPrefix(arg, "--bg="):
			bgPath = cleanArgParam(strings.TrimPrefix(arg, "--bg="))
		default:
			if !strings.HasPrefix(arg, "-") && rawSrc == "" {
				rawSrc = cleanArgParam(arg)
			}
		}
	}

	configPath := filepath.Join(".", "config.json")
	var err error
	appCfg, err = core.LoadConfig(configPath)
	if err != nil {
		appCfg = core.DefaultConfig()
	}

	if regShell {
		exe, _ := os.Executable()
		if err := core.RegisterShellMenus(exe, appCfg); err != nil {
			fmt.Printf("Shell registration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Windows Explorer context menu registered successfully.")
		os.Exit(0)
	}

	if unregShell {
		_ = core.UnregisterShellMenus()
		fmt.Println("Windows Explorer context menu unregistered.")
		os.Exit(0)
	}

	isItemClick := (rawSrc != "" || rawDest != "")
	isBackgroundClick := (!isItemClick && bgPath != "")

	var autoStage bool
	var pathsToStage []string

	if isItemClick {
		if stagedMode == "pull" || stagedMode == "move_from" {
			target := rawDest
			if target == "" {
				target = rawSrc
			}
			stagedDest = target
			targetItemToSelect = target
			stagedPath = target
			autoStage = false
		} else {
			if rawSrc != "" {
				stagedPath = rawSrc
				targetItemToSelect = rawSrc
				if stagedMode == "copy" || stagedMode == "move_to" {
					autoStage = true
					pathsToStage = []string{rawSrc}
				}
			}
			if rawDest != "" {
				stagedDest = rawDest
				targetItemToSelect = rawDest
			}
		}
	} else if isBackgroundClick {
		switch stagedMode {
		case "copy":
			// User directive: no file selected on copy -> background is destination
			stagedDest = bgPath
			stagedPath = bgPath
			autoStage = false
		case "move_to":
			// User directive: no file selected on move -> background is source
			stagedPath = bgPath
			stagedDest = ""
			autoStage = false
		case "move_from", "pull":
			stagedDest = bgPath
			stagedPath = bgPath
			autoStage = false
		default:
			stagedPath = bgPath
			autoStage = false
		}
	}

	actionRequested := (modeExplicit || isItemClick || isBackgroundClick) && stagedMode != "open"
	if actionRequested && (stagedMode == "copy" || stagedMode == "move_to" || stagedMode == "pull" || stagedMode == "move_from") {
		scheduleActionTrigger()
	}

	initialAutoStage = autoStage
	initialPathsToStage = pathsToStage

	if stagedPath == "" {
		stagedPath = `E:\Backup\pgwiz\rsync`
	}
	stagedPath = filepath.Clean(stagedPath)

	engineMgr = core.NewEngineManager(appCfg.PreferredIndexer)

	// SINGLE INSTANCE GUARD: Prevent multiple window recursion
	cls, _ := windows.UTF16PtrFromString("RsyncControllerWin32Class")
	existingHwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0)
	if existingHwnd != 0 {
		payload := core.LaunchPayload{
			Mode:      stagedMode,
			Server:    stagedServer,
			Src:       stagedPath,
			Dest:      stagedDest,
			Paths:     pathsToStage,
			AutoStage: autoStage,
		}
		_ = core.ForwardPayloadDirect(payload)
		bringWindowToForeground(w32.HWND(existingHwnd))
		os.Exit(0)
	}

	coordinator, err := core.CheckSingleInstance()
	if err != nil || !coordinator.IsMaster {
		payload := core.LaunchPayload{
			Mode:      stagedMode,
			Server:    stagedServer,
			Src:       stagedPath,
			Dest:      stagedDest,
			Paths:     pathsToStage,
			AutoStage: autoStage,
		}
		if coordinator != nil {
			_ = coordinator.ForwardPayload(payload)
		} else {
			_ = core.ForwardPayloadDirect(payload)
		}
		activateExistingWindow()
		os.Exit(0)
	}

	if coordinator != nil {
		_ = coordinator.StartPipeServer()
		defer coordinator.Close()
		coordinator.OnLaunchPayload = func(p core.LaunchPayload) {
			queueIncomingPayload(p)
		}
		coordinator.OnPathAdded = func(path string) {
			cleanP := cleanArgParam(path)
			queueIncomingPayload(core.LaunchPayload{
				Mode:      "copy",
				Src:       cleanP,
				Paths:     []string{cleanP},
				AutoStage: true,
			})
		}
	}

	runNativeWin32App()
}

func runNativeWin32App() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	procOleInitialize.Call(0)
	defer procOleUninitialize.Call()

	var icc w32.INITCOMMONCONTROLSEX
	icc.ICC = 0x00000001 | 0x00000008 | 0x00000004 | 0x00000020
	w32.InitCommonControlsEx(&icc)

	hInstance := w32.GetModuleHandle("")
	className, _ := windows.UTF16PtrFromString("RsyncControllerWin32Class")
	windowTitle, _ := windows.UTF16PtrFromString("rsyn_copy Controller")

	baseDir, _ := os.Getwd()
	iconPath := filepath.Join(baseDir, "assets", "icon.ico")
	pIconPath, _ := windows.UTF16PtrFromString(iconPath)
	hIcon := w32.HICON(w32.LoadImage(
		0,
		pIconPath,
		w32.IMAGE_ICON,
		0, 0,
		w32.LR_LOADFROMFILE|w32.LR_DEFAULTSIZE,
	))

	var wc w32.WNDCLASSEX
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.Style = w32.CS_HREDRAW | w32.CS_VREDRAW
	wc.WndProc = syscall.NewCallback(wndProc)
	wc.Instance = hInstance
	wc.Cursor = w32.LoadCursor(0, w32.MakeIntResource(w32.IDC_ARROW))
	wc.Background = w32.HBRUSH(w32.COLOR_BTNFACE + 1)
	wc.ClassName = className
	wc.Icon = hIcon
	wc.IconSm = hIcon

	w32.RegisterClassEx(&wc)

	hwndMain = w32.CreateWindowEx(
		w32.WS_EX_APPWINDOW,
		className,
		windowTitle,
		w32.WS_OVERLAPPEDWINDOW|w32.WS_CLIPCHILDREN,
		w32.CW_USEDEFAULT, w32.CW_USEDEFAULT,
		1100, 720,
		0, 0, hInstance, nil,
	)

	if hwndMain == 0 {
		fmt.Println("FAIL: Could not create main Win32 window.")
		return
	}

	w32.ShowWindow(hwndMain, w32.SW_SHOW)
	w32.UpdateWindow(hwndMain)

	var msg w32.MSG
	for w32.GetMessage(&msg, 0, 0, 0) != 0 {
		if msg.Message == w32.WM_KEYDOWN {
			if handleGlobalKeydown(msg.WParam, msg.Hwnd) {
				continue
			}
		}
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
}

func wndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case w32.WM_CREATE:
		onCreate(hwnd)
		return 0

	case w32.WM_SIZE:
		width := int(w32.LOWORD(uint32(lParam)))
		height := int(w32.HIWORD(uint32(lParam)))
		onResize(width, height)
		return 0

	case w32.WM_COMMAND:
		controlID := int(w32.LOWORD(uint32(wParam)))
		code := int(w32.HIWORD(uint32(wParam)))
		onCommand(hwnd, controlID, code)
		return 0

	case w32.WM_CTLCOLORSTATIC:
		if currentTheme != nil {
			hdc := w32.HDC(wParam)
			hCtrl := w32.HWND(lParam)
			if hCtrl == hwndEditLog {
				w32.SetTextColor(hdc, w32.COLORREF(currentTheme.TextColor))
				w32.SetBkColor(hdc, w32.COLORREF(currentTheme.SurfaceBg))
				return uintptr(currentTheme.SurfaceBrush)
			}
			if hCtrl == hwndLblOpBanner {
				w32.SetTextColor(hdc, w32.COLORREF(currentTheme.AccentColor))
				w32.SetBkColor(hdc, w32.COLORREF(currentTheme.CanvasBg))
				return uintptr(currentTheme.CanvasBrush)
			}
			w32.SetTextColor(hdc, w32.COLORREF(currentTheme.TextColor))
			w32.SetBkColor(hdc, w32.COLORREF(currentTheme.CanvasBg))
			return uintptr(currentTheme.CanvasBrush)
		}

	case w32.WM_CTLCOLOREDIT, w32.WM_CTLCOLORLISTBOX:
		if currentTheme != nil {
			hdc := w32.HDC(wParam)
			w32.SetTextColor(hdc, w32.COLORREF(currentTheme.TextColor))
			w32.SetBkColor(hdc, w32.COLORREF(currentTheme.SurfaceBg))
			return uintptr(currentTheme.SurfaceBrush)
		}

	case w32.WM_CTLCOLORBTN:
		if currentTheme != nil {
			hdc := w32.HDC(wParam)
			w32.SetTextColor(hdc, w32.COLORREF(currentTheme.TextColor))
			w32.SetBkColor(hdc, w32.COLORREF(currentTheme.CanvasBg))
			return uintptr(currentTheme.CanvasBrush)
		}

	case w32.WM_NOTIFY:
		nmhdr := (*w32.NMHDR)(unsafe.Pointer(lParam))
		if nmhdr.Code == NM_CUSTOMDRAW || int32(nmhdr.Code) == -12 {
			if nmhdr.HwndFrom == hwndListView || nmhdr.HwndFrom == hwndListViewUniv || nmhdr.HwndFrom == hwndListViewStaged {
				res := handleListViewCustomDraw(nmhdr.HwndFrom, lParam, currentTheme)
				if res != 0 {
					return res
				}
			} else if nmhdr.HwndFrom == hwndTreeView {
				res := handleTreeViewCustomDraw(nmhdr.HwndFrom, lParam, currentTheme)
				if res != 0 {
					return res
				}
			}
		}

		if nmhdr.HwndFrom == hwndTabControl {
			if nmhdr.Code == TCN_SELCHANGE || int32(nmhdr.Code) == -551 {
				onTabChanged()
			}
		} else if nmhdr.HwndFrom == hwndListView {
			if nmhdr.Code == NM_DBLCLK || int32(nmhdr.Code) == -3 || int32(nmhdr.Code) == -4 {
				onListViewDoubleClick(lParam)
			} else if nmhdr.Code == LVN_ITEMCHANGED || int32(nmhdr.Code) == -101 {
				onListViewItemChanged(lParam)
			} else if nmhdr.Code == uint32(0xFFFFFFFB) || int32(nmhdr.Code) == -5 { // NM_RCLICK
				showListViewContextMenu(hwndListView)
			}
		} else if nmhdr.HwndFrom == hwndListViewUniv {
			if nmhdr.Code == NM_DBLCLK || int32(nmhdr.Code) == -3 {
				onUnivListViewDoubleClick(lParam)
			} else if nmhdr.Code == LVN_ITEMCHANGED || int32(nmhdr.Code) == -101 {
				onUnivListViewItemChanged(lParam)
			}
		} else if nmhdr.HwndFrom == hwndListViewStaged {
			if nmhdr.Code == NM_DBLCLK || int32(nmhdr.Code) == -3 {
				onStagedListViewDoubleClick(lParam)
			} else if nmhdr.Code == uint32(0xFFFFFFFB) || int32(nmhdr.Code) == -5 { // NM_RCLICK
				showStagedContextMenu(hwndListViewStaged)
			}
		} else if nmhdr.HwndFrom == hwndTreeView {
			if nmhdr.Code == TVN_ITEMEXPANDINGW || int32(nmhdr.Code) == -460 {
				nmtv := (*NMTREEVIEWW)(unsafe.Pointer(lParam))
				hItem := nmtv.ItemNew.HItem
				if path, ok := treeNodePaths[hItem]; ok {
					populateTreeChildren(hwndTreeView, hItem, path)
				}
			} else if nmhdr.Code == TVN_SELCHANGEDW || int32(nmhdr.Code) == -456 {
				nmtv := (*NMTREEVIEWW)(unsafe.Pointer(lParam))
				hItem := nmtv.ItemNew.HItem
				if path, ok := treeNodePaths[hItem]; ok {
					if path != stagedPath && !isSyncingTree {
						navigateTo(path, true)
					}
				}
			}
		}
		return 0

	case 0x002B: // WM_DRAWITEM
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis.CtlType == ODT_BUTTON {
			drawOwnerButton(dis, currentTheme)
			return 1
		} else if dis.HwndItem == hwndTabControl {
			drawOwnerTab(dis, currentTheme)
			return 1
		}
		return 0

	case w32.WM_ERASEBKGND:
		if currentTheme != nil {
			hdc := w32.HDC(wParam)
			rc := w32.GetClientRect(hwnd)
			if rc != nil {
				w32.FillRect(hdc, rc, currentTheme.CanvasBrush)
				paintWindowDecorations(hdc, rc, currentTheme)
				return 1
			}
		}

	case WM_APP_PAYLOAD:
		payloadMutex.Lock()
		items := queuedPayloads
		queuedPayloads = nil
		payloadMutex.Unlock()

		for _, p := range items {
			handleLaunchPayload(p)
		}
		w32.ShowWindow(hwndMain, w32.SW_RESTORE)
		w32.SetForegroundWindow(hwndMain)
		return 0

	case WM_APP_TRIGGER_ACTION:
		executeContextAction()
		return 0

	case w32.WM_DROPFILES:
		hDrop := w32.HDROP(wParam)
		countRet, _, _ := procDragQueryFileW.Call(uintptr(hDrop), 0xFFFFFFFF, 0, 0)
		count := int(countRet)
		stagedAny := false

		for i := 0; i < count; i++ {
			filePath := w32.DragQueryFile(hDrop, uint(i))
			if filePath == "" {
				continue
			}
			cleanPath := cleanArgParam(filePath)
			fi, err := os.Stat(cleanPath)
			if err == nil {
				addToStaged(DisplayItem{
					Name:     filepath.Base(cleanPath),
					FullPath: cleanPath,
					Size:     fi.Size(),
					IsDir:    fi.IsDir(),
				})
				stagedAny = true
			}
		}
		w32.DragFinish(hDrop)

		if stagedAny {
			if !showStagedDrawer {
				showStagedDrawer = true
				rc := w32.GetClientRect(hwnd)
				if rc != nil {
					onResize(int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
				}
			}
			updateOperationBanner()
			syncCurrentListViewChecks()
			w32.InvalidateRect(hwnd, nil, true)
			playSystemChime()
		}
		return 0

	case WM_APP_SCAN_COMPLETE:
		onScanComplete()
		return 0

	case WM_APP_UNIV_COMPLETE:
		onUnivSearchComplete()
		return 0

	case WM_APP_RSYNC_LOG:
		textPtr := (*string)(unsafe.Pointer(lParam))
		if textPtr != nil {
			appendLog(*textPtr)
		}
		return 0

	case WM_APP_TRANSFER_COMPLETE:
		clearStaged()
		navigateTo(stagedPath, false)
		updateOperationBanner()
		return 0

	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
		return 0
	}

	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

func editSubclassProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == w32.WM_KEYDOWN && wParam == 13 { // VK_RETURN
		switch hwnd {
		case hwndTxtTarget:
			path := getWindowText(hwndTxtTarget)
			if path != "" {
				navigateTo(path, true)
			}
			return 0
		case hwndTxtTopUnivSearch:
			runTopUniversalSearch()
			return 0
		case hwndTxtUnivSearch:
			performUniversalSearch()
			return 0
		case hwndTxtFilter:
			filterFiles(getWindowText(hwndTxtFilter))
			return 0
		}
	}
	return callWindowProc(origEditProc, hwnd, msg, wParam, lParam)
}

func subclassEdit(hwnd w32.HWND) {
	cb := syscall.NewCallback(editSubclassProc)
	prev := setWindowLongPtr(hwnd, -4 /* GWLP_WNDPROC */, cb)
	if origEditProc == 0 {
		origEditProc = prev
	}
}

func runTopUniversalSearch() {
	query := strings.TrimSpace(getWindowText(hwndTxtTopUnivSearch))
	if query == "" {
		return
	}

	// Switch directly to Tab 1: Universal Search
	w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 1, 0)
	onTabChanged()

	// Sync query to tab input field
	setWindowText(hwndTxtUnivSearch, query)

	// Execute drive search immediately
	performUniversalSearch()
}

func onCreate(hwnd w32.HWND) {
	hInstance := w32.GetModuleHandle("")
	w32.DragAcceptFiles(hwnd, true)

	// Initialize theme engine
	activeThemeID := appCfg.ActiveTheme
	if activeThemeID == "" {
		activeThemeID = "win95-classic"
	}
	currentTheme = initTheme(activeThemeID, appCfg.CornerRadius, appCfg.ShadowDepth)

	// ==================== TOP NAVIGATION & UNIVERSAL SEARCH BAR (Y: 10) ====================
	hwndBtnBack = createButton("<", w32.WS_VISIBLE|w32.WS_CHILD, 12, 10, 28, 26, hwnd, ID_BTN_BACK, hInstance)
	setControlFont(hwndBtnBack, currentTheme.FontBold)

	hwndBtnFwd = createButton(">", w32.WS_VISIBLE|w32.WS_CHILD, 42, 10, 28, 26, hwnd, ID_BTN_FWD, hInstance)
	setControlFont(hwndBtnFwd, currentTheme.FontBold)

	hwndBtnUp = createButton("^", w32.WS_VISIBLE|w32.WS_CHILD, 72, 10, 28, 26, hwnd, ID_BTN_UP, hInstance)
	setControlFont(hwndBtnUp, currentTheme.FontBold)

	hwndTxtTarget = createControl("EDIT", stagedPath, w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|w32.ES_AUTOHSCROLL, 105, 11, 260, 24, hwnd, ID_TXT_TARGET, hInstance)
	setControlFont(hwndTxtTarget, currentTheme.FontNormal)
	subclassEdit(hwndTxtTarget)

	hwndBtnGo = createButton("GO", w32.WS_VISIBLE|w32.WS_CHILD, 370, 10, 36, 26, hwnd, ID_BTN_GO, hInstance)
	setControlFont(hwndBtnGo, currentTheme.FontBold)

	hwndBtnBrowse = createButton("BROWSE...", w32.WS_VISIBLE|w32.WS_CHILD, 410, 10, 85, 26, hwnd, ID_BTN_BROWSE, hInstance)
	setControlFont(hwndBtnBrowse, currentTheme.FontBold)

	hwndBtnJump = createButton("JUMP ▾", w32.WS_VISIBLE|w32.WS_CHILD, 500, 10, 65, 26, hwnd, ID_BTN_JUMP, hInstance)
	setControlFont(hwndBtnJump, currentTheme.FontBold)

	hwndTxtTopUnivSearch = createControl("EDIT", "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|w32.ES_AUTOHSCROLL, 500, 11, 200, 24, hwnd, ID_TXT_TOP_UNIV_SEARCH, hInstance)
	setControlFont(hwndTxtTopUnivSearch, currentTheme.FontNormal)
	setEditCueBanner(hwndTxtTopUnivSearch, "Search entire drive...")
	subclassEdit(hwndTxtTopUnivSearch)

	hwndBtnTopUnivSearch = createButton("SEARCH", w32.WS_VISIBLE|w32.WS_CHILD, 705, 10, 70, 26, hwnd, ID_BTN_TOP_UNIV_SEARCH, hInstance)
	setControlFont(hwndBtnTopUnivSearch, currentTheme.FontBold)

	hwndBtnThemeToggle = createButton("THEME: WIN95 ▾", w32.WS_VISIBLE|w32.WS_CHILD, 780, 10, 120, 26, hwnd, ID_BTN_THEME_TOGGLE, hInstance)
	setControlFont(hwndBtnThemeToggle, currentTheme.FontBold)
	updateThemeButtonLabel()

	hwndBtnRescan = createButton("RESCAN", w32.WS_VISIBLE|w32.WS_CHILD, 870, 10, 75, 26, hwnd, ID_BTN_RESCAN, hInstance)
	setControlFont(hwndBtnRescan, currentTheme.FontBold)

	hwndBtnSettings = createButton("OPTIONS", w32.WS_VISIBLE|w32.WS_CHILD, 950, 10, 80, 26, hwnd, ID_BTN_SETTINGS, hInstance)
	setControlFont(hwndBtnSettings, currentTheme.FontBold)

	hwndBtnDrawerToggle = createButton("HIDE DRAWER", w32.WS_VISIBLE|w32.WS_CHILD, 1035, 10, 80, 26, hwnd, ID_BTN_DRAWER_TOGGLE, hInstance)
	setControlFont(hwndBtnDrawerToggle, currentTheme.FontBold)

	// ==================== OPERATION BANNER (Y: 40) ====================
	hwndLblOpBanner = createControl("STATIC", "MODE: LOCAL FILE MANAGER - Ready for file staging and sync.", w32.WS_VISIBLE|w32.WS_CHILD, 12, 42, 850, 20, hwnd, ID_LBL_OP_BANNER, hInstance)
	setControlFont(hwndLblOpBanner, currentTheme.FontBold)

	hwndBtnOpAction = createButton("SELECT DEST", w32.WS_VISIBLE|w32.WS_CHILD, 870, 39, 160, 26, hwnd, ID_BTN_OP_ACTION, hInstance)
	setControlFont(hwndBtnOpAction, currentTheme.FontBold)

	// ==================== TABS (Y: 68) ====================
	hwndTabControl = createControl("SysTabControl32", "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_CLIPSIBLINGS|TCS_OWNERDRAWFIXED, 12, 68, 1050, 30, hwnd, ID_TAB_CONTROL, hInstance)
	subclassTabControl(hwndTabControl)
	setControlFont(hwndTabControl, currentTheme.FontNormal)

	w32.SendMessage(hwndTabControl, TCM_DELETEALLITEMS, 0, 0)
	addTabItem(hwndTabControl, 0, "  Local Explorer  ")
	addTabItem(hwndTabControl, 1, "  Universal Search  ")
	addTabItem(hwndTabControl, 2, "  Remote Sync  ")
	addTabItem(hwndTabControl, 3, "  Server Health  ")
	addTabItem(hwndTabControl, 4, "  Transfer Log  ")
	addTabItem(hwndTabControl, 5, "  Settings  ")

	// ==================== TAB 0: LOCAL EXPLORER CONTROLS ====================
	hwndLblFilter = createControl("STATIC", "Filter Folder:", w32.WS_VISIBLE|w32.WS_CHILD, 12, 104, 80, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblFilter, currentTheme.FontBold)

	hwndTxtFilter = createControl("EDIT", "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|w32.ES_AUTOHSCROLL, 95, 102, 450, 24, hwnd, ID_TXT_FILTER, hInstance)
	setControlFont(hwndTxtFilter, currentTheme.FontNormal)
	setEditCueBanner(hwndTxtFilter, "Filter files in current folder...")
	subclassEdit(hwndTxtFilter)

	btnTreeText := "TREE: ON"
	if !showTreeView {
		btnTreeText = "TREE: OFF"
	}
	hwndBtnToggleTree = createButton(btnTreeText, w32.WS_VISIBLE|w32.WS_CHILD, 555, 101, 105, 26, hwnd, ID_BTN_TOGGLE_TREE, hInstance)
	setControlFont(hwndBtnToggleTree, currentTheme.FontBold)

	hwndBtnToggleView = createButton("FOLDER VIEW", w32.WS_VISIBLE|w32.WS_CHILD, 665, 101, 150, 26, hwnd, ID_BTN_TOGGLE_VIEW, hInstance)
	setControlFont(hwndBtnToggleView, currentTheme.FontBold)

	hwndTreeView = createControl(WC_TREEVIEW, "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|TVS_HASBUTTONS|TVS_HASLINES|TVS_LINESATROOT|TVS_SHOWSELALWAYS, 12, 134, 210, 420, hwnd, ID_TREEVIEW, hInstance)
	setControlFont(hwndTreeView, currentTheme.FontNormal)
	populateTreeRoots(hwndTreeView)

	hwndListView = createControl("SysListView32", "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|0x0001|0x0008|w32.WS_VSCROLL, 228, 134, 534, 420, hwnd, ID_LISTVIEW, hInstance)
	setListViewExtendedStyle(hwndListView, LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES|LVS_EX_DOUBLEBUFFER|LVS_EX_CHECKBOXES)
	setControlFont(hwndListView, currentTheme.FontNormal)

	addListViewColumn(hwndListView, 0, "File / Folder Name", 300, true)
	addListViewColumn(hwndListView, 1, "Full / Relative Path", 250, true)
	addListViewColumn(hwndListView, 2, "Size", 90, false)
	addListViewColumn(hwndListView, 3, "Type", 70, true)
	addListViewColumn(hwndListView, 4, "Status", 80, true)

	hwndBtnMode = createButton(getModeButtonText(), w32.WS_VISIBLE|w32.WS_CHILD, 12, 565, 135, 30, hwnd, ID_BTN_MODE, hInstance)
	setControlFont(hwndBtnMode, currentTheme.FontBold)

	hwndBtnExecute = createButton("EXECUTE RSYNC", w32.WS_VISIBLE|w32.WS_CHILD, 153, 565, 140, 30, hwnd, ID_BTN_EXECUTE, hInstance)
	setControlFont(hwndBtnExecute, currentTheme.FontBold)

	hwndBtnDryRun = createButton("DRY RUN TEST", w32.WS_VISIBLE|w32.WS_CHILD, 299, 565, 120, 30, hwnd, ID_BTN_DRYRUN, hInstance)
	setControlFont(hwndBtnDryRun, currentTheme.FontBold)

	hwndBtnSelectAll = createButton("CHECK ALL", w32.WS_VISIBLE|w32.WS_CHILD, 425, 565, 95, 30, hwnd, ID_BTN_SELECTALL, hInstance)
	setControlFont(hwndBtnSelectAll, currentTheme.FontNormal)

	hwndBtnClear = createButton("UNCHECK ALL", w32.WS_VISIBLE|w32.WS_CHILD, 526, 565, 105, 30, hwnd, ID_BTN_CLEAR, hInstance)
	setControlFont(hwndBtnClear, currentTheme.FontNormal)

	// ==================== TAB 1: UNIVERSAL SEARCH CONTROLS ====================
	hwndLblUniv = createControl("STATIC", "Universal Search (Entire PC / Drive):", w32.WS_CHILD, 12, 104, 230, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblUniv, currentTheme.FontBold)

	hwndTxtUnivSearch = createControl("EDIT", "", w32.WS_CHILD|w32.WS_BORDER|w32.ES_AUTOHSCROLL, 245, 102, 350, 24, hwnd, ID_TXT_UNIV_SEARCH, hInstance)
	setControlFont(hwndTxtUnivSearch, currentTheme.FontNormal)
	setEditCueBanner(hwndTxtUnivSearch, "Enter file name or pattern to search...")
	subclassEdit(hwndTxtUnivSearch)

	hwndCmbUnivDrive = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 602, 102, 130, 100, hwnd, ID_CMB_UNIV_DRIVE, hInstance)
	setControlFont(hwndCmbUnivDrive, currentTheme.FontNormal)
	addComboItem(hwndCmbUnivDrive, "Entire C: Drive")
	addComboItem(hwndCmbUnivDrive, "Entire E: Drive")
	addComboItem(hwndCmbUnivDrive, "Current Target")
	w32.SendMessage(hwndCmbUnivDrive, 0x014E, 0, 0)

	hwndBtnUnivSearch = createButton("SEARCH DRIVE", w32.WS_CHILD, 738, 101, 130, 26, hwnd, ID_BTN_UNIV_SEARCH, hInstance)
	setControlFont(hwndBtnUnivSearch, currentTheme.FontBold)

	hwndListViewUniv = createControl("SysListView32", "", w32.WS_CHILD|w32.WS_BORDER|0x0001|0x0008|w32.WS_VSCROLL, 12, 134, 750, 420, hwnd, ID_LISTVIEW_UNIV, hInstance)
	setListViewExtendedStyle(hwndListViewUniv, LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES|LVS_EX_DOUBLEBUFFER|LVS_EX_CHECKBOXES)
	setControlFont(hwndListViewUniv, currentTheme.FontNormal)

	addListViewColumn(hwndListViewUniv, 0, "File / Folder Name", 280, true)
	addListViewColumn(hwndListViewUniv, 1, "Full System Path", 380, true)
	addListViewColumn(hwndListViewUniv, 2, "Size", 95, false)
	addListViewColumn(hwndListViewUniv, 3, "Type", 75, true)

	hwndBtnUnivStage = createButton("STAGE CHECKED", w32.WS_CHILD, 12, 565, 220, 32, hwnd, ID_BTN_UNIV_STAGE, hInstance)
	setControlFont(hwndBtnUnivStage, currentTheme.FontBold)

	hwndBtnUnivOpenDir = createButton("OPEN IN EXPLORER", w32.WS_CHILD, 238, 565, 160, 32, hwnd, ID_BTN_UNIV_OPENDIR, hInstance)
	setControlFont(hwndBtnUnivOpenDir, currentTheme.FontNormal)

	hwndBtnUnivCheckAll = createButton("CHECK ALL", w32.WS_CHILD, 404, 565, 105, 32, hwnd, ID_BTN_UNIV_CHECKALL, hInstance)
	setControlFont(hwndBtnUnivCheckAll, currentTheme.FontNormal)

	hwndBtnUnivClearAll = createButton("UNCHECK ALL", w32.WS_CHILD, 515, 565, 115, 32, hwnd, ID_BTN_UNIV_CLEARALL, hInstance)
	setControlFont(hwndBtnUnivClearAll, currentTheme.FontNormal)

	// ==================== TAB 2: REMOTE SERVER SYNC ====================
	hwndGrpRemote = createControl("BUTTON", "Remote SSH Server Target", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 104, 750, 200, hwnd, 0, hInstance)
	setControlFont(hwndGrpRemote, currentTheme.FontBold)

	hwndLblRemoteSrv = createControl("STATIC", "Configured Server:", w32.WS_CHILD, 30, 138, 120, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblRemoteSrv, currentTheme.FontNormal)

	hwndCmbRemoteSrv = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 160, 134, 240, 120, hwnd, ID_CMB_REMOTE_SRV, hInstance)
	setControlFont(hwndCmbRemoteSrv, currentTheme.FontNormal)
	for _, srv := range appCfg.Servers {
		addComboItem(hwndCmbRemoteSrv, fmt.Sprintf("%s (%s:%d)", srv.Name, srv.Host, srv.Port))
	}
	w32.SendMessage(hwndCmbRemoteSrv, 0x014E, 0, 0)

	hwndLblRemoteDest = createControl("STATIC", "Remote Destination Path:", w32.WS_CHILD, 30, 178, 150, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblRemoteDest, currentTheme.FontNormal)

	defaultRemote := "/data/backup"
	if len(appCfg.Servers) > 0 && appCfg.Servers[0].RemotePath != "" {
		defaultRemote = appCfg.Servers[0].RemotePath
	}
	hwndTxtRemoteDest = createControl("EDIT", defaultRemote, w32.WS_CHILD|w32.WS_BORDER|w32.ES_AUTOHSCROLL, 190, 176, 400, 24, hwnd, ID_TXT_REMOTE_DEST, hInstance)
	setControlFont(hwndTxtRemoteDest, currentTheme.FontNormal)

	hwndBtnRemotePush = createButton("PUSH TO SERVER", w32.WS_CHILD, 30, 228, 250, 34, hwnd, ID_BTN_REMOTE_PUSH, hInstance)
	setControlFont(hwndBtnRemotePush, currentTheme.FontBold)

	hwndBtnRemotePull = createButton("PULL FROM SERVER", w32.WS_CHILD, 290, 228, 250, 34, hwnd, ID_BTN_REMOTE_PULL, hInstance)
	setControlFont(hwndBtnRemotePull, currentTheme.FontBold)

	// ==================== TAB 3: SERVER HEALTH ====================
	hwndGrpHealth = createControl("BUTTON", "Server Connectivity & Latency Ping", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 104, 750, 160, hwnd, 0, hInstance)
	setControlFont(hwndGrpHealth, currentTheme.FontBold)

	hwndBtnPingTest = createButton("PING SERVERS", w32.WS_CHILD, 30, 138, 180, 30, hwnd, ID_BTN_PING_TEST, hInstance)
	setControlFont(hwndBtnPingTest, currentTheme.FontBold)

	hwndLblPingResult = createControl("STATIC", "Click PING SERVERS to verify SSH connectivity.", w32.WS_CHILD, 230, 144, 500, 24, hwnd, ID_LBL_PING_RESULT, hInstance)
	setControlFont(hwndLblPingResult, currentTheme.FontNormal)

	// ==================== TAB 4: TRANSFER LOG CONTROLS ====================
	hwndEditLog = createControl("EDIT", "", w32.WS_CHILD|w32.WS_BORDER|w32.ES_MULTILINE|w32.ES_READONLY|w32.WS_VSCROLL|w32.WS_HSCROLL, 12, 104, 750, 450, hwnd, ID_EDIT_LOG, hInstance)
	setControlFont(hwndEditLog, currentTheme.FontNormal)

	hwndBtnCopyLog = createButton("COPY LOG", w32.WS_CHILD, 12, 565, 220, 32, hwnd, ID_BTN_COPY_LOG, hInstance)
	setControlFont(hwndBtnCopyLog, currentTheme.FontBold)

	hwndBtnCopyCmd = createButton("COPY RSYNC COMMAND", w32.WS_CHILD, 240, 565, 230, 32, hwnd, ID_BTN_COPY_CMD, hInstance)
	setControlFont(hwndBtnCopyCmd, currentTheme.FontBold)

	// ==================== TAB 5: DEDICATED SETTINGS & OPTIONS TAB ====================
	hwndGrpShell = createControl("BUTTON", "1. Windows Explorer Context Menu Customization", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 104, 750, 130, hwnd, ID_GRP_SHELL_CFG, hInstance)
	setControlFont(hwndGrpShell, currentTheme.FontBold)

	hwndChkIcons = createControl("BUTTON", "Show Context Menu Icons (rp badge)", w32.WS_CHILD|w32.BS_AUTOCHECKBOX, 30, 130, 260, 24, hwnd, ID_CHK_ICONS, hInstance)
	setControlFont(hwndChkIcons, currentTheme.FontNormal)
	if appCfg.Shell.ShowIcons {
		w32.SendMessage(hwndChkIcons, 0x00F1, 1, 0)
	}

	hwndChkSeps = createControl("BUTTON", "Use Menu Visual Dividers (Separators)", w32.WS_CHILD|w32.BS_AUTOCHECKBOX, 30, 160, 260, 24, hwnd, ID_CHK_SEPARATORS, hInstance)
	setControlFont(hwndChkSeps, currentTheme.FontNormal)
	if appCfg.Shell.UseSeparators {
		w32.SendMessage(hwndChkSeps, 0x00F1, 1, 0)
	}

	hwndChkAccel = createControl("BUTTON", "Use Keyboard Accelerators (&Copy, &Pull)", w32.WS_CHILD|w32.BS_AUTOCHECKBOX, 30, 190, 280, 24, hwnd, ID_CHK_ACCELERATORS, hInstance)
	setControlFont(hwndChkAccel, currentTheme.FontNormal)
	if appCfg.Shell.UseAccelerators {
		w32.SendMessage(hwndChkAccel, 0x00F1, 1, 0)
	}

	hwndLblPosition = createControl("STATIC", "Menu Position:", w32.WS_CHILD, 340, 132, 100, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblPosition, currentTheme.FontNormal)

	hwndCmbPosition = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 445, 128, 180, 100, hwnd, ID_CMB_POSITION, hInstance)
	setControlFont(hwndCmbPosition, currentTheme.FontNormal)
	addComboItem(hwndCmbPosition, "Default Alphabetical")
	addComboItem(hwndCmbPosition, "Pin to Top")
	addComboItem(hwndCmbPosition, "Pin to Bottom")
	w32.SendMessage(hwndCmbPosition, 0x014E, 0, 0)

	// Group 2: Engine
	hwndGrpEngine = createControl("BUTTON", "2. NTFS Disk Indexer Engine", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 240, 750, 75, hwnd, ID_GRP_ENGINE_CFG, hInstance)
	setControlFont(hwndGrpEngine, currentTheme.FontBold)

	hwndLblEngine = createControl("STATIC", "Preferred Engine:", w32.WS_CHILD, 30, 270, 120, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblEngine, currentTheme.FontNormal)

	hwndCmbEngine = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 160, 266, 250, 120, hwnd, ID_CMB_ENGINE, hInstance)
	setControlFont(hwndCmbEngine, currentTheme.FontNormal)
	addComboItem(hwndCmbEngine, "USN Journal (Instant NTFS Indexing)")
	addComboItem(hwndCmbEngine, "Directory Crawler (Safe Fallback)")
	w32.SendMessage(hwndCmbEngine, 0x014E, 0, 0)

	// Group 3: Service
	hwndGrpService = createControl("BUTTON", "3. Background Windows Service Daemon", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 322, 750, 75, hwnd, ID_GRP_SERVICE_CFG, hInstance)
	setControlFont(hwndGrpService, currentTheme.FontBold)

	hwndLblService = createControl("STATIC", "Service Status: CHECKING...", w32.WS_CHILD, 30, 352, 280, 20, hwnd, ID_LBL_SERVICE, hInstance)
	setControlFont(hwndLblService, currentTheme.FontNormal)

	hwndBtnService = createButton("RESTART / INSTALL SERVICE", w32.WS_CHILD, 320, 347, 210, 28, hwnd, ID_BTN_SERVICE, hInstance)
	setControlFont(hwndBtnService, currentTheme.FontBold)

	// Group 4: Themes & Visual Customizer
	hwndGrpTheme = createControl("BUTTON", "4. Visual Theme, Palette & Shadows", w32.WS_CHILD|w32.BS_GROUPBOX, 12, 404, 750, 115, hwnd, ID_GRP_THEME_CFG, hInstance)
	setControlFont(hwndGrpTheme, currentTheme.FontBold)

	hwndLblTheme = createControl("STATIC", "Active Theme:", w32.WS_CHILD, 30, 434, 90, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblTheme, currentTheme.FontNormal)

	hwndCmbTheme = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 125, 430, 190, 140, hwnd, ID_CMB_THEME, hInstance)
	setControlFont(hwndCmbTheme, currentTheme.FontNormal)
	for _, tm := range allThemeMetas {
		addComboItem(hwndCmbTheme, tm.Name)
	}
	setComboThemeIndex(hwndCmbTheme, appCfg.ActiveTheme)

	hwndLblRadius = createControl("STATIC", "Radius:", w32.WS_CHILD, 325, 434, 55, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblRadius, currentTheme.FontNormal)

	hwndCmbRadius = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 385, 430, 160, 120, hwnd, ID_CMB_RADIUS, hInstance)
	setControlFont(hwndCmbRadius, currentTheme.FontNormal)
	addComboItem(hwndCmbRadius, "0px (Sharp Retro 80s/90s)")
	addComboItem(hwndCmbRadius, "4px (Subtle Modern)")
	addComboItem(hwndCmbRadius, "8px (Rounded Y2K)")
	addComboItem(hwndCmbRadius, "14px (Full Pill Aqua)")
	switch appCfg.CornerRadius {
	case 0:
		w32.SendMessage(hwndCmbRadius, 0x014E, 0, 0)
	case 4:
		w32.SendMessage(hwndCmbRadius, 0x014E, 1, 0)
	case 14:
		w32.SendMessage(hwndCmbRadius, 0x014E, 3, 0)
	default:
		w32.SendMessage(hwndCmbRadius, 0x014E, 2, 0)
	}

	hwndLblShadow = createControl("STATIC", "Shadow:", w32.WS_CHILD, 555, 434, 55, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblShadow, currentTheme.FontNormal)

	hwndCmbShadow = createControl("COMBOBOX", "", w32.WS_CHILD|w32.CBS_DROPDOWNLIST|w32.WS_VSCROLL, 615, 430, 140, 120, hwnd, ID_CMB_SHADOW, hInstance)
	setControlFont(hwndCmbShadow, currentTheme.FontNormal)
	addComboItem(hwndCmbShadow, "0px (Flat)")
	addComboItem(hwndCmbShadow, "1px (Subtle)")
	addComboItem(hwndCmbShadow, "2px (Standard)")
	addComboItem(hwndCmbShadow, "4px (Bold 3D)")
	addComboItem(hwndCmbShadow, "6px (Deep Comic)")
	switch appCfg.ShadowDepth {
	case 0:
		w32.SendMessage(hwndCmbShadow, 0x014E, 0, 0)
	case 1:
		w32.SendMessage(hwndCmbShadow, 0x014E, 1, 0)
	case 4:
		w32.SendMessage(hwndCmbShadow, 0x014E, 3, 0)
	case 6:
		w32.SendMessage(hwndCmbShadow, 0x014E, 4, 0)
	default:
		w32.SendMessage(hwndCmbShadow, 0x014E, 2, 0)
	}

	hwndBtnEditTheme = createButton("CUSTOMIZE PALETTE...", w32.WS_CHILD, 30, 470, 185, 30, hwnd, ID_BTN_EDIT_THEME, hInstance)
	setControlFont(hwndBtnEditTheme, currentTheme.FontBold)

	hwndBtnResetTheme = createButton("RESET PALETTE", w32.WS_CHILD, 222, 470, 125, 30, hwnd, ID_BTN_RESET_THEME, hInstance)
	setControlFont(hwndBtnResetTheme, currentTheme.FontNormal)

	hwndLblThemeHint = createControl("STATIC", "Custom palette: Canvas, Window, Button, Accent, Border & Depth Shadows.", w32.WS_CHILD, 355, 476, 395, 20, hwnd, 0, hInstance)
	setControlFont(hwndLblThemeHint, currentTheme.FontNormal)

	hwndBtnSaveCfg = createButton("APPLY & SAVE SETTINGS", w32.WS_CHILD, 12, 530, 280, 36, hwnd, ID_BTN_SAVESETTINGS, hInstance)
	setControlFont(hwndBtnSaveCfg, currentTheme.FontBold)

	// ==================== MINI RIGHT SUMMARY DRAWER (Persistent Across Tabs) ====================
	hwndGrpDrawer = createControl("BUTTON", "Staged Selection Summary", w32.WS_VISIBLE|w32.WS_CHILD|w32.BS_GROUPBOX, 780, 100, 290, 520, hwnd, ID_GRP_DRAWER, hInstance)
	setControlFont(hwndGrpDrawer, currentTheme.FontBold)

	hwndLblDrawerSummary = createControl("STATIC", "Staged: 0 items (0 B)", w32.WS_VISIBLE|w32.WS_CHILD, 792, 126, 266, 20, hwnd, ID_LBL_DRAWER_SUMMARY, hInstance)
	setControlFont(hwndLblDrawerSummary, currentTheme.FontBold)

	hwndListViewStaged = createControl("SysListView32", "", w32.WS_VISIBLE|w32.WS_CHILD|w32.WS_BORDER|0x0001|0x0008|w32.WS_VSCROLL, 792, 150, 266, 390, hwnd, ID_LISTVIEW_STAGED, hInstance)
	setListViewExtendedStyle(hwndListViewStaged, LVS_EX_FULLROWSELECT|LVS_EX_GRIDLINES|LVS_EX_DOUBLEBUFFER)
	setControlFont(hwndListViewStaged, currentTheme.FontNormal)

	addListViewColumn(hwndListViewStaged, 0, "Item Name", 125, true)
	addListViewColumn(hwndListViewStaged, 1, "Size", 60, false)
	addListViewColumn(hwndListViewStaged, 2, "Directory", 75, true)

	hwndBtnDrawerRemove = createButton("UNSTAGE", w32.WS_VISIBLE|w32.WS_CHILD, 792, 550, 80, 28, hwnd, ID_BTN_DRAWER_REMOVE, hInstance)
	setControlFont(hwndBtnDrawerRemove, currentTheme.FontNormal)

	hwndBtnDrawerClear = createButton("CLEAR", w32.WS_VISIBLE|w32.WS_CHILD, 876, 550, 75, 28, hwnd, ID_BTN_DRAWER_CLEAR, hInstance)
	setControlFont(hwndBtnDrawerClear, currentTheme.FontNormal)

	hwndBtnDrawerExecute = createButton("SYNC", w32.WS_VISIBLE|w32.WS_CHILD, 955, 550, 103, 28, hwnd, ID_BTN_DRAWER_EXECUTE, hInstance)
	setControlFont(hwndBtnDrawerExecute, currentTheme.FontBold)

	// Status Bar
	hwndStatusBar = createControl("msctls_statusbar32", "", w32.WS_VISIBLE|w32.WS_CHILD, 0, 0, 0, 0, hwnd, ID_STATUSBAR, hInstance)
	parts := []int32{240, 390, 560, 740, -1}
	setStatusBarParts(hwndStatusBar, parts)
	setStatusBarText(hwndStatusBar, 0, getDriveSpaceSummary(stagedPath))
	setStatusBarText(hwndStatusBar, 1, "ENGINE: USN SCANNER")
	setStatusBarText(hwndStatusBar, 2, "SERVICE: CHECKING")
	setStatusBarText(hwndStatusBar, 3, "Staged: 0 (0 B)")
	setStatusBarText(hwndStatusBar, 4, "READY")

	// Apply theme colors and attributes
	applyThemeToWindow(hwnd, currentTheme)

	// Process any payloads that arrived while the window was being initialized
	w32.PostMessage(hwnd, WM_APP_PAYLOAD, 0, 0)

	// Handle initial command line / context menu payload
	handleLaunchPayload(core.LaunchPayload{
		Mode:      stagedMode,
		Server:    stagedServer,
		Src:       stagedPath,
		Dest:      stagedDest,
		Paths:     initialPathsToStage,
		AutoStage: initialAutoStage,
	})

	go checkServiceStateAsync()
}

func onResize(width, height int) {
	if width < 720 || height < 400 {
		return
	}

	// Dynamic Top Navigation & Search Bar (Y: 10, H: 26)
	// Left fixed: 12 + 28 + 2 + 28 + 2 + 28 + 5 = 105px (Back, Fwd, Up)
	// Right buttons: Go(36)+Browse(80)+Search(65)+Theme(80)+Rescan(70)+Options(75)+Drawer(75) + margins (~45) = ~526px
	remW := width - 745
	if remW < 180 {
		remW = 180
	}
	addrW := (remW * 55) / 100
	if addrW < 110 {
		addrW = 110
	}
	univW := remW - addrW
	if univW < 80 {
		univW = 80
	}

	w32.MoveWindow(hwndBtnBack, 12, 10, 28, 26, true)
	w32.MoveWindow(hwndBtnFwd, 42, 10, 28, 26, true)
	w32.MoveWindow(hwndBtnUp, 72, 10, 28, 26, true)

	addrX := 105
	w32.MoveWindow(hwndTxtTarget, addrX, 11, addrW, 24, true)

	goX := addrX + addrW + 4
	w32.MoveWindow(hwndBtnGo, goX, 10, 36, 26, true)

	browseX := goX + 36 + 4
	w32.MoveWindow(hwndBtnBrowse, browseX, 10, 80, 26, true)

	jumpX := browseX + 80 + 4
	w32.MoveWindow(hwndBtnJump, jumpX, 10, 65, 26, true)

	univX := jumpX + 65 + 6
	w32.MoveWindow(hwndTxtTopUnivSearch, univX, 11, univW, 24, true)

	searchX := univX + univW + 4
	w32.MoveWindow(hwndBtnTopUnivSearch, searchX, 10, 65, 26, true)

	themeX := searchX + 65 + 4
	w32.MoveWindow(hwndBtnThemeToggle, themeX, 10, 120, 26, true)

	rescanX := themeX + 120 + 4
	w32.MoveWindow(hwndBtnRescan, rescanX, 10, 70, 26, true)

	optX := rescanX + 70 + 4
	w32.MoveWindow(hwndBtnSettings, optX, 10, 75, 26, true)

	drawerToggleX := optX + 75 + 4
	w32.MoveWindow(hwndBtnDrawerToggle, drawerToggleX, 10, 75, 26, true)

	// Operation Banner (Y: 40, H: 24)
	bannerActionW := 160
	w32.MoveWindow(hwndLblOpBanner, 12, 42, width-24-bannerActionW-8, 20, true)
	w32.MoveWindow(hwndBtnOpAction, width-12-bannerActionW, 39, bannerActionW, 26, true)

	// Tab Control (Y: 68, H: 28)
	w32.MoveWindow(hwndTabControl, 12, 68, width-24, 28, true)

	contentTop := 102
	bottomBarY := height - 68
	listHeight := bottomBarY - (contentTop + 32) - 8

	// Layout with or without Right Summary Drawer
	drawerW := 290
	leftW := width - 24
	if showStagedDrawer {
		leftW = width - 24 - drawerW - 10
		if leftW < 400 {
			leftW = 400
		}
		drawerX := width - 12 - drawerW
		drawerListH := (listHeight + 32 + 36) - 24 - 24 - 36 - 8

		w32.ShowWindow(hwndGrpDrawer, w32.SW_SHOW)
		w32.ShowWindow(hwndLblDrawerSummary, w32.SW_SHOW)
		w32.ShowWindow(hwndListViewStaged, w32.SW_SHOW)
		w32.ShowWindow(hwndBtnDrawerRemove, w32.SW_SHOW)
		w32.ShowWindow(hwndBtnDrawerClear, w32.SW_SHOW)
		w32.ShowWindow(hwndBtnDrawerExecute, w32.SW_SHOW)

		w32.MoveWindow(hwndGrpDrawer, drawerX, contentTop, drawerW, listHeight+32+36, true)
		w32.MoveWindow(hwndLblDrawerSummary, drawerX+10, contentTop+24, drawerW-20, 20, true)
		w32.MoveWindow(hwndListViewStaged, drawerX+10, contentTop+48, drawerW-20, drawerListH, true)

		btnY := contentTop + 48 + drawerListH + 6
		w32.MoveWindow(hwndBtnDrawerRemove, drawerX+10, btnY, 80, 28, true)
		w32.MoveWindow(hwndBtnDrawerClear, drawerX+94, btnY, 70, 28, true)
		w32.MoveWindow(hwndBtnDrawerExecute, drawerX+168, btnY, drawerW-10-168, 28, true)
	} else {
		w32.ShowWindow(hwndGrpDrawer, w32.SW_HIDE)
		w32.ShowWindow(hwndLblDrawerSummary, w32.SW_HIDE)
		w32.ShowWindow(hwndListViewStaged, w32.SW_HIDE)
		w32.ShowWindow(hwndBtnDrawerRemove, w32.SW_HIDE)
		w32.ShowWindow(hwndBtnDrawerClear, w32.SW_HIDE)
		w32.ShowWindow(hwndBtnDrawerExecute, w32.SW_HIDE)
	}

	// Tab 0: Local Explorer controls
	w32.MoveWindow(hwndLblFilter, 12, contentTop+2, 80, 20, true)
	filterW := leftW - 85 - 105 - 150 - 20
	if filterW < 80 {
		filterW = 80
	}
	w32.MoveWindow(hwndTxtFilter, 95, contentTop, filterW, 24, true)
	w32.MoveWindow(hwndBtnToggleTree, 95+filterW+6, contentTop-1, 105, 26, true)
	w32.MoveWindow(hwndBtnToggleView, 95+filterW+6+110, contentTop-1, 150, 26, true)

	if showTreeView {
		treeW := 210
		if leftW < 450 {
			treeW = 160
		}
		w32.ShowWindow(hwndTreeView, w32.SW_SHOW)
		w32.MoveWindow(hwndTreeView, 12, contentTop+32, treeW, listHeight, true)
		w32.MoveWindow(hwndListView, 12+treeW+6, contentTop+32, leftW-treeW-6, listHeight, true)
	} else {
		w32.ShowWindow(hwndTreeView, w32.SW_HIDE)
		w32.MoveWindow(hwndListView, 12, contentTop+32, leftW, listHeight, true)
	}

	w32.MoveWindow(hwndBtnMode, 12, bottomBarY, 135, 30, true)
	w32.MoveWindow(hwndBtnExecute, 153, bottomBarY, 140, 30, true)
	w32.MoveWindow(hwndBtnDryRun, 299, bottomBarY, 120, 30, true)
	w32.MoveWindow(hwndBtnSelectAll, 425, bottomBarY, 95, 30, true)
	w32.MoveWindow(hwndBtnClear, 526, bottomBarY, 105, 30, true)

	// Tab 1: Universal Search controls
	w32.MoveWindow(hwndLblUniv, 12, contentTop+2, 230, 20, true)
	univInputW := leftW - 235 - 145 - 135 - 10
	if univInputW < 100 {
		univInputW = 100
	}
	w32.MoveWindow(hwndTxtUnivSearch, 245, contentTop, univInputW, 24, true)
	w32.MoveWindow(hwndCmbUnivDrive, 245+univInputW+6, contentTop, 140, 100, true)
	w32.MoveWindow(hwndBtnUnivSearch, 245+univInputW+6+146, contentTop-1, 130, 26, true)
	w32.MoveWindow(hwndListViewUniv, 12, contentTop+32, leftW, listHeight, true)

	w32.MoveWindow(hwndBtnUnivStage, 12, bottomBarY, 220, 30, true)
	w32.MoveWindow(hwndBtnUnivOpenDir, 238, bottomBarY, 160, 30, true)
	w32.MoveWindow(hwndBtnUnivCheckAll, 404, bottomBarY, 105, 30, true)
	w32.MoveWindow(hwndBtnUnivClearAll, 515, bottomBarY, 115, 30, true)

	// Tab 2: Remote Sync
	w32.MoveWindow(hwndGrpRemote, 12, contentTop, leftW, 200, true)

	// Tab 3: Server Health
	w32.MoveWindow(hwndGrpHealth, 12, contentTop, leftW, 160, true)

	// Tab 4: Transfer Log
	w32.MoveWindow(hwndEditLog, 12, contentTop, leftW, bottomBarY-contentTop-8, true)
	w32.MoveWindow(hwndBtnCopyLog, 12, bottomBarY, 220, 30, true)
	w32.MoveWindow(hwndBtnCopyCmd, 240, bottomBarY, 230, 30, true)

	// Tab 5: Settings Tab Controls
	w32.MoveWindow(hwndGrpShell, 12, contentTop, leftW, 130, true)
	w32.MoveWindow(hwndChkIcons, 30, contentTop+30, 260, 24, true)
	w32.MoveWindow(hwndChkSeps, 30, contentTop+60, 260, 24, true)
	w32.MoveWindow(hwndChkAccel, 30, contentTop+90, 280, 24, true)
	w32.MoveWindow(hwndLblPosition, 340, contentTop+32, 100, 20, true)
	w32.MoveWindow(hwndCmbPosition, 445, contentTop+28, 180, 100, true)

	w32.MoveWindow(hwndGrpEngine, 12, contentTop+138, leftW, 75, true)
	w32.MoveWindow(hwndLblEngine, 30, contentTop+168, 120, 20, true)
	w32.MoveWindow(hwndCmbEngine, 160, contentTop+164, 250, 120, true)

	w32.MoveWindow(hwndGrpService, 12, contentTop+220, leftW, 75, true)
	w32.MoveWindow(hwndLblService, 30, contentTop+250, 280, 20, true)
	w32.MoveWindow(hwndBtnService, 320, contentTop+245, 210, 28, true)

	w32.MoveWindow(hwndGrpTheme, 12, contentTop+302, leftW, 115, true)
	w32.MoveWindow(hwndLblTheme, 30, contentTop+332, 90, 20, true)
	w32.MoveWindow(hwndCmbTheme, 125, contentTop+328, 190, 140, true)
	w32.MoveWindow(hwndLblRadius, 325, contentTop+332, 55, 20, true)
	w32.MoveWindow(hwndCmbRadius, 385, contentTop+328, 160, 120, true)
	w32.MoveWindow(hwndLblShadow, 555, contentTop+332, 55, 20, true)
	w32.MoveWindow(hwndCmbShadow, 615, contentTop+328, 140, 120, true)

	w32.MoveWindow(hwndBtnEditTheme, 30, contentTop+368, 185, 30, true)
	w32.MoveWindow(hwndBtnResetTheme, 222, contentTop+368, 125, 30, true)
	w32.MoveWindow(hwndLblThemeHint, 355, contentTop+374, 395, 20, true)

	w32.MoveWindow(hwndBtnSaveCfg, 12, contentTop+428, 280, 36, true)

	// Status bar
	parts := []int32{240, 390, 560, 740, -1}
	setStatusBarParts(hwndStatusBar, parts)
	w32.SendMessage(hwndStatusBar, w32.WM_SIZE, 0, 0)
}

func onCommand(hwnd w32.HWND, controlID, code int) {
	switch controlID {
	// Navigation Bar
	case ID_BTN_BACK:
		navBack()

	case ID_BTN_FWD:
		navForward()

	case ID_BTN_UP:
		navUp()

	case ID_BTN_GO:
		path := getWindowText(hwndTxtTarget)
		if path != "" {
			navigateTo(path, true)
		}

	case ID_BTN_BROWSE:
		if path, ok := BrowseFolderDialog(hwnd, "Select Target Folder"); ok {
			navigateTo(path, true)
		}

	case ID_BTN_JUMP:
		showJumpMenu(hwndBtnJump)

	case ID_BTN_TOP_UNIV_SEARCH:
		runTopUniversalSearch()

	case ID_BTN_THEME_TOGGLE:
		showThemePopupMenu(hwnd, hwndBtnThemeToggle)

	case ID_BTN_RESCAN:
		populateTreeRoots(hwndTreeView)
		syncTreeSelection(hwndTreeView, stagedPath)
		triggerScanAsync(stagedPath)

	case ID_BTN_SETTINGS:
		w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 5, 0)
		onTabChanged()

	case ID_BTN_DRAWER_TOGGLE:
		showStagedDrawer = !showStagedDrawer
		rc := w32.GetClientRect(hwndMain)
		if rc != nil {
			onResize(int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
		}
		refreshStagedDrawer()

	case ID_BTN_OP_ACTION:
		handleOperationBannerAction()

	case ID_BTN_SAVESETTINGS:
		saveSettingsFromUI()

	case ID_BTN_SERVICE:
		go func() {
			svcExe, _ := filepath.Abs("rsync-service.exe")
			if core.IsSelfElevated() {
				_ = core.InstallAndStartService(svcExe)
			} else {
				psCmd := fmt.Sprintf(`Start-Process -FilePath '%s' -ArgumentList '--install' -Verb RunAs -Wait`, svcExe)
				_ = exec.Command("powershell", "-NoProfile", "-Command", psCmd).Run()
			}
			checkServiceStateAsync()
		}()

	// Tab 0: Local Explorer
	case ID_BTN_TOGGLE_TREE:
		showTreeView = !showTreeView
		if showTreeView {
			setWindowText(hwndBtnToggleTree, "TREE: ON")
		} else {
			setWindowText(hwndBtnToggleTree, "TREE: OFF")
		}
		rc := w32.GetClientRect(hwndMain)
		if rc != nil {
			onResize(int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
		}
		w32.InvalidateRect(hwndMain, nil, true)

	case ID_BTN_TOGGLE_VIEW:
		showAllRecursive = !showAllRecursive
		if showAllRecursive {
			setWindowText(hwndBtnToggleView, "VIEW: ALL RECURSIVE")
		} else {
			setWindowText(hwndBtnToggleView, "FOLDER VIEW")
		}
		filterFiles(getWindowText(hwndTxtFilter))

	case ID_TXT_FILTER:
		if code == 0x0300 {
			filterFiles(getWindowText(hwndTxtFilter))
		}

	case ID_BTN_SELECTALL:
		for _, item := range currentListItems {
			if !item.IsParent {
				addToStaged(item)
			}
		}
		syncCurrentListViewChecks()
		updateOperationBanner()

	case ID_BTN_CLEAR:
		for _, item := range currentListItems {
			removeFromStaged(item.FullPath)
		}
		syncCurrentListViewChecks()
		updateOperationBanner()

	case ID_BTN_MODE:
		showModePopupMenu(hwndBtnMode)

	case ID_BTN_EXECUTE:
		runRsyncTransfer(false)

	case ID_BTN_DRYRUN:
		runRsyncTransfer(true)

	// Tab 1: Universal Search
	case ID_BTN_UNIV_SEARCH:
		performUniversalSearch()

	case ID_TXT_UNIV_SEARCH:
		if code == 0x0300 {
			performUniversalSearch()
		}

	case ID_BTN_UNIV_STAGE:
		stageCheckedUniversalItems()

	case ID_BTN_UNIV_OPENDIR:
		openSelectedUniversalFileDir()

	case ID_BTN_UNIV_CHECKALL:
		setAllListViewCheckboxes(hwndListViewUniv, true)

	case ID_BTN_UNIV_CLEARALL:
		setAllListViewCheckboxes(hwndListViewUniv, false)

	// Tab 2: Remote Server Sync
	case ID_BTN_REMOTE_PUSH:
		stagedMode = "copy"
		dest := getWindowText(hwndTxtRemoteDest)
		if dest != "" {
			stagedDest = dest
		}
		runRsyncTransfer(false)

	case ID_BTN_REMOTE_PULL:
		stagedMode = "pull"
		runRsyncTransfer(false)

	// Tab 3: Server Health
	case ID_BTN_PING_TEST:
		testServerPingAsync()

	// Tab 4: Transfer Log controls
	case ID_BTN_COPY_LOG:
		logText := getWindowText(hwndEditLog)
		if logText != "" {
			setClipboardText(hwndMain, logText)
			setStatusBarText(hwndStatusBar, 4, "LOG COPIED TO CLIPBOARD")
			playSystemChime()
		}

	case ID_BTN_COPY_CMD:
		copyRsyncCliCommandToClipboard()

	// Mini Right Drawer controls
	case ID_BTN_DRAWER_REMOVE:
		selected := getSelectedListViewItems(hwndListViewStaged)
		if len(selected) > 0 {
			var toRemove []string
			for _, selIdx := range selected {
				if selIdx >= 0 && selIdx < len(stagedListPaths) {
					toRemove = append(toRemove, stagedListPaths[selIdx])
				}
			}
			for _, p := range toRemove {
				removeFromStaged(p)
			}
			updateOperationBanner()
		}

	case ID_BTN_DRAWER_CLEAR:
		clearStaged()
		updateOperationBanner()

	case ID_BTN_DRAWER_EXECUTE:
		handleOperationBannerAction()

	case ID_CMB_THEME:
		if code == 1 /* CBN_SELCHANGE */ {
			themeSel := int(w32.SendMessage(hwndCmbTheme, 0x0147, 0, 0))
			if themeSel >= 0 && themeSel < len(availableThemes) {
				appCfg.ActiveTheme = availableThemes[themeSel]
				switchTheme(hwndMain, appCfg.ActiveTheme)
				updateThemeButtonLabel()
			}
		}

	case ID_CMB_RADIUS:
		if code == 1 /* CBN_SELCHANGE */ {
			radSel := int(w32.SendMessage(hwndCmbRadius, 0x0147, 0, 0))
			rad := 8
			switch radSel {
			case 0:
				rad = 0
			case 1:
				rad = 4
			case 2:
				rad = 8
			case 3:
				rad = 14
			}
			appCfg.CornerRadius = rad
			updateCornerRadius(hwndMain, rad)
			_ = core.SaveConfig("./config.json", appCfg)
		}

	case ID_CMB_SHADOW:
		if code == 1 /* CBN_SELCHANGE */ {
			shadowSel := int(w32.SendMessage(hwndCmbShadow, 0x0147, 0, 0))
			depth := 2
			switch shadowSel {
			case 0:
				depth = 0
			case 1:
				depth = 1
			case 2:
				depth = 2
			case 3:
				depth = 4
			case 4:
				depth = 6
			}
			appCfg.ShadowDepth = depth
			updateShadowDepth(hwndMain, depth)
			_ = core.SaveConfig("./config.json", appCfg)
		}

	case ID_BTN_EDIT_THEME:
		showThemeCustomizerMenu(hwndMain, hwndBtnEditTheme)

	case ID_BTN_RESET_THEME:
		resetCustomTheme(hwndMain)
	}
}

func executeContextAction() {
	if hwndMain != 0 {
		w32.ShowWindow(hwndMain, w32.SW_RESTORE)
		w32.SetForegroundWindow(hwndMain)
	}

	mode := strings.ToLower(stagedMode)
	switch mode {
	case "copy":
		if stagedServer != "" {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 2, 0)
			onTabChanged()
			selectServerInCombo(stagedServer)

			srvName := stagedServer
			for _, s := range appCfg.Servers {
				if strings.EqualFold(s.ID, stagedServer) || strings.EqualFold(s.Name, stagedServer) {
					srvName = s.Name
					break
				}
			}

			count := len(stagedRegistry)
			if count == 0 && stagedPath != "" {
				count = 1
			}

			msg := fmt.Sprintf("Ready to push %d staged item(s) to remote server %s?\n\nExecute Rsync transfer now?", count, srvName)
			ans := w32.MessageBox(hwndMain, msg, "Confirm Remote Rsync Push", w32.MB_YESNO|w32.MB_ICONQUESTION)
			if ans == w32.IDYES {
				runRsyncTransfer(false)
			} else {
				setStatusBarText(hwndStatusBar, 4, "PUSH STAGED - CLICK EXECUTE WHEN READY")
			}
			return
		}

		if stagedDest != "" && len(stagedRegistry) == 0 {
			src, ok := BrowseFolderDialog(hwndMain, "Select Source Folder to Copy From")
			if !ok || strings.TrimSpace(src) == "" {
				setStatusBarText(hwndStatusBar, 4, "COPY CANCELLED")
				return
			}
			stagedPath = filepath.Clean(src)
			updateOperationBanner()

			msg := fmt.Sprintf("Ready to copy files from:\n%s\n\nInto destination folder:\n%s\n\nExecute Rsync copy now?", stagedPath, stagedDest)
			ans := w32.MessageBox(hwndMain, msg, "Confirm Rsync Copy", w32.MB_YESNO|w32.MB_ICONQUESTION)
			if ans == w32.IDYES {
				runRsyncTransfer(false)
			} else {
				setStatusBarText(hwndStatusBar, 4, "COPY PRIMED - CLICK EXECUTE WHEN READY")
			}
			return
		}

		count := len(stagedRegistry)
		if count == 0 && stagedPath != "" {
			fi, err := os.Stat(stagedPath)
			if err == nil {
				addToStaged(DisplayItem{
					Name:     filepath.Base(stagedPath),
					FullPath: stagedPath,
					Size:     fi.Size(),
					IsDir:    fi.IsDir(),
				})
				count = len(stagedRegistry)
			}
		}

		if stagedDest == "" {
			dest, ok := BrowseFolderDialog(hwndMain, "Select Destination Folder for Rsync Copy")
			if !ok || strings.TrimSpace(dest) == "" {
				setStatusBarText(hwndStatusBar, 4, "COPY CANCELLED - ITEMS REMAIN STAGED")
				return
			}
			stagedDest = filepath.Clean(dest)
			updateOperationBanner()
		}

		msg := fmt.Sprintf("Ready to copy %d staged item(s) to:\n%s\n\nExecute Rsync transfer now?", count, stagedDest)
		ans := w32.MessageBox(hwndMain, msg, "Confirm Rsync Copy", w32.MB_YESNO|w32.MB_ICONQUESTION)
		if ans == w32.IDYES {
			runRsyncTransfer(false)
		} else {
			setStatusBarText(hwndStatusBar, 4, "COPY PRIMED - CLICK EXECUTE WHEN READY")
		}

	case "move_to":
		if stagedDest == "" {
			dest, ok := BrowseFolderDialog(hwndMain, "Select Destination Folder for Rsync Move")
			if !ok || strings.TrimSpace(dest) == "" {
				setStatusBarText(hwndStatusBar, 4, "MOVE CANCELLED - ITEMS REMAIN STAGED")
				return
			}
			stagedDest = filepath.Clean(dest)
			updateOperationBanner()
		}

		sourceDesc := fmt.Sprintf("%d staged item(s)", len(stagedRegistry))
		if len(stagedRegistry) == 0 {
			sourceDesc = fmt.Sprintf("all files in:\n%s", stagedPath)
		}

		msg := fmt.Sprintf("Ready to MOVE %s to:\n%s\n\nSource items will be removed upon successful transfer (--remove-source-files).\n\nExecute Rsync move now?", sourceDesc, stagedDest)
		ans := w32.MessageBox(hwndMain, msg, "Confirm Rsync Move", w32.MB_YESNO|w32.MB_ICONWARNING)
		if ans == w32.IDYES {
			runRsyncTransfer(false)
		} else {
			setStatusBarText(hwndStatusBar, 4, "MOVE PRIMED - CLICK EXECUTE WHEN READY")
		}

	case "move_from":
		src, ok := BrowseFolderDialog(hwndMain, "Select Source Folder to Move From")
		if !ok || strings.TrimSpace(src) == "" {
			setStatusBarText(hwndStatusBar, 4, "MOVE CANCELLED")
			return
		}
		stagedPath = filepath.Clean(src)
		updateOperationBanner()

		destFolder := stagedDest
		if destFolder == "" {
			destFolder = stagedPath
		}

		msg := fmt.Sprintf("Ready to MOVE files from:\n%s\n\nInto destination folder:\n%s\n\nSource items will be removed upon successful transfer (--remove-source-files).\n\nExecute Rsync move now?", stagedPath, destFolder)
		ans := w32.MessageBox(hwndMain, msg, "Confirm Rsync Move", w32.MB_YESNO|w32.MB_ICONWARNING)
		if ans == w32.IDYES {
			runRsyncTransfer(false)
		} else {
			setStatusBarText(hwndStatusBar, 4, "MOVE PRIMED - CLICK EXECUTE WHEN READY")
		}

	case "pull":
		if stagedServer != "" {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 2, 0)
			onTabChanged()
			selectServerInCombo(stagedServer)

			srvName := stagedServer
			for _, s := range appCfg.Servers {
				if strings.EqualFold(s.ID, stagedServer) || strings.EqualFold(s.Name, stagedServer) {
					srvName = s.Name
					break
				}
			}

			destFolder := stagedDest
			if destFolder == "" {
				destFolder = stagedPath
			}

			msg := fmt.Sprintf("Ready to pull from remote server %s\nInto destination directory:\n%s\n\nExecute Rsync transfer now?", srvName, destFolder)
			ans := w32.MessageBox(hwndMain, msg, "Confirm Remote Rsync Pull", w32.MB_YESNO|w32.MB_ICONQUESTION)
			if ans == w32.IDYES {
				runRsyncTransfer(false)
			} else {
				setStatusBarText(hwndStatusBar, 4, "PULL PRIMED - CLICK EXECUTE WHEN READY")
			}
			return
		}

		src, ok := BrowseFolderDialog(hwndMain, "Select Source Folder to Pull From")
		if !ok || strings.TrimSpace(src) == "" {
			setStatusBarText(hwndStatusBar, 4, "PULL CANCELLED")
			return
		}
		stagedPath = filepath.Clean(src)
		updateOperationBanner()

		destFolder := stagedDest
		if destFolder == "" {
			destFolder = stagedPath
		}

		msg := fmt.Sprintf("Ready to pull files from:\n%s\n\nInto destination folder:\n%s\n\nExecute Rsync pull now?", stagedPath, destFolder)
		ans := w32.MessageBox(hwndMain, msg, "Confirm Rsync Pull", w32.MB_YESNO|w32.MB_ICONQUESTION)
		if ans == w32.IDYES {
			runRsyncTransfer(false)
		} else {
			setStatusBarText(hwndStatusBar, 4, "PULL PRIMED - CLICK EXECUTE WHEN READY")
		}

	default:
		runRsyncTransfer(false)
	}
}

func handleOperationBannerAction() {
	executeContextAction()
}

func copyRsyncCliCommandToClipboard() {
	var selectedPaths []string
	if len(stagedRegistry) > 0 {
		for p := range stagedRegistry {
			selectedPaths = append(selectedPaths, p)
		}
	} else {
		selectedPaths = []string{stagedPath}
	}
	dest := stagedDest
	if dest == "" {
		dest = `C:\Backup\Target`
	}

	isWin := core.IsRsyncWin(appCfg.RsyncBinaryPath)
	var cmdStr string

	if isWin {
		var cmds []string
		cleanDest := filepath.ToSlash(dest)
		for _, src := range selectedPaths {
			cleanSrc := filepath.ToSlash(src)
			var parts []string
			parts = append(parts, "rsync", "-a", "-v", "--progress")
			if stagedServer != "" {
				for _, s := range appCfg.Servers {
					if strings.EqualFold(s.ID, stagedServer) || strings.EqualFold(s.Name, stagedServer) {
						if s.Port != 0 && s.Port != 22 {
							parts = append(parts, "--ssh-port", strconv.Itoa(s.Port))
						}
						if s.KeyPath != "" {
							parts = append(parts, "-i", fmt.Sprintf(`"%s"`, filepath.ToSlash(s.KeyPath)))
						}
						if stagedMode == "pull" || stagedMode == "move_from" {
							remoteSrc := fmt.Sprintf(`"%s@%s:%s"`, s.User, s.Host, s.RemotePath)
							parts = append(parts, "--src", remoteSrc, "--dest", fmt.Sprintf(`"%s"`, cleanDest))
						} else {
							remoteDest := fmt.Sprintf(`"%s@%s:%s"`, s.User, s.Host, s.RemotePath)
							parts = append(parts, "--src", fmt.Sprintf(`"%s"`, cleanSrc), "--dest", remoteDest)
						}
						break
					}
				}
			} else {
				parts = append(parts, "--src", fmt.Sprintf(`"%s"`, cleanSrc), "--dest", fmt.Sprintf(`"%s"`, cleanDest))
			}
			cmds = append(cmds, strings.Join(parts, " "))
		}
		cmdStr = strings.Join(cmds, " && ")
	} else {
		var parts []string
		parts = append(parts, "rsync", "-av", "--progress")
		if stagedMode == "move_to" || stagedMode == "move_from" {
			parts = append(parts, "--remove-source-files")
		}
		if stagedServer != "" {
			for _, s := range appCfg.Servers {
				if strings.EqualFold(s.ID, stagedServer) || strings.EqualFold(s.Name, stagedServer) {
					sshCmd := fmt.Sprintf(`ssh -p %d`, s.Port)
					if s.KeyPath != "" {
						sshCmd += fmt.Sprintf(` -i "%s"`, s.KeyPath)
					}
					parts = append(parts, "-e", fmt.Sprintf(`"%s"`, sshCmd))
					if stagedMode == "pull" || stagedMode == "move_from" {
						parts = append(parts, fmt.Sprintf(`"%s@%s:%s"`, s.User, s.Host, s.RemotePath), fmt.Sprintf(`"%s"`, dest))
					} else {
						for _, src := range selectedPaths {
							parts = append(parts, fmt.Sprintf(`"%s"`, src))
						}
						parts = append(parts, fmt.Sprintf(`"%s@%s:%s"`, s.User, s.Host, s.RemotePath))
					}
					break
				}
			}
		} else {
			for _, src := range selectedPaths {
				parts = append(parts, fmt.Sprintf(`"%s"`, src))
			}
			parts = append(parts, fmt.Sprintf(`"%s"`, dest))
		}
		cmdStr = strings.Join(parts, " ")
	}

	setClipboardText(hwndMain, cmdStr)
	setStatusBarText(hwndStatusBar, 4, "RSYNC COMMAND COPIED TO CLIPBOARD")
	playSystemChime()
	w32.MessageBox(hwndMain, "Rsync CLI command copied to clipboard:\n\n"+cmdStr, "Rsync CLI Command", w32.MB_OK|w32.MB_ICONINFORMATION)
}

// Navigation helpers
func navigateTo(path string, recordHistory bool) {
	cleanPath := filepath.Clean(path)
	fi, err := os.Stat(cleanPath)
	if err != nil {
		return
	}
	if !fi.IsDir() {
		cleanPath = filepath.Dir(cleanPath)
	}

	stagedPath = cleanPath
	setWindowText(hwndTxtTarget, stagedPath)
	updateDriveSpaceStatus()

	if recordHistory {
		if navIndex < len(navHistory)-1 {
			navHistory = navHistory[:navIndex+1]
		}
		navHistory = append(navHistory, stagedPath)
		navIndex = len(navHistory) - 1
	}

	updateNavButtons()
	syncTreeSelection(hwndTreeView, stagedPath)
	filterFiles(getWindowText(hwndTxtFilter))
	triggerScanAsync(stagedPath)
}

func navBack() {
	if navIndex > 0 {
		navIndex--
		navigateTo(navHistory[navIndex], false)
	} else {
		navUp()
	}
}

func navForward() {
	if navIndex < len(navHistory)-1 {
		navIndex++
		navigateTo(navHistory[navIndex], false)
	}
}

func navUp() {
	parent := filepath.Dir(stagedPath)
	if parent != stagedPath && parent != "" {
		navigateTo(parent, true)
	}
}

func updateNavButtons() {
	canGoUp := filepath.Dir(stagedPath) != stagedPath && stagedPath != ""
	canGoBack := navIndex > 0 || canGoUp
	canGoFwd := navIndex < len(navHistory)-1

	w32.EnableWindow(hwndBtnBack, canGoBack)
	w32.EnableWindow(hwndBtnFwd, canGoFwd)
	w32.EnableWindow(hwndBtnUp, canGoUp)
}

func onListViewDoubleClick(lParam uintptr) {
	nmia := (*NMITEMACTIVATE)(unsafe.Pointer(lParam))
	rowIndex := int(nmia.IItem)
	if rowIndex < 0 || rowIndex >= len(currentListItems) {
		return
	}

	item := currentListItems[rowIndex]
	if item.IsParent {
		navUp()
		return
	}

	if item.IsDir {
		navigateTo(item.FullPath, true)
		return
	}

	checked := isListViewItemChecked(hwndListView, rowIndex)
	setListViewItemChecked(hwndListView, rowIndex, !checked)
}

func onListViewItemChanged(lParam uintptr) {
	if isPopulatingListView {
		return
	}
	nmlv := (*NMLISTVIEW)(unsafe.Pointer(lParam))
	if (nmlv.UChanged & LVIF_STATE) == 0 {
		return
	}
	oldChecked := (nmlv.UOldState & 0x2000) != 0
	newChecked := (nmlv.UNewState & 0x2000) != 0
	if oldChecked == newChecked {
		return
	}

	idx := int(nmlv.IItem)
	if idx < 0 || idx >= len(currentListItems) {
		return
	}
	item := currentListItems[idx]
	if item.IsParent {
		return
	}

	if newChecked {
		addToStaged(item)
	} else {
		removeFromStaged(item.FullPath)
	}
	updateOperationBanner()
}

func onUnivListViewDoubleClick(lParam uintptr) {
	nmia := (*NMITEMACTIVATE)(unsafe.Pointer(lParam))
	rowIndex := int(nmia.IItem)
	if rowIndex < 0 || rowIndex >= len(univResults) {
		return
	}

	res := univResults[rowIndex]
	parentFolder := filepath.Dir(res.FullPath)
	navigateTo(parentFolder, true)

	w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 0, 0)
	onTabChanged()
}

func onUnivListViewItemChanged(lParam uintptr) {
	nmlv := (*NMLISTVIEW)(unsafe.Pointer(lParam))
	if (nmlv.UChanged & LVIF_STATE) == 0 {
		return
	}
	oldChecked := (nmlv.UOldState & 0x2000) != 0
	newChecked := (nmlv.UNewState & 0x2000) != 0
	if oldChecked == newChecked {
		return
	}

	idx := int(nmlv.IItem)
	if idx < 0 || idx >= len(univResults) {
		return
	}
	res := univResults[idx]
	name := filepath.Base(res.FullPath)
	size := int64(0)
	isDir := false
	if res.Record != nil {
		name = res.Record.Name
		size = res.Record.Size
		isDir = res.Record.IsDir
	}
	item := DisplayItem{
		Name:     name,
		FullPath: res.FullPath,
		Size:     size,
		IsDir:    isDir,
	}

	if newChecked {
		addToStaged(item)
	} else {
		removeFromStaged(item.FullPath)
	}
	updateOperationBanner()
}

func onStagedListViewDoubleClick(lParam uintptr) {
	nmia := (*NMITEMACTIVATE)(unsafe.Pointer(lParam))
	rowIndex := int(nmia.IItem)
	if rowIndex >= 0 && rowIndex < len(stagedListPaths) {
		p := stagedListPaths[rowIndex]
		removeFromStaged(p)
		updateOperationBanner()
	}
}

func onTabChanged() {
	activeTab = getSelectedTab(hwndTabControl)

	// Tab 0: Local Explorer
	isTab0 := (activeTab == 0)
	w32.ShowWindow(hwndLblFilter, boolToCmd(isTab0))
	w32.ShowWindow(hwndTxtFilter, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnToggleTree, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnToggleView, boolToCmd(isTab0))
	w32.ShowWindow(hwndTreeView, boolToCmd(isTab0 && showTreeView))
	w32.ShowWindow(hwndListView, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnMode, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnExecute, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnDryRun, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnSelectAll, boolToCmd(isTab0))
	w32.ShowWindow(hwndBtnClear, boolToCmd(isTab0))

	// Tab 1: Universal Search
	isTab1 := (activeTab == 1)
	w32.ShowWindow(hwndLblUniv, boolToCmd(isTab1))
	w32.ShowWindow(hwndTxtUnivSearch, boolToCmd(isTab1))
	w32.ShowWindow(hwndCmbUnivDrive, boolToCmd(isTab1))
	w32.ShowWindow(hwndBtnUnivSearch, boolToCmd(isTab1))
	w32.ShowWindow(hwndListViewUniv, boolToCmd(isTab1))
	w32.ShowWindow(hwndBtnUnivStage, boolToCmd(isTab1))
	w32.ShowWindow(hwndBtnUnivOpenDir, boolToCmd(isTab1))
	w32.ShowWindow(hwndBtnUnivCheckAll, boolToCmd(isTab1))
	w32.ShowWindow(hwndBtnUnivClearAll, boolToCmd(isTab1))

	// Tab 2: Remote Sync
	isTab2 := (activeTab == 2)
	w32.ShowWindow(hwndGrpRemote, boolToCmd(isTab2))
	w32.ShowWindow(hwndLblRemoteSrv, boolToCmd(isTab2))
	w32.ShowWindow(hwndCmbRemoteSrv, boolToCmd(isTab2))
	w32.ShowWindow(hwndLblRemoteDest, boolToCmd(isTab2))
	w32.ShowWindow(hwndTxtRemoteDest, boolToCmd(isTab2))
	w32.ShowWindow(hwndBtnRemotePush, boolToCmd(isTab2))
	w32.ShowWindow(hwndBtnRemotePull, boolToCmd(isTab2))

	// Tab 3: Server Health
	isTab3 := (activeTab == 3)
	w32.ShowWindow(hwndGrpHealth, boolToCmd(isTab3))
	w32.ShowWindow(hwndBtnPingTest, boolToCmd(isTab3))
	w32.ShowWindow(hwndLblPingResult, boolToCmd(isTab3))

	// Tab 4: Transfer Log
	isTab4 := (activeTab == 4)
	w32.ShowWindow(hwndEditLog, boolToCmd(isTab4))
	w32.ShowWindow(hwndBtnCopyLog, boolToCmd(isTab4))
	w32.ShowWindow(hwndBtnCopyCmd, boolToCmd(isTab4))

	// Tab 5: Settings Tab
	isTab5 := (activeTab == 5)
	w32.ShowWindow(hwndGrpShell, boolToCmd(isTab5))
	w32.ShowWindow(hwndChkIcons, boolToCmd(isTab5))
	w32.ShowWindow(hwndChkSeps, boolToCmd(isTab5))
	w32.ShowWindow(hwndChkAccel, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblPosition, boolToCmd(isTab5))
	w32.ShowWindow(hwndCmbPosition, boolToCmd(isTab5))

	w32.ShowWindow(hwndGrpEngine, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblEngine, boolToCmd(isTab5))
	w32.ShowWindow(hwndCmbEngine, boolToCmd(isTab5))

	w32.ShowWindow(hwndGrpService, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblService, boolToCmd(isTab5))
	w32.ShowWindow(hwndBtnService, boolToCmd(isTab5))

	w32.ShowWindow(hwndGrpTheme, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblTheme, boolToCmd(isTab5))
	w32.ShowWindow(hwndCmbTheme, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblRadius, boolToCmd(isTab5))
	w32.ShowWindow(hwndCmbRadius, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblShadow, boolToCmd(isTab5))
	w32.ShowWindow(hwndCmbShadow, boolToCmd(isTab5))
	w32.ShowWindow(hwndBtnEditTheme, boolToCmd(isTab5))
	w32.ShowWindow(hwndBtnResetTheme, boolToCmd(isTab5))
	w32.ShowWindow(hwndLblThemeHint, boolToCmd(isTab5))

	w32.ShowWindow(hwndBtnSaveCfg, boolToCmd(isTab5))

	w32.InvalidateRect(hwndTabControl, nil, true)
	w32.InvalidateRect(hwndMain, nil, true)
	w32.UpdateWindow(hwndMain)
}

func boolToCmd(v bool) int {
	if v {
		return w32.SW_SHOW
	}
	return w32.SW_HIDE
}

func triggerScanAsync(target string) {
	scanLock.Lock()
	if isScanning {
		scanLock.Unlock()
		return
	}
	isScanning = true
	scanLock.Unlock()

	setStatusBarText(hwndStatusBar, 4, "SCANNING DISK...")

	go func() {
		defer func() {
			scanLock.Lock()
			isScanning = false
			scanLock.Unlock()
		}()

		idx, _, engineName, err := engineMgr.Scan(target)
		if err != nil {
			setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("SCAN ERROR: %v", err))
			return
		}

		currentScanIndex = idx
		setStatusBarText(hwndStatusBar, 1, fmt.Sprintf("ENGINE: %s", strings.ToUpper(engineName)))
		w32.PostMessage(hwndMain, WM_APP_SCAN_COMPLETE, 0, 0)
	}()
}

func onScanComplete() {
	filterFiles(getWindowText(hwndTxtFilter))
	setStatusBarText(hwndStatusBar, 4, "READY")
}

func filterFiles(pattern string) {
	isPopulatingListView = true
	defer func() { isPopulatingListView = false }()

	if currentScanIndex == nil && showAllRecursive {
		return
	}

	clearListView(hwndListView)
	currentListItems = nil

	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Parent folder entry
	parentDir := filepath.Dir(stagedPath)
	if parentDir != stagedPath && parentDir != "" {
		currentListItems = append(currentListItems, DisplayItem{
			Name:     ".. (Parent Directory)",
			FullPath: parentDir,
			RelPath:  "..",
			Size:     0,
			IsDir:    true,
			IsParent: true,
		})
	}

	if showAllRecursive {
		for _, rec := range currentScanIndex.Records {
			if rec == nil {
				continue
			}
			if pattern != "" && !strings.Contains(strings.ToLower(rec.Name), pattern) {
				continue
			}
			full := currentScanIndex.ResolvePath(rec.ID)
			currentListItems = append(currentListItems, DisplayItem{
				Name:     rec.Name,
				FullPath: full,
				RelPath:  full,
				Size:     rec.Size,
				IsDir:    rec.IsDir,
			})
		}
	} else {
		dirClean := filepath.Clean(stagedPath)
		entries, err := os.ReadDir(dirClean)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if pattern != "" && !strings.Contains(strings.ToLower(name), pattern) {
					continue
				}

				info, _ := e.Info()
				var size int64
				if info != nil {
					size = info.Size()
				}
				full := filepath.Join(dirClean, name)
				currentListItems = append(currentListItems, DisplayItem{
					Name:     name,
					FullPath: full,
					RelPath:  name,
					Size:     size,
					IsDir:    e.IsDir(),
				})
			}
		}
	}

	for i, item := range currentListItems {
		typeStr := "File"
		if item.IsDir {
			typeStr = "Folder"
		}
		if item.IsParent {
			typeStr = "Up"
		}

		sizeStr := formatBytes(item.Size)
		if item.IsDir {
			sizeStr = "-"
		}

		staged := false
		if _, exists := stagedRegistry[filepath.Clean(item.FullPath)]; exists {
			staged = true
		}

		addListViewRow(hwndListView, int32(i), []string{
			item.Name,
			item.RelPath,
			sizeStr,
			typeStr,
			"Ready",
		}, staged)
	}

	if targetItemToSelect != "" {
		targetClean := strings.ToLower(filepath.Clean(targetItemToSelect))
		targetBase := strings.ToLower(filepath.Base(targetItemToSelect))
		for i, item := range currentListItems {
			if strings.ToLower(filepath.Clean(item.FullPath)) == targetClean || strings.ToLower(item.Name) == targetBase {
				selectAndFocusListViewItem(hwndListView, i)
				targetItemToSelect = ""
				break
			}
		}
	}

	refreshStagedDrawer()
}

// Staging Helpers
func addToStaged(item DisplayItem) {
	if item.IsParent || item.FullPath == "" {
		return
	}
	clean := filepath.Clean(item.FullPath)
	if _, exists := stagedRegistry[clean]; !exists {
		item.FullPath = clean
		stagedRegistry[clean] = item
		stagedTotalBytes += item.Size
		refreshStagedDrawer()
	}
}

func removeFromStaged(fullPath string) {
	clean := filepath.Clean(fullPath)
	if item, exists := stagedRegistry[clean]; exists {
		stagedTotalBytes -= item.Size
		if stagedTotalBytes < 0 {
			stagedTotalBytes = 0
		}
		delete(stagedRegistry, clean)
		refreshStagedDrawer()
		syncCurrentListViewChecks()
	}
}

func clearStaged() {
	stagedRegistry = make(map[string]DisplayItem)
	stagedTotalBytes = 0
	refreshStagedDrawer()
	syncCurrentListViewChecks()
}

func refreshStagedDrawer() {
	if hwndListViewStaged == 0 {
		return
	}
	clearListView(hwndListViewStaged)

	stagedListPaths = make([]string, 0, len(stagedRegistry))
	for p := range stagedRegistry {
		stagedListPaths = append(stagedListPaths, p)
	}
	sort.Strings(stagedListPaths)

	for row, p := range stagedListPaths {
		item := stagedRegistry[p]
		sizeStr := "Folder"
		if !item.IsDir {
			sizeStr = formatBytes(item.Size)
		}
		folder := filepath.Dir(item.FullPath)
		addListViewRow(hwndListViewStaged, int32(row), []string{item.Name, sizeStr, folder}, false)
	}

	summary := fmt.Sprintf("Staged: %d items (%s)", len(stagedRegistry), formatBytes(stagedTotalBytes))
	setWindowText(hwndLblDrawerSummary, summary)
	setStatusBarText(hwndStatusBar, 3, fmt.Sprintf("Staged: %d (%s)", len(stagedRegistry), formatBytes(stagedTotalBytes)))

	if showStagedDrawer {
		setWindowText(hwndBtnDrawerToggle, "HIDE DRAWER")
	} else {
		setWindowText(hwndBtnDrawerToggle, fmt.Sprintf("STAGED (%d)", len(stagedRegistry)))
	}
}

func syncCurrentListViewChecks() {
	if hwndListView == 0 {
		return
	}
	isPopulatingListView = true
	defer func() { isPopulatingListView = false }()
	for i, item := range currentListItems {
		clean := filepath.Clean(item.FullPath)
		_, staged := stagedRegistry[clean]
		setListViewItemChecked(hwndListView, i, staged)
	}
}

func handleLaunchPayload(p core.LaunchPayload) {
	if p.Mode != "" {
		stagedMode = strings.ToLower(cleanArgParam(p.Mode))
	}
	if p.Server != "" {
		stagedServer = strings.TrimSpace(p.Server)
		for len(stagedServer) >= 2 && stagedServer[0] == '"' && stagedServer[len(stagedServer)-1] == '"' {
			stagedServer = stagedServer[1 : len(stagedServer)-1]
		}
	}
	if p.Src != "" {
		stagedPath = cleanArgParam(p.Src)
	}
	if p.Dest != "" {
		stagedDest = cleanArgParam(p.Dest)
	}

	// Auto-stage provided paths or src
	autoStagedAny := false
	if p.AutoStage {
		if len(p.Paths) > 0 {
			for _, itemPath := range p.Paths {
				cleanItem := cleanArgParam(itemPath)
				if cleanItem == "" {
					continue
				}
				fi, err := os.Stat(cleanItem)
				if err == nil {
					targetItemToSelect = cleanItem
					addToStaged(DisplayItem{
						Name:     filepath.Base(cleanItem),
						FullPath: cleanItem,
						Size:     fi.Size(),
						IsDir:    fi.IsDir(),
					})
					autoStagedAny = true
				}
			}
		} else if p.Src != "" {
			cleanItem := cleanArgParam(p.Src)
			fi, err := os.Stat(cleanItem)
			if err == nil {
				targetItemToSelect = cleanItem
				addToStaged(DisplayItem{
					Name:     filepath.Base(cleanItem),
					FullPath: cleanItem,
					Size:     fi.Size(),
					IsDir:    fi.IsDir(),
				})
				autoStagedAny = true
			}
		}
	}

	if autoStagedAny && !showStagedDrawer {
		showStagedDrawer = true
		if hwndMain != 0 {
			rc := w32.GetClientRect(hwndMain)
			if rc != nil {
				onResize(int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
			}
		}
	}

	updateOperationBanner()

	// Direct action navigation
	switch strings.ToLower(stagedMode) {
	case "copy", "move_to":
		if p.Server != "" {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 2, 0)
			onTabChanged()
			selectServerInCombo(p.Server)
		} else {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 0, 0)
			onTabChanged()
			if p.Src != "" {
				cleanSrc := cleanArgParam(p.Src)
				fi, err := os.Stat(cleanSrc)
				if err == nil && !fi.IsDir() {
					targetItemToSelect = cleanSrc
					navigateTo(filepath.Dir(cleanSrc), true)
				} else if err == nil {
					targetItemToSelect = cleanSrc
					navigateTo(cleanSrc, true)
				}
			} else if stagedPath != "" {
				navigateTo(stagedPath, true)
			}
		}

	case "pull", "move_from":
		if p.Server != "" {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 2, 0)
			onTabChanged()
			selectServerInCombo(p.Server)
		} else {
			w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 0, 0)
			onTabChanged()
			if p.Dest != "" {
				cleanDest := cleanArgParam(p.Dest)
				targetItemToSelect = cleanDest
				navigateTo(cleanDest, true)
			} else if stagedDest != "" {
				navigateTo(stagedDest, true)
			}
		}
	default:
		if stagedPath != "" {
			navigateTo(stagedPath, true)
		}
	}

	updateModeButtonLabel()

	if hwndMain != 0 {
		bringWindowToForeground(hwndMain)
		w32.InvalidateRect(hwndMain, nil, true)
	}

	modeLower := strings.ToLower(stagedMode)
	if p.Mode != "open" && p.Mode != "" && (modeLower == "copy" || modeLower == "move_to" || modeLower == "pull" || modeLower == "move_from") {
		scheduleActionTrigger()
	}
}

func selectServerInCombo(srvID string) {
	if hwndCmbRemoteSrv == 0 {
		return
	}
	for i, s := range appCfg.Servers {
		if strings.EqualFold(s.ID, srvID) || strings.EqualFold(s.Name, srvID) {
			w32.SendMessage(hwndCmbRemoteSrv, 0x014E /* CB_SETCURSEL */, uintptr(i), 0)
			if s.RemotePath != "" {
				setWindowText(hwndTxtRemoteDest, s.RemotePath)
			}
			break
		}
	}
}

func updateOperationBanner() {
	if hwndLblOpBanner == 0 {
		return
	}

	switch strings.ToLower(stagedMode) {
	case "copy":
		if stagedServer != "" {
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: SERVER PUSH -> %s: %d item(s) staged. Click PUSH TO SERVER or review below.", strings.ToUpper(stagedServer), len(stagedRegistry)))
			setWindowText(hwndBtnOpAction, "PUSH TO SERVER")
		} else if stagedDest != "" && len(stagedRegistry) == 0 {
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL COPY: Dest locked: %s | Select source folder or files to copy.", stagedDest))
			setWindowText(hwndBtnOpAction, "CHOOSE SOURCE & COPY")
		} else {
			destDisplay := stagedDest
			if destDisplay == "" {
				destDisplay = "(Not chosen - click Choose Dest)"
				setWindowText(hwndBtnOpAction, "CHOOSE DEST & COPY")
			} else {
				setWindowText(hwndBtnOpAction, "START RSYNC COPY")
			}
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL COPY: %d item(s) staged -> Dest: %s", len(stagedRegistry), destDisplay))
		}

	case "move_to":
		destDisplay := stagedDest
		if destDisplay == "" {
			destDisplay = "(Not chosen - click Choose Dest)"
			setWindowText(hwndBtnOpAction, "CHOOSE DEST & MOVE")
		} else {
			setWindowText(hwndBtnOpAction, "START RSYNC MOVE")
		}
		if len(stagedRegistry) == 0 {
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL MOVE TO: Source: %s -> Dest: %s (Source deleted on completion)", stagedPath, destDisplay))
		} else {
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL MOVE TO: %d item(s) staged -> Dest: %s (Source deleted on completion)", len(stagedRegistry), destDisplay))
		}

	case "pull":
		if stagedServer != "" {
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: SERVER PULL -> %s: Destination locked: %s. Click PULL FROM SERVER.", strings.ToUpper(stagedServer), stagedDest))
			setWindowText(hwndBtnOpAction, "PULL FROM SERVER")
		} else {
			destDisplay := stagedDest
			if destDisplay == "" {
				destDisplay = stagedPath
			}
			setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL PULL: Dest locked: %s | Select source directory and sync.", destDisplay))
			setWindowText(hwndBtnOpAction, "CHOOSE SOURCE & PULL")
		}

	case "move_from":
		destDisplay := stagedDest
		if destDisplay == "" {
			destDisplay = stagedPath
		}
		setWindowText(hwndLblOpBanner, fmt.Sprintf("ACTIVE: LOCAL MOVE FROM: Dest locked: %s | Select source directory to move (Source deleted on completion).", destDisplay))
		setWindowText(hwndBtnOpAction, "CHOOSE SOURCE & MOVE")

	default:
		setWindowText(hwndLblOpBanner, fmt.Sprintf("MODE: LOCAL FILE MANAGER: %d item(s) staged across folders. Ready for Rsync transfer.", len(stagedRegistry)))
		setWindowText(hwndBtnOpAction, "EXECUTE RSYNC")
	}
}

// Universal Search implementation
func performUniversalSearch() {
	query := strings.TrimSpace(getWindowText(hwndTxtUnivSearch))
	if query == "" {
		clearListView(hwndListViewUniv)
		univResults = nil
		setStatusBarText(hwndStatusBar, 4, "UNIVERSAL SEARCH: IDLE")
		return
	}

	driveSel := int(w32.SendMessage(hwndCmbUnivDrive, 0x0147, 0, 0))
	targetDrive := "C:"
	if driveSel == 1 {
		targetDrive = "E:"
	} else if driveSel == 2 {
		targetDrive = stagedPath
	}

	setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("SEARCHING %s FOR '%s' ...", targetDrive, query))

	go func() {
		start := time.Now()
		idx, _, _, err := engineMgr.Scan(targetDrive)
		if err != nil {
			setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("SEARCH ERROR: %v", err))
			return
		}

		res := idx.SearchAll(query, 300)
		univResults = res

		elapsed := time.Since(start)
		w32.PostMessage(hwndMain, WM_APP_UNIV_COMPLETE, 0, 0)
		setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("FOUND %d MATCHES IN %v", len(res), elapsed.Round(time.Millisecond)))
	}()
}

func onUnivSearchComplete() {
	clearListView(hwndListViewUniv)

	for i, r := range univResults {
		if r.Record == nil {
			continue
		}
		fType := "File"
		if r.Record.IsDir {
			fType = "Folder"
		}

		staged := false
		if _, exists := stagedRegistry[r.FullPath]; exists {
			staged = true
		}

		name := filepath.Base(r.FullPath)
		size := int64(0)
		if r.Record != nil {
			name = r.Record.Name
			size = r.Record.Size
		}
		addListViewRow(hwndListViewUniv, int32(i), []string{
			name,
			r.FullPath,
			formatBytes(size),
			fType,
		}, staged)
	}
}

func stageCheckedUniversalItems() {
	count := int(w32.SendMessage(hwndListViewUniv, LVM_GETITEMCOUNT, 0, 0))
	added := 0

	for i := 0; i < count; i++ {
		if isListViewItemChecked(hwndListViewUniv, i) && i < len(univResults) {
			res := univResults[i]
			name := filepath.Base(res.FullPath)
			size := int64(0)
			isDir := false
			if res.Record != nil {
				name = res.Record.Name
				size = res.Record.Size
				isDir = res.Record.IsDir
			}
			addToStaged(DisplayItem{
				Name:     name,
				FullPath: res.FullPath,
				Size:     size,
				IsDir:    isDir,
			})
			added++
		}
	}

	updateOperationBanner()
	w32.MessageBox(hwndMain, fmt.Sprintf("Successfully staged %d search item(s) into Rsync queue!", added), "Items Staged", w32.MB_OK|w32.MB_ICONINFORMATION)
}

func openSelectedUniversalFileDir() {
	selected := getSelectedListViewItems(hwndListViewUniv)
	if len(selected) == 0 {
		return
	}
	res := univResults[selected[0]]
	parent := filepath.Dir(res.FullPath)
	_ = exec.Command("explorer.exe", parent).Start()
}

func runRsyncTransfer(dryRun bool) {
	var selectedPaths []string
	if len(stagedRegistry) > 0 {
		for p := range stagedRegistry {
			selectedPaths = append(selectedPaths, p)
		}
	} else if len(currentListItems) > 0 {
		count := int(w32.SendMessage(hwndListView, LVM_GETITEMCOUNT, 0, 0))
		for i := 0; i < count; i++ {
			if isListViewItemChecked(hwndListView, i) && i < len(currentListItems) {
				item := currentListItems[i]
				if !item.IsParent {
					selectedPaths = append(selectedPaths, item.FullPath)
				}
			}
		}
	}

	w32.SendMessage(hwndTabControl, TCM_SETCURSEL, 4, 0) // Tab 4 = Transfer Log
	onTabChanged()

	appendLog("\r\n========================================================\r\n")
	tag := "SYNC RUN"
	if dryRun {
		tag = "DRY RUN TEST"
	}
	appendLog(fmt.Sprintf("%s - Starting Rsync transfer at %s\r\n", tag, time.Now().Format("15:04:05")))

	if len(selectedPaths) == 0 {
		appendLog("WARN: No files explicitly staged! Using entire active folder as target source.\r\n")
		selectedPaths = []string{stagedPath}
	} else {
		appendLog(fmt.Sprintf("INFO: Syncing %d explicitly selected/checked item(s):\r\n", len(selectedPaths)))
		for _, p := range selectedPaths {
			appendLog(fmt.Sprintf("       -> %s\r\n", p))
		}
	}

	go func() {
		ctx := context.Background()
		dest := stagedDest
		if dest == "" {
			dest = `C:\Backup\Target`
		}

		var srv *core.ServerConfig
		if stagedServer != "" {
			for _, s := range appCfg.Servers {
				if strings.EqualFold(s.ID, stagedServer) || strings.EqualFold(s.Name, stagedServer) {
					srv = s
					break
				}
			}
		}

		job := &core.RsyncJob{
			ID:          fmt.Sprintf("job_%d", time.Now().Unix()),
			Sources:     selectedPaths,
			Destination: dest,
			IsPull:      (stagedMode == "pull" || stagedMode == "move_from"),
			IsMove:      (stagedMode == "move_to" || stagedMode == "move_from"),
			Server:      srv,
			CreatedAt:   time.Now(),
		}

		setStatusBarText(hwndStatusBar, 4, "TRANSFER RUNNING")

		err := core.RunRsyncWithOutput(ctx, appCfg.RsyncBinaryPath, job, func(p core.SyncProgress) {
			setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("%d%% - %s | ETA: %s", p.Percentage, p.Speed, p.ETA))
		}, func(line string) {
			l := line + "\r\n"
			w32.SendMessage(hwndMain, WM_APP_RSYNC_LOG, 0, uintptr(unsafe.Pointer(&l)))
		})

		if err != nil {
			errLine := fmt.Sprintf("\r\nFAIL: Transfer finished with error: %v\r\n", err)
			w32.SendMessage(hwndMain, WM_APP_RSYNC_LOG, 0, uintptr(unsafe.Pointer(&errLine)))
			setStatusBarText(hwndStatusBar, 4, "TRANSFER FAILED")
		} else {
			okLine := "\r\nOK: Transfer completed successfully!\r\n"
			w32.SendMessage(hwndMain, WM_APP_RSYNC_LOG, 0, uintptr(unsafe.Pointer(&okLine)))
			setStatusBarText(hwndStatusBar, 4, "TRANSFER COMPLETE")
			playSuccessChime()
			w32.PostMessage(hwndMain, WM_APP_TRANSFER_COMPLETE, 0, 0)
		}
	}()
}

func testServerPingAsync() {
	setWindowText(hwndLblPingResult, "Testing connection to configured servers...")
	go func() {
		if len(appCfg.Servers) == 0 {
			setWindowText(hwndLblPingResult, "No remote servers configured in config.json")
			return
		}
		srv := appCfg.Servers[0]
		addr := fmt.Sprintf("%s:%d", srv.Host, srv.Port)
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, time.Duration(appCfg.NetworkTimeoutMs)*time.Millisecond)
		elapsed := time.Since(start)

		if err != nil {
			setWindowText(hwndLblPingResult, fmt.Sprintf("OFFLINE: Server '%s' (%s) unreachable: %v", srv.Name, addr, err))
		} else {
			conn.Close()
			setWindowText(hwndLblPingResult, fmt.Sprintf("ONLINE: Server '%s' reachable! Latency: %v", srv.Name, elapsed.Round(time.Millisecond)))
		}
	}()
}

func appendLog(msg string) {
	curr := getWindowText(hwndEditLog)
	setWindowText(hwndEditLog, curr+msg)
	w32.SendMessage(hwndEditLog, 0x0115, 7, 0)
}

func checkServiceStateAsync() {
	running := core.IsDaemonServiceRunning()

	statusText := "SERVICE: STOPPED"
	if running {
		statusText = "SERVICE: RUNNING"
	}

	setStatusBarText(hwndStatusBar, 2, statusText)
	if hwndLblService != 0 {
		setWindowText(hwndLblService, fmt.Sprintf("Service Status: %s", statusText))
	}
}

func saveSettingsFromUI() {
	chkIcons := w32.SendMessage(hwndChkIcons, 0x00F0, 0, 0) != 0
	chkSeps := w32.SendMessage(hwndChkSeps, 0x00F0, 0, 0) != 0
	chkAccel := w32.SendMessage(hwndChkAccel, 0x00F0, 0, 0) != 0

	appCfg.Shell.ShowIcons = chkIcons
	appCfg.Shell.UseSeparators = chkSeps
	appCfg.Shell.UseAccelerators = chkAccel

	posSel := int(w32.SendMessage(hwndCmbPosition, 0x0147, 0, 0))
	switch posSel {
	case 1:
		appCfg.Shell.Position = "Top"
	case 2:
		appCfg.Shell.Position = "Bottom"
	default:
		appCfg.Shell.Position = "Default"
	}

	engSel := int(w32.SendMessage(hwndCmbEngine, 0x0147, 0, 0))
	if engSel == 1 {
		appCfg.PreferredIndexer = core.EngineCrawler
	} else {
		appCfg.PreferredIndexer = core.EngineUSN
	}

	themeSel := int(w32.SendMessage(hwndCmbTheme, 0x0147, 0, 0))
	if themeSel >= 0 && themeSel < len(availableThemes) {
		appCfg.ActiveTheme = availableThemes[themeSel]
		switchTheme(hwndMain, appCfg.ActiveTheme)
		updateThemeButtonLabel()
	}

	radSel := int(w32.SendMessage(hwndCmbRadius, 0x0147, 0, 0))
	switch radSel {
	case 0:
		appCfg.CornerRadius = 0
	case 1:
		appCfg.CornerRadius = 4
	case 3:
		appCfg.CornerRadius = 14
	default:
		appCfg.CornerRadius = 8
	}
	updateCornerRadius(hwndMain, appCfg.CornerRadius)

	shadowSel := int(w32.SendMessage(hwndCmbShadow, 0x0147, 0, 0))
	switch shadowSel {
	case 0:
		appCfg.ShadowDepth = 0
	case 1:
		appCfg.ShadowDepth = 1
	case 3:
		appCfg.ShadowDepth = 4
	case 4:
		appCfg.ShadowDepth = 6
	default:
		appCfg.ShadowDepth = 2
	}
	updateShadowDepth(hwndMain, appCfg.ShadowDepth)

	_ = core.SaveConfig("./config.json", appCfg)

	exe, _ := os.Executable()
	_ = core.RegisterShellMenus(exe, appCfg)

	if !core.IsSelfElevated() {
		rsCopyExe, _ := filepath.Abs("rs_copy.exe")
		if _, err := os.Stat(rsCopyExe); err == nil {
			psCmd := fmt.Sprintf(`Start-Process -FilePath '%s' -ArgumentList '--repair' -Verb RunAs -Wait`, rsCopyExe)
			_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Start()
		}
	}

	appendLog("INFO: Settings saved and Explorer context menus refreshed.\r\n")
	setStatusBarText(hwndStatusBar, 4, "SETTINGS SAVED")

	w32.MessageBox(hwndMain, "Settings have been successfully applied and Windows Explorer context menus updated!", "Rsync Controller - Settings Saved", w32.MB_OK|w32.MB_ICONINFORMATION)
}

func setComboThemeIndex(hwnd w32.HWND, themeID string) {
	for i, t := range availableThemes {
		if strings.EqualFold(t, themeID) {
			w32.SendMessage(hwnd, 0x014E, uintptr(i), 0)
			return
		}
	}
	w32.SendMessage(hwnd, 0x014E, 0, 0)
}

func showThemePopupMenu(hwndParent w32.HWND, btnHwnd w32.HWND) {
	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		switchNextTheme()
		return
	}
	defer w32.DestroyMenu(hMenu)

	for i, t := range allThemeMetas {
		flags := uint(w32.MF_STRING)
		if strings.EqualFold(t.ID, appCfg.ActiveTheme) {
			flags |= w32.MF_CHECKED
		}
		pName, _ := windows.UTF16PtrFromString(t.Name)
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(flags), uintptr(1000+i), uintptr(unsafe.Pointer(pName)))
	}

	// Separator and Direct Customizer option
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_SEPARATOR), 0, 0)
	pCustomOpt, _ := windows.UTF16PtrFromString("Customize Palette & Shadows...")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), uintptr(2000), uintptr(unsafe.Pointer(pCustomOpt)))

	rc := w32.GetWindowRect(btnHwnd)
	if rc == nil {
		switchNextTheme()
		return
	}

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002), // TPM_RETURNCMD | TPM_RIGHTBUTTON
		uintptr(rc.Left),
		uintptr(rc.Bottom),
		0,
		uintptr(hwndParent),
		0,
	)

	if cmd == 2000 {
		showThemeCustomizerMenu(hwndParent, btnHwnd)
		return
	}

	selIdx := int(cmd) - 1000
	if selIdx >= 0 && selIdx < len(allThemeMetas) {
		chosenID := allThemeMetas[selIdx].ID
		appCfg.ActiveTheme = chosenID
		switchTheme(hwndMain, chosenID)
		setComboThemeIndex(hwndCmbTheme, chosenID)
		updateThemeButtonLabel()
		setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("THEME: %s APPLIED", allThemeMetas[selIdx].Name))
		_ = core.SaveConfig("./config.json", appCfg)
	}
}

func showThemeCustomizerMenu(hwndOwner w32.HWND, btnHwnd w32.HWND) {
	if appCfg.CustomTheme == nil {
		appCfg.CustomTheme = core.DefaultCustomTheme()
	}

	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer w32.DestroyMenu(hMenu)

	hSubShadow := w32.CreatePopupMenu()
	shadowDepths := []struct {
		depth int
		label string
	}{
		{0, "0px (Flat - No Shadow)"},
		{1, "1px (Subtle Depth)"},
		{2, "2px (Standard Tactile)"},
		{4, "4px (Bold 3D Retro)"},
		{6, "6px (Deep Comic Shadow)"},
	}
	for i, sd := range shadowDepths {
		flags := uint(w32.MF_STRING)
		if appCfg.ShadowDepth == sd.depth {
			flags |= w32.MF_CHECKED
		}
		pName, _ := windows.UTF16PtrFromString(sd.label)
		_, _, _ = procAppendMenuW.Call(uintptr(hSubShadow), uintptr(flags), uintptr(300+i), uintptr(unsafe.Pointer(pName)))
	}

	hSubRadius := w32.CreatePopupMenu()
	radiuses := []struct {
		rad   int
		label string
	}{
		{0, "0px (Sharp Retro 80s/90s)"},
		{4, "4px (Subtle Modern)"},
		{8, "8px (Rounded Y2K)"},
		{14, "14px (Full Pill Aqua)"},
		{20, "20px (Extra Round Bubble)"},
	}
	for i, rd := range radiuses {
		flags := uint(w32.MF_STRING)
		if appCfg.CornerRadius == rd.rad {
			flags |= w32.MF_CHECKED
		}
		pName, _ := windows.UTF16PtrFromString(rd.label)
		_, _, _ = procAppendMenuW.Call(uintptr(hSubRadius), uintptr(flags), uintptr(400+i), uintptr(unsafe.Pointer(pName)))
	}

	appendMenuItem := func(id uint, text string) {
		pText, _ := windows.UTF16PtrFromString(text)
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), uintptr(id), uintptr(unsafe.Pointer(pText)))
	}
	appendSeparator := func() {
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_SEPARATOR), 0, 0)
	}

	appendMenuItem(101, fmt.Sprintf("Canvas Background Color... (#%s)", appCfg.CustomTheme.CanvasBg))
	appendMenuItem(102, fmt.Sprintf("Window Surface Color... (#%s)", appCfg.CustomTheme.SurfaceBg))
	appendMenuItem(103, fmt.Sprintf("Button Background Color... (#%s)", appCfg.CustomTheme.ButtonBg))
	appendMenuItem(104, fmt.Sprintf("Button Text Color... (#%s)", appCfg.CustomTheme.ButtonText))
	appendMenuItem(105, fmt.Sprintf("Accent / Highlight Color... (#%s)", appCfg.CustomTheme.AccentColor))
	appendMenuItem(106, fmt.Sprintf("Window Border Color... (#%s)", appCfg.CustomTheme.BorderColor))
	appendSeparator()

	pShadowTitle, _ := windows.UTF16PtrFromString("Button Shadow Depth")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_POPUP), uintptr(hSubShadow), uintptr(unsafe.Pointer(pShadowTitle)))

	pRadiusTitle, _ := windows.UTF16PtrFromString("Corner Radius")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_POPUP), uintptr(hSubRadius), uintptr(unsafe.Pointer(pRadiusTitle)))

	appendSeparator()
	appendMenuItem(107, "Reset Palette to Default")

	rc := w32.GetWindowRect(btnHwnd)
	x, y := 100, 100
	if rc != nil {
		x = int(rc.Left)
		y = int(rc.Bottom)
	}

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002), // TPM_RETURNCMD | TPM_RIGHTBUTTON
		uintptr(x),
		uintptr(y),
		0,
		uintptr(hwndOwner),
		0,
	)

	switch {
	case cmd == 101:
		curVal, _ := parseHexColor(appCfg.CustomTheme.CanvasBg)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.CanvasBg = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd == 102:
		curVal, _ := parseHexColor(appCfg.CustomTheme.SurfaceBg)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.SurfaceBg = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd == 103:
		curVal, _ := parseHexColor(appCfg.CustomTheme.ButtonBg)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.ButtonBg = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd == 104:
		curVal, _ := parseHexColor(appCfg.CustomTheme.ButtonText)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.ButtonText = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd == 105:
		curVal, _ := parseHexColor(appCfg.CustomTheme.AccentColor)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.AccentColor = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd == 106:
		curVal, _ := parseHexColor(appCfg.CustomTheme.BorderColor)
		if chosen, ok := chooseColorDialog(hwndOwner, curVal); ok {
			appCfg.CustomTheme.BorderColor = colorToHex(chosen)
			applyCustomThemeLive(hwndOwner)
		}
	case cmd >= 300 && cmd < 300+uintptr(len(shadowDepths)):
		idx := int(cmd - 300)
		appCfg.ShadowDepth = shadowDepths[idx].depth
		updateShadowDepth(hwndOwner, appCfg.ShadowDepth)
		if hwndCmbShadow != 0 {
			w32.SendMessage(hwndCmbShadow, 0x014E, uintptr(idx), 0)
		}
		_ = core.SaveConfig("./config.json", appCfg)
	case cmd >= 400 && cmd < 400+uintptr(len(radiuses)):
		idx := int(cmd - 400)
		appCfg.CornerRadius = radiuses[idx].rad
		updateCornerRadius(hwndOwner, appCfg.CornerRadius)
		if hwndCmbRadius != 0 && idx < 4 {
			w32.SendMessage(hwndCmbRadius, 0x014E, uintptr(idx), 0)
		}
		_ = core.SaveConfig("./config.json", appCfg)
	case cmd == 107:
		resetCustomTheme(hwndOwner)
	}
}

func applyCustomThemeLive(hwndOwner w32.HWND) {
	appCfg.ActiveTheme = "custom"
	switchTheme(hwndOwner, "custom")
	setComboThemeIndex(hwndCmbTheme, "custom")
	updateThemeButtonLabel()
	setStatusBarText(hwndStatusBar, 4, "CUSTOM THEME PALETTE APPLIED")
	_ = core.SaveConfig("./config.json", appCfg)
}

func resetCustomTheme(hwndOwner w32.HWND) {
	appCfg.CustomTheme = core.DefaultCustomTheme()
	applyCustomThemeLive(hwndOwner)
	setStatusBarText(hwndStatusBar, 4, "PALETTE RESET TO DEFAULTS")
}

func switchNextTheme() {
	themes := availableThemes
	curr := appCfg.ActiveTheme
	next := themes[0]
	for i, t := range themes {
		if strings.EqualFold(t, curr) {
			next = themes[(i+1)%len(themes)]
			break
		}
	}
	appCfg.ActiveTheme = next
	switchTheme(hwndMain, next)
	setComboThemeIndex(hwndCmbTheme, next)
	updateThemeButtonLabel()
	setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("THEME: %s APPLIED", next))
	_ = core.SaveConfig("./config.json", appCfg)
}

func updateThemeButtonLabel() {
	if hwndBtnThemeToggle == 0 {
		return
	}
	btnTag := "WIN95"
	for _, tm := range allThemeMetas {
		if strings.EqualFold(tm.ID, appCfg.ActiveTheme) {
			btnTag = tm.BtnLabel
			break
		}
	}
	setWindowText(hwndBtnThemeToggle, "THEME: "+btnTag+" ▾")
}

func updateDriveSpaceStatus() {
	if hwndStatusBar != 0 {
		setStatusBarText(hwndStatusBar, 0, getDriveSpaceSummary(stagedPath))
	}
}

func showJumpMenu(btnHwnd w32.HWND) {
	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer w32.DestroyMenu(hMenu)

	homeDir, _ := os.UserHomeDir()
	desktopDir := filepath.Join(homeDir, "Desktop")
	downloadsDir := filepath.Join(homeDir, "Downloads")
	documentsDir := filepath.Join(homeDir, "Documents")

	type jumpTarget struct {
		Label string
		Path  string
	}

	targets := []jumpTarget{
		{"User Home (" + filepath.Base(homeDir) + ")", homeDir},
		{"Desktop", desktopDir},
		{"Downloads", downloadsDir},
		{"Documents", documentsDir},
	}

	drives := getSystemDrives()
	for _, drv := range drives {
		targets = append(targets, jumpTarget{
			Label: "Drive " + drv,
			Path:  drv,
		})
	}

	for i, t := range targets {
		pText, _ := windows.UTF16PtrFromString(t.Label)
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), uintptr(2000+i), uintptr(unsafe.Pointer(pText)))
	}

	rc := w32.GetWindowRect(btnHwnd)
	if rc == nil {
		return
	}

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002), // TPM_RETURNCMD | TPM_RIGHTBUTTON
		uintptr(rc.Left),
		uintptr(rc.Bottom),
		0,
		uintptr(hwndMain),
		0,
	)

	sel := int(cmd) - 2000
	if sel >= 0 && sel < len(targets) {
		targetPath := targets[sel].Path
		if _, err := os.Stat(targetPath); err == nil {
			navigateTo(targetPath, true)
		}
	}
}

func showListViewContextMenu(hwndLV w32.HWND) {
	sel := int(w32.SendMessage(hwndLV, LVM_GETNEXTITEM, ^uintptr(0), uintptr(LVNI_SELECTED)))
	if sel < 0 || sel >= len(currentListItems) {
		return
	}
	item := currentListItems[sel]

	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer w32.DestroyMenu(hMenu)

	stageLabel := "Stage Item for Sync"
	if _, ok := stagedRegistry[item.FullPath]; ok {
		stageLabel = "Unstage Item"
	}
	pStage, _ := windows.UTF16PtrFromString(stageLabel)
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3001, uintptr(unsafe.Pointer(pStage)))

	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_SEPARATOR), 0, 0)

	pCopyAction, _ := windows.UTF16PtrFromString("Copy to... (Stage & Copy)")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3004, uintptr(unsafe.Pointer(pCopyAction)))

	pMoveAction, _ := windows.UTF16PtrFromString("Move to... (Stage & Move)")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3005, uintptr(unsafe.Pointer(pMoveAction)))

	pPullAction, _ := windows.UTF16PtrFromString("Pull from... (Pull into this folder)")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3006, uintptr(unsafe.Pointer(pPullAction)))

	pMoveFromAction, _ := windows.UTF16PtrFromString("Move from... (Move into this folder)")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3007, uintptr(unsafe.Pointer(pMoveFromAction)))

	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_SEPARATOR), 0, 0)

	pExplorer, _ := windows.UTF16PtrFromString("Reveal in Windows Explorer")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3002, uintptr(unsafe.Pointer(pExplorer)))

	pCopy, _ := windows.UTF16PtrFromString("Copy Full Path")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3003, uintptr(unsafe.Pointer(pCopy)))

	ptX, ptY, _ := w32.GetCursorPos()

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002),
		uintptr(ptX),
		uintptr(ptY),
		0,
		uintptr(hwndMain),
		0,
	)

	switch cmd {
	case 3001:
		if _, ok := stagedRegistry[item.FullPath]; ok {
			removeFromStaged(item.FullPath)
		} else {
			addToStaged(item)
		}
		syncCurrentListViewChecks()
		updateOperationBanner()
	case 3004:
		addToStaged(item)
		syncCurrentListViewChecks()
		stagedMode = "copy"
		updateModeButtonLabel()
		updateOperationBanner()
		executeContextAction()
	case 3005:
		addToStaged(item)
		syncCurrentListViewChecks()
		stagedMode = "move_to"
		updateModeButtonLabel()
		updateOperationBanner()
		executeContextAction()
	case 3006:
		stagedDest = item.FullPath
		if !item.IsDir {
			stagedDest = filepath.Dir(item.FullPath)
		}
		stagedMode = "pull"
		updateModeButtonLabel()
		updateOperationBanner()
		executeContextAction()
	case 3007:
		stagedDest = item.FullPath
		if !item.IsDir {
			stagedDest = filepath.Dir(item.FullPath)
		}
		stagedMode = "move_from"
		updateModeButtonLabel()
		updateOperationBanner()
		executeContextAction()
	case 3002:
		revealInExplorer(item.FullPath)
	case 3003:
		setClipboardText(hwndMain, item.FullPath)
		setStatusBarText(hwndStatusBar, 4, "PATH COPIED TO CLIPBOARD")
	}
}

func getModeButtonText() string {
	switch strings.ToLower(stagedMode) {
	case "move_to":
		return "MODE: MOVE TO ▾"
	case "pull":
		return "MODE: PULL FROM ▾"
	case "move_from":
		return "MODE: MOVE FROM ▾"
	default:
		return "MODE: COPY TO ▾"
	}
}

func updateModeButtonLabel() {
	if hwndBtnMode != 0 {
		setWindowText(hwndBtnMode, getModeButtonText())
	}
}

func showModePopupMenu(btnHwnd w32.HWND) {
	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer w32.DestroyMenu(hMenu)

	type modeOption struct {
		ID    uintptr
		Mode  string
		Label string
	}

	modes := []modeOption{
		{3201, "copy", "Copy to... (Copy staged files to destination)"},
		{3202, "move_to", "Move to... (Move staged files to destination, delete source)"},
		{3203, "pull", "Pull from... (Pull files from source into active folder)"},
		{3204, "move_from", "Move from... (Move files from source into active folder, delete source)"},
	}

	curr := strings.ToLower(stagedMode)
	for _, m := range modes {
		flags := uint(w32.MF_STRING)
		if curr == m.Mode {
			flags |= w32.MF_CHECKED
		}
		pLabel, _ := windows.UTF16PtrFromString(m.Label)
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(flags), m.ID, uintptr(unsafe.Pointer(pLabel)))
	}

	rc := w32.GetWindowRect(btnHwnd)
	if rc == nil {
		return
	}

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002|0x0020), // TPM_RETURNCMD | TPM_RIGHTBUTTON | TPM_BOTTOMALIGN
		uintptr(rc.Left),
		uintptr(rc.Top),
		0,
		uintptr(hwndMain),
		0,
	)

	switch cmd {
	case 3201:
		stagedMode = "copy"
	case 3202:
		stagedMode = "move_to"
	case 3203:
		stagedMode = "pull"
	case 3204:
		stagedMode = "move_from"
	default:
		return
	}

	updateModeButtonLabel()
	updateOperationBanner()
	if hwndMain != 0 {
		w32.InvalidateRect(hwndMain, nil, true)
	}
}

func showStagedContextMenu(hwndLV w32.HWND) {
	sel := int(w32.SendMessage(hwndLV, LVM_GETNEXTITEM, ^uintptr(0), uintptr(LVNI_SELECTED)))
	hMenu := w32.CreatePopupMenu()
	if hMenu == 0 {
		return
	}
	defer w32.DestroyMenu(hMenu)

	var selectedItem *DisplayItem
	if sel >= 0 && sel < len(stagedListPaths) {
		p := stagedListPaths[sel]
		if it, ok := stagedRegistry[p]; ok {
			selectedItem = &it
		}
	}

	if selectedItem != nil {
		pUnstage, _ := windows.UTF16PtrFromString("Unstage " + selectedItem.Name)
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3101, uintptr(unsafe.Pointer(pUnstage)))

		pJump, _ := windows.UTF16PtrFromString("Jump to Folder in Explorer")
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3102, uintptr(unsafe.Pointer(pJump)))

		pReveal, _ := windows.UTF16PtrFromString("Reveal in Windows Explorer")
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3103, uintptr(unsafe.Pointer(pReveal)))

		pCopy, _ := windows.UTF16PtrFromString("Copy Full Path")
		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3104, uintptr(unsafe.Pointer(pCopy)))

		_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_SEPARATOR), 0, 0)
	}

	pCopyAll, _ := windows.UTF16PtrFromString("Copy All Staged Paths")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3105, uintptr(unsafe.Pointer(pCopyAll)))

	pClear, _ := windows.UTF16PtrFromString("Clear All Staged Items")
	_, _, _ = procAppendMenuW.Call(uintptr(hMenu), uintptr(w32.MF_STRING), 3106, uintptr(unsafe.Pointer(pClear)))

	ptX, ptY, _ := w32.GetCursorPos()

	cmd, _, _ := procTrackPopupMenu.Call(
		uintptr(hMenu),
		uintptr(0x0100|0x0002),
		uintptr(ptX),
		uintptr(ptY),
		0,
		uintptr(hwndMain),
		0,
	)

	switch cmd {
	case 3101:
		if selectedItem != nil {
			removeFromStaged(selectedItem.FullPath)
			syncCurrentListViewChecks()
			updateOperationBanner()
		}
	case 3102:
		if selectedItem != nil {
			dir := filepath.Dir(selectedItem.FullPath)
			if selectedItem.IsDir {
				dir = selectedItem.FullPath
			}
			navigateTo(dir, true)
		}
	case 3103:
		if selectedItem != nil {
			revealInExplorer(selectedItem.FullPath)
		}
	case 3104:
		if selectedItem != nil {
			setClipboardText(hwndMain, selectedItem.FullPath)
			setStatusBarText(hwndStatusBar, 4, "PATH COPIED TO CLIPBOARD")
		}
	case 3105:
		var sb strings.Builder
		for _, p := range stagedListPaths {
			sb.WriteString(p + "\r\n")
		}
		setClipboardText(hwndMain, sb.String())
		setStatusBarText(hwndStatusBar, 4, fmt.Sprintf("%d STAGED PATHS COPIED", len(stagedListPaths)))
	case 3106:
		clearStaged()
		updateOperationBanner()
	}
}

func revealInExplorer(targetPath string) {
	clean := filepath.Clean(targetPath)
	if _, err := os.Stat(clean); err != nil {
		return
	}
	go func() {
		_ = exec.Command("explorer.exe", "/select,"+clean).Start()
	}()
}

func isEditHwnd(h w32.HWND) bool {
	if h == hwndTxtTarget || h == hwndTxtFilter || h == hwndTxtTopUnivSearch || h == hwndTxtUnivSearch || h == hwndTxtRemoteDest || h == hwndEditLog {
		return true
	}
	var className [64]uint16
	procGetClassNameW.Call(uintptr(h), uintptr(unsafe.Pointer(&className[0])), 64)
	cls := strings.ToLower(windows.UTF16ToString(className[:]))
	return strings.Contains(cls, "edit")
}

func handleGlobalKeydown(wParam uintptr, focusedHwnd w32.HWND) bool {
	isCtrl := (w32.GetKeyState(w32.VK_CONTROL) & 0x8000) != 0
	isAlt := (w32.GetKeyState(w32.VK_MENU) & 0x8000) != 0

	switch wParam {
	case 116: // VK_F5
		triggerScanAsync(stagedPath)
		return true

	case 8: // VK_BACK
		if !isEditHwnd(focusedHwnd) {
			navUp()
			return true
		}

	case 37: // VK_LEFT
		if isAlt {
			navBack()
			return true
		}

	case 39: // VK_RIGHT
		if isAlt {
			navForward()
			return true
		}

	case 46: // VK_DELETE
		if focusedHwnd == hwndListViewStaged {
			onCommand(hwndMain, ID_BTN_DRAWER_REMOVE, 0)
			return true
		} else if focusedHwnd == hwndListView && !isEditHwnd(focusedHwnd) {
			onCommand(hwndMain, ID_BTN_DRAWER_REMOVE, 0)
			return true
		}

	case 65: // 'A' with Ctrl
		if isCtrl && (focusedHwnd == hwndListView || !isEditHwnd(focusedHwnd)) {
			onCommand(hwndMain, ID_BTN_SELECTALL, 0)
			return true
		}

	case 70: // 'F' with Ctrl
		if isCtrl {
			w32.SetFocus(hwndTxtFilter)
			w32.SendMessage(hwndTxtFilter, 0x00B1 /* EM_SETSEL */, 0, ^uintptr(0))
			return true
		}

	case 68: // 'D' with Ctrl
		if isCtrl {
			onCommand(hwndMain, ID_BTN_DRYRUN, 0)
			return true
		}

	case 69: // 'E' with Ctrl
		if isCtrl {
			onCommand(hwndMain, ID_BTN_EXECUTE, 0)
			return true
		}

	case 32: // VK_SPACE
		if focusedHwnd == hwndListView && !isEditHwnd(focusedHwnd) {
			sel := int(w32.SendMessage(hwndListView, LVM_GETNEXTITEM, ^uintptr(0), uintptr(LVNI_SELECTED)))
			if sel >= 0 && sel < len(currentListItems) {
				item := currentListItems[sel]
				if _, ok := stagedRegistry[item.FullPath]; ok {
					removeFromStaged(item.FullPath)
				} else {
					addToStaged(item)
				}
				syncCurrentListViewChecks()
				updateOperationBanner()
				return true
			}
		}
	}
	return false
}

func createButton(text string, style uint, x, y, w, h int, parent w32.HWND, id int, hInst w32.HINSTANCE) w32.HWND {
	btnStyle := (style &^ (w32.BS_PUSHBUTTON | w32.BS_DEFPUSHBUTTON)) | w32.BS_OWNERDRAW
	return createControl("BUTTON", text, btnStyle, x, y, w, h, parent, id, hInst)
}

func createControl(className, text string, style uint, x, y, w, h int, parent w32.HWND, id int, hInst w32.HINSTANCE) w32.HWND {
	pClass, _ := windows.UTF16PtrFromString(className)
	pText, _ := windows.UTF16PtrFromString(text)
	return w32.CreateWindowEx(
		0,
		pClass,
		pText,
		style,
		x, y, w, h,
		parent,
		w32.HMENU(uintptr(id)),
		hInst,
		nil,
	)
}

func addComboItem(hwnd w32.HWND, text string) {
	pText, _ := windows.UTF16PtrFromString(text)
	w32.SendMessage(hwnd, 0x0143, 0, uintptr(unsafe.Pointer(pText)))
}

func setWindowText(hwnd w32.HWND, text string) {
	w32.SetWindowText(hwnd, text)
}

func getWindowText(hwnd w32.HWND) string {
	return w32.GetWindowText(hwnd)
}
