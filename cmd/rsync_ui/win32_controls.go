//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"
)

var (
	shell32DLL          = windows.NewLazySystemDLL("shell32.dll")
	shBrowseForFolderW  = shell32DLL.NewProc("SHBrowseForFolderW")
	shGetPathFromIDList = shell32DLL.NewProc("SHGetPathFromIDListW")
	procDragQueryFileW  = shell32DLL.NewProc("DragQueryFileW")
	ole32DLL            = windows.NewLazySystemDLL("ole32.dll")
	procOleInitialize   = ole32DLL.NewProc("OleInitialize")
	procOleUninitialize = ole32DLL.NewProc("OleUninitialize")
	coInitialize        = ole32DLL.NewProc("CoInitialize")
	coUninitialize      = ole32DLL.NewProc("CoUninitialize")
	gdi32DLL            = windows.NewLazySystemDLL("gdi32.dll")
	procCreateFontW     = gdi32DLL.NewProc("CreateFontW")
	procRoundRect       = gdi32DLL.NewProc("RoundRect")
	kernel32DLL           = windows.NewLazySystemDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = kernel32DLL.NewProc("GetDiskFreeSpaceExW")
	user32DLL           = windows.NewLazySystemDLL("user32.dll")
	procSetWindowLongPtrW = user32DLL.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = user32DLL.NewProc("CallWindowProcW")
	procMessageBeep       = user32DLL.NewProc("MessageBeep")
	procDrawEdge          = user32DLL.NewProc("DrawEdge")
	procDrawFocusRect     = user32DLL.NewProc("DrawFocusRect")
	procFrameRect         = user32DLL.NewProc("FrameRect")
	procAppendMenuW       = user32DLL.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32DLL.NewProc("TrackPopupMenu")
	procGetClassNameW    = user32DLL.NewProc("GetClassNameW")
	procFindWindowW      = user32DLL.NewProc("FindWindowW")
	procSetForegroundWindow = user32DLL.NewProc("SetForegroundWindow")
	procShowWindow       = user32DLL.NewProc("ShowWindow")
	procAttachThreadInput = user32DLL.NewProc("AttachThreadInput")
	procGetForegroundWindow = user32DLL.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32DLL.NewProc("GetWindowThreadProcessId")
)

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	_pad       uint32
	HwndItem   w32.HWND
	Hdc        w32.HDC
	RcItem     w32.RECT
	ItemData   uintptr
}

const (
	ODT_MENU     = 1
	ODT_LISTBOX  = 2
	ODT_COMBOBOX = 3
	ODT_BUTTON   = 4
	ODT_STATIC   = 5

	ODS_SELECTED = 0x0001
	ODS_GRAYED   = 0x0002
	ODS_DISABLED = 0x0004
	ODS_CHECKED  = 0x0008
	ODS_FOCUS    = 0x0010
	ODS_DEFAULT  = 0x0020

	BDR_RAISEDOUTER = 0x0001
	BDR_SUNKENOUTER = 0x0002
	BDR_RAISEDINNER = 0x0004
	BDR_SUNKENINNER = 0x0008

	EDGE_RAISED = (BDR_RAISEDOUTER | BDR_RAISEDINNER)
	EDGE_SUNKEN = (BDR_SUNKENOUTER | BDR_SUNKENINNER)
	EDGE_ETCHED = (BDR_SUNKENOUTER | BDR_RAISEDINNER)
	EDGE_BUMP   = (BDR_RAISEDOUTER | BDR_SUNKENINNER)

	BF_LEFT   = 0x0001
	BF_TOP    = 0x0002
	BF_RIGHT  = 0x0004
	BF_BOTTOM = 0x0008
	BF_RECT   = (BF_LEFT | BF_TOP | BF_RIGHT | BF_BOTTOM)

	TCS_OWNERDRAWFIXED = 0x2000
)

func drawEdge(hdc w32.HDC, rc *w32.RECT, edge, flags uint32) bool {
	r, _, _ := procDrawEdge.Call(uintptr(hdc), uintptr(unsafe.Pointer(rc)), uintptr(edge), uintptr(flags))
	return r != 0
}

func drawFocusRect(hdc w32.HDC, rc *w32.RECT) bool {
	r, _, _ := procDrawFocusRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rc)))
	return r != 0
}

func frameRect(hdc w32.HDC, rc *w32.RECT, hbr w32.HBRUSH) bool {
	r, _, _ := procFrameRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rc)), uintptr(hbr))
	return r != 0
}

