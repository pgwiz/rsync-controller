//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"
)

const (
	WC_TREEVIEW = "SysTreeView32"

	// Styles
	TVS_HASBUTTONS    = 0x0001
	TVS_HASLINES      = 0x0002
	TVS_LINESATROOT   = 0x0004
	TVS_SHOWSELALWAYS = 0x0020
	TVS_TRACKSELECT   = 0x0200

	// Messages
	TV_FIRST         = 0x1100
	TVM_INSERTITEMW  = TV_FIRST + 50
	TVM_DELETEITEM   = TV_FIRST + 1
	TVM_EXPAND       = TV_FIRST + 2
	TVM_GETITEMW     = TV_FIRST + 62
	TVM_SETITEMW     = TV_FIRST + 63
	TVM_SELECTITEM   = TV_FIRST + 11
	TVM_GETNEXTITEM  = TV_FIRST + 10
	TVM_SETBKCOLOR   = TV_FIRST + 29
	TVM_SETTEXTCOLOR = TV_FIRST + 30
	TVM_SETLINECOLOR = TV_FIRST + 40

	// Notification codes
	TVN_FIRST          = uint32(0xFFFFFFFF) - 400 + 1
	TVN_SELCHANGEDW    = TVN_FIRST - 50 // 0xFFFFFE38 / int32 -456
	TVN_ITEMEXPANDINGW = TVN_FIRST - 54 // 0xFFFFFE34 / int32 -460

	// Item flags
	TVIF_TEXT     = 0x0001
	TVIF_PARAM    = 0x0004
	TVIF_CHILDREN = 0x0040
	TVIF_HANDLE   = 0x0010

	// Special handles
	TVI_ROOT = uintptr(0xFFFF0000)
	TVI_LAST = uintptr(0xFFFF0002)

	// GetNextItem flags
	TVGN_ROOT     = 0x0000
	TVGN_NEXT     = 0x0001
	TVGN_PREVIOUS = 0x0002
	TVGN_PARENT   = 0x0003
	TVGN_CHILD    = 0x0004
	TVGN_CARET    = 0x0009

	// Expand flags
	TVE_EXPAND = 0x0002
)

type TVITEMW struct {
	Mask           uint32
	HItem          uintptr
	State          uint32
	StateMask      uint32
	PszText        *uint16
	CchTextMax     int32
	IImage         int32
	ISelectedImage int32
	CChildren      int32
	LParam         uintptr
}

type TVINSERTSTRUCTW struct {
	HParent      uintptr
	HInsertAfter uintptr
	Item         TVITEMW
}

type NMTREEVIEWW struct {
	Hdr     w32.NMHDR
	Action  uint32
	ItemOld TVITEMW
	ItemNew TVITEMW
	PtDrag  w32.POINT
}

var (
	hwndTreeView      w32.HWND
	showTreeView      = true
	hwndBtnToggleTree w32.HWND

	treeNodePaths     = make(map[uintptr]string) // hItem -> full path
	pathToTreeNode    = make(map[string]uintptr) // full path -> hItem
	treePopulatedDirs = make(map[uintptr]bool)   // hItem -> children populated
	isSyncingTree     bool

	procGetLogicalDrives      = kernel32DLL.NewProc("GetLogicalDrives")
	procGetDriveTypeW         = kernel32DLL.NewProc("GetDriveTypeW")
	procGetVolumeInformationW = kernel32DLL.NewProc("GetVolumeInformationW")
	procSetErrorMode          = kernel32DLL.NewProc("SetErrorMode")
)

type DriveInfo struct {
	Letter string
	Root   string
	Label  string
}

