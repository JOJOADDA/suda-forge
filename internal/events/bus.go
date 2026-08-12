package events

import (
	"context"
	"sync"
)

type Event struct {
	Type      string `json:"type"`
	ProjectID string `json:"project_id,omitempty"`
	RuntimeID string `json:"runtime_id,omitempty"`
	Data      any    `json:"data,omitempty"`
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

func NewBus() *Bus { return &Bus{subscribers: make(map[chan Event]struct{})} }
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
func (b *Bus) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, 32)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	go func() { <-ctx.Done(); b.mu.Lock(); delete(b.subscribers, ch); close(ch); b.mu.Unlock() }()
	return ch
}