// Win32 ListView messages and structures
const (
	LVM_FIRST           = 0x1000
	LVM_INSERTCOLUMNW   = LVM_FIRST + 97
	LVM_INSERTITEMW     = LVM_FIRST + 77
	LVM_SETITEMTEXTW    = LVM_FIRST + 116
	LVM_DELETEALLITEMS  = LVM_FIRST + 9
	LVM_GETITEMCOUNT    = LVM_FIRST + 4
	LVM_GETNEXTITEM     = LVM_FIRST + 12
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVM_SETITEMSTATE    = LVM_FIRST + 43
	LVM_GETITEMSTATE    = LVM_FIRST + 44
	LVM_ENSUREVISIBLE   = LVM_FIRST + 19

	LVS_EX_FULLROWSELECT = 0x00000020
	LVS_EX_GRIDLINES     = 0x00000001
	LVS_EX_DOUBLEBUFFER  = 0x00010000
	LVS_EX_CHECKBOXES    = 0x00000004

	LVCF_FMT     = 0x0001
	LVCF_WIDTH   = 0x0002
	LVCF_TEXT    = 0x0004
	LVCF_SUBITEM = 0x0008

	LVCFMT_LEFT  = 0x0000
	LVCFMT_RIGHT = 0x0001

	LVIF_TEXT    = 0x0001
	LVIF_PARAM   = 0x0004
	LVIF_STATE   = 0x0008

	LVIS_FOCUSED        = 0x0001
	LVIS_SELECTED       = 0x0002
	LVIS_STATEIMAGEMASK = 0x3000
	LVNI_SELECTED       = 0x0002

	// Notification codes
	NM_CLICK      = ^uint32(1) // -2
	NM_DBLCLK     = ^uint32(2) // -3
	NM_CUSTOMDRAW = ^uint32(11) // -12 (0xFFFFFFF4)

	CDDS_PREPAINT      = 0x00000001
	CDDS_POSTPAINT     = 0x00000002
	CDDS_PREERASE      = 0x00000003
	CDDS_POSTERASE     = 0x00000004
	CDDS_ITEM          = 0x00010000
	CDDS_ITEMPREPAINT  = (CDDS_ITEM | CDDS_PREPAINT)
	CDDS_ITEMPOSTPAINT = (CDDS_ITEM | CDDS_POSTPAINT)

	CDRF_DODEFAULT       = 0x00000000
	CDRF_NEWFONT         = 0x00000002
	CDRF_SKIPDEFAULT     = 0x00000004
	CDRF_NOTIFYPOSTPAINT = 0x00000010
	CDRF_NOTIFYITEMDRAW  = 0x00000020

	CDIS_SELECTED = 0x0001
	CDIS_GRAYED   = 0x0002
	CDIS_DISABLED = 0x0004
	CDIS_CHECKED  = 0x0008
	CDIS_FOCUS    = 0x0010
	CDIS_DEFAULT  = 0x0020
	CDIS_HOT      = 0x0040

	// ListView notification codes
	LVN_FIRST       = uint32(0xFFFFFF9C) // -100
	LVN_ITEMCHANGED = uint32(0xFFFFFF9B) // -101 (LVN_FIRST - 1)

	// Tab control messages
	TCM_FIRST           = 0x1300
	TCM_INSERTITEM      = TCM_FIRST + 62
	TCM_GETCURSEL       = TCM_FIRST + 11
	TCM_SETCURSEL       = TCM_FIRST + 12
	TCM_DELETEALLITEMS  = TCM_FIRST + 9
	TCIF_TEXT           = 0x0001

	// Tab control notification codes
	TCN_FIRST           = uint32(0xFFFFFDD6) // -550
	TCN_SELCHANGE       = uint32(0xFFFFFDD9) // -551 (TCN_FIRST - 1)
	TCN_SELCHANGING     = uint32(0xFFFFFDD8) // -552 (TCN_FIRST - 2)

	// Status bar messages
	SB_SETPARTS = 0x0404
	SB_SETTEXTW = 0x040B
)

type LVCOLUMNW struct {
	Mask       uint32
	Fmt        int32
	Cx         int32
	PszText    *uint16
	CchTextMax int32
	ISubItem   int32
	IImage     int32
	IOrder     int32
}

type LVITEMW struct {
	Mask       uint32
	IItem      int32
	ISubItem   int32
	State      uint32
	StateMask  uint32
	PszText    *uint16
	CchTextMax int32
	IImage     int32
	LParam     uintptr
	IIndent    int32
	GroupId    int32
	CColumns   uint32
	PuColumns  *uint32
}

