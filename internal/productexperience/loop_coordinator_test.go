package productexperience

import (
	"context"
	"sync"
	"testing"
	"time"
)

type loopMemoryStore struct {
	mu     sync.Mutex
	values map[string]LoopExecution
}

func (s *loopMemoryStore) SaveLoopExecution(_ context.Context, l LoopExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]LoopExecution{}
	}
	s.values[l.ID] = l
	return nil
}
func (s *loopMemoryStore) GetLoopExecution(_ context.Context, id string) (LoopExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[id], nil
}
func (s *loopMemoryStore) ListRunnableLoopExecutions(_ context.Context) ([]LoopExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []LoopExecution{}
	for _, l := range s.values {
		if l.Status == "RUNNING" || l.Status == "BLOCKED" {
			out = append(out, l)
		}
	}
	return out, nil
}

type recordingExecutor struct {
	mu     sync.Mutex
	stages []LoopStage
	block  LoopStage
}

func (e *recordingExecutor) ExecuteLoopStage(_ context.Context, _ LoopExecution, stage LoopStage) (LoopStageResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stages = append(e.stages, stage)
	state := "COMPLETED"
	message := "completed"
	if stage == e.block {
		state = "BLOCKED_BY_ENVIRONMENT"
		message = "runtime unavailable"
	}
	now := time.Now().UTC()
	return LoopStageResult{Stage: stage, State: state, Message: message, StartedAt: now, EndedAt: now}, nil
}

func TestCoordinatorPersistsCheckpointsAndResumes(t *testing.T) {
	store := &loopMemoryStore{}
	exec := &recordingExecutor{block: VisualQA}
	now := func() time.Time { return time.Unix(100, 0) }
	c := NewCoordinator(store, exec, now)
	loop := LoopExecution{LoopPlan: LoopPlan{ID: "loop_p", ProjectID: "p", Stages: []LoopStage{Plan, Build, VisualQA, Deploy}}, Results: map[LoopStage]LoopStageResult{}}
	if err := c.Start(context.Background(), loop); err != nil {
		t.Fatal(err)
	}
	waitForLoop(t, store, "loop_p", "BLOCKED")
	got, _ := store.GetLoopExecution(context.Background(), "loop_p")
	if got.CurrentStage != VisualQA || got.Results[Build].State != "COMPLETED" {
		t.Fatalf("checkpoint not persisted: %+v", got)
	}
	exec.block = ""
	if err := c.Resume(context.Background(), "loop_p"); err != nil {
		t.Fatal(err)
	}
	waitForLoop(t, store, "loop_p", "COMPLETED")
	got, _ = store.GetLoopExecution(context.Background(), "loop_p")
	if len(got.Results) != 4 {
		t.Fatalf("expected all results after resume, got %d", len(got.Results))
	}
}

func waitForLoop(t *testing.T, store *loopMemoryStore, id, status string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		got, _ := store.GetLoopExecution(context.Background(), id)
		if got.Status == status {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	got, _ := store.GetLoopExecution(context.Background(), id)
	t.Fatalf("timed out waiting for %s, got %s", status, got.Status)
}
