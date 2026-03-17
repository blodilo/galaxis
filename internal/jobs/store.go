// Package jobs provides an in-memory store for galaxy generation jobs.
package jobs

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Status is the lifecycle state of a generation job.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// ProgressEvent carries incremental progress for SSE streaming.
type ProgressEvent struct {
	Seq   int    `json:"seq"`
	Step  string `json:"step"`  // "morphology"|"spectral"|"objects"|"planets"
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Msg   string `json:"msg,omitempty"`
}

const progressBufSize = 64

// progState holds the progress channel and replay buffer for one job.
type progState struct {
	mu     sync.Mutex
	ch     chan ProgressEvent
	events []ProgressEvent // all emitted events for replay on reconnect
	seq    int
	closed bool
}

// Job holds the state of one generation run.
type Job struct {
	ID        string     `json:"job_id"`
	Status    Status     `json:"status"`
	GalaxyID  *uuid.UUID `json:"galaxy_id"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Store is a thread-safe in-memory job store.
type Store struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	progMu  sync.RWMutex
	progMap map[string]*progState
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		jobs:    make(map[string]*Job),
		progMap: make(map[string]*progState),
	}
}

// Create inserts a new pending job, pre-allocates a progress channel, and returns it.
func (s *Store) Create() *Job {
	s.mu.Lock()
	j := &Job{
		ID:        uuid.New().String(),
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.jobs[j.ID] = j
	s.mu.Unlock()

	s.progMu.Lock()
	s.progMap[j.ID] = &progState{ch: make(chan ProgressEvent, progressBufSize)}
	s.progMu.Unlock()

	return j
}

// Get returns a snapshot copy of the job, or false if not found.
func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

// SetRunning transitions a job to running.
func (s *Store) SetRunning(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = StatusRunning
		j.UpdatedAt = time.Now()
	}
}

// SetDone transitions a job to done, records the generated galaxy ID, and closes the progress channel.
func (s *Store) SetDone(id string, galaxyID uuid.UUID) {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.Status = StatusDone
		gid := galaxyID
		j.GalaxyID = &gid
		j.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	s.closeProgress(id)
}

// SetError transitions a job to error, records the error message, and closes the progress channel.
func (s *Store) SetError(id string, errMsg string) {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok {
		j.Status = StatusError
		j.Error = errMsg
		j.UpdatedAt = time.Now()
	}
	s.mu.Unlock()
	s.closeProgress(id)
}

// Emit records a progress event and sends it to any listening SSE client.
// Dropped silently if the channel buffer is full (no subscriber reading).
func (s *Store) Emit(jobID, step string, done, total int, msg string) {
	s.progMu.RLock()
	ps, ok := s.progMap[jobID]
	s.progMu.RUnlock()
	if !ok {
		return
	}
	ps.mu.Lock()
	ps.seq++
	ev := ProgressEvent{Seq: ps.seq, Step: step, Done: done, Total: total, Msg: msg}
	ps.events = append(ps.events, ev)
	ps.mu.Unlock()

	// Non-blocking send — drop if no subscriber is reading.
	select {
	case ps.ch <- ev:
	default:
	}
}

// Subscribe returns stored events with Seq > afterSeq (for reconnect replay)
// and the live channel for new events. Returns false if job not found.
func (s *Store) Subscribe(jobID string, afterSeq int) ([]ProgressEvent, <-chan ProgressEvent, bool) {
	s.progMu.RLock()
	ps, ok := s.progMap[jobID]
	s.progMu.RUnlock()
	if !ok {
		return nil, nil, false
	}
	ps.mu.Lock()
	replay := make([]ProgressEvent, 0, len(ps.events))
	for _, ev := range ps.events {
		if ev.Seq > afterSeq {
			replay = append(replay, ev)
		}
	}
	ps.mu.Unlock()
	return replay, ps.ch, true
}

// closeProgress closes the progress channel exactly once.
func (s *Store) closeProgress(jobID string) {
	s.progMu.RLock()
	ps, ok := s.progMap[jobID]
	s.progMu.RUnlock()
	if !ok {
		return
	}
	ps.mu.Lock()
	if !ps.closed {
		ps.closed = true
		close(ps.ch)
	}
	ps.mu.Unlock()
}
