package core

import (
	"sort"
	"strings"
)

type SearchResult struct {
	Record   *FileRecord `json:"record"`
	FullPath string      `json:"fullPath"`
}

// SearchSubtree searches for matching records within a specific directory subtree.
func (idx *VolumeIndex) SearchSubtree(parentID uint64, query string, maxResults int) []*SearchResult {
	query = strings.ToLower(query)
	var results []*SearchResult

	var traverse func(id uint64)
	traverse = func(id uint64) {
		if maxResults > 0 && len(results) >= maxResults {
			return
		}
		children := idx.GetChildren(id)
		for _, child := range children {
			if query == "" || strings.Contains(strings.ToLower(child.Name), query) {
				results = append(results, &SearchResult{
					Record:   child,
					FullPath: idx.ResolvePath(child.ID),
				})
			}
			if child.IsDir {
				traverse(child.ID)
			}
		}
	}

	traverse(parentID)

	// Sort results: directories first, then alphabetical
	sort.Slice(results, func(i, j int) bool {
		if results[i].Record.IsDir != results[j].Record.IsDir {
			return results[i].Record.IsDir
		}
		return strings.ToLower(results[i].Record.Name) < strings.ToLower(results[j].Record.Name)
	})

	return results
}

// SearchAll searches the entire volume index for records matching the query string.
func (idx *VolumeIndex) SearchAll(query string, maxResults int) []*SearchResult {
	query = strings.ToLower(query)
	var results []*SearchResult

	for _, rec := range idx.Records {
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
		if query == "" || strings.Contains(strings.ToLower(rec.Name), query) {
			results = append(results, &SearchResult{
				Record:   rec,
				FullPath: idx.ResolvePath(rec.ID),
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Record.IsDir != results[j].Record.IsDir {
			return results[i].Record.IsDir
		}
		return strings.ToLower(results[i].Record.Name) < strings.ToLower(results[j].Record.Name)
	})

	return results
}
