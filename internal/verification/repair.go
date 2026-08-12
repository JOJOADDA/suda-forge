package verification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"suda-forge/internal/orchestration"
)

type FailureAnalyzer interface {
	Analyze(context.Context, VerificationRun, []FailureReport, ProjectContext, TaskContext) (RepairPlan, error)
}
type DeterministicFailureAnalyzer struct{}

func (DeterministicFailureAnalyzer) Analyze(_ context.Context, _ VerificationRun, failures []FailureReport, _ ProjectContext, _ TaskContext) (RepairPlan, error) {
	if len(failures) == 0 {
		return RepairPlan{}, errors.New("no failures to analyze")
	}
	files := []string{}
	targets := []ID{}
	changes := []string{}
	for _, f := range failures {
		files = append(files, f.AffectedFiles...)
		targets = append(targets, f.CheckID)
		changes = append(changes, "inspect and repair the failing verification target "+string(f.CheckID))
	}
	return RepairPlan{Problem: failures[0].SuspectedCause, RootCauseHypothesis: "failure requires inspection of the reported verification evidence", AffectedFiles: files, RecommendedChanges: changes, VerificationTargets: targets, Risk: "MEDIUM", Confidence: 0.5}, nil
}

type RepairExecutor interface {
	ExecuteRepair(context.Context, RepairPlan, ProjectContext, TaskContext) error
}
type OrchestrationRepairExecutor struct{ Executor orchestration.TaskExecutor }

func (e OrchestrationRepairExecutor) ExecuteRepair(ctx context.Context, plan RepairPlan, project ProjectContext, taskContext TaskContext) error {
	if e.Executor == nil {
		return errors.New("repair executor unavailable")
	}
	task := orchestration.Task{ID: orchestration.ID("repair_" + taskContext.TaskID), WorkflowID: orchestration.ID(taskContext.WorkflowID), ProjectID: project.ProjectID, Title: "Repair verification failure", Description: plan.Problem, TaskType: "REPAIR", Status: orchestration.TaskPending, AssignedAgent: ""}
	_, err := e.Executor.Execute(ctx, task, orchestration.TaskRun{ID: orchestration.ID("repair_run_" + task.ID), TaskID: task.ID, Status: orchestration.TaskRunning, Attempt: 1})
	return err
}

type RepairLoop struct {
	Engine      *Engine
	Analyzer    FailureAnalyzer
	Executor    RepairExecutor
	MaxAttempts int
	Events      EventSink
	Now         func() time.Time
}

func (l *RepairLoop) Run(ctx context.Context, request CheckRequest, project ProjectContext, task TaskContext) (VerificationRun, error) {
	if l.Engine == nil {
		return VerificationRun{}, errors.New("verification engine is required")
	}
	if l.Analyzer == nil {
		l.Analyzer = DeterministicFailureAnalyzer{}
	}
	max := l.MaxAttempts
	if max < 1 {
		max = 1
	}
	run, err := l.Engine.Run(ctx, request, project)
	if err != nil {
		return run, err
	}
	if run.Status == Passed || l.Executor == nil {
		return run, nil
	}
	for attempt := 1; attempt <= max && run.Status != Passed; attempt++ {
		plan, analysisErr := l.Analyzer.Analyze(ctx, run, run.Failures, project, task)
		repair := RepairAttempt{ID: ID(fmt.Sprintf("repair_%s_%d", run.ID, attempt)), RunID: run.ID, Attempt: attempt, Status: Running, Plan: plan, StartedAt: time.Now().UTC()}
		if l.Events != nil {
			l.Events.Publish("repair.started", request.ProjectID, repair)
		}
		if analysisErr == nil {
			analysisErr = l.Executor.ExecuteRepair(ctx, plan, project, task)
		}
		completed := time.Now().UTC()
		repair.CompletedAt = &completed
		if analysisErr != nil {
			repair.Status = Failed
			repair.Error = analysisErr.Error()
			if l.Events != nil {
				l.Events.Publish("repair.failed", request.ProjectID, repair)
			}
		} else {
			repair.Status = Passed
			if l.Events != nil {
				l.Events.Publish("repair.completed", request.ProjectID, repair)
			}
		}
		run.Repairs = append(run.Repairs, repair)
		if analysisErr != nil {
			break
		}
		priorRepairs := append([]RepairAttempt(nil), run.Repairs...)
		run, err = l.Engine.Run(ctx, request, project)
		if err != nil {
			return run, err
		}
		run.Repairs = append(priorRepairs, run.Repairs...)
	}
	if run.Status != Passed && len(run.Repairs) >= max && l.Events != nil {
		l.Events.Publish("repair.max_attempts", request.ProjectID, map[string]any{"run_id": run.ID, "max_attempts": max})
	}
	return run, nil
}
