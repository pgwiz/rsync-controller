package core

// FileRecord represents metadata of a single file or directory indexed from NTFS.
type FileRecord struct {
	ID         uint64 `json:"id"`
	ParentID   uint64 `json:"parentId"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Attributes uint32 `json:"attributes"`
	MTime      int64  `json:"mtime"`
	IsDir      bool   `json:"isDir"`
}

// VolumeIndex holds the complete in-memory parent-child tree for an entire drive.
type VolumeIndex struct {
	DriveLetter string                  `json:"driveLetter"`
	Records     map[uint64]*FileRecord  `json:"-"`
	Children    map[uint64][]uint64     `json:"-"` // ParentID -> slice of ChildIDs
	RootID      uint64                  `json:"rootId"`
	PathToID    map[string]uint64       `json:"-"` // Cache of resolved directory paths
}

// NewVolumeIndex initializes an empty index for a drive letter (e.g. "E:").
func NewVolumeIndex(driveLetter string) *VolumeIndex {
	return &VolumeIndex{
		DriveLetter: driveLetter,
		Records:     make(map[uint64]*FileRecord),
		Children:    make(map[uint64][]uint64),
		PathToID:    make(map[string]uint64),
	}
}

// AddRecord adds a record and registers it under its parent's child list.
func (idx *VolumeIndex) AddRecord(rec *FileRecord) {
	idx.Records[rec.ID] = rec
	idx.Children[rec.ParentID] = append(idx.Children[rec.ParentID], rec.ID)
}

// GetChildren returns all direct child records for a given parent record ID.
func (idx *VolumeIndex) GetChildren(parentID uint64) []*FileRecord {
	childIDs, exists := idx.Children[parentID]
	if !exists {
		return nil
	}
	out := make([]*FileRecord, 0, len(childIDs))
	for _, id := range childIDs {
		if rec, ok := idx.Records[id]; ok {
			out = append(out, rec)
		}
	}
	return out
}

// ResolvePath builds the full absolute path of a record by climbing up to the root.
func (idx *VolumeIndex) ResolvePath(id uint64) string {
	curr, ok := idx.Records[id]
	if !ok {
		return ""
	}

	parts := []string{}
	visited := make(map[uint64]bool)

	for curr != nil && !visited[curr.ID] {
		visited[curr.ID] = true
		if curr.ID == idx.RootID || curr.Name == "" {
			break
		}
		parts = append([]string{curr.Name}, parts...)
		if curr.ParentID == curr.ID || curr.ParentID == 0 {
			break
		}
		curr = idx.Records[curr.ParentID]
	}

	prefix := idx.DriveLetter
	if len(prefix) > 0 && prefix[len(prefix)-1] != '\\' {
		prefix += "\\"
	}

	for i, part := range parts {
		if i == 0 {
			prefix += part
		} else {
			prefix += "\\" + part
		}
	}
	return prefix
}
