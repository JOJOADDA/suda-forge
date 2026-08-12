package orchestration

import (
	"errors"
	"time"
)

type ID string
type TaskStatus string

const (
	TaskPending   TaskStatus = "PENDING"
	TaskReady     TaskStatus = "READY"
	TaskRunning   TaskStatus = "RUNNING"
	TaskWaiting   TaskStatus = "WAITING"
	TaskBlocked   TaskStatus = "BLOCKED"
	TaskSucceeded TaskStatus = "SUCCEEDED"
	TaskFailed    TaskStatus = "FAILED"
	TaskCancelled TaskStatus = "CANCELLED"
	TaskTimedOut  TaskStatus = "TIMED_OUT"
)

type WorkflowStatus string

const (
	WorkflowCreated    WorkflowStatus = "CREATED"
	WorkflowPlanning   WorkflowStatus = "PLANNING"
	WorkflowPlanned    WorkflowStatus = "PLANNED"
	WorkflowRunning    WorkflowStatus = "RUNNING"
	WorkflowWaiting    WorkflowStatus = "WAITING"
	WorkflowPaused     WorkflowStatus = "PAUSED"
	WorkflowCompleting WorkflowStatus = "COMPLETING"
	WorkflowSucceeded  WorkflowStatus = "SUCCEEDED"
	WorkflowFailed     WorkflowStatus = "FAILED"
	WorkflowCancelled  WorkflowStatus = "CANCELLED"
)

type ExecutionStrategy string

const (
	Sequential       ExecutionStrategy = "SEQUENTIAL"
	Parallel         ExecutionStrategy = "PARALLEL"
	DependencyDriven ExecutionStrategy = "DEPENDENCY_DRIVEN"
)

type FailurePolicy string

const (
	FailFast            FailurePolicy = "FAIL_FAST"
	ContinueIndependent FailurePolicy = "CONTINUE_INDEPENDENT"
	AllowDependent      FailurePolicy = "ALLOW_DEPENDENT"
)

type RetryPolicy struct {
	MaxAttempts     int      `json:"max_attempts"`
	RetryableErrors []string `json:"retryable_errors,omitempty"`
}
type Task struct {
	ID             ID                `json:"id"`
	WorkflowID     ID                `json:"workflow_id"`
	ProjectID      string            `json:"project_id"`
	ParentTaskID   ID                `json:"parent_task_id,omitempty"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	TaskType       string            `json:"task_type"`
	Priority       int               `json:"priority"`
	Status         TaskStatus        `json:"status"`
	Dependencies   []ID              `json:"dependencies"`
	AssignedAgent  string            `json:"assigned_agent,omitempty"`
	RoutingRequest map[string]any    `json:"routing_request,omitempty"`
	Constraints    map[string]string `json:"constraints,omitempty"`
	Retry          RetryPolicy       `json:"retry_policy"`
	Deadline       *time.Time        `json:"deadline,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
}
type TaskGraph struct {
	Tasks        map[ID]Task `json:"tasks"`
	Dependencies map[ID][]ID `json:"dependencies"`
}
type Workflow struct {
	ID            ID                `json:"id"`
	ProjectID     string            `json:"project_id"`
	Goal          string            `json:"goal"`
	Status        WorkflowStatus    `json:"status"`
	Strategy      ExecutionStrategy `json:"execution_strategy"`
	FailurePolicy FailurePolicy     `json:"failure_policy"`
	MaxParallel   int               `json:"max_parallel_tasks"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Graph         TaskGraph         `json:"graph"`
}
type TaskRun struct {
	ID          ID         `json:"id"`
	TaskID      ID         `json:"task_id"`
	Attempt     int        `json:"attempt"`
	Status      TaskStatus `json:"status"`
	AgentID     string     `json:"agent_id,omitempty"`
	ModelID     string     `json:"model_id,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}
type TaskArtifact struct {
	ID        ID             `json:"id"`
	ProjectID string         `json:"project_id"`
	TaskID    ID             `json:"task_id"`
	TaskRunID ID             `json:"task_run_id"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path,omitempty"`
	Commit    string         `json:"commit,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
type TaskResult struct {
	Status       TaskStatus     `json:"status"`
	Summary      string         `json:"summary"`
	Artifacts    []TaskArtifact `json:"artifacts,omitempty"`
	Commit       string         `json:"commit,omitempty"`
	ChangedFiles []string       `json:"changed_files,omitempty"`
	Tests        []string       `json:"tests,omitempty"`
	Warnings     []string       `json:"warnings,omitempty"`
	Errors       []string       `json:"errors,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}
type Approval struct {
	ID          ID         `json:"id"`
	WorkflowID  ID         `json:"workflow_id"`
	TaskID      ID         `json:"task_id"`
	Permission  string     `json:"permission"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}
type TaskEvent struct {
	ID         ID             `json:"id"`
	WorkflowID ID             `json:"workflow_id"`
	TaskID     ID             `json:"task_id,omitempty"`
	Type       string         `json:"type"`
	Data       map[string]any `json:"data,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}
type UserIntent struct {
	ProjectID string         `json:"project_id"`
	Goal      string         `json:"goal"`
	Context   map[string]any `json:"context,omitempty"`
}
type PlannerInput struct {
	Intent             UserIntent     `json:"intent"`
	ProjectContext     map[string]any `json:"project_context,omitempty"`
	EnvironmentContext map[string]any `json:"environment_context,omitempty"`
	AvailableAgents    []string       `json:"available_agents,omitempty"`
	RoutingPolicy      string         `json:"routing_policy,omitempty"`
}
type TaskPlan struct {
	Goal                string            `json:"goal"`
	Tasks               []Task            `json:"tasks"`
	Dependencies        map[ID][]ID       `json:"dependencies"`
	Strategy            ExecutionStrategy `json:"execution_strategy"`
	EstimatedComplexity string            `json:"estimated_complexity"`
}
type Planner interface {
	Plan(PlannerInput) (TaskPlan, error)
}

var (
	ErrCycle               = errors.New("task graph contains a cycle")
	ErrMissingDependency   = errors.New("task graph references a missing dependency")
	ErrDuplicateDependency = errors.New("task graph contains duplicate dependency")
	ErrInvalidParent       = errors.New("task has invalid parent relationship")
)
