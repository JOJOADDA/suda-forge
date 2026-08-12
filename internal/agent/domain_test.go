package agent

import (
	"context"
	"testing"
	"time"
)

func TestSessionLifecycleRejectsInvalidTransitions(t *testing.T) {
	now := time.Now()
	s := Session{ID: "s1", Status: SessionCreated, UpdatedAt: now}
	if err := s.Transition(SessionRunning, now); err == nil {
		t.Fatal("created -> running must be rejected")
	}
	if err := s.Transition(SessionStarting, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(SessionRunning, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(SessionCompleting, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(SessionCompleted, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(SessionRunning, now); err == nil {
		t.Fatal("completed -> running must be rejected")
	}
}

func TestMockAdapterEmitsNormalizedEvents(t *testing.T) {
	adapter := NewMockAdapter()
	session := Session{ID: "s1", Status: SessionCreated}
	if err := adapter.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	ch, err := adapter.StreamEvents(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := <-ch
	second := <-ch
	if first.Type != EventSessionStarted || second.Type != EventMessage {
		t.Fatalf("events = %s, %s", first.Type, second.Type)
	}
	if err := adapter.SendMessage(context.Background(), session.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	if got := <-ch; got.Type != EventMessage {
		t.Fatalf("message event = %s", got.Type)
	}
	if err := adapter.Cancel(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
}
