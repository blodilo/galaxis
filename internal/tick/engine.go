// Package tick implements the strategy tick engine.
// Each tick fires all registered TickHandlers in sequence.
package tick

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Handler is a function called once per strategy tick.
// ctx is cancelled when the engine is stopping.
// tickN is the monotonically increasing tick counter (starts at 1).
type Handler func(ctx context.Context, tickN int64)

// Engine drives the strategy tick loop.
type Engine struct {
	duration  time.Duration
	handlers  []Handler
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
	// tickCounter is shared between the timer loop and Advance().
	// The timer loop increments it on each scheduled tick; Advance() does the same
	// for manual ticks. This ensures tick numbers are globally monotonic.
	tickCounter atomic.Int64
}

// NewEngine creates a new Engine with the given tick duration.
func NewEngine(duration time.Duration) *Engine {
	return &Engine{
		duration: duration,
		stopCh:   make(chan struct{}),
	}
}

// Register adds a handler to be called every tick.
// Must be called before Start.
func (e *Engine) Register(h Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, h)
}

// Start launches the tick loop in a background goroutine.
func (e *Engine) Start(ctx context.Context) {
	e.wg.Add(1)
	go e.run(ctx)
}

// Stop signals the tick loop to exit and waits for it to finish.
func (e *Engine) Stop() {
	close(e.stopCh)
	e.wg.Wait()
}

// Advance fires one manual tick immediately, incrementing the shared tick counter.
// Used by POST /admin/tick/advance in the MVP to drive production without
// waiting for the real-time timer. Safe to call concurrently with the timer loop.
func (e *Engine) Advance(ctx context.Context) {
	n := e.tickCounter.Add(1)
	e.fireTick(ctx, n, time.Now())
}

func (e *Engine) run(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.duration)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			n := e.tickCounter.Add(1)
			e.fireTick(ctx, n, t)
		}
	}
}

func (e *Engine) fireTick(ctx context.Context, tickN int64, t time.Time) {
	e.mu.RLock()
	handlers := make([]Handler, len(e.handlers))
	copy(handlers, e.handlers)
	e.mu.RUnlock()

	log.Printf("tick #%d at %s (%d handlers)", tickN, t.Format(time.RFC3339), len(handlers))

	for _, h := range handlers {
		// Each handler runs synchronously in tick order.
		// Long-running work should be dispatched to goroutines internally.
		h(ctx, tickN)
	}
}
