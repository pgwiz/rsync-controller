package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVolumeIndexOperations(t *testing.T) {
	idx := NewVolumeIndex("E:")

	root := &FileRecord{ID: 5, ParentID: 0, Name: "", IsDir: true}
	folderA := &FileRecord{ID: 100, ParentID: 5, Name: "Backup", IsDir: true}
	folderB := &FileRecord{ID: 101, ParentID: 100, Name: "pgwiz", IsDir: true}
	file1 := &FileRecord{ID: 200, ParentID: 101, Name: "data.iso", Size: 4096, IsDir: false}
	file2 := &FileRecord{ID: 201, ParentID: 101, Name: "notes.txt", Size: 128, IsDir: false}

	idx.AddRecord(root)
	idx.AddRecord(folderA)
	idx.AddRecord(folderB)
	idx.AddRecord(file1)
	idx.AddRecord(file2)

	// Test GetChildren
	children := idx.GetChildren(101)
	if len(children) != 2 {
		t.Fatalf("expected 2 children under folderB, got %d", len(children))
	}

	// Test ResolvePath
	p := idx.ResolvePath(200)
	expected := `E:\Backup\pgwiz\data.iso`
	if p != expected {
		t.Errorf("expected path %q, got %q", expected, p)
	}

	// Test SearchSubtree
	results := idx.SearchSubtree(100, "data", 10)
	if len(results) != 1 || results[0].Record.Name != "data.iso" {
		t.Errorf("expected 1 match for 'data', got %d", len(results))
	}

	// Test SearchAll
	allResults := idx.SearchAll("notes", 10)
	if len(allResults) != 1 || allResults[0].Record.Name != "notes.txt" {
		t.Errorf("expected 1 match for 'notes', got %d", len(allResults))
	}
}

func TestLaunchPayloadSerialization(t *testing.T) {
	payload := LaunchPayload{
		Mode:      "copy",
		Server:    "alpha",
		Src:       `E:\Backup\test`,
		Dest:      `/var/backup`,
		Paths:     []string{`E:\Backup\test\file1.txt`, `E:\Backup\test\file2.txt`},
		AutoStage: true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var recovered LaunchPayload
	if err := json.Unmarshal(data, &recovered); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if recovered.Mode != "copy" || recovered.Server != "alpha" || recovered.AutoStage != true {
		t.Errorf("payload fields mismatch: %+v", recovered)
	}
	if len(recovered.Paths) != 2 || recovered.Paths[0] != `E:\Backup\test\file1.txt` {
		t.Errorf("recovered paths mismatch: %+v", recovered.Paths)
	}
}

func TestPipeCommunication(t *testing.T) {
	testPipeName := `\\.\pipe\rsync-ui-test-pipe`
	ln, err := OpenPipeListener(testPipeName)
	if err != nil {
		t.Fatalf("OpenPipeListener failed: %v", err)
	}
	defer ln.Close()

	done := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- ""
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		done <- string(buf[:n])
	}()

	clientConn, err := DialPipe(testPipeName, 2*time.Second)
	if err != nil {
		t.Fatalf("DialPipe failed: %v", err)
	}
	defer clientConn.Close()

	testMsg := "HELO_RSYNC_PIPE\n"
	_, err = clientConn.Write([]byte(testMsg))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	received := <-done
	if received != testMsg {
		t.Errorf("expected %q, got %q", testMsg, received)
	}
}

func TestRsyncJobMoveSupport(t *testing.T) {
	job := &RsyncJob{
		ID:          "job_move_1",
		Sources:     []string{`E:\test\file.txt`},
		Destination: `E:\dest`,
		IsMove:      true,
		IsPull:      false,
	}

	if !job.IsMove {
		t.Errorf("expected IsMove to be true")
	}

	payloadMoveTo := LaunchPayload{
		Mode:      "move_to",
		Src:       `E:\test\file.txt`,
		AutoStage: true,
	}
	data, err := json.Marshal(payloadMoveTo)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	var recovered LaunchPayload
	if err := json.Unmarshal(data, &recovered); err != nil || recovered.Mode != "move_to" {
		t.Fatalf("unexpected recovered mode: %v", recovered.Mode)
	}
}

func TestThemeConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ShadowDepth != 2 {
		t.Errorf("expected default ShadowDepth to be 2, got %d", cfg.ShadowDepth)
	}
	if cfg.CustomTheme == nil {
		t.Fatalf("expected CustomTheme to be initialized")
	}
	if cfg.CustomTheme.ShadowDepth != 2 {
		t.Errorf("expected CustomTheme.ShadowDepth to be 2, got %d", cfg.CustomTheme.ShadowDepth)
	}
	if cfg.CustomTheme.CornerRadius != 8 {
		t.Errorf("expected CustomTheme.CornerRadius to be 8, got %d", cfg.CustomTheme.CornerRadius)
	}
}

func TestRsyncWinDetection(t *testing.T) {
	isWin := IsRsyncWin("rsync")
	t.Logf("rsync binary detected as rsync-win: %v", isWin)
}

func TestRsyncExecutionLive(t *testing.T) {
	tmpDest := filepath.Join(os.TempDir(), "rsync_test_dest")
	_ = os.MkdirAll(tmpDest, 0755)
	defer os.RemoveAll(tmpDest)

	srcFile := filepath.Join(".", "core.go")
	if _, err := os.Stat("config.go"); err == nil {
		srcFile = "config.go"
	}

	job := &RsyncJob{
		ID:          "test_job_1",
		Sources:     []string{srcFile},
		Destination: tmpDest,
		IsMove:      false,
	}

	var outputLines []string
	cfg := DefaultConfig()
	err := RunRsyncWithOutput(context.Background(), cfg.RsyncBinaryPath, job, nil, func(line string) {
		outputLines = append(outputLines, line)
	})

	if err != nil {
		t.Fatalf("RunRsyncWithOutput failed: %v\nOutput:\n%s", err, strings.Join(outputLines, "\n"))
	}
	t.Logf("Sync successful! Output lines: %d", len(outputLines))
}

