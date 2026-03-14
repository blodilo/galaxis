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
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

// Create inserts a new pending job and returns it.
func (s *Store) Create() *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := &Job{
		ID:        uuid.New().String(),
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.jobs[j.ID] = j
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

// SetDone transitions a job to done and records the generated galaxy ID.
func (s *Store) SetDone(id string, galaxyID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = StatusDone
		gid := galaxyID
		j.GalaxyID = &gid
		j.UpdatedAt = time.Now()
	}
}

// SetError transitions a job to error and records the error message.
func (s *Store) SetError(id string, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.Status = StatusError
		j.Error = errMsg
		j.UpdatedAt = time.Now()
	}
}
