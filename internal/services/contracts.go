package services

import "context"

type ProcessManager interface {
	Start(context.Context, ProcessSpec) (Process, error)
	Stop(context.Context, string) error
	Restart(context.Context, string) error
	Inspect(context.Context, string) (Process, error)
	Logs(context.Context, string) (string, error)
}

type ProcessSpec struct {
	RuntimeID   string
	Name        string
	Argv        []string
	Workdir     string
	Environment []string
	Port        int
}
type Process struct {
	ID        string `json:"id"`
	RuntimeID string `json:"runtime_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type GitService interface {
	Status(context.Context, string) (GitStatus, error)
	Diff(context.Context, string) (string, error)
	Init(context.Context, string) error
	Clone(context.Context, string, string) error
	Branch(context.Context, string) ([]string, error)
	Checkout(context.Context, string, string) error
	Commit(context.Context, string, string) error
	Log(context.Context, string) (string, error)
}
type GitStatus struct {
	Branch string `json:"branch"`
	Clean  bool   `json:"clean"`
	Output string `json:"output"`
}

type PortDiscovery interface {
	Discover(context.Context, string) ([]Port, error)
}
type Port struct {
	RuntimeID string `json:"runtime_id"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Status    string `json:"status"`
	Service   string `json:"service,omitempty"`
}

type PreviewProvider interface {
	AddRoute(context.Context, Route) error
	RemoveRoute(context.Context, string) error
	URL(Route) string
}
type Route struct {
	ID          string `json:"id"`
	ProjectSlug string `json:"project_slug"`
	Port        int    `json:"port"`
	Hostname    string `json:"hostname"`
	Target      string `json:"target"`
}
