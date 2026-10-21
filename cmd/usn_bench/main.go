//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"rsync-controller/core"
)

func main() {
	drive := "E:"
	targetFolder := `E:\Backup\pgwiz\rsync`
	if len(os.Args) > 1 {
		drive = os.Args[1]
	}

	fmt.Printf("INFO: Attempting elevated USN Journal scan on volume %s...\n", drive)

	start := time.Now()
	idx, err := core.ScanVolumeUSN(drive)
	elapsed := time.Since(start)

	var folderID uint64

	if err != nil {
		fmt.Printf("NOTICE: Direct volume USN access restricted: %v\n", err)
		fmt.Printf("INFO: Falling back to non-elevated folder scanner on: %s\n", targetFolder)

		fStart := time.Now()
		var fErr error
		idx, folderID, fErr = core.ScanFolderFallback(targetFolder)
		fElapsed := time.Since(fStart)

		if fErr != nil {
			fmt.Printf("FAIL: Fallback scan failed: %v\n", fErr)
			os.Exit(1)
		}

		fmt.Printf("SUCCESS: Fallback scan indexed %d items in %v\n", len(idx.Records), fElapsed)
	} else {
		recordCount := len(idx.Records)
		fmt.Printf("SUCCESS: Elevated USN scan completed: %d records in %v (%.0f records/sec)\n",
			recordCount, elapsed, float64(recordCount)/elapsed.Seconds())

		var found bool
		folderID, found = idx.FindFolderID(targetFolder)
		if !found {
			fmt.Printf("WARN: Folder not resolved in USN index: %s\n", targetFolder)
		}
	}

	// List children of target folder
	fmt.Printf("\nINFO: Direct children of target folder:\n")
	children := idx.GetChildren(folderID)
	for _, child := range children {
		kind := "FILE:"
		if child.IsDir {
			kind = "DIR: "
		}
		fmt.Printf("   %s ID=%-6d Size=%-10d %s\n", kind, child.ID, child.Size, child.Name)
	}

	// Test instant in-memory search
	query := "theme"
	fmt.Printf("\nINFO: Running instant in-memory search for query: %q\n", query)
	sStart := time.Now()
	results := idx.SearchAll(query, 10)
	sElapsed := time.Since(sStart)

	fmt.Printf("SUCCESS: Found %d matches in %v:\n", len(results), sElapsed)
	for i, res := range results {
		fmt.Printf("   (%02d) %s -> %s\n", i+1, res.Record.Name, res.FullPath)
	}
}
