//go:build windows

package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"

	"rsync-controller/core"
)

var (
	dwmapiDLL                 = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapiDLL.NewProc("DwmSetWindowAttribute")
	uxthemeDLL                = windows.NewLazySystemDLL("uxtheme.dll")
	procSetWindowTheme        = uxthemeDLL.NewProc("SetWindowTheme")
)

func setWindowTheme(hwnd w32.HWND, appName, idList string) {
	if hwnd == 0 {
		return
	}
	var pApp, pID *uint16
	if appName != "" {
		pApp, _ = windows.UTF16PtrFromString(appName)
	} else {
		empty := uint16(0)
		pApp = &empty
	}
	if idList != "" {
		pID, _ = windows.UTF16PtrFromString(idList)
	} else {
		empty := uint16(0)
		pID = &empty
	}
	_, _, _ = procSetWindowTheme.Call(uintptr(hwnd), uintptr(unsafe.Pointer(pApp)), uintptr(unsafe.Pointer(pID)))
}

func applyUxThemeMode(hwnd w32.HWND, theme *ThemeStyle) {
	if hwnd == 0 || theme == nil {
		return
	}
	if theme.ID == "dark-obsidian" {
		setWindowTheme(hwnd, "DarkMode_Explorer", "")
	} else if theme.ID == "win95-classic" || theme.ID == "monochrome-mac" || theme.ID == "vaporwave-cyber" || theme.ID == "y2k-candy" {
		setWindowTheme(hwnd, "", "")
	} else {
		setWindowTheme(hwnd, "Explorer", "")
	}
}

type ThemeStyle struct {
	ID            string
	Name          string
	CanvasBg      uint32 // COLORREF
	SurfaceBg     uint32 // COLORREF
	TextColor     uint32 // COLORREF
	AccentColor   uint32 // COLORREF
	AccentText    uint32 // COLORREF
	BorderColor   uint32 // COLORREF
	ButtonBg      uint32 // COLORREF
	ButtonText    uint32 // COLORREF
	ButtonBorder  uint32 // COLORREF
	HighlightBg   uint32 // COLORREF
	HighlightText uint32 // COLORREF
	DarkTitlebar  bool
	Radius        int
	ShadowDepth   int    // 0=flat, 1=1px, 2=2px, 4=4px, 6=6px
	ShadowColor   uint32 // COLORREF
	FontName      string

	// GDI Cached Brushes
	CanvasBrush    w32.HBRUSH
	SurfaceBrush   w32.HBRUSH
	AccentBrush    w32.HBRUSH
	ButtonBrush    w32.HBRUSH
	HighlightBrush w32.HBRUSH
	BorderBrush    w32.HBRUSH
	FontNormal     w32.HFONT
	FontBold       w32.HFONT
	FontHeader     w32.HFONT
}

var (
	currentTheme *ThemeStyle
	availableThemes = []string{
		"win95-classic",
		"dark-obsidian",
		"y2k-candy",
		"vaporwave-cyber",
		"monochrome-mac",
		"aqua-win32",
		"warm-handdrawn",
		"pastel-grid",
		"nostalgic-media",
		"y2k-pink",
		"retrowave-pc",
		"win95-figma",
		"y2k-blue",
		"pixel-pc",
		"music-chat",
		"player-clock",
		"dialog-green",
		"dialog-sunset",
		"custom",
	}
)

func RGB(r, g, b uint8) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

