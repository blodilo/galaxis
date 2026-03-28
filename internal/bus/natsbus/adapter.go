// Package natsbus implements bus.Bus backed by NATS (nats.go, Apache-2.0).
package natsbus

import (
	"context"
	"fmt"
	"time"

	"galaxis/internal/bus"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSAdapter wraps a NATS connection and JetStream context.
type NATSAdapter struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// New connects to NATS at url and returns a bus.Bus implementation.
func New(url string) (*NATSAdapter, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				// reconnect is handled automatically by nats.go
				_ = err
			}
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("natsbus: connect %s: %w", url, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("natsbus: jetstream init: %w", err)
	}

	return &NATSAdapter{nc: nc, js: js}, nil
}

// --- Tier 1 ---

func (a *NATSAdapter) Publish(_ context.Context, msg bus.Message) error {
	nm := nats.NewMsg(msg.Subject)
	nm.Data = msg.Payload
	for k, v := range msg.Headers {
		nm.Header.Set(k, v)
	}
	return a.nc.PublishMsg(nm)
}

func (a *NATSAdapter) Subscribe(_ context.Context, subject string, h bus.MsgHandler) (bus.Subscription, error) {
	sub, err := a.nc.Subscribe(subject, func(nm *nats.Msg) {
		h(toMsg(nm))
	})
	if err != nil {
		return nil, err
	}
	return &coreSub{sub: sub}, nil
}

func (a *NATSAdapter) QueueSubscribe(_ context.Context, subject, queue string, h bus.MsgHandler) (bus.Subscription, error) {
	sub, err := a.nc.QueueSubscribe(subject, queue, func(nm *nats.Msg) {
		h(toMsg(nm))
	})
	if err != nil {
		return nil, err
	}
	return &coreSub{sub: sub}, nil
}

func (a *NATSAdapter) Request(ctx context.Context, msg bus.Message, timeout time.Duration) (bus.Message, error) {
	nm := nats.NewMsg(msg.Subject)
	nm.Data = msg.Payload
	for k, v := range msg.Headers {
		nm.Header.Set(k, v)
	}
	reply, err := a.nc.RequestMsgWithContext(ctx, nm)
	if err != nil {
		return bus.Message{}, err
	}
	_ = timeout // nats.go uses ctx deadline; timeout param kept for interface compat
	return toMsg(reply), nil
}

func (a *NATSAdapter) Reply(_ context.Context, to bus.Message, reply bus.Message) error {
	replySubj := ""
	if to.Headers != nil {
		replySubj = to.Headers["_nats_reply"]
	}
	if replySubj == "" {
		return fmt.Errorf("natsbus: no reply subject on incoming message")
	}
	return a.nc.Publish(replySubj, reply.Payload)
}

// --- Tier 2 ---

func (a *NATSAdapter) EnsureStream(ctx context.Context, cfg bus.StreamConfig) error {
	jsCfg := jetstream.StreamConfig{
		Name:     cfg.Name,
		Subjects: cfg.Subjects,
		MaxAge:   cfg.MaxAge,
		MaxBytes: cfg.MaxBytes,
		Storage:  jetstream.FileStorage,
	}
	_, err := a.js.CreateOrUpdateStream(ctx, jsCfg)
	return err
}

func (a *NATSAdapter) PublishDurable(ctx context.Context, _ string, msg bus.Message) error {
	nm := nats.NewMsg(msg.Subject)
	nm.Data = msg.Payload
	for k, v := range msg.Headers {
		nm.Header.Set(k, v)
	}
	_, err := a.js.PublishMsg(ctx, nm)
	return err
}

func (a *NATSAdapter) SubscribeDurable(ctx context.Context, streamName, consumer string, startSeq uint64, h bus.AckHandler) (bus.Subscription, error) {
	st, err := a.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("natsbus: stream %s: %w", streamName, err)
	}

	ccfg := jetstream.ConsumerConfig{
		Name:      consumer,
		Durable:   consumer,
		AckPolicy: jetstream.AckExplicitPolicy,
	}
	if startSeq > 0 {
		ccfg.DeliverPolicy = jetstream.DeliverByStartSequencePolicy
		ccfg.OptStartSeq = startSeq
	}

	cons, err := st.CreateOrUpdateConsumer(ctx, ccfg)
	if err != nil {
		return nil, fmt.Errorf("natsbus: consumer %s: %w", consumer, err)
	}

	mc, err := cons.Messages()
	if err != nil {
		return nil, fmt.Errorf("natsbus: messages iter: %w", err)
	}

	go func() {
		for {
			jm, err := mc.Next()
			if err != nil {
				return // context cancelled or connection closed
			}
			msg := bus.Message{
				Subject: jm.Subject(),
				Payload: jm.Data(),
				Headers: jsHeaders(jm),
			}
			h(msg, func() error {
				return jm.Ack()
			})
		}
	}()

	return &durableSub{mc: mc}, nil
}

func (a *NATSAdapter) Close() error {
	_ = a.nc.Drain()
	return nil
}

// --- helper types ---

type coreSub struct{ sub *nats.Subscription }

func (s *coreSub) Unsubscribe() error { return s.sub.Unsubscribe() }

type durableSub struct{ mc jetstream.MessagesContext }

func (s *durableSub) Unsubscribe() error {
	s.mc.Stop()
	return nil
}

// --- helpers ---

func toMsg(nm *nats.Msg) bus.Message {
	headers := make(map[string]string, len(nm.Header))
	for k := range nm.Header {
		headers[k] = nm.Header.Get(k)
	}
	if nm.Reply != "" {
		headers["_nats_reply"] = nm.Reply
	}
	return bus.Message{
		Subject: nm.Subject,
		Payload: nm.Data,
		Headers: headers,
	}
}

func jsHeaders(jm jetstream.Msg) map[string]string {
	out := make(map[string]string)
	for k := range jm.Headers() {
		out[k] = jm.Headers().Get(k)
	}
	return out
}