type NMITEMACTIVATE struct {
	Hdr       w32.NMHDR
	IItem     int32
	ISubItem  int32
	UNewState uint32
	UOldState uint32
	UChanged  uint32
	PtAction  w32.POINT
	LParam    uintptr
	UKeyFlags uint32
}

type NMLISTVIEW struct {
	Hdr       w32.NMHDR
	IItem     int32
	ISubItem  int32
	UNewState uint32
	UOldState uint32
	UChanged  uint32
	PtAction  w32.POINT
	LParam    uintptr
}

type NMCUSTOMDRAW struct {
	Hdr         w32.NMHDR
	DwDrawStage uint32
	Hdc         w32.HDC
	Rc          w32.RECT
	DwItemSpec  uintptr
	UItemState  uint32
	LItemlParam uintptr
}

type NMLVCUSTOMDRAW struct {
	Nmcd        NMCUSTOMDRAW
	ClrText     uint32
	ClrTextBk   uint32
	ISubItem    int32
	DwItemType  uint32
	ClrFace     uint32
	IIconEffect int32
	IIconPhase  int32
	IPartId     int32
	IStateId    int32
	RcText      w32.RECT
	UAlign      uint32
}

type NMTVCUSTOMDRAW struct {
	Nmcd      NMCUSTOMDRAW
	ClrText   uint32
	ClrTextBk uint32
	ILevel    int32
}

type TCITEMW struct {
	Mask        uint32
	DwState     uint32
	DwStateMask uint32
	PszText     *uint16
	CchTextMax  int32
	IImage      int32
	LParam      uintptr
}

type BROWSEINFOW struct {
	HwndOwner      w32.HWND
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

// ListView helpers
func setListViewExtendedStyle(hwnd w32.HWND, exStyle uint32) {
	w32.SendMessage(hwnd, LVM_SETEXTENDEDLISTVIEWSTYLE, 0, uintptr(exStyle))
}

func addListViewColumn(hwnd w32.HWND, colIndex int32, title string, width int32, alignLeft bool) {
	fmtFlag := LVCFMT_LEFT
	if !alignLeft {
		fmtFlag = LVCFMT_RIGHT
	}

	pTitle, _ := windows.UTF16PtrFromString(title)
	col := LVCOLUMNW{
		Mask:     LVCF_FMT | LVCF_WIDTH | LVCF_TEXT | LVCF_SUBITEM,
		Fmt:      int32(fmtFlag),
		Cx:       width,
		PszText:  pTitle,
		ISubItem: colIndex,
	}
	w32.SendMessage(hwnd, LVM_INSERTCOLUMNW, uintptr(colIndex), uintptr(unsafe.Pointer(&col)))
}

func addListViewRow(hwnd w32.HWND, rowIndex int32, values []string, checked bool) {
	if len(values) == 0 {
		return
	}

	state := uint32(0x1000)
	if checked {
		state = 0x2000
	}

	pText0, _ := windows.UTF16PtrFromString(values[0])
	item := LVITEMW{
		Mask:      LVIF_TEXT | LVIF_STATE,
		IItem:     rowIndex,
		ISubItem:  0,
		State:     state,
		StateMask: LVIS_STATEIMAGEMASK,
		PszText:   pText0,
	}
	w32.SendMessage(hwnd, LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&item)))

	for col := 1; col < len(values); col++ {
		pTextCol, _ := windows.UTF16PtrFromString(values[col])
		subItem := LVITEMW{
			Mask:     LVIF_TEXT,
			IItem:    rowIndex,
			ISubItem: int32(col),
			PszText:  pTextCol,
		}
		w32.SendMessage(hwnd, LVM_SETITEMTEXTW, uintptr(rowIndex), uintptr(unsafe.Pointer(&subItem)))
	}
}

func clearListView(hwnd w32.HWND) {
	prev := isPopulatingListView
	isPopulatingListView = true
	defer func() { isPopulatingListView = prev }()
	w32.SendMessage(hwnd, LVM_DELETEALLITEMS, 0, 0)
}

func setListViewItemChecked(hwnd w32.HWND, index int, checked bool) {
	prev := isPopulatingListView
	isPopulatingListView = true
	defer func() { isPopulatingListView = prev }()

	state := uint32(0x1000)
	if checked {
		state = 0x2000
	}
	item := LVITEMW{
		State:     state,
		StateMask: LVIS_STATEIMAGEMASK,
	}
	w32.SendMessage(hwnd, LVM_SETITEMSTATE, uintptr(index), uintptr(unsafe.Pointer(&item)))
}