func initTheme(themeID string, customRadius int, customShadow int) *ThemeStyle {
	t := &ThemeStyle{}
	t.ShadowDepth = 2

	switch themeID {
	case "dark-obsidian":
		t.ID = "dark-obsidian"
		t.Name = "Dark Obsidian (Cyber Night)"
		t.CanvasBg = RGB(24, 24, 37)       // #181825
		t.SurfaceBg = RGB(37, 37, 56)      // #252538
		t.TextColor = RGB(248, 250, 252)   // #F8FAFC
		t.AccentColor = RGB(0, 120, 215)   // #0078D7
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(69, 71, 90)    // #45475A
		t.ButtonBg = RGB(42, 42, 62)       // #2A2A3E
		t.ButtonText = RGB(248, 250, 252)
		t.ButtonBorder = RGB(69, 71, 90)
		t.HighlightBg = RGB(0, 120, 215)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = true
		t.Radius = 4
		t.FontName = "Segoe UI"

	case "y2k-candy":
		t.ID = "y2k-candy"
		t.Name = "Y2K Cyber Candy (Retro Gel)"
		t.CanvasBg = RGB(255, 238, 245)    // #FFEEF5 Bubblegum Pink
		t.SurfaceBg = RGB(255, 255, 255)   // #FFFFFF Crisp White
		t.TextColor = RGB(26, 26, 46)      // #1A1A2E Deep Midnight Navy
		t.AccentColor = RGB(214, 0, 110)   // #D6006E Vivid Candy Raspberry
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(26, 26, 46)    // #1A1A2E Deep Midnight Navy
		t.ButtonBg = RGB(255, 255, 255)    // #FFFFFF
		t.ButtonText = RGB(26, 26, 46)     // #1A1A2E
		t.ButtonBorder = RGB(26, 26, 46)
		t.HighlightBg = RGB(0, 240, 255)   // #00F0FF Electric Cyan
		t.HighlightText = RGB(26, 26, 46)
		t.DarkTitlebar = false
		t.Radius = 8
		t.FontName = "Segoe UI"

	case "vaporwave-cyber":
		t.ID = "vaporwave-cyber"
		t.Name = "Cyber Vaporwave (Neon 80s)"
		t.CanvasBg = RGB(24, 9, 40)        // #180928
		t.SurfaceBg = RGB(13, 4, 23)       // #0D0417
		t.TextColor = RGB(0, 240, 255)     // #00F0FF
		t.AccentColor = RGB(255, 0, 127)   // #FF007F
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(121, 40, 202)  // #7928CA
		t.ButtonBg = RGB(37, 13, 62)       // #250D3E
		t.ButtonText = RGB(0, 240, 255)
		t.ButtonBorder = RGB(255, 0, 127)
		t.HighlightBg = RGB(255, 0, 127)
		t.HighlightText = RGB(0, 0, 0)
		t.DarkTitlebar = true
		t.Radius = 8
		t.FontName = "Consolas"

	case "monochrome-mac":
		t.ID = "monochrome-mac"
		t.Name = "Monochrome 1-Bit (Retro Classic)"
		t.CanvasBg = RGB(255, 255, 255)    // #FFFFFF
		t.SurfaceBg = RGB(255, 255, 255)   // #FFFFFF
		t.TextColor = RGB(0, 0, 0)         // #000000
		t.AccentColor = RGB(0, 0, 0)       // #000000
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(0, 0, 0)
		t.ButtonBg = RGB(255, 255, 255)
		t.ButtonText = RGB(0, 0, 0)
		t.ButtonBorder = RGB(0, 0, 0)
		t.HighlightBg = RGB(0, 0, 0)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 0
		t.FontName = "Courier New"

	case "aqua-win32":
		t.ID = "aqua-win32"
		t.Name = "Y2K Aqua (Win32 Teal & Cerulean)"
		t.CanvasBg = RGB(192, 192, 192)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(0, 0, 0)
		t.AccentColor = RGB(10, 36, 106)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(3, 105, 161)
		t.ButtonBg = RGB(212, 208, 200)
		t.ButtonText = RGB(0, 0, 0)
		t.ButtonBorder = RGB(3, 105, 161)
		t.HighlightBg = RGB(0, 128, 128)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 4
		t.FontName = "Tahoma"

	case "warm-handdrawn":
		t.ID = "warm-handdrawn"
		t.Name = "Warm Handdrawn Retro PC (Parchment & Amber)"
		t.CanvasBg = RGB(255, 250, 240)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(74, 53, 37)
		t.AccentColor = RGB(245, 158, 11)
		t.AccentText = RGB(34, 25, 20)
		t.BorderColor = RGB(74, 53, 37)
		t.ButtonBg = RGB(254, 243, 199)
		t.ButtonText = RGB(74, 53, 37)
		t.ButtonBorder = RGB(74, 53, 37)
		t.HighlightBg = RGB(245, 158, 11)
		t.HighlightText = RGB(34, 25, 20)
		t.DarkTitlebar = false
		t.Radius = 4
		t.FontName = "Segoe UI"

	case "pastel-grid":
		t.ID = "pastel-grid"
		t.Name = "Pastel Retro Grid (Pink & Lavender)"
		t.CanvasBg = RGB(254, 240, 138)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(49, 46, 129)
		t.AccentColor = RGB(236, 72, 153)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(219, 39, 119)
		t.ButtonBg = RGB(251, 207, 232)
		t.ButtonText = RGB(131, 24, 67)
		t.ButtonBorder = RGB(219, 39, 119)
		t.HighlightBg = RGB(236, 72, 153)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 6
		t.FontName = "Segoe UI"

	case "nostalgic-media":
		t.ID = "nostalgic-media"
		t.Name = "Nostalgic Media Popup UI (Cobalt & Ivory)"
		t.CanvasBg = RGB(253, 246, 226)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(30, 58, 138)
		t.AccentColor = RGB(59, 130, 246)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(29, 78, 216)
		t.ButtonBg = RGB(224, 242, 254)
		t.ButtonText = RGB(30, 58, 138)
		t.ButtonBorder = RGB(59, 130, 246)
		t.HighlightBg = RGB(59, 130, 246)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 4
		t.FontName = "Tahoma"

	case "y2k-pink":
		t.ID = "y2k-pink"
		t.Name = "Pink & Yellow Y2K Pop (Raspberry & Lemon)"
		t.CanvasBg = RGB(254, 240, 138)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(136, 19, 55)
		t.AccentColor = RGB(236, 72, 153)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(190, 18, 60)
		t.ButtonBg = RGB(252, 231, 243)
		t.ButtonText = RGB(136, 19, 55)
		t.ButtonBorder = RGB(236, 72, 153)
		t.HighlightBg = RGB(13, 148, 136)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 8
		t.FontName = "Segoe UI"

	case "retrowave-pc":
		t.ID = "retrowave-pc"
		t.Name = "Retrowave Vintage PC (Hot Magenta & Neon)"
		t.CanvasBg = RGB(34, 0, 68)
		t.SurfaceBg = RGB(17, 0, 34)
		t.TextColor = RGB(0, 255, 255)
		t.AccentColor = RGB(255, 0, 255)
		t.AccentText = RGB(255, 255, 0)
		t.BorderColor = RGB(255, 0, 255)
		t.ButtonBg = RGB(51, 0, 102)
		t.ButtonText = RGB(255, 255, 0)
		t.ButtonBorder = RGB(255, 0, 255)
		t.HighlightBg = RGB(255, 0, 255)
		t.HighlightText = RGB(255, 255, 0)
		t.DarkTitlebar = true
		t.Radius = 6
		t.FontName = "Consolas"

	case "win95-figma":
		t.ID = "win95-figma"
		t.Name = "Classic Win95/98 Teal Desktop"
		t.CanvasBg = RGB(0, 128, 128)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(0, 0, 0)
		t.AccentColor = RGB(0, 0, 128)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(128, 128, 128)
		t.ButtonBg = RGB(192, 192, 192)
		t.ButtonText = RGB(0, 0, 0)
		t.ButtonBorder = RGB(0, 0, 0)
		t.HighlightBg = RGB(0, 0, 128)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 0
		t.FontName = "Tahoma"

	case "y2k-blue":
		t.ID = "y2k-blue"
		t.Name = "Y2K Classic Desktop Blue Folders"
		t.CanvasBg = RGB(248, 250, 252)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(15, 23, 42)
		t.AccentColor = RGB(37, 99, 235)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(29, 78, 216)
		t.ButtonBg = RGB(224, 242, 254)
		t.ButtonText = RGB(30, 58, 138)
		t.ButtonBorder = RGB(37, 99, 235)
		t.HighlightBg = RGB(37, 99, 235)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 6
		t.FontName = "Segoe UI"

	case "pixel-pc":
		t.ID = "pixel-pc"
		t.Name = "Vintage Pixel PC Screens (Ruby Crimson)"
		t.CanvasBg = RGB(255, 251, 235)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(69, 26, 3)
		t.AccentColor = RGB(220, 38, 38)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(153, 27, 27)
		t.ButtonBg = RGB(254, 243, 199)
		t.ButtonText = RGB(120, 53, 15)
		t.ButtonBorder = RGB(220, 38, 38)
		t.HighlightBg = RGB(220, 38, 38)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 0
		t.FontName = "Courier New"

	case "music-chat":
		t.ID = "music-chat"
		t.Name = "Y2K Music Chat Desktop (Coral Peach)"
		t.CanvasBg = RGB(250, 235, 215)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(58, 37, 16)
		t.AccentColor = RGB(255, 127, 80)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(139, 69, 19)
		t.ButtonBg = RGB(255, 228, 196)
		t.ButtonText = RGB(90, 62, 27)
		t.ButtonBorder = RGB(255, 127, 80)
		t.HighlightBg = RGB(255, 127, 80)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 8
		t.FontName = "Segoe UI"

	case "player-clock":
		t.ID = "player-clock"
		t.Name = "Y2K Desktop Music Player & Clock (Emerald)"
		t.CanvasBg = RGB(245, 255, 250)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(11, 56, 35)
		t.AccentColor = RGB(46, 139, 87)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(14, 58, 29)
		t.ButtonBg = RGB(224, 238, 224)
		t.ButtonText = RGB(11, 56, 35)
		t.ButtonBorder = RGB(46, 139, 87)
		t.HighlightBg = RGB(46, 139, 87)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 8
		t.FontName = "Segoe UI"

	case "dialog-green":
		t.ID = "dialog-green"
		t.Name = "Y2K System Dialogs (Sky Blue & Soft Mint)"
		t.CanvasBg = RGB(240, 253, 244)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(6, 95, 70)
		t.AccentColor = RGB(2, 132, 199)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(3, 105, 161)
		t.ButtonBg = RGB(224, 242, 254)
		t.ButtonText = RGB(3, 105, 161)
		t.ButtonBorder = RGB(2, 132, 199)
		t.HighlightBg = RGB(2, 132, 199)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 6
		t.FontName = "Segoe UI"

	case "dialog-sunset":
		t.ID = "dialog-sunset"
		t.Name = "Y2K System Dialogs (Sunset Magenta & Rose)"
		t.CanvasBg = RGB(255, 241, 242)
		t.SurfaceBg = RGB(255, 255, 255)
		t.TextColor = RGB(136, 19, 55)
		t.AccentColor = RGB(219, 39, 119)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(159, 18, 57)
		t.ButtonBg = RGB(252, 231, 243)
		t.ButtonText = RGB(136, 19, 55)
		t.ButtonBorder = RGB(219, 39, 119)
		t.HighlightBg = RGB(219, 39, 119)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 6
		t.FontName = "Segoe UI"

	case "custom":
		t.ID = "custom"
		t.Name = "Custom Theme (User Palette)"
		t.CanvasBg = RGB(26, 35, 50)
		t.SurfaceBg = RGB(192, 192, 192)
		t.TextColor = RGB(0, 0, 0)
		t.AccentColor = RGB(10, 36, 106)
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(15, 23, 42)
		t.ButtonBg = RGB(212, 208, 200)
		t.ButtonText = RGB(0, 0, 0)
		t.ButtonBorder = RGB(15, 23, 42)
		t.HighlightBg = RGB(0, 128, 128)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 8
		t.ShadowDepth = 2
		t.FontName = "Segoe UI"

		if appCfg != nil && appCfg.CustomTheme != nil {
			ct := appCfg.CustomTheme
			if ct.Name != "" {
				t.Name = ct.Name
			}
			if c, ok := parseHexColor(ct.CanvasBg); ok {
				t.CanvasBg = c
			}
			if c, ok := parseHexColor(ct.SurfaceBg); ok {
				t.SurfaceBg = c
			}
			if c, ok := parseHexColor(ct.TextColor); ok {
				t.TextColor = c
			}
			if c, ok := parseHexColor(ct.AccentColor); ok {
				t.AccentColor = c
			}
			if c, ok := parseHexColor(ct.AccentText); ok {
				t.AccentText = c
			}
			if c, ok := parseHexColor(ct.BorderColor); ok {
				t.BorderColor = c
				t.ButtonBorder = c
			}
			if c, ok := parseHexColor(ct.ButtonBg); ok {
				t.ButtonBg = c
			}
			if c, ok := parseHexColor(ct.ButtonText); ok {
				t.ButtonText = c
			}
			if ct.ShadowColor != "" {
				if c, ok := parseHexColor(ct.ShadowColor); ok {
					t.ShadowColor = c
				}
			}
			if ct.CornerRadius >= 0 {
				t.Radius = ct.CornerRadius
			}
			if ct.ShadowDepth >= 0 {
				t.ShadowDepth = ct.ShadowDepth
			}
		}

	default: // "win95-classic"
		t.ID = "win95-classic"
		t.Name = "Windows 95 Classic (Retro Bevel)"
		t.CanvasBg = RGB(192, 192, 192)    // #C0C0C0
		t.SurfaceBg = RGB(255, 255, 255)   // #FFFFFF
		t.TextColor = RGB(0, 0, 0)         // #000000
		t.AccentColor = RGB(0, 0, 128)     // #000080
		t.AccentText = RGB(255, 255, 255)
		t.BorderColor = RGB(128, 128, 128)
		t.ButtonBg = RGB(192, 192, 192)
		t.ButtonText = RGB(0, 0, 0)
		t.ButtonBorder = RGB(0, 0, 0)
		t.HighlightBg = RGB(0, 0, 128)
		t.HighlightText = RGB(255, 255, 255)
		t.DarkTitlebar = false
		t.Radius = 0
		t.FontName = "Tahoma"
	}

	if customRadius >= 0 {
		t.Radius = customRadius
	}
	if customShadow >= 0 {
		t.ShadowDepth = customShadow
	}

	t.CanvasBrush = w32.CreateSolidBrush(t.CanvasBg)
	t.SurfaceBrush = w32.CreateSolidBrush(t.SurfaceBg)
	t.AccentBrush = w32.CreateSolidBrush(t.AccentColor)
	t.ButtonBrush = w32.CreateSolidBrush(t.ButtonBg)
	t.HighlightBrush = w32.CreateSolidBrush(t.HighlightBg)
	t.BorderBrush = w32.CreateSolidBrush(t.BorderColor)

	t.FontNormal = createFontCustom(t.FontName, -12, false)
	t.FontBold = createFontCustom(t.FontName, -12, true)
	t.FontHeader = createFontCustom(t.FontName, -14, true)

	return t
}

