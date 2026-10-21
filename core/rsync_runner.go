package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SyncProgress struct {
	Percentage int    `json:"percentage"`
	Speed      string `json:"speed"`
	ETA        string `json:"eta"`
	Raw        string `json:"raw"`
}

type RsyncJob struct {
	ID          string        `json:"id"`
	Sources     []string      `json:"sources"`
	Destination string        `json:"destination"`
	IsPull      bool          `json:"isPull"`
	IsMove      bool          `json:"isMove"`
	Server      *ServerConfig `json:"server,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
}

// Progress pattern matching both GNU rsync and rsync-win output:
// e.g. "   12,451,840  45%   10.42MB/s    0:00:14"
// e.g. "        4,298,240 100%   43.28MB/s    0:00:00 (xfr#1, to-chk=0/1)"
var progressRegex = regexp.MustCompile(`\s+(\d+)%\s+([\d\.]+[A-Za-z]+/s)\s+([\d:]+)`)

var (
	rsyncWinCacheLock sync.RWMutex
	rsyncWinCache     = make(map[string]bool)
)

// IsRsyncWin detects whether the target rsync binary is the Windows-native 'rsync-win'
// flavor, which requires --src/--dest flags and forwards slashes instead of GNU rsync syntax.
func IsRsyncWin(rsyncPath string) bool {
	if rsyncPath == "" {
		rsyncPath = "rsync"
	}

	rsyncWinCacheLock.RLock()
	val, ok := rsyncWinCache[rsyncPath]
	rsyncWinCacheLock.RUnlock()
	if ok {
		return val
	}

	out, err := exec.Command(rsyncPath, "--version").Output()
	isWin := false
	if err == nil {
		isWin = strings.Contains(strings.ToLower(string(out)), "rsync-win")
	}

	rsyncWinCacheLock.Lock()
	rsyncWinCache[rsyncPath] = isWin
	rsyncWinCacheLock.Unlock()

	return isWin
}

// RunRsync executes the rsync binary with progress tracking.
func RunRsync(ctx context.Context, rsyncPath string, job *RsyncJob, onProgress func(p SyncProgress)) error {
	return RunRsyncWithOutput(ctx, rsyncPath, job, onProgress, nil)
}

// RunRsyncWithOutput executes the rsync binary with progress tracking and streaming raw log output.
func RunRsyncWithOutput(ctx context.Context, rsyncPath string, job *RsyncJob, onProgress func(p SyncProgress), onOutput func(line string)) error {
	if rsyncPath == "" {
		rsyncPath = "rsync"
	}

	if IsRsyncWin(rsyncPath) {
		return runRsyncWin(ctx, rsyncPath, job, onProgress, onOutput)
	}
	return runRsyncGnu(ctx, rsyncPath, job, onProgress, onOutput)
}

func runRsyncWin(ctx context.Context, rsyncPath string, job *RsyncJob, onProgress func(p SyncProgress), onOutput func(line string)) error {
	cleanDest := filepath.ToSlash(job.Destination)

	sources := job.Sources
	if len(sources) == 0 {
		return fmt.Errorf("no source paths provided for rsync transfer")
	}

	for _, src := range sources {
		cleanSrc := filepath.ToSlash(src)

		var args []string
		args = append(args, "-a", "-v", "--progress")

		if job.Server != nil {
			if job.Server.Port != 0 && job.Server.Port != 22 {
				args = append(args, "--ssh-port", strconv.Itoa(job.Server.Port))
			}
			if job.Server.KeyPath != "" {
				args = append(args, "-i", filepath.ToSlash(job.Server.KeyPath))
			}
			if job.IsPull {
				remoteSrc := fmt.Sprintf("%s@%s:%s", job.Server.User, job.Server.Host, job.Server.RemotePath)
				args = append(args, "--src", remoteSrc, "--dest", cleanDest)
			} else {
				remoteDest := fmt.Sprintf("%s@%s:%s", job.Server.User, job.Server.Host, job.Server.RemotePath)
				args = append(args, "--src", cleanSrc, "--dest", remoteDest)
			}
		} else {
			args = append(args, "--src", cleanSrc, "--dest", cleanDest)
		}

		if onOutput != nil {
			onOutput(fmt.Sprintf("RUN: %s %s\r\n", rsyncPath, strings.Join(args, " ")))
		}

		cmd := exec.CommandContext(ctx, rsyncPath, args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to open stdout pipe: %w", err)
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start %s: %w", rsyncPath, err)
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if onOutput != nil {
				onOutput(line)
			}
			matches := progressRegex.FindStringSubmatch(line)
			if len(matches) == 4 && onProgress != nil {
				pct, _ := strconv.Atoi(matches[1])
				onProgress(SyncProgress{
					Percentage: pct,
					Speed:      matches[2],
					ETA:        matches[3],
					Raw:        strings.TrimSpace(line),
				})
			}
		}

		if err := cmd.Wait(); err != nil {
			return err
		}

		if job.IsMove {
			_ = os.RemoveAll(src)
		}
	}

	return nil
}

func runRsyncGnu(ctx context.Context, rsyncPath string, job *RsyncJob, onProgress func(p SyncProgress), onOutput func(line string)) error {
	var args []string
	args = append(args, "-av", "--progress")
	if job.IsMove {
		args = append(args, "--remove-source-files")
	}

	if job.Server != nil {
		sshCmd := fmt.Sprintf("ssh -p %d", job.Server.Port)
		if job.Server.KeyPath != "" {
			sshCmd += fmt.Sprintf(" -i %s", job.Server.KeyPath)
		}
		args = append(args, "-e", sshCmd)

		if job.IsPull {
			remoteSrc := fmt.Sprintf("%s@%s:%s", job.Server.User, job.Server.Host, job.Server.RemotePath)
			args = append(args, remoteSrc, job.Destination)
		} else {
			args = append(args, job.Sources...)
			remoteDest := fmt.Sprintf("%s@%s:%s", job.Server.User, job.Server.Host, job.Server.RemotePath)
			args = append(args, remoteDest)
		}
	} else {
		args = append(args, job.Sources...)
		args = append(args, job.Destination)
	}

	if onOutput != nil {
		onOutput(fmt.Sprintf("RUN: %s %s\r\n", rsyncPath, strings.Join(args, " ")))
	}

	cmd := exec.CommandContext(ctx, rsyncPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", rsyncPath, err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if onOutput != nil {
			onOutput(line)
		}
		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) == 4 && onProgress != nil {
			pct, _ := strconv.Atoi(matches[1])
			onProgress(SyncProgress{
				Percentage: pct,
				Speed:      matches[2],
				ETA:        matches[3],
				Raw:        strings.TrimSpace(line),
			})
		}
	}

	return cmd.Wait()
}
