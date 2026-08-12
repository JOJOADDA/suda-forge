package projectcomputer

import (
	"context"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type ID string
type Status string

const (
	Provisioning         Status = "PROVISIONING"
	Ready                Status = "READY"
	Degraded             Status = "DEGRADED"
	Stopped              Status = "STOPPED"
	Failed               Status = "FAILED"
	Destroying           Status = "DESTROYING"
	Destroyed            Status = "DESTROYED"
	BlockedByEnvironment Status = "BLOCKED_BY_ENVIRONMENT"
)

type Readiness string

const (
	CoreReady    Readiness = "CORE_READY"
	AgentReady   Readiness = "AGENT_READY"
	BrowserReady Readiness = "BROWSER_READY"
	BuildReady   Readiness = "BUILD_READY"
	DeployReady  Readiness = "DEPLOY_READY"
	FullyReady   Readiness = "FULLY_READY"
)

type Capability string

const (
	Filesystem Capability = "filesystem"
	Process    Capability = "process"
	Network    Capability = "network"
	Ports      Capability = "ports"
	PTY        Capability = "pty"
	Browser    Capability = "browser"
	Git        Capability = "git"
	Containers Capability = "containers"
	GPU        Capability = "gpu"
	Snapshots  Capability = "snapshots"
)

type CapabilityStatus string

const (
	Supported        CapabilityStatus = "SUPPORTED"
	Unsupported      CapabilityStatus = "UNSUPPORTED"
	Blocked          CapabilityStatus = "BLOCKED_BY_ENVIRONMENT"
	CapabilityFailed CapabilityStatus = "FAILED"
)

type CapabilityCheck struct {
	Capability Capability       `json:"capability"`
	Status     CapabilityStatus `json:"status"`
	Evidence   string           `json:"evidence,omitempty"`
	CheckedAt  time.Time        `json:"checked_at"`
}
type ResourceSnapshot struct {
	CPU            int   `json:"cpu"`
	MemoryBytes    int64 `json:"memory_bytes"`
	DiskBytes      int64 `json:"disk_bytes"`
	GPU            bool  `json:"gpu"`
	GPUMemoryBytes int64 `json:"gpu_memory_bytes"`
	Network        bool  `json:"network"`
}
type ProjectComputer struct {
	ID                     ID                              `json:"id"`
	ProjectID              string                          `json:"project_id"`
	RuntimeProvider        string                          `json:"runtime_provider"`
	RuntimeID              string                          `json:"runtime_id,omitempty"`
	Image                  string                          `json:"image"`
	ImageVersion           string                          `json:"image_version"`
	Status                 Status                          `json:"status"`
	Resources              environment.ResourceRequirement `json:"resources"`
	EnvironmentFingerprint string                          `json:"environment_fingerprint,omitempty"`
	Readiness              []Readiness                     `json:"readiness"`
	Capabilities           []CapabilityCheck               `json:"capabilities"`
	CreatedAt              time.Time                       `json:"created_at"`
	UpdatedAt              time.Time                       `json:"updated_at"`
	Metadata               map[string]string               `json:"metadata,omitempty"`
}
type Operation string

const (
	OpCreate  Operation = "CREATE"
	OpStart   Operation = "START"
	OpStop    Operation = "STOP"
	OpRestart Operation = "RESTART"
	OpDestroy Operation = "DESTROY"
	OpRebuild Operation = "REBUILD"
	OpRepair  Operation = "REPAIR"
	OpVerify  Operation = "VERIFY"
)

type OperationRecord struct {
	ID          string     `json:"id"`
	ComputerID  ID         `json:"computer_id"`
	ProjectID   string     `json:"project_id"`
	Operation   Operation  `json:"operation"`
	Status      string     `json:"status"`
	RunID       string     `json:"run_id,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
type Store interface {
	Save(context.Context, ProjectComputer) error
	Get(context.Context, ID) (ProjectComputer, error)
	List(context.Context, string) ([]ProjectComputer, error)
	SaveOperation(context.Context, OperationRecord) error
}
type Provider interface{ runtime.Provider }