func createFontCustom(faceName string, size int, bold bool) w32.HFONT {
	weight := 400
	if bold {
		weight = 700
	}
	face, _ := windows.UTF16PtrFromString(faceName)
	h, _, _ := procCreateFontW.Call(
		uintptr(size), 0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		uintptr(w32.DEFAULT_CHARSET),
		uintptr(w32.OUT_DEFAULT_PRECIS),
		uintptr(w32.CLIP_DEFAULT_PRECIS),
		uintptr(w32.CLEARTYPE_QUALITY),
		uintptr(w32.DEFAULT_PITCH|w32.FF_DONTCARE),
		uintptr(unsafe.Pointer(face)),
	)
	return w32.HFONT(h)
}

func applyThemeToWindow(hwnd w32.HWND, theme *ThemeStyle) {
	if theme == nil || hwnd == 0 {
		return
	}

	// 1. Title bar immersive dark mode (Windows 10 build 1809+ / Windows 11)
	val := int32(0)
	if theme.DarkTitlebar {
		val = 1
	}
	// DWMWA_USE_IMMERSIVE_DARK_MODE (20 on Win 10 2004+, 19 on 1809)
	_, _, _ = procDwmSetWindowAttribute.Call(uintptr(hwnd), 20, uintptr(unsafe.Pointer(&val)), 4)
	_, _, _ = procDwmSetWindowAttribute.Call(uintptr(hwnd), 19, uintptr(unsafe.Pointer(&val)), 4)

	// Window corner preference (Windows 11 build 22000+)
	// DWMWA_WINDOW_CORNER_PREFERENCE = 33
	cornerPref := int32(0)
	if theme.Radius == 0 {
		cornerPref = 1 // DWMWCP_DONOTROUND (Crisp retro rectangular corners)
	} else if theme.Radius <= 4 {
		cornerPref = 3 // DWMWCP_ROUNDSMALL (Subtle modern rounded corners)
	} else {
		cornerPref = 2 // DWMWCP_ROUND (Chic Y2K rounded aesthetic)
	}
	_, _, _ = procDwmSetWindowAttribute.Call(uintptr(hwnd), 33, uintptr(unsafe.Pointer(&cornerPref)), 4)

	// Caption & text color styling on Windows 11 build 22000+
	captionColor := theme.CanvasBg
	captionTextColor := theme.TextColor
	if theme.ID == "y2k-candy" {
		captionColor = RGB(255, 238, 245)
		captionTextColor = RGB(26, 26, 46)
	} else if theme.ID == "vaporwave-cyber" {
		captionColor = RGB(24, 9, 40)
		captionTextColor = RGB(0, 240, 255)
	} else if theme.ID == "dark-obsidian" {
		captionColor = RGB(24, 24, 37)
		captionTextColor = RGB(248, 250, 252)
	}
	_, _, _ = procDwmSetWindowAttribute.Call(uintptr(hwnd), 35 /* DWMWA_CAPTION_COLOR */, uintptr(unsafe.Pointer(&captionColor)), 4)
	_, _, _ = procDwmSetWindowAttribute.Call(uintptr(hwnd), 36 /* DWMWA_TEXT_COLOR */, uintptr(unsafe.Pointer(&captionTextColor)), 4)

	// 2. Set window background class brush
	w32.SetClassLongPtr(hwnd, -10 /* GCLP_HBRBACKGROUND */, uintptr(theme.CanvasBrush))

	// 3. Apply colors to listviews & treeviews
	applyThemeToListView(hwndListView, theme)
	applyThemeToListView(hwndListViewUniv, theme)
	if hwndListViewStaged != 0 {
		applyThemeToListView(hwndListViewStaged, theme)
	}
	if hwndTreeView != 0 {
		applyUxThemeMode(hwndTreeView, theme)
		applyThemeToTreeView(hwndTreeView, theme)
	}
	if hwndStatusBar != 0 {
		applyUxThemeMode(hwndStatusBar, theme)
		w32.SendMessage(hwndStatusBar, 0x2001 /* SB_SETBKCOLOR */, 0, uintptr(theme.CanvasBg))
	}
	if hwndTabControl != 0 {
		applyUxThemeMode(hwndTabControl, theme)
		w32.InvalidateRect(hwndTabControl, nil, true)
	}

	// 4. Update fonts across all child controls
	w32.EnumChildWindows(hwnd, func(child w32.HWND) bool {
		w32.SendMessage(child, w32.WM_SETFONT, uintptr(theme.FontNormal), 1)
		return true
	})

	// 5. Force full repaint across window, tab control, and child controls
	w32.InvalidateRect(hwnd, nil, true)
	w32.UpdateWindow(hwnd)
}