func isListViewItemChecked(hwnd w32.HWND, index int) bool {
	res := w32.SendMessage(hwnd, LVM_GETITEMSTATE, uintptr(index), uintptr(LVIS_STATEIMAGEMASK))
	return (res & 0x2000) != 0
}

func setAllListViewCheckboxes(hwnd w32.HWND, checked bool) {
	count := int(w32.SendMessage(hwnd, LVM_GETITEMCOUNT, 0, 0))
	for i := 0; i < count; i++ {
		setListViewItemChecked(hwnd, i, checked)
	}
}

func getSelectedListViewItems(hwnd w32.HWND) []int {
	var selected []int
	idx := -1
	for {
		idx = int(w32.SendMessage(hwnd, LVM_GETNEXTITEM, uintptr(idx), uintptr(LVNI_SELECTED)))
		if idx == -1 {
			break
		}
		selected = append(selected, idx)
	}
	return selected
}

// TabControl helpers
func addTabItem(hwnd w32.HWND, index int, title string) {
	pText, _ := windows.UTF16PtrFromString(title)
	item := TCITEMW{
		Mask:    TCIF_TEXT,
		PszText: pText,
	}
	w32.SendMessage(hwnd, TCM_INSERTITEM, uintptr(index), uintptr(unsafe.Pointer(&item)))
}

func getSelectedTab(hwnd w32.HWND) int {
	return int(w32.SendMessage(hwnd, TCM_GETCURSEL, 0, 0))
}

// StatusBar helpers
func setStatusBarParts(hwnd w32.HWND, rightEdges []int32) {
	w32.SendMessage(hwnd, SB_SETPARTS, uintptr(len(rightEdges)), uintptr(unsafe.Pointer(&rightEdges[0])))
}

func setStatusBarText(hwnd w32.HWND, partIndex int, text string) {
	pText, _ := windows.UTF16PtrFromString(text)
	w32.SendMessage(hwnd, SB_SETTEXTW, uintptr(partIndex), uintptr(unsafe.Pointer(pText)))
}

// System font helper: creates clean Segoe UI via gdi32 CreateFontW
func createSegoeFont(size int, bold bool) w32.HFONT {
	weight := 400
	if bold {
		weight = 700
	}
	face, _ := windows.UTF16PtrFromString("Segoe UI")
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

func setControlFont(hwnd w32.HWND, hFont w32.HFONT) {
	w32.SendMessage(hwnd, w32.WM_SETFONT, uintptr(hFont), 1)
}

// BrowseFolderDialog opens standard Windows folder picker
func BrowseFolderDialog(parent w32.HWND, title string) (string, bool) {
	pTitle, _ := windows.UTF16PtrFromString(title)
	displayBuf := make([]uint16, 260)

	bi := BROWSEINFOW{
		HwndOwner:      parent,
		PszDisplayName: &displayBuf[0],
		LpszTitle:      pTitle,
		UlFlags:        0x00000001 | 0x00000040, // BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE
	}

	pidl, _, _ := shBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", false
	}

	pathBuf := make([]uint16, 260)
	ret, _, _ := shGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ret == 0 {
		return "", false
	}

	return windows.UTF16ToString(pathBuf), true
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func setEditCueBanner(hwnd w32.HWND, text string) {
	pText, _ := windows.UTF16PtrFromString(text)
	w32.SendMessage(hwnd, 0x1501, 1, uintptr(unsafe.Pointer(pText)))
}

func setWindowLongPtr(hwnd w32.HWND, index int, newLong uintptr) uintptr {
	ret, _, _ := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index), newLong)
	return ret
}

