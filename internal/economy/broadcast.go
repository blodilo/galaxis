package economy

import (
	"sync"
)

// TickEvent is the payload pushed to SSE subscribers after each production tick.
type TickEvent struct {
	TickN int64  `json:"tick"`
	// StarID scopes the event to one system so subscribers can filter.
	StarID string `json:"star_id"`
	// Message is a human-readable summary line for the event log UI.
	Message string `json:"message,omitempty"`
}

// Broadcaster distributes tick events to any number of SSE subscribers.
// Each subscriber gets its own unbuffered channel; slow readers are dropped.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan TickEvent]struct{}
}

// NewBroadcaster creates a ready-to-use Broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan TickEvent]struct{}),
	}
}

// Subscribe returns a channel that will receive TickEvents.
// The caller must call Unsubscribe when done (e.g. on client disconnect).
func (b *Broadcaster) Subscribe() chan TickEvent {
	ch := make(chan TickEvent, 8)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (b *Broadcaster) Unsubscribe(ch chan TickEvent) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	close(ch)
}

// Publish sends ev to all current subscribers.
// Slow subscribers whose buffer is full are silently dropped (non-blocking send).
func (b *Broadcaster) Publish(ev TickEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- ev:
		default: // subscriber too slow → skip
		}
	}
}