func applyThemeToListView(hwndLV w32.HWND, theme *ThemeStyle) {
	if hwndLV == 0 || theme == nil {
		return
	}
	applyUxThemeMode(hwndLV, theme)

	// Update header font and theme
	hwndHdr := w32.HWND(w32.SendMessage(hwndLV, 0x101F /* LVM_GETHEADER */, 0, 0))
	if hwndHdr != 0 {
		applyUxThemeMode(hwndHdr, theme)
		w32.SendMessage(hwndHdr, w32.WM_SETFONT, uintptr(theme.FontBold), 1)
	}

	// LVM_SETBKCOLOR = 0x1001
	w32.SendMessage(hwndLV, 0x1001, 0, uintptr(theme.SurfaceBg))
	// LVM_SETTEXTCOLOR = 0x1024
	w32.SendMessage(hwndLV, 0x1024, 0, uintptr(theme.TextColor))
	// LVM_SETTEXTBKCOLOR = 0x1026
	w32.SendMessage(hwndLV, 0x1026, 0, uintptr(theme.SurfaceBg))
	w32.InvalidateRect(hwndLV, nil, true)
}

var (
	oldTabWndProc       uintptr
	tabWndProcCallback  = windows.NewCallback(tabSubclassProc)
)

func subclassTabControl(hwnd w32.HWND) {
	if hwnd == 0 {
		return
	}
	oldTabWndProc = w32.SetWindowLongPtr(hwnd, w32.GWLP_WNDPROC, tabWndProcCallback)
}

func tabSubclassProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == w32.WM_ERASEBKGND {
		if currentTheme != nil {
			hdc := w32.HDC(wParam)
			rc := w32.GetClientRect(hwnd)
			if rc != nil {
				w32.FillRect(hdc, rc, currentTheme.CanvasBrush)
				return 1
			}
		}
	}
	return w32.CallWindowProc(oldTabWndProc, hwnd, msg, wParam, lParam)
}

func handleListViewCustomDraw(hwndLV w32.HWND, lParam uintptr, theme *ThemeStyle) uintptr {
	if theme == nil || hwndLV == 0 {
		return CDRF_DODEFAULT
	}
	lvcd := (*NMLVCUSTOMDRAW)(unsafe.Pointer(lParam))
	switch lvcd.Nmcd.DwDrawStage {
	case CDDS_PREPAINT:
		return CDRF_NOTIFYITEMDRAW
	case CDDS_ITEMPREPAINT:
		itemIdx := int(lvcd.Nmcd.DwItemSpec)
		state := w32.SendMessage(hwndLV, LVM_GETITEMSTATE, uintptr(itemIdx), uintptr(LVIS_SELECTED))
		if (state & LVIS_SELECTED) != 0 {
			// ONLY the individually selected item receives the theme highlight
			lvcd.ClrText = theme.HighlightText
			lvcd.ClrTextBk = theme.HighlightBg
		} else {
			// All other items remain on the clean surface background
			lvcd.ClrText = theme.TextColor
			lvcd.ClrTextBk = theme.SurfaceBg
		}
		return CDRF_NEWFONT
	}
	return CDRF_DODEFAULT
}

func handleTreeViewCustomDraw(hwndTV w32.HWND, lParam uintptr, theme *ThemeStyle) uintptr {
	if theme == nil || hwndTV == 0 {
		return CDRF_DODEFAULT
	}
	tvcd := (*NMTVCUSTOMDRAW)(unsafe.Pointer(lParam))
	switch tvcd.Nmcd.DwDrawStage {
	case CDDS_PREPAINT:
		return CDRF_NOTIFYITEMDRAW
	case CDDS_ITEMPREPAINT:
		hItem := tvcd.Nmcd.DwItemSpec
		hSel := w32.SendMessage(hwndTV, TVM_GETNEXTITEM, TVGN_CARET, 0)
		if hItem != 0 && hItem == hSel {
			// ONLY the individually selected tree node receives the theme highlight
			tvcd.ClrText = theme.HighlightText
			tvcd.ClrTextBk = theme.HighlightBg
		} else {
			// All other tree nodes remain on the clean surface background
			tvcd.ClrText = theme.TextColor
			tvcd.ClrTextBk = theme.SurfaceBg
		}
		return CDRF_NEWFONT
	}
	return CDRF_DODEFAULT
}

func switchTheme(hwnd w32.HWND, themeID string) {
	if currentTheme != nil {
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.CanvasBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.SurfaceBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.AccentBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.ButtonBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.HighlightBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.BorderBrush))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.FontNormal))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.FontBold))
		w32.DeleteObject(w32.HGDIOBJ(currentTheme.FontHeader))
	}
	currentTheme = initTheme(themeID, appCfg.CornerRadius, appCfg.ShadowDepth)
	appCfg.ActiveTheme = themeID
	_ = core.SaveConfig("./config.json", appCfg)
	applyThemeToWindow(hwnd, currentTheme)
}

func updateCornerRadius(hwnd w32.HWND, radius int) {
	if currentTheme == nil {
		return
	}
	currentTheme.Radius = radius
	appCfg.CornerRadius = radius
	if appCfg.CustomTheme != nil {
		appCfg.CustomTheme.CornerRadius = radius
	}
	_ = core.SaveConfig("./config.json", appCfg)
	applyThemeToWindow(hwnd, currentTheme)
}

func updateShadowDepth(hwnd w32.HWND, depth int) {
	if currentTheme == nil {
		return
	}
	currentTheme.ShadowDepth = depth
	appCfg.ShadowDepth = depth
	if appCfg.CustomTheme != nil {
		appCfg.CustomTheme.ShadowDepth = depth
	}
	_ = core.SaveConfig("./config.json", appCfg)
	if hwnd != 0 {
		w32.InvalidateRect(hwnd, nil, true)
	}
}

func parseHexColor(s string) (uint32, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, false
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return RGB(uint8(r), uint8(g), uint8(b)), true
}

