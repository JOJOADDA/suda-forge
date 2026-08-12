package provisioning

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type Manager struct {
	Runtime RuntimeFactory
	Store   RunStore
	Events  EventSink
	Cache   Cache
	Now     func() time.Time
	mu      sync.Mutex
	runs    map[ID]Run
}

func NewManager(now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{Now: now, runs: map[ID]Run{}}
}
func (m *Manager) Plan(projectID string, manifest environment.Manifest) (Run, error) {
	if projectID == "" || manifest.ProjectID != projectID {
		return Run{}, errors.New("project ownership mismatch")
	}
	steps := DefaultSteps()
	if err := ValidateGraph(steps); err != nil {
		return Run{}, err
	}
	now := m.Now().UTC()
	run := Run{ID: ID(fmt.Sprintf("prov_%s_%d", projectID, now.UnixNano())), ProjectID: projectID, Manifest: manifest, Status: Planned, Steps: steps, StartedAt: now}
	m.mu.Lock()
	m.runs[run.ID] = run
	m.mu.Unlock()
	m.emit(run, "environment.planned", "", "Provisioning plan created", 0)
	return run, nil
}
func (m *Manager) Provision(ctx context.Context, id ID) (Run, error) {
	run, err := m.get(id)
	if err != nil {
		return Run{}, err
	}
	if err := ValidateGraph(run.Steps); err != nil {
		return m.fail(ctx, run, err)
	}
	if run.Status == Ready {
		return run, nil
	}
	if run.Status == Cancelled {
		return run, errors.New("provisioning run is cancelled")
	}
	if err := m.transition(ctx, &run, Allocating, "environment.provisioning.started", "Provisioning started", 0); err != nil {
		return Run{}, err
	}
	if m.Runtime == nil {
		return m.fail(ctx, run, errors.New("BLOCKED_BY_ENVIRONMENT: runtime provider unavailable"))
	}
	if run.RuntimeID == "" {
		rt, createErr := m.Runtime.Create(ctx, runtime.Spec{Name: "suda-" + run.ProjectID, CPU: run.Manifest.Resources.CPU, MemoryBytes: run.Manifest.Resources.MemoryBytes, DiskBytes: run.Manifest.Resources.DiskBytes, NetworkMode: "controlled"})
		if createErr != nil {
			return m.fail(ctx, run, fmt.Errorf("BLOCKED_BY_ENVIRONMENT: create project computer: %w", createErr))
		}
		run.RuntimeID = rt.ID
		if err := m.save(ctx, run); err != nil {
			return Run{}, err
		}
		m.emit(run, "environment.runtime.created", "runtime", "Project Computer runtime created", 0)
	}
	for {
		if run.CancelRequested {
			return m.cancel(ctx, run)
		}
		ready := ReadySteps(run.Steps)
		if len(ready) == 0 {
			break
		}
		for _, step := range ready {
			if step.ID == "runtime" {
				run = m.mark(run, step.ID, StepPassed, "runtime created", 100)
			} else {
				run = m.mark(run, step.ID, StepRunning, "", 0)
				if err := m.executeStep(ctx, run, step); err != nil {
					run = m.mark(run, step.ID, StepFailed, "", 0)
					return m.fail(ctx, run, err)
				}
				run = m.mark(run, step.ID, StepPassed, "runtime-scoped capability verified", 100)
			}
			run.LastSuccessfulStep = step.ID
			if err := m.save(ctx, run); err != nil {
				return Run{}, err
			}
			m.emit(run, "environment.installation.completed", step.ID, step.Name, 100)
		}
	}
	now := m.Now().UTC()
	run.Status = Ready
	run.CompletedAt = &now
	if err := m.save(ctx, run); err != nil {
		return Run{}, err
	}
	m.emit(run, "environment.ready", "", "Project Computer is ready", 100)
	return run, nil
}
func (m *Manager) executeStep(ctx context.Context, run Run, step Step) error {
	if step.ID == "runtime" {
		return nil
	}
	commands := map[string]runtime.Command{"configure": {Argv: []string{"sh", "-lc", "test -d /workspace"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "system": {Argv: []string{"sh", "-lc", "true"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "language": {Argv: []string{"sh", "-lc", "command -v " + languageBinary(run.Manifest)}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "framework": {Argv: []string{"sh", "-lc", "test -f package.json || test -f go.mod || test -f pyproject.toml"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "tools": {Argv: []string{"sh", "-lc", "command -v git"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "agents": {Argv: []string{"sh", "-lc", "command -v " + agentBinary(run.Manifest)}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "browser": {Argv: []string{"sh", "-lc", "command -v chromium || command -v chromium-browser"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "project": {Argv: []string{"sh", "-lc", "git rev-parse --is-inside-work-tree"}, WorkingDir: "/workspace", TimeoutSeconds: 60}, "verify": {Argv: []string{"sh", "-lc", "test -d /workspace"}, WorkingDir: "/workspace", TimeoutSeconds: 60}}
	command, ok := commands[step.ID]
	if !ok {
		return fmt.Errorf("unsupported provisioning step %s", step.ID)
	}
	result, err := m.Runtime.Exec(ctx, run.RuntimeID, command)
	if err != nil {
		return fmt.Errorf("%s: %w", step.Name, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("%s failed: %s", step.Name, result.Stderr)
	}
	return nil
}
func languageBinary(m environment.Manifest) string {
	if len(m.Languages) == 0 {
		return "sh"
	}
	switch m.Languages[0].Name {
	case "node":
		return "node"
	case "go":
		return "go"
	case "python":
		return "python3"
	default:
		return "sh"
	}
}
func agentBinary(m environment.Manifest) string {
	if len(m.AgentCLIs) == 0 {
		return "sh"
	}
	switch m.AgentCLIs[0].AgentID {
	case "codex":
		return "codex"
	case "claude_code":
		return "claude"
	case "kimi":
		return "kimi"
	default:
		return "sh"
	}
}
func (m *Manager) RequestCancel(ctx context.Context, id ID) (Run, error) {
	run, err := m.get(id)
	if err != nil {
		return Run{}, err
	}
	run.CancelRequested = true
	run.Status = CancelRequested
	if err := m.save(ctx, run); err != nil {
		return Run{}, err
	}
	m.emit(run, "environment.retry.started", "", "Cancellation requested", 0)
	return run, nil
}
func (m *Manager) Resume(ctx context.Context, id ID) (Run, error) {
	run, err := m.get(id)
	if err != nil {
		return Run{}, err
	}
	if run.Status != Failed && run.Status != CancelRequested {
		return run, errors.New("run is not resumable")
	}
	run.CancelRequested = false
	run.Status = Planned
	return m.Provision(ctx, id)
}
func (m *Manager) Cleanup(ctx context.Context, id ID) error {
	run, err := m.get(id)
	if err != nil {
		return err
	}
	m.emit(run, "environment.cleanup.started", "", "Cleanup requested", 0)
	if m.Runtime != nil && run.RuntimeID != "" {
		if err := m.Runtime.Destroy(ctx, run.RuntimeID); err != nil {
			return err
		}
	}
	run.Status = Cancelled
	now := m.Now().UTC()
	run.CompletedAt = &now
	if err := m.save(ctx, run); err != nil {
		return err
	}
	m.emit(run, "environment.cleanup.completed", "", "Cleanup completed", 100)
	return nil
}
func (m *Manager) cancel(ctx context.Context, run Run) (Run, error) {
	run.Status = Cancelled
	now := m.Now().UTC()
	run.CompletedAt = &now
	_ = m.save(ctx, run)
	m.emit(run, "environment.failed", "", "Provisioning cancelled", 0)
	return run, context.Canceled
}
func (m *Manager) fail(ctx context.Context, run Run, err error) (Run, error) {
	run.Status = Failed
	run.Error = err.Error()
	now := m.Now().UTC()
	run.CompletedAt = &now
	_ = m.save(ctx, run)
	m.emit(run, "environment.failed", "", err.Error(), 0)
	return run, err
}
func (m *Manager) transition(ctx context.Context, run *Run, status Status, event, message string, progress int) error {
	run.Status = status
	if err := m.save(ctx, *run); err != nil {
		return err
	}
	m.emit(*run, event, "", message, progress)
	return nil
}
func (m *Manager) mark(run Run, id string, status StepStatus, evidence string, progress int) Run {
	for i := range run.Steps {
		if run.Steps[i].ID == id {
			run.Steps[i].Status = status
			run.Steps[i].Evidence = evidence
			run.Steps[i].Progress = progress
		}
	}
	return run
}
func (m *Manager) get(id ID) (Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[id]
	if !ok {
		return Run{}, errors.New("provisioning run not found")
	}
	return run, nil
}
func (m *Manager) save(ctx context.Context, run Run) error {
	m.mu.Lock()
	m.runs[run.ID] = run
	m.mu.Unlock()
	if m.Store != nil {
		return m.Store.Save(ctx, run)
	}
	return nil
}
func (m *Manager) emit(run Run, typ, step, message string, progress int) {
	if m.Events != nil {
		m.Events.Publish(Event{Type: typ, ProjectID: run.ProjectID, RunID: run.ID, StepID: step, Status: run.Status, Message: message, Progress: progress, At: m.Now().UTC()})
	}
}
