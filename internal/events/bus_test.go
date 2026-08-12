package events

import (
	"context"
	"testing"
	"time"
)

func TestBusPublishesAndClosesOnContext(t *testing.T) {
	bus := NewBus()
	ctx, cancel := context.WithCancel(context.Background())
	ch := bus.Subscribe(ctx)
	bus.Publish(Event{Type: "project.state", ProjectID: "p1"})
	select {
	case got := <-ch:
		if got.Type != "project.state" || got.ProjectID != "p1" {
			t.Fatalf("unexpected event: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("subscription did not close")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close")
	}
}