func colorToHex(c uint32) string {
	r := uint8(c & 0xFF)
	g := uint8((c >> 8) & 0xFF)
	b := uint8((c >> 16) & 0xFF)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func drawOwnerButton(dis *DRAWITEMSTRUCT, theme *ThemeStyle) {
	if theme == nil || dis == nil {
		return
	}
	hdc := dis.Hdc
	rc := dis.RcItem
	isPressed := (dis.ItemState & ODS_SELECTED) != 0
	isDisabled := (dis.ItemState & ODS_DISABLED) != 0
	isFocused := (dis.ItemState & ODS_FOCUS) != 0

	isPrimary := (dis.CtlID == ID_BTN_EXECUTE || dis.CtlID == ID_BTN_OP_ACTION || dis.CtlID == ID_BTN_DRAWER_EXECUTE || dis.CtlID == ID_BTN_REMOTE_PUSH || dis.CtlID == ID_BTN_REMOTE_PULL)

	btnText := getWindowText(dis.HwndItem)

	switch theme.ID {
	case "win95-classic", "win95-figma":
		w32.FillRect(hdc, &rc, theme.ButtonBrush)

		if isPressed {
			drawEdge(hdc, &rc, EDGE_SUNKEN, BF_RECT)
			rc.Left++
			rc.Top++
		} else {
			if isPrimary {
				blackBrush := w32.CreateSolidBrush(RGB(0, 0, 0))
				frameRect(hdc, &rc, blackBrush)
				w32.DeleteObject(w32.HGDIOBJ(blackBrush))
				rcInner := rc
				rcInner.Left++
				rcInner.Top++
				rcInner.Right--
				rcInner.Bottom--
				drawEdge(hdc, &rcInner, EDGE_RAISED, BF_RECT)
			} else {
				drawEdge(hdc, &rc, EDGE_RAISED, BF_RECT)
			}
		}

		if isFocused && !isPressed {
			focusRc := rc
			focusRc.Left += 3
			focusRc.Top += 3
			focusRc.Right -= 3
			focusRc.Bottom -= 3
			drawFocusRect(hdc, &focusRc)
		}

		w32.SetBkMode(hdc, w32.TRANSPARENT)
		w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))

		if isDisabled {
			rcShadow := rc
			rcShadow.Left++
			rcShadow.Top++
			w32.SetTextColor(hdc, w32.COLORREF(RGB(255, 255, 255)))
			w32.DrawText(hdc, btnText, &rcShadow, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(128, 128, 128)))
			w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		} else {
			w32.SetTextColor(hdc, w32.COLORREF(theme.ButtonText))
			w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		}

	case "vaporwave-cyber", "retrowave-pc":
		var bgBrush w32.HBRUSH
		var txtColor uint32
		if isPressed {
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			rc.Left += 2
			rc.Top += 2
		} else if isPrimary {
			bgBrush = theme.AccentBrush
			txtColor = theme.AccentText
		} else {
			bgBrush = theme.ButtonBrush
			txtColor = theme.ButtonText
		}

		w32.FillRect(hdc, &rc, bgBrush)

		if isPrimary {
			cyanBrush := w32.CreateSolidBrush(RGB(0, 240, 255))
			frameRect(hdc, &rc, cyanBrush)
			rcInner := rc
			rcInner.Left++
			rcInner.Top++
			rcInner.Right--
			rcInner.Bottom--
			frameRect(hdc, &rcInner, theme.AccentBrush)
			w32.DeleteObject(w32.HGDIOBJ(cyanBrush))
		} else {
			frameRect(hdc, &rc, theme.AccentBrush)
		}

		w32.SetBkMode(hdc, w32.TRANSPARENT)
		w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))
		if isDisabled {
			w32.SetTextColor(hdc, w32.COLORREF(RGB(100, 50, 130)))
		} else {
			w32.SetTextColor(hdc, w32.COLORREF(txtColor))
		}
		w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)

	case "y2k-candy", "y2k-pink", "warm-handdrawn", "pastel-grid", "nostalgic-media", "y2k-blue", "music-chat", "player-clock", "dialog-green", "dialog-sunset", "custom", "aqua-win32":
		var bgBrush w32.HBRUSH
		var txtColor uint32
		sh := theme.ShadowDepth

		if isPressed {
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			shift := int32(sh)
			if shift <= 0 {
				shift = 1
			}
			rc.Left += shift
			rc.Top += shift
		} else if isPrimary {
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			if sh > 0 {
				rad := theme.Radius
				if rad <= 0 {
					rad = 2
				}
				shadowCol := theme.BorderColor
				if theme.ShadowColor != 0 {
					shadowCol = theme.ShadowColor
				}
				shadowBrush := w32.CreateSolidBrush(shadowCol)
				shadowPen := w32.CreatePen(w32.PS_SOLID, 1, w32.COLORREF(shadowCol))
				oldB := w32.SelectObject(hdc, w32.HGDIOBJ(shadowBrush))
				oldP := w32.SelectObject(hdc, w32.HGDIOBJ(shadowPen))
				_, _, _ = procRoundRect.Call(uintptr(hdc), uintptr(rc.Left+int32(sh)), uintptr(rc.Top+int32(sh)), uintptr(rc.Right), uintptr(rc.Bottom), uintptr(rad), uintptr(rad))
				w32.SelectObject(hdc, oldB)
				w32.SelectObject(hdc, oldP)
				w32.DeleteObject(w32.HGDIOBJ(shadowBrush))
				w32.DeleteObject(w32.HGDIOBJ(shadowPen))
				rc.Right -= int32(sh)
				rc.Bottom -= int32(sh)
			}
		} else {
			bgBrush = theme.ButtonBrush
			txtColor = theme.ButtonText
			if sh > 0 {
				rad := theme.Radius
				if rad <= 0 {
					rad = 2
				}
				shadowCol := theme.BorderColor
				if theme.ShadowColor != 0 {
					shadowCol = theme.ShadowColor
				}
				shadowBrush := w32.CreateSolidBrush(shadowCol)
				shadowPen := w32.CreatePen(w32.PS_SOLID, 1, w32.COLORREF(shadowCol))
				oldB := w32.SelectObject(hdc, w32.HGDIOBJ(shadowBrush))
				oldP := w32.SelectObject(hdc, w32.HGDIOBJ(shadowPen))
				_, _, _ = procRoundRect.Call(uintptr(hdc), uintptr(rc.Left+int32(sh)), uintptr(rc.Top+int32(sh)), uintptr(rc.Right), uintptr(rc.Bottom), uintptr(rad), uintptr(rad))
				w32.SelectObject(hdc, oldB)
				w32.SelectObject(hdc, oldP)
				w32.DeleteObject(w32.HGDIOBJ(shadowBrush))
				w32.DeleteObject(w32.HGDIOBJ(shadowPen))
				rc.Right -= int32(sh)
				rc.Bottom -= int32(sh)
			}
		}

		borderPen := w32.CreatePen(w32.PS_SOLID, 2, w32.COLORREF(theme.BorderColor))
		oldB := w32.SelectObject(hdc, w32.HGDIOBJ(bgBrush))
		oldP := w32.SelectObject(hdc, w32.HGDIOBJ(borderPen))
		rad := theme.Radius
		if rad <= 0 {
			w32.Rectangle(hdc, int(rc.Left), int(rc.Top), int(rc.Right), int(rc.Bottom))
		} else {
			_, _, _ = procRoundRect.Call(uintptr(hdc), uintptr(rc.Left), uintptr(rc.Top), uintptr(rc.Right), uintptr(rc.Bottom), uintptr(rad), uintptr(rad))
		}
		w32.SelectObject(hdc, oldB)
		w32.SelectObject(hdc, oldP)
		w32.DeleteObject(w32.HGDIOBJ(borderPen))

		w32.SetBkMode(hdc, w32.TRANSPARENT)
		w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))
		if isDisabled {
			w32.SetTextColor(hdc, w32.COLORREF(RGB(180, 180, 200)))
		} else {
			w32.SetTextColor(hdc, w32.COLORREF(txtColor))
		}
		w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)

	case "monochrome-mac", "pixel-pc":
		var bgBrush w32.HBRUSH
		var txtColor uint32
		sh := theme.ShadowDepth
		if isPressed {
			// Inverted solid black face on click!
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			shift := int32(sh)
			if shift <= 0 {
				shift = 1
			}
			rc.Left += shift
			rc.Top += shift
		} else if isPrimary {
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			if sh > 0 {
				shadowBrush := w32.CreateSolidBrush(theme.BorderColor)
				w32.FillRect(hdc, &w32.RECT{
					Left:   rc.Left + int32(sh),
					Top:    rc.Top + int32(sh),
					Right:  rc.Right,
					Bottom: rc.Bottom,
				}, shadowBrush)
				w32.DeleteObject(w32.HGDIOBJ(shadowBrush))
				rc.Right -= int32(sh)
				rc.Bottom -= int32(sh)
			}
		} else {
			bgBrush = theme.ButtonBrush
			txtColor = theme.ButtonText
			if sh > 0 {
				shadowBrush := w32.CreateSolidBrush(theme.BorderColor)
				w32.FillRect(hdc, &w32.RECT{
					Left:   rc.Left + int32(sh),
					Top:    rc.Top + int32(sh),
					Right:  rc.Right,
					Bottom: rc.Bottom,
				}, shadowBrush)
				w32.DeleteObject(w32.HGDIOBJ(shadowBrush))
				rc.Right -= int32(sh)
				rc.Bottom -= int32(sh)
			}
		}

		w32.FillRect(hdc, &rc, bgBrush)

		borderBrush := w32.CreateSolidBrush(RGB(0, 0, 0))
		frameRect(hdc, &rc, borderBrush)
		if isPrimary || isPressed {
			rcInner := rc
			rcInner.Left++
			rcInner.Top++
			rcInner.Right--
			rcInner.Bottom--
			frameRect(hdc, &rcInner, borderBrush)
		}
		w32.DeleteObject(w32.HGDIOBJ(borderBrush))

		w32.SetBkMode(hdc, w32.TRANSPARENT)
		w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))
		if isDisabled {
			w32.SetTextColor(hdc, w32.COLORREF(RGB(160, 160, 160)))
		} else {
			w32.SetTextColor(hdc, w32.COLORREF(txtColor))
		}
		w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)

	default: // dark-obsidian
		var bgBrush w32.HBRUSH
		var txtColor uint32
		if isPressed {
			bgBrush = theme.HighlightBrush
			txtColor = theme.HighlightText
			rc.Left++
			rc.Top++
		} else if isPrimary {
			bgBrush = theme.AccentBrush
			txtColor = theme.AccentText
		} else {
			bgBrush = theme.ButtonBrush
			txtColor = theme.ButtonText
		}

		w32.FillRect(hdc, &rc, bgBrush)

		borderBrush := w32.CreateSolidBrush(theme.BorderColor)
		if isPrimary {
			w32.DeleteObject(w32.HGDIOBJ(borderBrush))
			borderBrush = w32.CreateSolidBrush(theme.AccentColor)
		}
		frameRect(hdc, &rc, borderBrush)
		w32.DeleteObject(w32.HGDIOBJ(borderBrush))

		w32.SetBkMode(hdc, w32.TRANSPARENT)
		w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))
		if isDisabled {
			w32.SetTextColor(hdc, w32.COLORREF(RGB(120, 120, 140)))
		} else {
			w32.SetTextColor(hdc, w32.COLORREF(txtColor))
		}
		w32.DrawText(hdc, btnText, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
	}
}

