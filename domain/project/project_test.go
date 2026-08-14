package project

import (
	"testing"
	"time"
)

func TestProjectLifecycle(t *testing.T) {
	now := time.Unix(100, 0)
	p, err := New("Hello World", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "hello-world" {
		t.Fatalf("slug = %q", p.Slug)
	}
	for _, next := range []Status{StatusPreparing, StatusReady, StatusRunning, StatusStopping, StatusStopped, StatusRunning} {
		if err := p.Transition(next, now); err != nil {
			t.Fatalf("transition to %s: %v", next, err)
		}
	}
}

func TestProjectRejectsIllegalTransition(t *testing.T) {
	p, err := New("demo", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Transition(StatusRunning, time.Now()); err == nil {
		t.Fatal("expected illegal transition error")
	}
}