func callWindowProc(lpPrevWndFunc uintptr, hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procCallWindowProcW.Call(lpPrevWndFunc, uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func playSystemChime() {
	procMessageBeep.Call(0) // MB_OK standard chime
}

func setClipboardText(hwnd w32.HWND, text string) bool {
	if !w32.OpenClipboard(hwnd) {
		return false
	}
	defer w32.CloseClipboard()
	w32.EmptyClipboard()

	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return false
	}

	bytesLen := uint32(len(utf16) * 2)
	hMem := w32.GlobalAlloc(0x0002 /* GMEM_MOVEABLE */, bytesLen)
	if hMem == 0 {
		return false
	}

	ptr := w32.GlobalLock(hMem)
	if ptr == nil {
		return false
	}
	slice := unsafe.Slice((*uint16)(ptr), len(utf16))
	copy(slice, utf16)
	w32.GlobalUnlock(hMem)

	return w32.SetClipboardData(13 /* CF_UNICODETEXT */, w32.HANDLE(hMem)) != 0
}

func playSuccessChime() {
	procMessageBeep.Call(0x40 /* MB_ICONASTERISK */)
}

func getDriveSpaceSummary(path string) string {
	vol := filepath.VolumeName(path)
	if vol == "" {
		vol = "C:"
	}
	root := vol + `\`
	pRoot, _ := windows.UTF16PtrFromString(root)
	var freeAvail, totalBytes, totalFree uint64
	r1, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pRoot)),
		uintptr(unsafe.Pointer(&freeAvail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r1 != 0 && totalBytes > 0 {
		freeGB := float64(freeAvail) / (1024 * 1024 * 1024)
		totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
		pctFree := (float64(freeAvail) / float64(totalBytes)) * 100
		return fmt.Sprintf("DRIVE %s %.1f GB free / %.1f GB (%.0f%% free)", vol, freeGB, totalGB, pctFree)
	}
	return "DRIVE " + vol
}

func bringWindowToForeground(hwnd w32.HWND) {
	if hwnd == 0 {
		return
	}
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	var fgThreadId uint32
	if fgHwnd != 0 {
		fgThreadIdRet, _, _ := procGetWindowThreadProcessId.Call(fgHwnd, 0)
		fgThreadId = uint32(fgThreadIdRet)
	}
	curThreadId := windows.GetCurrentThreadId()

	if fgThreadId != 0 && fgThreadId != curThreadId {
		procAttachThreadInput.Call(uintptr(curThreadId), uintptr(fgThreadId), 1)
		procShowWindow.Call(uintptr(hwnd), uintptr(w32.SW_RESTORE))
		procSetForegroundWindow.Call(uintptr(hwnd))
		procAttachThreadInput.Call(uintptr(curThreadId), uintptr(fgThreadId), 0)
	} else {
		procShowWindow.Call(uintptr(hwnd), uintptr(w32.SW_RESTORE))
		procSetForegroundWindow.Call(uintptr(hwnd))
	}
}

func activateExistingWindow() bool {
	cls, _ := windows.UTF16PtrFromString("RsyncControllerWin32Class")
	r0, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cls)), 0)
	if r0 != 0 {
		bringWindowToForeground(w32.HWND(r0))
		return true
	}
	title, _ := windows.UTF16PtrFromString("Rsync Controller")
	r0, _, _ = procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if r0 != 0 {
		bringWindowToForeground(w32.HWND(r0))
		return true
	}
	return false
}

func selectAndFocusListViewItem(hwndLV w32.HWND, index int) {
	if hwndLV == 0 || index < 0 {
		return
	}
	var lviClear LVITEMW
	lviClear.Mask = 0x0008 // LVIF_STATE
	lviClear.StateMask = LVIS_SELECTED | LVIS_FOCUSED
	lviClear.State = 0
	w32.SendMessage(hwndLV, LVM_SETITEMSTATE, ^uintptr(0), uintptr(unsafe.Pointer(&lviClear)))

	var lvi LVITEMW
	lvi.Mask = 0x0008 // LVIF_STATE
	lvi.StateMask = LVIS_SELECTED | LVIS_FOCUSED
	lvi.State = LVIS_SELECTED | LVIS_FOCUSED
	w32.SendMessage(hwndLV, LVM_SETITEMSTATE, uintptr(index), uintptr(unsafe.Pointer(&lvi)))
	w32.SendMessage(hwndLV, LVM_ENSUREVISIBLE, uintptr(index), 0)
	w32.SetFocus(hwndLV)
}

var (
	comdlg32DLL      = windows.NewLazySystemDLL("comdlg32.dll")
	procChooseColorW = comdlg32DLL.NewProc("ChooseColorW")
	customColors     [16]uint32
)

type CHOOSECOLORW struct {
	LStructSize    uint32
	HwndOwner      w32.HWND
	HInstance      w32.HWND
	RgbResult      uint32
	LpCustColors   *uint32
	Flags          uint32
	LCustData      uintptr
	LpfnHook       uintptr
	LpTemplateName *uint16
}

func chooseColorDialog(hwndOwner w32.HWND, initialColor uint32) (uint32, bool) {
	var cc CHOOSECOLORW
	cc.LStructSize = uint32(unsafe.Sizeof(cc))
	cc.HwndOwner = hwndOwner
	cc.RgbResult = initialColor
	cc.LpCustColors = &customColors[0]
	cc.Flags = 0x00000001 | 0x00000002 // CC_RGBINIT | CC_FULLOPEN

	r0, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&cc)))
	if r0 != 0 {
		return cc.RgbResult, true
	}
	return initialColor, false
}

