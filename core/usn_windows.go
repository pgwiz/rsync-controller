//go:build windows

package core

import (
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	FSCTL_ENUM_USN_DATA      = 0x000900b3
	FSCTL_QUERY_USN_JOURNAL  = 0x000900f4
	USN_PAGE_SIZE            = 64 * 1024 // 64 KB buffer per IO call
	FILE_ATTRIBUTE_DIRECTORY = 0x00000010
)

type MftEnumDataV0 struct {
	StartFileReferenceNumber uint64
	LowUsn                   int64
	HighUsn                  int64
}

type UsnJournalDataV0 struct {
	UsnJournalID    uint64
	LowestUsn       int64
	HighestUsn      int64
	MaximumUsn      int64
	MaximumSize     int64
	AllocationDelta int64
}

// OpenVolumeHandle opens an elevated handle to the target volume (e.g. `\\.\C:` or `\\.\E:`).
func OpenVolumeHandle(driveLetter string) (windows.Handle, error) {
	driveLetter = strings.TrimSuffix(driveLetter, "\\")
	driveLetter = strings.TrimSuffix(driveLetter, "/")
	if !strings.HasSuffix(driveLetter, ":") {
		driveLetter += ":"
	}
	volPath := fmt.Sprintf(`\\.\%s`, driveLetter)

	pathPtr, err := windows.UTF16PtrFromString(volPath)
	if err != nil {
		return windows.InvalidHandle, err
	}

	h, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		// Fallback to read-only access if write permission is restricted
		h, err = windows.CreateFile(
			pathPtr,
			windows.FILE_READ_DATA|windows.FILE_READ_ATTRIBUTES,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_BACKUP_SEMANTICS,
			0,
		)
	}
	return h, err
}

// ScanVolumeUSN enumerates all files and directories on the given drive via USN Journal.
func ScanVolumeUSN(driveLetter string) (*VolumeIndex, error) {
	hVol, err := OpenVolumeHandle(driveLetter)
	if err != nil {
		return nil, fmt.Errorf("failed to open volume handle for %s: %w (administrator rights required)", driveLetter, err)
	}
	defer windows.CloseHandle(hVol)

	// Query USN Journal to find HighUsn
	var journalData UsnJournalDataV0
	var bytesReturned uint32
	highUsn := int64(math.MaxInt64)

	err = windows.DeviceIoControl(
		hVol,
		FSCTL_QUERY_USN_JOURNAL,
		nil,
		0,
		(*byte)(unsafe.Pointer(&journalData)),
		uint32(unsafe.Sizeof(journalData)),
		&bytesReturned,
		nil,
	)
	if err == nil && journalData.HighestUsn > 0 {
		highUsn = journalData.HighestUsn
	}

	enumData := MftEnumDataV0{
		StartFileReferenceNumber: 0,
		LowUsn:                   0,
		HighUsn:                  highUsn,
	}

	index := NewVolumeIndex(driveLetter)
	buf := make([]byte, USN_PAGE_SIZE)

	for {
		err = windows.DeviceIoControl(
			hVol,
			FSCTL_ENUM_USN_DATA,
			(*byte)(unsafe.Pointer(&enumData)),
			uint32(unsafe.Sizeof(enumData)),
			&buf[0],
			uint32(len(buf)),
			&bytesReturned,
			nil,
		)

		if err != nil {
			if err == windows.ERROR_HANDLE_EOF || err == syscall.Errno(38) {
				break // Normal end of MFT enumeration
			}
			return nil, fmt.Errorf("FSCTL_ENUM_USN_DATA error: %w", err)
		}

		if bytesReturned < 8 {
			break
		}

		// Next starting reference number is in the first 8 bytes
		enumData.StartFileReferenceNumber = binary.LittleEndian.Uint64(buf[0:8])

		offset := 8
		for offset+60 <= int(bytesReturned) {
			recLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
			if recLen == 0 || offset+int(recLen) > int(bytesReturned) {
				break
			}

			// Parse USN_RECORD_V2
			fileRefNum := binary.LittleEndian.Uint64(buf[offset+8 : offset+16])
			parentRefNum := binary.LittleEndian.Uint64(buf[offset+16 : offset+24])
			fileAttr := binary.LittleEndian.Uint32(buf[offset+52 : offset+56])
			nameLen := binary.LittleEndian.Uint16(buf[offset+56 : offset+58])
			nameOffset := binary.LittleEndian.Uint16(buf[offset+58 : offset+60])

			// Mask the upper 16 bits of FileReferenceNumber (sequence number) to leave the 48-bit MFT index
			fileRefClean := fileRefNum & 0x0000FFFFFFFFFFFF
			parentRefClean := parentRefNum & 0x0000FFFFFFFFFFFF

			if int(nameOffset)+int(nameLen) <= int(recLen) {
				nameBytes := buf[offset+int(nameOffset) : offset+int(nameOffset)+int(nameLen)]
				u16s := make([]uint16, len(nameBytes)/2)
				for i := 0; i < len(u16s); i++ {
					u16s[i] = binary.LittleEndian.Uint16(nameBytes[i*2 : i*2+2])
				}
				fileName := string(utf16.Decode(u16s))

				isDir := (fileAttr & FILE_ATTRIBUTE_DIRECTORY) != 0

				rec := &FileRecord{
					ID:         fileRefClean,
					ParentID:   parentRefClean,
					Name:       fileName,
					Attributes: fileAttr,
					IsDir:      isDir,
				}
				index.AddRecord(rec)
			}

			offset += int(recLen)
		}
	}

	return index, nil
}

// FindFolderID resolves a full directory path (e.g. `E:\Backup\pgwiz\rsync`) to its FileRecord ID in the index.
func (idx *VolumeIndex) FindFolderID(folderPath string) (uint64, bool) {
	cleanPath := filepath.Clean(folderPath)
	vol := filepath.VolumeName(cleanPath)
	rel := strings.TrimPrefix(cleanPath, vol)
	rel = strings.TrimPrefix(rel, "\\")
	rel = strings.TrimPrefix(rel, "/")

	if rel == "" {
		// Root folder has standard MFT index 5 in NTFS
		return 5, true
	}

	parts := strings.Split(rel, "\\")
	currentID := uint64(5) // NTFS root directory MFT record index

	for _, part := range parts {
		if part == "" {
			continue
		}
		children := idx.GetChildren(currentID)
		found := false
		for _, child := range children {
			if child.IsDir && strings.EqualFold(child.Name, part) {
				currentID = child.ID
				found = true
				break
			}
		}
		if !found {
			return 0, false
		}
	}

	return currentID, true
}
