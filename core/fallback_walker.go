package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// ScanFolderFallback crawls a specific directory tree when volume-level USN access is unavailable.
func ScanFolderFallback(rootPath string) (*VolumeIndex, uint64, error) {
	cleanRoot := filepath.Clean(rootPath)
	vol := filepath.VolumeName(cleanRoot)
	if vol == "" {
		vol = "C:"
	}

	idx := NewVolumeIndex(vol)
	var nextID uint64 = 1000

	// Create root record
	rootID := atomic.AddUint64(&nextID, 1)
	idx.RootID = rootID
	idx.Records[rootID] = &FileRecord{
		ID:       rootID,
		ParentID: 0,
		Name:     filepath.Base(cleanRoot),
		IsDir:    true,
	}

	pathToID := make(map[string]uint64)
	pathToID[strings.ToLower(cleanRoot)] = rootID

	err := filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Skip unreadable folders
		}
		if path == cleanRoot {
			return nil
		}

		parentPath := filepath.Dir(path)
		parentID, exists := pathToID[strings.ToLower(parentPath)]
		if !exists {
			parentID = rootID
		}

		info, err := d.Info()
		var size int64
		var mtime int64
		if err == nil {
			size = info.Size()
			mtime = info.ModTime().Unix()
		}

		recID := atomic.AddUint64(&nextID, 1)
		rec := &FileRecord{
			ID:       recID,
			ParentID: parentID,
			Name:     d.Name(),
			Size:     size,
			MTime:    mtime,
			IsDir:    d.IsDir(),
		}
		idx.AddRecord(rec)

		if d.IsDir() {
			pathToID[strings.ToLower(path)] = recID
		}

		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("fallback scan error on %s: %w", rootPath, err)
	}

	return idx, rootID, nil
}
