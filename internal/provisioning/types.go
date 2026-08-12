package provisioning

import (
	"context"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type ID string
type Status string

const (
	Planned             Status = "PLANNED"
	Allocating          Status = "ALLOCATING"
	CreatingRuntime     Status = "CREATING_RUNTIME"
	Configuring         Status = "CONFIGURING"
	InstallingSystem    Status = "INSTALLING_SYSTEM"
	InstallingLanguage  Status = "INSTALLING_LANGUAGE"
	InstallingFramework Status = "INSTALLING_FRAMEWORK"
	InstallingTools     Status = "INSTALLING_TOOLS"
	InstallingAgents    Status = "INSTALLING_AGENTS"
	InstallingBrowser   Status = "INSTALLING_BROWSER"
	InstallingProject   Status = "INSTALLING_PROJECT"
	Verifying           Status = "VERIFYING"
	Ready               Status = "READY"
	Failed              Status = "FAILED"
	CancelRequested     Status = "CANCEL_REQUESTED"
	Cancelled           Status = "CANCELLED"
)

type StepStatus string

const (
	StepPending StepStatus = "PENDING"
	StepRunning StepStatus = "RUNNING"
	StepPassed  StepStatus = "PASSED"
	StepFailed  StepStatus = "FAILED"
	StepSkipped StepStatus = "SKIPPED"
)

type Run struct {
	ID                 ID                   `json:"id"`
	ProjectID          string               `json:"project_id"`
	RuntimeID          string               `json:"runtime_id,omitempty"`
	Manifest           environment.Manifest `json:"manifest"`
	Status             Status               `json:"status"`
	LastSuccessfulStep string               `json:"last_successful_step,omitempty"`
	Steps              []Step               `json:"steps"`
	Error              string               `json:"error,omitempty"`
	StartedAt          time.Time            `json:"started_at"`
	CompletedAt        *time.Time           `json:"completed_at,omitempty"`
	CancelRequested    bool                 `json:"cancel_requested"`
}
type Step struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Status       StepStatus `json:"status"`
	Dependencies []string   `json:"dependencies,omitempty"`
	Required     bool       `json:"required"`
	Progress     int        `json:"progress"`
	Evidence     string     `json:"evidence,omitempty"`
	Error        string     `json:"error,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}
type Event struct {
	Type      string    `json:"type"`
	ProjectID string    `json:"project_id"`
	RunID     ID        `json:"run_id"`
	StepID    string    `json:"step_id,omitempty"`
	Status    Status    `json:"status"`
	Message   string    `json:"message"`
	Progress  int       `json:"progress,omitempty"`
	At        time.Time `json:"at"`
}
type Executor interface {
	Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error)
}
type RuntimeFactory interface {
	Create(context.Context, runtime.Spec) (runtime.Runtime, error)
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Destroy(context.Context, string) error
	Status(context.Context, string) (runtime.Status, error)
	Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error)
}
type EventSink interface{ Publish(Event) }
type Cache interface {
	Get(context.Context, string) (string, bool)
	Put(context.Context, string, string)
}
type RunStore interface {
	Save(context.Context, Run) error
	Get(context.Context, ID) (Run, error)
	List(context.Context, string) ([]Run, error)
}
