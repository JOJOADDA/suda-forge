package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type TaskExecutor interface {
	Execute(context.Context, Task, TaskRun) (TaskResult, error)
	Cancel(context.Context, TaskRun) error
}
type Scheduler struct {
	Executor    TaskExecutor
	MaxParallel int
	mu          sync.Mutex
	locks       map[ID]bool
}

func NewScheduler(executor TaskExecutor, maxParallel int) *Scheduler {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &Scheduler{Executor: executor, MaxParallel: maxParallel, locks: map[ID]bool{}}
}
func (s *Scheduler) lockTask(id ID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locks[id] {
		return false
	}
	s.locks[id] = true
	return true
}
func (s *Scheduler) unlockTask(id ID) { s.mu.Lock(); defer s.mu.Unlock(); delete(s.locks, id) }
func (s *Scheduler) Run(ctx context.Context, workflow *Workflow) error {
	if err := ValidateGraph(workflow.Graph); err != nil {
		return err
	}
	if s.Executor == nil {
		return errors.New("task executor is required")
	}
	workflow.Status = WorkflowRunning
	sem := make(chan struct{}, s.MaxParallel)
	for {
		if err := ctx.Err(); err != nil {
			workflow.Status = WorkflowCancelled
			return err
		}
		PropagateFailures(&workflow.Graph)
		ready := ReadyTasks(workflow.Graph)
		if len(ready) == 0 {
			pending := false
			failed := false
			for _, task := range workflow.Graph.Tasks {
				if task.Status == TaskPending || task.Status == TaskReady || task.Status == TaskRunning {
					pending = true
				}
				if task.Status == TaskFailed || task.Status == TaskTimedOut {
					failed = true
				}
			}
			if pending {
				return nil
			}
			if failed {
				workflow.Status = WorkflowFailed
			} else {
				workflow.Status = WorkflowSucceeded
			}
			return nil
		}
		var wg sync.WaitGroup
		for _, id := range ready {
			task := workflow.Graph.Tasks[id]
			task.Status = TaskRunning
			workflow.Graph.Tasks[id] = task
			if !s.lockTask(id) {
				continue
			}
			wg.Add(1)
			go func(task Task) {
				defer wg.Done()
				defer s.unlockTask(task.ID)
				sem <- struct{}{}
				defer func() { <-sem }()
				run := TaskRun{ID: ID(fmt.Sprintf("run_%s_%d", task.ID, time.Now().UnixNano())), TaskID: task.ID, Attempt: 1, Status: TaskRunning}
				result, err := s.executeWithRetry(ctx, task, run)
				updated := workflow.Graph.Tasks[task.ID]
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						updated.Status = TaskTimedOut
					} else if errors.Is(err, context.Canceled) {
						updated.Status = TaskCancelled
					} else {
						updated.Status = TaskFailed
					}
					_ = result
				} else {
					updated.Status = result.Status
					if updated.Status == "" {
						updated.Status = TaskSucceeded
					}
				}
				workflow.Graph.Tasks[task.ID] = updated
			}(task)
		}
		wg.Wait()
		allDone := true
		for _, task := range workflow.Graph.Tasks {
			if task.Status == TaskPending || task.Status == TaskReady || task.Status == TaskRunning {
				allDone = false
				break
			}
		}
		if allDone {
			failed := false
			for _, task := range workflow.Graph.Tasks {
				if task.Status == TaskFailed || task.Status == TaskTimedOut {
					failed = true
					break
				}
			}
			if failed {
				workflow.Status = WorkflowFailed
			} else {
				workflow.Status = WorkflowSucceeded
			}
			return nil
		}
	}
}
func (s *Scheduler) executeWithRetry(parent context.Context, task Task, run TaskRun) (TaskResult, error) {
	attempts := task.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		run.Attempt = attempt
		ctx := parent
		cancel := func() {}
		if task.Deadline != nil {
			ctx, cancel = context.WithDeadline(parent, *task.Deadline)
		}
		result, err := s.Executor.Execute(ctx, task, run)
		cancel()
		if err == nil {
			return result, nil
		}
		last = err
		if !retryable(task, err) || attempt == attempts {
			break
		}
	}
	return TaskResult{}, last
}
func retryable(task Task, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := err.Error()
	for _, item := range task.Retry.RetryableErrors {
		if item != "" && containsText(message, item) {
			return true
		}
	}
	return false
}
func containsText(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
