// Package inprocbus implements bus.Bus fully in-memory for tests and dev mode.
// No external process required.
package inprocbus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"galaxis/internal/bus"
)

type InProcAdapter struct {
	mu      sync.RWMutex
	subs    []*sub
	streams map[string]*stream
	// replySubject -> channel for Request/Reply
	replies map[string]chan bus.Message
}

func New() *InProcAdapter {
	return &InProcAdapter{
		streams: make(map[string]*stream),
		replies: make(map[string]chan bus.Message),
	}
}

// --- Tier 1 ---

func (a *InProcAdapter) Publish(_ context.Context, msg bus.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, s := range a.subs {
		if matchSubject(s.subject, msg.Subject) {
			s.handler(msg)
		}
	}
	return nil
}

func (a *InProcAdapter) Subscribe(_ context.Context, subject string, h bus.MsgHandler) (bus.Subscription, error) {
	s := &sub{adapter: a, subject: subject, handler: h}
	a.mu.Lock()
	a.subs = append(a.subs, s)
	a.mu.Unlock()
	return s, nil
}

func (a *InProcAdapter) QueueSubscribe(_ context.Context, subject, _ string, h bus.MsgHandler) (bus.Subscription, error) {
	// Simplified: queue groups behave like regular subscriptions in-proc
	return a.Subscribe(context.Background(), subject, h)
}

func (a *InProcAdapter) Request(ctx context.Context, msg bus.Message, timeout time.Duration) (bus.Message, error) {
	replySubj := fmt.Sprintf("_INBOX.%d", time.Now().UnixNano())
	ch := make(chan bus.Message, 1)

	a.mu.Lock()
	a.replies[replySubj] = ch
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.replies, replySubj)
		a.mu.Unlock()
	}()

	enriched := bus.Message{
		Subject: msg.Subject,
		Payload: msg.Payload,
		Headers: copyHeaders(msg.Headers),
	}
	if enriched.Headers == nil {
		enriched.Headers = make(map[string]string)
	}
	enriched.Headers["_reply"] = replySubj

	if err := a.Publish(ctx, enriched); err != nil {
		return bus.Message{}, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case reply := <-ch:
		return reply, nil
	case <-timer.C:
		return bus.Message{}, fmt.Errorf("inprocbus: request timeout on %s", msg.Subject)
	case <-ctx.Done():
		return bus.Message{}, ctx.Err()
	}
}

func (a *InProcAdapter) Reply(_ context.Context, to bus.Message, reply bus.Message) error {
	replySubj := ""
	if to.Headers != nil {
		replySubj = to.Headers["_reply"]
	}
	if replySubj == "" {
		return fmt.Errorf("inprocbus: no reply subject in message headers")
	}
	a.mu.RLock()
	ch, ok := a.replies[replySubj]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("inprocbus: reply subject %s not found", replySubj)
	}
	ch <- reply
	return nil
}

// --- Tier 2 ---

func (a *InProcAdapter) EnsureStream(_ context.Context, cfg bus.StreamConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.streams[cfg.Name]; !ok {
		a.streams[cfg.Name] = newStream(cfg)
	}
	return nil
}

func (a *InProcAdapter) PublishDurable(ctx context.Context, streamName string, msg bus.Message) error {
	a.mu.RLock()
	st, ok := a.streams[streamName]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("inprocbus: stream %s not found", streamName)
	}
	st.append(msg)
	// Also fan out to Tier-1 subscribers (mirrors NATS behaviour)
	return a.Publish(ctx, msg)
}

func (a *InProcAdapter) SubscribeDurable(_ context.Context, streamName, consumer string, startSeq uint64, h bus.AckHandler) (bus.Subscription, error) {
	a.mu.RLock()
	st, ok := a.streams[streamName]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("inprocbus: stream %s not found", streamName)
	}
	return st.subscribe(consumer, startSeq, h), nil
}

func (a *InProcAdapter) Close() error { return nil }

// --- internal sub ---

type sub struct {
	adapter *InProcAdapter
	subject string
	handler bus.MsgHandler
}

func (s *sub) Unsubscribe() error {
	s.adapter.mu.Lock()
	defer s.adapter.mu.Unlock()
	updated := s.adapter.subs[:0]
	for _, existing := range s.adapter.subs {
		if existing != s {
			updated = append(updated, existing)
		}
	}
	s.adapter.subs = updated
	return nil
}

// --- internal stream ---

type storedMsg struct {
	seq uint64
	msg bus.Message
}

type durableSub struct {
	stream   *stream
	consumer string
	handler  bus.AckHandler
	quit     chan struct{}
}

func (d *durableSub) Unsubscribe() error {
	close(d.quit)
	d.stream.removeDurable(d.consumer)
	return nil
}

type stream struct {
	mu       sync.Mutex
	cfg      bus.StreamConfig
	messages []storedMsg
	seq      uint64
	durable  map[string]*durableSub
	notify   chan struct{}
}

func newStream(cfg bus.StreamConfig) *stream {
	return &stream{
		cfg:     cfg,
		durable: make(map[string]*durableSub),
		notify:  make(chan struct{}, 1),
	}
}

func (st *stream) append(msg bus.Message) {
	st.mu.Lock()
	st.seq++
	st.messages = append(st.messages, storedMsg{seq: st.seq, msg: msg})
	st.mu.Unlock()
	select {
	case st.notify <- struct{}{}:
	default:
	}
}

func (st *stream) subscribe(consumer string, startSeq uint64, h bus.AckHandler) bus.Subscription {
	d := &durableSub{
		stream:   st,
		consumer: consumer,
		handler:  h,
		quit:     make(chan struct{}),
	}
	st.mu.Lock()
	st.durable[consumer] = d
	st.mu.Unlock()

	go func() {
		next := startSeq
		for {
			select {
			case <-d.quit:
				return
			case <-st.notify:
			}
			st.mu.Lock()
			msgs := make([]storedMsg, len(st.messages))
			copy(msgs, st.messages)
			st.mu.Unlock()
			for _, m := range msgs {
				if m.seq > next {
					next = m.seq
					acked := make(chan struct{})
					h(m.msg, func() error {
						close(acked)
						return nil
					})
					<-acked
				}
			}
		}
	}()

	return d
}

func (st *stream) removeDurable(consumer string) {
	st.mu.Lock()
	delete(st.durable, consumer)
	st.mu.Unlock()
}

// --- subject matching (NATS wildcard semantics) ---

func matchSubject(pattern, subject string) bool {
	pp := strings.Split(pattern, ".")
	sp := strings.Split(subject, ".")
	return matchTokens(pp, sp)
}

func matchTokens(pp, sp []string) bool {
	for i, p := range pp {
		if p == ">" {
			return true
		}
		if i >= len(sp) {
			return false
		}
		if p != "*" && p != sp[i] {
			return false
		}
	}
	return len(pp) == len(sp)
}

func copyHeaders(h map[string]string) map[string]string {
	if h == nil {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}
