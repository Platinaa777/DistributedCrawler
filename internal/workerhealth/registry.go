package workerhealth

import (
	"sync"
	"time"
)

type Status string

const (
	StatusUnknown  Status = "UNKNOWN"
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
	StatusDraining Status = "DRAINING"
	StatusDead     Status = "DEAD"
)

type CommandType string

const (
	CommandDrain     CommandType = "DRAIN"
	CommandForceKill CommandType = "FORCE_KILL"
)

type HeartbeatInfo struct {
	WorkerID    string
	WorkerType  string
	Status      Status
	ActiveTasks int32
	Timestamp   time.Time
	StartedAt   time.Time
}

type WorkerRecord struct {
	WorkerID        string
	WorkerType      string
	Status          Status
	LastHeartbeatAt time.Time
	ActiveTasks     int32
	StartedAt       time.Time
	LastStatusAt    time.Time
}

type WorkerSnapshot struct {
	WorkerRecord
	Uptime time.Duration
}

type Command struct {
	Type     CommandType
	Reason   string
	IssuedAt time.Time
}

type Registry struct {
	mu                sync.RWMutex
	workers           map[string]*WorkerRecord
	streams           map[string]chan Command
	pendingCommands   map[string][]Command
	inactiveThreshold time.Duration
}

func NewRegistry(inactiveThreshold time.Duration) *Registry {
	return &Registry{
		workers:           make(map[string]*WorkerRecord),
		streams:           make(map[string]chan Command),
		pendingCommands:   make(map[string][]Command),
		inactiveThreshold: inactiveThreshold,
	}
}

func (r *Registry) UpdateHeartbeat(info HeartbeatInfo) {
	now := info.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.markInactiveLocked(now)

	record, ok := r.workers[info.WorkerID]
	if !ok {
		status := normalizeStatus(info.Status)
		r.workers[info.WorkerID] = &WorkerRecord{
			WorkerID:        info.WorkerID,
			WorkerType:      info.WorkerType,
			Status:          status,
			LastHeartbeatAt: now,
			ActiveTasks:     info.ActiveTasks,
			StartedAt:       safeTime(info.StartedAt, now),
			LastStatusAt:    now,
		}
		return
	}

	record.WorkerType = firstNonEmpty(info.WorkerType, record.WorkerType)
	record.LastHeartbeatAt = now
	record.ActiveTasks = info.ActiveTasks
	if !info.StartedAt.IsZero() {
		record.StartedAt = info.StartedAt
	}

	newStatus := normalizeStatus(info.Status)
	if record.Status == StatusDead {
		newStatus = StatusDead
	}
	if record.Status == StatusDraining && newStatus == StatusActive {
		newStatus = StatusDraining
	}
	if newStatus != record.Status {
		record.Status = newStatus
		record.LastStatusAt = now
	}
}

func (r *Registry) AttachStream(workerID string) (chan Command, []Command) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.streams[workerID]; ok {
		close(existing)
	}

	stream := make(chan Command, 8)
	r.streams[workerID] = stream

	pending := r.pendingCommands[workerID]
	delete(r.pendingCommands, workerID)

	return stream, pending
}

func (r *Registry) DetachStream(workerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ch, ok := r.streams[workerID]; ok {
		close(ch)
		delete(r.streams, workerID)
	}
}

func (r *Registry) RequestDrain(workerID, reason string) bool {
	return r.sendCommand(workerID, Command{
		Type:     CommandDrain,
		Reason:   reason,
		IssuedAt: time.Now().UTC(),
	}, StatusDraining)
}

func (r *Registry) RequestForceKill(workerID, reason string) bool {
	return r.sendCommand(workerID, Command{
		Type:     CommandForceKill,
		Reason:   reason,
		IssuedAt: time.Now().UTC(),
	}, StatusDead)
}

func (r *Registry) List(now time.Time) []WorkerSnapshot {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	r.mu.Lock()
	r.markInactiveLocked(now)
	workers := make([]WorkerSnapshot, 0, len(r.workers))
	for _, worker := range r.workers {
		workers = append(workers, WorkerSnapshot{
			WorkerRecord: *worker,
			Uptime:       uptimeFor(worker, now),
		})
	}
	r.mu.Unlock()

	return workers
}

func (r *Registry) sendCommand(workerID string, cmd Command, status Status) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := r.ensureWorkerLocked(workerID)
	if record.Status != status {
		record.Status = status
		record.LastStatusAt = cmd.IssuedAt
	}

	if ch, ok := r.streams[workerID]; ok {
		select {
		case ch <- cmd:
			return true
		default:
			r.pendingCommands[workerID] = append(r.pendingCommands[workerID], cmd)
			return false
		}
	}

	r.pendingCommands[workerID] = append(r.pendingCommands[workerID], cmd)
	return false
}

func (r *Registry) ensureWorkerLocked(workerID string) *WorkerRecord {
	if record, ok := r.workers[workerID]; ok {
		return record
	}

	now := time.Now().UTC()
	record := &WorkerRecord{
		WorkerID:        workerID,
		Status:          StatusUnknown,
		LastHeartbeatAt: time.Time{},
		ActiveTasks:     0,
		StartedAt:       now,
		LastStatusAt:    now,
	}
	r.workers[workerID] = record
	return record
}

func (r *Registry) markInactiveLocked(now time.Time) {
	if r.inactiveThreshold == 0 {
		return
	}

	for _, worker := range r.workers {
		if worker.Status == StatusDead {
			continue
		}
		if worker.LastHeartbeatAt.IsZero() {
			continue
		}
		if now.Sub(worker.LastHeartbeatAt) > r.inactiveThreshold {
			if worker.Status != StatusInactive {
				worker.Status = StatusInactive
				worker.LastStatusAt = now
			}
		}
	}
}

func normalizeStatus(status Status) Status {
	switch status {
	case StatusActive, StatusInactive, StatusDraining, StatusDead:
		return status
	default:
		return StatusUnknown
	}
}

func uptimeFor(worker *WorkerRecord, now time.Time) time.Duration {
	if worker.StartedAt.IsZero() {
		return 0
	}
	if now.Before(worker.StartedAt) {
		return 0
	}
	return now.Sub(worker.StartedAt)
}

func safeTime(t time.Time, fallback time.Time) time.Time {
	if !t.IsZero() {
		return t
	}
	return fallback
}

func firstNonEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
