package productexperience

import (
	"context"
	"errors"
	"time"
)

// DelegatedStageExecutor is an adapter over existing services. It deliberately
// does not implement a second orchestrator, verifier, or runtime boundary.
type DelegatedStageExecutor struct {
	Orchestrate func(context.Context, LoopExecution) (map[string]any, error)
	Verify      func(context.Context, LoopExecution, LoopStage) (map[string]any, error)
	RuntimeOK   func(LoopStage) bool
	Now         func() time.Time
}

func (e DelegatedStageExecutor) ExecuteLoopStage(ctx context.Context, execution LoopExecution, stage LoopStage) (LoopStageResult, error) {
	now := time.Now
	if e.Now != nil {
		now = e.Now
	}
	started := now().UTC()
	result := LoopStageResult{Stage: stage, StartedAt: started}
	if e.RuntimeOK != nil && !e.RuntimeOK(stage) {
		result.State = "BLOCKED_BY_ENVIRONMENT"
		result.Message = "required runtime capability is unavailable"
		result.EndedAt = now().UTC()
		return result, nil
	}
	var data map[string]any
	var err error
	switch stage {
	case Plan, Architect, Implement:
		if e.Orchestrate == nil {
			result.State = "BLOCKED_BY_ENVIRONMENT"
			result.Message = "orchestration delegate is unavailable"
		} else {
			data, err = e.Orchestrate(ctx, execution)
		}
	case Build, Test, Verify, VisualQA, Security, PostDeployVerify:
		if e.Verify == nil {
			result.State = "BLOCKED_BY_ENVIRONMENT"
			result.Message = "verification delegate is unavailable"
		} else {
			data, err = e.Verify(ctx, execution, stage)
		}
	default:
		result.State = "BLOCKED_BY_ENVIRONMENT"
		result.Message = "stage requires a configured existing product delegate"
	}
	if err != nil {
		if errors.Is(err, ErrLoopBlocked) {
			result.State = "BLOCKED_BY_ENVIRONMENT"
			result.Message = err.Error()
			result.EndedAt = now().UTC()
			return result, nil
		}
		return LoopStageResult{}, err
	}
	if result.State == "" {
		result.State = "COMPLETED"
		result.Message = "existing delegate completed stage"
		result.Data = data
	}
	result.EndedAt = now().UTC()
	return result, nil
}

var ErrLoopBlocked = errors.New("autonomous loop blocked by environment")
