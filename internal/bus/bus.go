package bus

import (
	"context"
	"time"
)

// Message ist die einheitliche Nachricht über alle Broker-Grenzen hinweg.
type Message struct {
	Subject string
	Payload []byte
	Headers map[string]string
}

// MsgHandler wird für jeden eingehenden Message aufgerufen.
type MsgHandler func(msg Message)

// AckHandler wird für durable Messages aufgerufen; Ack muss explizit bestätigt werden.
type AckHandler func(msg Message, ack func() error)

// Subscription repräsentiert ein aktives Abonnement.
type Subscription interface {
	Unsubscribe() error
}

// StreamConfig beschreibt einen persistenten Kanal (JetStream-Stream).
type StreamConfig struct {
	Name     string
	Subjects []string
	MaxAge   time.Duration
	MaxBytes int64
}

// Bus ist die einzige öffentliche Abstraktion für alle Messaging-Operationen.
type Bus interface {
	// --- Tier 1: At-most-once (fire-and-forget) ---
	Publish(ctx context.Context, msg Message) error
	Subscribe(ctx context.Context, subject string, h MsgHandler) (Subscription, error)
	QueueSubscribe(ctx context.Context, subject, queue string, h MsgHandler) (Subscription, error)
	Request(ctx context.Context, msg Message, timeout time.Duration) (Message, error)
	Reply(ctx context.Context, to Message, reply Message) error

	// --- Tier 2: At-least-once (durable, persistent) ---
	EnsureStream(ctx context.Context, cfg StreamConfig) error
	PublishDurable(ctx context.Context, stream string, msg Message) error
	SubscribeDurable(ctx context.Context, stream, consumer string, startSeq uint64, h AckHandler) (Subscription, error)

	// --- Lifecycle ---
	Close() error
}
