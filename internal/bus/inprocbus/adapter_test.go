package inprocbus_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"galaxis/internal/bus"
	"galaxis/internal/bus/inprocbus"
)

func TestPublishSubscribe(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	received := make(chan bus.Message, 1)
	sub, err := a.Subscribe(ctx, "galaxis.tick.advance", func(msg bus.Message) {
		received <- msg
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.tick.advance", Payload: []byte(`{"tick":1}`)})

	select {
	case msg := <-received:
		if string(msg.Payload) != `{"tick":1}` {
			t.Errorf("unexpected payload: %s", msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: no message received")
	}
}

func TestWildcardStar(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	var got []string
	var mu sync.Mutex
	_, _ = a.Subscribe(ctx, "galaxis.economy.*.stock", func(msg bus.Message) {
		mu.Lock()
		got = append(got, msg.Subject)
		mu.Unlock()
	})

	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.economy.star1.stock"})
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.economy.star2.stock"})
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.economy.star1.order"}) // no match

	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Errorf("expected 2 matches, got %d: %v", len(got), got)
	}
}

func TestWildcardGreater(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	count := 0
	var mu sync.Mutex
	_, _ = a.Subscribe(ctx, "galaxis.economy.>", func(_ bus.Message) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.economy.star1.stock"})
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.economy.star1.order.42"})
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.tick.advance"}) // no match

	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestRequestReply(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	_, _ = a.Subscribe(ctx, "galaxis.action.ship.move", func(msg bus.Message) {
		_ = a.Reply(ctx, msg, bus.Message{
			Subject: msg.Subject,
			Payload: []byte(`{"ok":true}`),
		})
	})

	reply, err := a.Request(ctx, bus.Message{Subject: "galaxis.action.ship.move", Payload: []byte(`{}`)}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(reply.Payload) != `{"ok":true}` {
		t.Errorf("unexpected reply: %s", reply.Payload)
	}
}

func TestRequestTimeout(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	_, err := a.Request(ctx, bus.Message{Subject: "galaxis.action.nobody"}, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestUnsubscribe(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	count := 0
	sub, _ := a.Subscribe(ctx, "galaxis.tick.advance", func(_ bus.Message) { count++ })
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.tick.advance"})
	_ = sub.Unsubscribe()
	_ = a.Publish(ctx, bus.Message{Subject: "galaxis.tick.advance"})

	time.Sleep(10 * time.Millisecond)
	if count != 1 {
		t.Errorf("expected 1 delivery before unsubscribe, got %d", count)
	}
}

func TestDurableStream(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	_ = a.EnsureStream(ctx, bus.StreamConfig{
		Name:     "ECONOMY",
		Subjects: []string{"galaxis.economy.>"},
		MaxAge:   7 * 24 * time.Hour,
	})

	received := make(chan bus.Message, 10)
	_, err := a.SubscribeDurable(ctx, "ECONOMY", "test-consumer", 0, func(msg bus.Message, ack func() error) {
		received <- msg
		_ = ack()
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = a.PublishDurable(ctx, "ECONOMY", bus.Message{
		Subject: "galaxis.economy.star1.stock",
		Payload: []byte(`{"iron":100}`),
	})

	select {
	case msg := <-received:
		if string(msg.Payload) != `{"iron":100}` {
			t.Errorf("unexpected payload: %s", msg.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: durable message not received")
	}
}

func TestEnsureStreamIdempotent(t *testing.T) {
	a := inprocbus.New()
	ctx := context.Background()

	cfg := bus.StreamConfig{Name: "TICK", Subjects: []string{"galaxis.tick.>"}}
	if err := a.EnsureStream(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := a.EnsureStream(ctx, cfg); err != nil {
		t.Fatal("EnsureStream should be idempotent")
	}
}