func drawOwnerTab(dis *DRAWITEMSTRUCT, theme *ThemeStyle) {
	if theme == nil || dis == nil {
		return
	}
	hdc := dis.Hdc
	rc := dis.RcItem
	tabIndex := int(dis.ItemID)
	isSelected := (w32.SendMessage(hwndTabControl, TCM_GETCURSEL, 0, 0) == uintptr(tabIndex))

	tabTitles := []string{
		"Local Explorer",
		"Universal Search",
		"Remote Sync",
		"Server Health",
		"Transfer Log",
		"Settings",
	}
	title := ""
	if tabIndex >= 0 && tabIndex < len(tabTitles) {
		title = tabTitles[tabIndex]
	}

	w32.SelectObject(hdc, w32.HGDIOBJ(theme.FontBold))
	w32.SetBkMode(hdc, w32.TRANSPARENT)

	switch theme.ID {
	case "win95-classic", "win95-figma":
		w32.FillRect(hdc, &rc, theme.ButtonBrush)
		if isSelected {
			rcActive := rc
			rcActive.Top -= 2
			rcActive.Bottom += 2
			w32.FillRect(hdc, &rcActive, theme.ButtonBrush)
			drawEdge(hdc, &rcActive, EDGE_RAISED, BF_LEFT|BF_TOP|BF_RIGHT)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(0, 0, 0)))
			w32.DrawText(hdc, title, &rcActive, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		} else {
			drawEdge(hdc, &rc, EDGE_RAISED, BF_RECT)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(64, 64, 64)))
			w32.DrawText(hdc, title, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		}

	case "vaporwave-cyber", "retrowave-pc":
		if isSelected {
			w32.FillRect(hdc, &rc, theme.ButtonBrush)
			topRc := rc
			topRc.Bottom = topRc.Top + 3
			cyanBrush := w32.CreateSolidBrush(RGB(0, 240, 255))
			w32.FillRect(hdc, &topRc, cyanBrush)
			w32.DeleteObject(w32.HGDIOBJ(cyanBrush))
			frameRect(hdc, &rc, theme.AccentBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(255, 255, 255)))
		} else {
			w32.FillRect(hdc, &rc, theme.SurfaceBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(121, 40, 202)))
			frameRect(hdc, &rc, theme.ButtonBrush)
		}
		w32.DrawText(hdc, title, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)

	case "y2k-candy", "y2k-pink", "warm-handdrawn", "pastel-grid", "nostalgic-media", "y2k-blue", "music-chat", "player-clock", "dialog-green", "dialog-sunset":
		borderBrush := w32.CreateSolidBrush(theme.BorderColor)
		if isSelected {
			rcActive := rc
			rcActive.Top -= 1
			rcActive.Bottom += 2
			w32.FillRect(hdc, &rcActive, theme.HighlightBrush) // Candy Cyan
			frameRect(hdc, &rcActive, borderBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(26, 26, 46)))
			w32.DrawText(hdc, title, &rcActive, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		} else {
			w32.FillRect(hdc, &rc, theme.SurfaceBrush)
			frameRect(hdc, &rc, borderBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(100, 100, 120)))
			w32.DrawText(hdc, title, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
		}
		w32.DeleteObject(w32.HGDIOBJ(borderBrush))

	case "monochrome-mac", "pixel-pc":
		blackBrush := w32.CreateSolidBrush(RGB(0, 0, 0))
		if isSelected {
			w32.FillRect(hdc, &rc, blackBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(255, 255, 255)))
		} else {
			w32.FillRect(hdc, &rc, theme.SurfaceBrush)
			frameRect(hdc, &rc, blackBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(0, 0, 0)))
		}
		w32.DeleteObject(w32.HGDIOBJ(blackBrush))
		w32.DrawText(hdc, title, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)

	default: // dark-obsidian
		borderBrush := w32.CreateSolidBrush(theme.BorderColor)
		if isSelected {
			w32.FillRect(hdc, &rc, theme.SurfaceBrush)
			topRc := rc
			topRc.Bottom = topRc.Top + 3
			w32.FillRect(hdc, &topRc, theme.AccentBrush)
			frameRect(hdc, &rc, borderBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(255, 255, 255)))
		} else {
			w32.FillRect(hdc, &rc, theme.CanvasBrush)
			frameRect(hdc, &rc, borderBrush)
			w32.SetTextColor(hdc, w32.COLORREF(RGB(160, 160, 180)))
		}
		w32.DeleteObject(w32.HGDIOBJ(borderBrush))
		w32.DrawText(hdc, title, &rc, w32.DT_CENTER|w32.DT_VCENTER|w32.DT_SINGLELINE)
	}
}

