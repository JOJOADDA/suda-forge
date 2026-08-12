package runtime

import "context"

type Provider interface {
	Name() string
	Create(context.Context, Spec) (Runtime, error)
	Start(context.Context, string) error
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	Destroy(context.Context, string) error
	Status(context.Context, string) (Status, error)
	Exec(context.Context, string, Command) (ExecResult, error)
}

type Spec struct {
	Name          string
	WorkspacePath string
	CPU           int
	MemoryBytes   int64
	DiskBytes     int64
	NetworkMode   string
}

type Runtime struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Status   Status `json:"status"`
}

type Status string

const (
	StatusUnknown Status = "UNKNOWN"
	StatusStopped Status = "STOPPED"
	StatusRunning Status = "RUNNING"
	StatusFailed  Status = "FAILED"
)

type Command struct {
	Argv           []string
	WorkingDir     string
	Environment    []string
	Interactive    bool
	TimeoutSeconds int
}

type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}
