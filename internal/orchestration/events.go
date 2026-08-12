package orchestration

import (
	"context"
	"sync"
	"time"
)

type EventSink interface{ Publish(TaskEvent) }
type MemoryEventSink struct {
	mu    sync.RWMutex
	items []TaskEvent
}

func (s *MemoryEventSink) Publish(event TaskEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, event)
}
func (s *MemoryEventSink) Events() []TaskEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TaskEvent, len(s.items))
	copy(out, s.items)
	return out
}

type ObservedExecutor struct {
	Inner TaskExecutor
	Sink  EventSink
	Now   func() time.Time
}

func (e ObservedExecutor) Execute(ctx context.Context, task Task, run TaskRun) (TaskResult, error) {
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now()
	}
	if e.Sink != nil {
		e.Sink.Publish(TaskEvent{ID: ID("event_started_" + string(run.ID)), WorkflowID: task.WorkflowID, TaskID: task.ID, Type: "task.started", CreatedAt: now})
	}
	result, err := e.Inner.Execute(ctx, task, run)
	if e.Sink != nil {
		typ := "task.completed"
		if err != nil {
			typ = "task.failed"
		}
		e.Sink.Publish(TaskEvent{ID: ID("event_done_" + string(run.ID)), WorkflowID: task.WorkflowID, TaskID: task.ID, Type: typ, Data: map[string]any{"error": errorString(err), "summary": result.Summary}, CreatedAt: now})
	}
	return result, err
}
func (e ObservedExecutor) Cancel(ctx context.Context, run TaskRun) error {
	if e.Inner == nil {
		return nil
	}
	return e.Inner.Cancel(ctx, run)
}
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
