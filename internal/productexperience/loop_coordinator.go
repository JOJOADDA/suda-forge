package productexperience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type LoopStageResult struct {
	Stage     LoopStage      `json:"stage"`
	State     string         `json:"state"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data,omitempty"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at"`
}

type LoopExecution struct {
	LoopPlan
	CurrentStage LoopStage                     `json:"current_stage,omitempty"`
	Results      map[LoopStage]LoopStageResult `json:"results,omitempty"`
	Error        string                        `json:"error,omitempty"`
	UpdatedAt    time.Time                     `json:"updated_at"`
}

type LoopStore interface {
	SaveLoopExecution(context.Context, LoopExecution) error
	GetLoopExecution(context.Context, string) (LoopExecution, error)
	ListRunnableLoopExecutions(context.Context) ([]LoopExecution, error)
}

type StageExecutor interface {
	ExecuteLoopStage(context.Context, LoopExecution, LoopStage) (LoopStageResult, error)
}

type Coordinator struct {
	Store    LoopStore
	Executor StageExecutor
	Now      func() time.Time
	mu       sync.Mutex
	running  map[string]bool
}

func NewCoordinator(store LoopStore, executor StageExecutor, now func() time.Time) *Coordinator {
	if now == nil {
		now = time.Now
	}
	return &Coordinator{Store: store, Executor: executor, Now: now, running: map[string]bool{}}
}

func (c *Coordinator) Start(ctx context.Context, execution LoopExecution) error {
	if c.Store == nil || c.Executor == nil {
		return errors.New("loop coordinator is unavailable")
	}
	if execution.ID == "" || execution.ProjectID == "" {
		return errors.New("loop id and project_id are required")
	}
	if len(execution.Stages) == 0 {
		return errors.New("loop requires at least one stage")
	}
	if execution.CurrentStage == "" {
		execution.CurrentStage = execution.Stages[0]
	}
	if execution.Results == nil {
		execution.Results = map[LoopStage]LoopStageResult{}
	}
	execution.Status = "RUNNING"
	execution.UpdatedAt = c.Now().UTC()
	if err := c.Store.SaveLoopExecution(ctx, execution); err != nil {
		return err
	}
	go c.run(context.WithoutCancel(ctx), execution.ID)
	return nil
}

func (c *Coordinator) Resume(ctx context.Context, id string) error {
	if c.Store == nil {
		return errors.New("loop store is unavailable")
	}
	execution, err := c.Store.GetLoopExecution(ctx, id)
	if err != nil {
		return err
	}
	if execution.Status == "COMPLETED" {
		return nil
	}
	return c.Start(ctx, execution)
}

func (c *Coordinator) Recover(ctx context.Context) error {
	executions, err := c.Store.ListRunnableLoopExecutions(ctx)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		if err := c.Resume(ctx, execution.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Coordinator) run(ctx context.Context, id string) {
	c.mu.Lock()
	if c.running[id] {
		c.mu.Unlock()
		return
	}
	c.running[id] = true
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.running, id); c.mu.Unlock() }()

	execution, err := c.Store.GetLoopExecution(ctx, id)
	if err != nil {
		return
	}
	for _, stage := range execution.Stages {
		if result, ok := execution.Results[stage]; ok && result.State == "COMPLETED" {
			continue
		}
		execution.CurrentStage = stage
		execution.UpdatedAt = c.Now().UTC()
		_ = c.Store.SaveLoopExecution(ctx, execution)
		result, execErr := c.Executor.ExecuteLoopStage(ctx, execution, stage)
		if execErr != nil {
			result = LoopStageResult{Stage: stage, State: "FAILED", Message: execErr.Error(), StartedAt: c.Now().UTC(), EndedAt: c.Now().UTC()}
			execution.Error = execErr.Error()
			execution.Status = "FAILED"
			execution.Results[stage] = result
			execution.UpdatedAt = c.Now().UTC()
			_ = c.Store.SaveLoopExecution(ctx, execution)
			return
		}
		execution.Results[stage] = result
		execution.UpdatedAt = c.Now().UTC()
		if result.State == "BLOCKED_BY_ENVIRONMENT" || result.State == "BLOCKED" {
			execution.Status = "BLOCKED"
			_ = c.Store.SaveLoopExecution(ctx, execution)
			return
		}
		if result.State != "COMPLETED" {
			execution.Status = "FAILED"
			execution.Error = fmt.Sprintf("stage %s ended in %s", stage, result.State)
			_ = c.Store.SaveLoopExecution(ctx, execution)
			return
		}
		if err := c.Store.SaveLoopExecution(ctx, execution); err != nil {
			return
		}
	}
	execution.CurrentStage = ""
	execution.Status = "COMPLETED"
	execution.UpdatedAt = c.Now().UTC()
	_ = c.Store.SaveLoopExecution(ctx, execution)
}