func normalizeDriveRoot(p string) string {
	clean := filepath.Clean(p)
	if len(clean) == 2 && clean[1] == ':' {
		return clean + `\`
	}
	return clean
}

func getAllSystemDrives() []DriveInfo {
	// Suppress modal system alerts for unready removable or optical drives
	oldMode, _, _ := procSetErrorMode.Call(0x0001 | 0x8000) // SEM_FAILCRITICALERRORS | SEM_NOOPENFILEERRORBOX
	defer procSetErrorMode.Call(oldMode)

	r0, _, _ := procGetLogicalDrives.Call()
	mask := uint32(r0)

	var drives []DriveInfo
	for i := 0; i < 26; i++ {
		if (mask & (1 << i)) != 0 {
			letter := string(rune('A' + i))
			root := letter + `:\`
			pRoot, _ := windows.UTF16PtrFromString(root)

			// Get Volume Label
			volBuf := make([]uint16, 260)
			fsBuf := make([]uint16, 260)
			var maxCompLen, fsFlags uint32
			r1, _, _ := procGetVolumeInformationW.Call(
				uintptr(unsafe.Pointer(pRoot)),
				uintptr(unsafe.Pointer(&volBuf[0])),
				uintptr(len(volBuf)),
				0,
				uintptr(unsafe.Pointer(&maxCompLen)),
				uintptr(unsafe.Pointer(&fsFlags)),
				uintptr(unsafe.Pointer(&fsBuf[0])),
				uintptr(len(fsBuf)),
			)

			volName := ""
			if r1 != 0 {
				volName = windows.UTF16ToString(volBuf)
			}

			// Get Drive Type
			driveTypeRet, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(pRoot)))
			driveType := uint32(driveTypeRet)

			label := ""
			if volName != "" {
				label = fmt.Sprintf("%s (%s:)", volName, letter)
			} else {
				switch driveType {
				case 2: // DRIVE_REMOVABLE
					label = fmt.Sprintf("USB Drive (%s:)", letter)
				case 4: // DRIVE_REMOTE
					label = fmt.Sprintf("Network Drive (%s:)", letter)
				case 5: // DRIVE_CDROM
					label = fmt.Sprintf("CD/DVD Drive (%s:)", letter)
				case 6: // DRIVE_RAMDISK
					label = fmt.Sprintf("RAM Disk (%s:)", letter)
				default:
					label = fmt.Sprintf("Local Disk (%s:)", letter)
				}
			}

			drives = append(drives, DriveInfo{
				Letter: letter,
				Root:   root,
				Label:  label,
			})
		}
	}

	if len(drives) == 0 {
		for _, l := range "CDEABFGHIJKLMNOPQRSTUVWXYZ" {
			d := string(l) + `:\`
			if _, err := os.Stat(d); err == nil {
				drives = append(drives, DriveInfo{
					Letter: string(l),
					Root:   d,
					Label:  fmt.Sprintf("Drive %s:", string(l)),
				})
			}
		}
	}

	return drives
}

func getSystemDrives() []string {
	all := getAllSystemDrives()
	var roots []string
	for _, d := range all {
		roots = append(roots, d.Root)
	}
	return roots
}

func insertTreeItem(hwndTV w32.HWND, hParent uintptr, label string, fullPath string, hasChildren bool) uintptr {
	pText, _ := windows.UTF16PtrFromString(label)
	var tvis TVINSERTSTRUCTW
	tvis.HParent = hParent
	tvis.HInsertAfter = TVI_LAST
	tvis.Item.Mask = TVIF_TEXT | TVIF_PARAM
	tvis.Item.PszText = pText

	if hasChildren {
		tvis.Item.Mask |= TVIF_CHILDREN
		tvis.Item.CChildren = 1
	}

	ret := w32.SendMessage(hwndTV, TVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&tvis)))
	if ret != 0 {
		clean := normalizeDriveRoot(fullPath)
		treeNodePaths[ret] = clean
		pathToTreeNode[strings.ToLower(clean)] = ret
	}
	return ret
}

func populateTreeRoots(hwndTV w32.HWND) {
	if hwndTV == 0 {
		return
	}
	w32.SendMessage(hwndTV, TVM_DELETEITEM, 0, TVI_ROOT)
	treeNodePaths = make(map[uintptr]string)
	pathToTreeNode = make(map[string]uintptr)
	treePopulatedDirs = make(map[uintptr]bool)

	drives := getAllSystemDrives()
	for _, drv := range drives {
		clean := normalizeDriveRoot(drv.Root)
		hDrv := insertTreeItem(hwndTV, TVI_ROOT, drv.Label, clean, true)
		populateTreeChildren(hwndTV, hDrv, clean)
	}
}

func populateTreeChildren(hwndTV w32.HWND, hParent uintptr, parentPath string) {
	if treePopulatedDirs[hParent] {
		return
	}
	treePopulatedDirs[hParent] = true

	cleanParent := normalizeDriveRoot(parentPath)
	entries, err := os.ReadDir(cleanParent)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "$") || strings.EqualFold(name, "System Volume Information") {
			continue
		}
		subPath := filepath.Join(cleanParent, name)
		hasSub := false
		if subEntries, subErr := os.ReadDir(subPath); subErr == nil {
			for _, se := range subEntries {
				if se.IsDir() {
					hasSub = true
					break
				}
			}
		}
		insertTreeItem(hwndTV, hParent, name, subPath, hasSub)
	}
}

func syncTreeSelection(hwndTV w32.HWND, targetPath string) {
	if hwndTV == 0 || isSyncingTree {
		return
	}
	isSyncingTree = true
	defer func() { isSyncingTree = false }()

	clean := filepath.Clean(targetPath)
	vol := filepath.VolumeName(clean)
	if vol == "" {
		return
	}

	root := vol + `\`
	rel, err := filepath.Rel(root, clean)
	if err != nil {
		return
	}

	curr := root
	segments := []string{curr}
	if rel != "." && rel != "" {
		parts := strings.Split(rel, string(filepath.Separator))
		for _, p := range parts {
			curr = filepath.Join(curr, p)
			segments = append(segments, curr)
		}
	}

	var lastItem uintptr
	for i, seg := range segments {
		cleanSeg := strings.ToLower(normalizeDriveRoot(seg))
		hItem, ok := pathToTreeNode[cleanSeg]
		if !ok {
			break
		}
		lastItem = hItem
		if i < len(segments)-1 {
			populateTreeChildren(hwndTV, hItem, normalizeDriveRoot(seg))
			w32.SendMessage(hwndTV, TVM_EXPAND, TVE_EXPAND, hItem)
		}
	}

	if lastItem != 0 {
		w32.SendMessage(hwndTV, TVM_SELECTITEM, TVGN_CARET, lastItem)
	}
}

func applyThemeToTreeView(hwndTV w32.HWND, theme *ThemeStyle) {
	if hwndTV == 0 || theme == nil {
		return
	}
	w32.SendMessage(hwndTV, TVM_SETBKCOLOR, 0, uintptr(theme.SurfaceBg))
	w32.SendMessage(hwndTV, TVM_SETTEXTCOLOR, 0, uintptr(theme.TextColor))
	w32.SendMessage(hwndTV, TVM_SETLINECOLOR, 0, uintptr(theme.BorderColor))
	w32.InvalidateRect(hwndTV, nil, true)
}
