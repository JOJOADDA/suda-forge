package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"suda-forge/internal/orchestration"
)

type simulatedExecutor struct{}

func (simulatedExecutor) Execute(ctx context.Context, task orchestration.Task, _ orchestration.TaskRun) (orchestration.TaskResult, error) {
	select {
	case <-time.After(8 * time.Millisecond):
		return orchestration.TaskResult{Status: orchestration.TaskSucceeded, Summary: "simulated: " + task.Title, Tests: []string{"deterministic simulator"}}, nil
	case <-ctx.Done():
		return orchestration.TaskResult{}, ctx.Err()
	}
}
func (simulatedExecutor) Cancel(context.Context, orchestration.TaskRun) error { return nil }
func main() {
	planner := orchestration.DeterministicPlanner{}
	workflow, err := (orchestration.Orchestrator{Planner: planner, Now: time.Now}).Plan(orchestration.PlannerInput{Intent: orchestration.UserIntent{ProjectID: "simulator", Goal: "exercise orchestration"}})
	if err != nil {
		panic(err)
	}
	scheduler := orchestration.NewScheduler(simulatedExecutor{}, 2)
	err = scheduler.Run(context.Background(), &workflow)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(workflow)
}
