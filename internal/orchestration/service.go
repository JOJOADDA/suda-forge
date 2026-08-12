package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"suda-forge/internal/agent"
	"suda-forge/internal/routing"
)

type DeterministicPlanner struct{}

func (DeterministicPlanner) Plan(input PlannerInput) (TaskPlan, error) {
	if input.Intent.Goal == "" {
		return TaskPlan{}, errors.New("goal is required")
	}
	now := time.Now().UTC()
	root := Task{ID: "analyze", ProjectID: input.Intent.ProjectID, Title: "Analyze request", Description: input.Intent.Goal, TaskType: "ARCHITECTURE", Status: TaskPending, CreatedAt: now}
	implement := Task{ID: "implement", ProjectID: input.Intent.ProjectID, Title: "Implement solution", Description: input.Intent.Goal, TaskType: "CODE", Status: TaskPending, Dependencies: []ID{root.ID}, CreatedAt: now}
	verify := Task{ID: "verify", ProjectID: input.Intent.ProjectID, Title: "Verify implementation", Description: "Run deterministic verification", TaskType: "TEST", Status: TaskPending, Dependencies: []ID{implement.ID}, CreatedAt: now}
	return TaskPlan{Goal: input.Intent.Goal, Tasks: []Task{root, implement, verify}, Dependencies: map[ID][]ID{root.ID: {}, implement.ID: {root.ID}, verify.ID: {implement.ID}}, Strategy: DependencyDriven, EstimatedComplexity: "MEDIUM"}, nil
}
func ValidatePlan(plan TaskPlan) (TaskGraph, error) {
	if plan.Goal == "" || len(plan.Tasks) == 0 {
		return TaskGraph{}, errors.New("plan requires goal and tasks")
	}
	graph := TaskGraph{Tasks: map[ID]Task{}, Dependencies: plan.Dependencies}
	for _, task := range plan.Tasks {
		if _, ok := graph.Tasks[task.ID]; ok {
			return TaskGraph{}, errors.New("duplicate task id")
		}
		graph.Tasks[task.ID] = task
	}
	for id, deps := range graph.Dependencies {
		graph.Tasks[id] = graph.Tasks[id]
		if _, ok := graph.Tasks[id]; !ok {
			return TaskGraph{}, ErrMissingDependency
		}
		_ = deps
	}
	if err := ValidateGraph(graph); err != nil {
		return TaskGraph{}, err
	}
	return graph, nil
}

type AgentAssignment struct {
	TaskID  ID                     `json:"task_id"`
	AgentID string                 `json:"agent_id"`
	Model   routing.ModelReference `json:"model"`
	Reason  string                 `json:"reason"`
}
type AgentSelector interface {
	Select(Task, []agent.AgentDefinition, routing.RoutingDecision) (AgentAssignment, error)
}
type CapabilityAgentSelector struct{}

func (CapabilityAgentSelector) Select(task Task, agents []agent.AgentDefinition, decision routing.RoutingDecision) (AgentAssignment, error) {
	for _, a := range agents {
		if a.Status == "DISABLED" {
			continue
		}
		return AgentAssignment{TaskID: task.ID, AgentID: string(a.ID), Model: decision.Selected, Reason: "selected compatible registered agent"}, nil
	}
	return AgentAssignment{}, errors.New("no compatible agent")
}

type Handoff struct {
	Summary      string   `json:"summary"`
	ChangedFiles []string `json:"changed_files"`
	Commit       string   `json:"commit"`
	Tests        []string `json:"tests"`
	Limitations  []string `json:"limitations"`
}
type Orchestrator struct {
	Planner   Planner
	Router    func(routing.RoutingRequest) (routing.RoutingDecision, error)
	Scheduler *Scheduler
	Now       func() time.Time
}

func (o Orchestrator) Plan(input PlannerInput) (Workflow, error) {
	if o.Planner == nil {
		return Workflow{}, errors.New("planner is required")
	}
	plan, err := o.Planner.Plan(input)
	if err != nil {
		return Workflow{}, err
	}
	graph, err := ValidatePlan(plan)
	if err != nil {
		return Workflow{}, err
	}
	now := time.Now().UTC()
	if o.Now != nil {
		now = o.Now()
	}
	workflowID := ID(fmt.Sprintf("wf_%d", now.UnixNano()))
	remapped := TaskGraph{Tasks: map[ID]Task{}, Dependencies: map[ID][]ID{}}
	idMap := map[ID]ID{}
	for id := range graph.Tasks {
		idMap[id] = ID(string(workflowID) + "_" + string(id))
	}
	for id, task := range graph.Tasks {
		newID := idMap[id]
		task.ID = newID
		task.WorkflowID = workflowID
		task.Dependencies = nil
		for _, dep := range graph.Dependencies[id] {
			task.Dependencies = append(task.Dependencies, idMap[dep])
		}
		remapped.Tasks[newID] = task
		remapped.Dependencies[newID] = append([]ID(nil), task.Dependencies...)
	}
	return Workflow{ID: workflowID, ProjectID: input.Intent.ProjectID, Goal: plan.Goal, Status: WorkflowPlanned, Strategy: plan.Strategy, FailurePolicy: ContinueIndependent, MaxParallel: 3, CreatedAt: now, UpdatedAt: now, Graph: remapped}, nil
}
func SortedTaskIDs(graph TaskGraph) []ID {
	ids := make([]ID, 0, len(graph.Tasks))
	for id := range graph.Tasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// RuntimeAgentExecutor is deliberately adapter-only: it receives a runtime-safe agent boundary and never invokes host shell.
type RuntimeAgentExecutor struct {
	Start      func(context.Context, Task, TaskRun) (TaskResult, error)
	CancelFunc func(context.Context, TaskRun) error
}

func (e RuntimeAgentExecutor) Execute(ctx context.Context, task Task, run TaskRun) (TaskResult, error) {
	if e.Start == nil {
		return TaskResult{}, errors.New("runtime agent executor is unavailable")
	}
	return e.Start(ctx, task, run)
}
func (e RuntimeAgentExecutor) Cancel(ctx context.Context, run TaskRun) error {
	if e.CancelFunc == nil {
		return nil
	}
	return e.CancelFunc(ctx, run)
}
