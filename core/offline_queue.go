package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type QueuedJob struct {
	ID          string        `json:"id"`
	CreatedAt   time.Time     `json:"createdAt"`
	Sources     []string      `json:"sources"`
	Destination string        `json:"destination"`
	IsPull      bool          `json:"isPull"`
	ServerID    string        `json:"serverId"`
	ServerName  string        `json:"serverName"`
	Status      string        `json:"status"` // [QUEUED], [RUNNING], [COMPLETED], [FAILED]
	LastError   string        `json:"lastError,omitempty"`
}

type OfflineQueue struct {
	filePath string
	mu       sync.Mutex
	Jobs     []*QueuedJob `json:"jobs"`
}

func NewOfflineQueue(storagePath string) (*OfflineQueue, error) {
	q := &OfflineQueue{
		filePath: storagePath,
		Jobs:     make([]*QueuedJob, 0),
	}
	if err := q.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load offline queue: %w", err)
	}
	return q, nil
}

func (q *OfflineQueue) load() error {
	data, err := os.ReadFile(q.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &q.Jobs)
}

func (q *OfflineQueue) save() error {
	dir := filepath.Dir(q.filePath)
	if dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	data, err := json.MarshalIndent(q.Jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(q.filePath, data, 0644)
}

func (q *OfflineQueue) Enqueue(job *RsyncJob) (*QueuedJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	serverID := ""
	serverName := "Local"
	if job.Server != nil {
		serverID = job.Server.ID
		serverName = job.Server.Name
	}

	qJob := &QueuedJob{
		ID:          fmt.Sprintf("job-%d", time.Now().UnixNano()),
		CreatedAt:   time.Now(),
		Sources:     job.Sources,
		Destination: job.Destination,
		IsPull:      job.IsPull,
		ServerID:    serverID,
		ServerName:  serverName,
		Status:      "QUEUED",
	}

	q.Jobs = append(q.Jobs, qJob)
	return qJob, q.save()
}

func (q *OfflineQueue) GetPending() []*QueuedJob {
	q.mu.Lock()
	defer q.mu.Unlock()

	var pending []*QueuedJob
	for _, j := range q.Jobs {
		if j.Status == "QUEUED" || j.Status == "FAILED" || j.Status == "[QUEUED]" || j.Status == "[FAILED]" {
			pending = append(pending, j)
		}
	}
	return pending
}

func (q *OfflineQueue) UpdateStatus(jobID, status, lastErr string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, j := range q.Jobs {
		if j.ID == jobID {
			j.Status = status
			j.LastError = lastErr
			return q.save()
		}
	}
	return fmt.Errorf("job %s not found in queue", jobID)
}
