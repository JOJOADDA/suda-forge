package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"suda-forge/internal/runtime"
)

type CLIAdapter struct {
	Name             string
	Binary           string
	Process          ProcessManager
	CapabilitiesList []Capability
	mu               sync.Mutex
	processes        map[SessionID]string
}

func NewCLIAdapter(name, binary string, process ProcessManager, capabilities []Capability) *CLIAdapter {
	return &CLIAdapter{Name: name, Binary: binary, Process: process, CapabilitiesList: capabilities, processes: map[SessionID]string{}}
}
func (a *CLIAdapter) ID() string { return a.Name }
func (a *CLIAdapter) Capabilities() []Capability {
	return append([]Capability(nil), a.CapabilitiesList...)
}
func (a *CLIAdapter) Start(ctx context.Context, s Session) error {
	if a.Process == nil {
		return errors.New("runtime process manager is required")
	}
	id, err := a.Process.Start(ctx, runtime.Spec{Name: s.RuntimeID}, runtime.Command{Argv: []string{a.Binary}, WorkingDir: s.WorkingDirectory})
	if err != nil {
		return fmt.Errorf("%s start: %w", a.Name, err)
	}
	a.mu.Lock()
	a.processes[s.ID] = id
	a.mu.Unlock()
	return nil
}
func (a *CLIAdapter) Stop(ctx context.Context, id SessionID) error { return a.stop(ctx, id) }
func (a *CLIAdapter) Resume(ctx context.Context, s Session) error  { return a.Start(ctx, s) }
func (a *CLIAdapter) SendMessage(ctx context.Context, id SessionID, message string) error {
	a.mu.Lock()
	processID := a.processes[id]
	a.mu.Unlock()
	if processID == "" {
		return errors.New("agent process not found")
	}
	return a.Process.Write(ctx, processID, []byte(message+"\n"))
}

func (a *CLIAdapter) StreamEvents(ctx context.Context, id SessionID) (<-chan Event, error) {
	a.mu.Lock()
	processID := a.processes[id]
	a.mu.Unlock()
	if processID == "" {
		return nil, errors.New("agent process not found")
	}
	output, err := a.Process.Output(ctx, processID)
	if err != nil {
		return nil, err
	}
	events := make(chan Event, 16)
	go func() {
		defer close(events)
		for line := range output {
			events <- Normalize(id, a.Name, line, time.Now())
		}
	}()
	return events, nil
}
func (a *CLIAdapter) Cancel(ctx context.Context, id SessionID) error { return a.stop(ctx, id) }
func (a *CLIAdapter) Status(ctx context.Context, id SessionID) (SessionStatus, error) {
	a.mu.Lock()
	processID := a.processes[id]
	a.mu.Unlock()
	if processID == "" {
		return "", errors.New("agent process not found")
	}
	status, err := a.Process.Status(ctx, processID)
	if err != nil {
		return "", err
	}
	if status == "running" {
		return SessionRunning, nil
	}
	return SessionCompleted, nil
}
func (a *CLIAdapter) stop(ctx context.Context, id SessionID) error {
	a.mu.Lock()
	processID := a.processes[id]
	a.mu.Unlock()
	if processID == "" {
		return errors.New("agent process not found")
	}
	return a.Process.Stop(ctx, processID)
}

func NewCodexAdapter(process ProcessManager) *CLIAdapter {
	return NewCLIAdapter("codex", "codex", process, []Capability{CapabilityTerminal, CapabilityFilesystem, CapabilityGit, CapabilityStreaming, CapabilityInteractive})
}
func NewClaudeCodeAdapter(process ProcessManager) *CLIAdapter {
	return NewCLIAdapter("claude-code", "claude", process, []Capability{CapabilityTerminal, CapabilityFilesystem, CapabilityGit, CapabilityStreaming, CapabilityInteractive, CapabilityToolCalling})
}
func NewKimiAdapter(process ProcessManager) *CLIAdapter {
	return NewCLIAdapter("kimi", "kimi", process, []Capability{CapabilityTerminal, CapabilityFilesystem, CapabilityGit, CapabilityStreaming, CapabilityInteractive})
}