func paintWindowDecorations(hdc w32.HDC, rc *w32.RECT, theme *ThemeStyle) {
	if theme == nil || rc == nil {
		return
	}
	width := rc.Right - rc.Left
	height := rc.Bottom - rc.Top

	rcDiv1 := w32.RECT{Left: 12, Top: 37, Right: width - 12, Bottom: 39}
	rcDiv2 := w32.RECT{Left: 12, Top: 65, Right: width - 12, Bottom: 67}
	bottomBarY := height - 68
	rcDiv3 := w32.RECT{Left: 12, Top: bottomBarY - 6, Right: width - 12, Bottom: bottomBarY - 4}

	switch theme.ID {
	case "win95-classic":
		drawEdge(hdc, &rcDiv1, EDGE_ETCHED, BF_RECT)
		drawEdge(hdc, &rcDiv2, EDGE_ETCHED, BF_RECT)
		drawEdge(hdc, &rcDiv3, EDGE_ETCHED, BF_RECT)

	case "vaporwave-cyber":
		neonBrush := w32.CreateSolidBrush(theme.AccentColor)
		w32.FillRect(hdc, &rcDiv1, neonBrush)
		w32.FillRect(hdc, &rcDiv2, neonBrush)
		w32.FillRect(hdc, &rcDiv3, neonBrush)
		w32.DeleteObject(w32.HGDIOBJ(neonBrush))

	case "y2k-candy":
		navyBrush := w32.CreateSolidBrush(theme.BorderColor)
		w32.FillRect(hdc, &rcDiv1, navyBrush)
		w32.FillRect(hdc, &rcDiv2, navyBrush)
		w32.FillRect(hdc, &rcDiv3, navyBrush)
		w32.DeleteObject(w32.HGDIOBJ(navyBrush))

	case "monochrome-mac":
		blackBrush := w32.CreateSolidBrush(RGB(0, 0, 0))
		w32.FillRect(hdc, &rcDiv1, blackBrush)
		w32.FillRect(hdc, &rcDiv2, blackBrush)
		w32.FillRect(hdc, &rcDiv3, blackBrush)
		w32.DeleteObject(w32.HGDIOBJ(blackBrush))

	default: // dark-obsidian
		slateBrush := w32.CreateSolidBrush(theme.BorderColor)
		w32.FillRect(hdc, &rcDiv1, slateBrush)
		w32.FillRect(hdc, &rcDiv2, slateBrush)
		w32.FillRect(hdc, &rcDiv3, slateBrush)
		w32.DeleteObject(w32.HGDIOBJ(slateBrush))
	}
}
