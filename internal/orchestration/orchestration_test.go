package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func graph(tasks ...Task) TaskGraph {
	m := map[ID]Task{}
	deps := map[ID][]ID{}
	for _, task := range tasks {
		m[task.ID] = task
		deps[task.ID] = task.Dependencies
	}
	return TaskGraph{Tasks: m, Dependencies: deps}
}
func TestGraphRejectsCycleAndDuplicateEdges(t *testing.T) {
	g := graph(Task{ID: "a", Dependencies: []ID{"b"}}, Task{ID: "b", Dependencies: []ID{"a"}})
	if err := ValidateGraph(g); err != ErrCycle {
		t.Fatalf("cycle error=%v", err)
	}
	g = graph(Task{ID: "a", Dependencies: []ID{"b", "b"}}, Task{ID: "b"})
	if err := ValidateGraph(g); err != ErrDuplicateDependency {
		t.Fatalf("duplicate error=%v", err)
	}
}
func TestReadyAndFailurePropagation(t *testing.T) {
	g := graph(Task{ID: "a", Status: TaskSucceeded}, Task{ID: "b", Status: TaskPending, Dependencies: []ID{"a"}}, Task{ID: "c", Status: TaskPending, Dependencies: []ID{"b"}})
	ready := ReadyTasks(g)
	if len(ready) != 1 || ready[0] != "b" {
		t.Fatalf("ready=%v", ready)
	}
	g.Tasks["b"] = Task{ID: "b", Status: TaskFailed, Dependencies: []ID{"a"}}
	PropagateFailures(&g)
	if g.Tasks["c"].Status != TaskBlocked {
		t.Fatalf("c=%s", g.Tasks["c"].Status)
	}
}

type fakeExec struct {
	mu          sync.Mutex
	active, max int
	calls       map[ID]int
	failOnce    bool
}

func (f *fakeExec) Execute(ctx context.Context, task Task, _ TaskRun) (TaskResult, error) {
	f.mu.Lock()
	f.active++
	if f.active > f.max {
		f.max = f.active
	}
	f.calls[task.ID]++
	call := f.calls[task.ID]
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.active--; f.mu.Unlock() }()
	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
		return TaskResult{}, ctx.Err()
	}
	if f.failOnce && task.ID == "a" && call == 1 {
		return TaskResult{}, errors.New("transient")
	}
	return TaskResult{Status: TaskSucceeded, Summary: task.Title}, nil
}
func (f *fakeExec) Cancel(context.Context, TaskRun) error { return nil }
func TestSchedulerRespectsConcurrencyAndRetries(t *testing.T) {
	f := &fakeExec{calls: map[ID]int{}, failOnce: true}
	s := NewScheduler(f, 2)
	now := time.Now()
	w := Workflow{Graph: graph(Task{ID: "a", Status: TaskPending, Title: "A", Retry: RetryPolicy{MaxAttempts: 2, RetryableErrors: []string{"transient"}}, CreatedAt: now}, Task{ID: "b", Status: TaskPending, Title: "B", CreatedAt: now}, Task{ID: "c", Status: TaskPending, Title: "C", Dependencies: []ID{"a", "b"}, CreatedAt: now})}
	if err := s.Run(context.Background(), &w); err != nil {
		t.Fatal(err)
	}
	if w.Graph.Tasks["c"].Status != TaskSucceeded {
		t.Fatalf("c=%s", w.Graph.Tasks["c"].Status)
	}
	if f.max > 2 {
		t.Fatalf("max active=%d", f.max)
	}
	if f.calls["a"] != 2 {
		t.Fatalf("a calls=%d", f.calls["a"])
	}
}
