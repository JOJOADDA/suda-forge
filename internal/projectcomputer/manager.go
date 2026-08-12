package projectcomputer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type EventSink interface{ Publish(Event) }
type Event struct {
	Type       string    `json:"type"`
	ProjectID  string    `json:"project_id"`
	ComputerID ID        `json:"computer_id"`
	Operation  Operation `json:"operation,omitempty"`
	Status     Status    `json:"status"`
	Message    string    `json:"message"`
	At         time.Time `json:"at"`
}
type Manager struct {
	Provider  runtime.Provider
	Store     Store
	Events    EventSink
	Now       func() time.Time
	mu        sync.RWMutex
	computers map[ID]ProjectComputer
}

func NewManager(now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{Now: now, computers: map[ID]ProjectComputer{}}
}
func (m *Manager) Create(ctx context.Context, projectID string, manifest environment.Manifest, available ResourceSnapshot) (ProjectComputer, error) {
	if projectID == "" || manifest.ProjectID != projectID {
		return ProjectComputer{}, errors.New("project ownership mismatch")
	}
	if err := checkResources(manifest.Resources, available); err != nil {
		return ProjectComputer{}, err
	}
	now := m.Now().UTC()
	computer := ProjectComputer{ID: ID(fmt.Sprintf("pc_%s_%d", projectID, now.UnixNano())), ProjectID: projectID, RuntimeProvider: "", Image: manifest.BaseImage, ImageVersion: manifest.Version, Status: Provisioning, Resources: manifest.Resources, CreatedAt: now, UpdatedAt: now, Readiness: []Readiness{}, Metadata: map[string]string{"manifest_id": manifest.ID}}
	if m.Provider == nil {
		computer.Status = BlockedByEnvironment
		m.save(ctx, computer)
		m.publish(computer, OpCreate, "project_computer.failed", "RuntimeProvider unavailable")
		return computer, fmt.Errorf("BLOCKED_BY_ENVIRONMENT: RuntimeProvider unavailable")
	}
	computer.RuntimeProvider = m.Provider.Name()
	rt, err := m.Provider.Create(ctx, runtime.Spec{Name: "suda-" + projectID, CPU: manifest.Resources.CPU, MemoryBytes: manifest.Resources.MemoryBytes, DiskBytes: manifest.Resources.DiskBytes, NetworkMode: "controlled"})
	if err != nil {
		computer.Status = BlockedByEnvironment
		computer.Metadata["evidence"] = err.Error()
		m.save(ctx, computer)
		m.publish(computer, OpCreate, "project_computer.failed", err.Error())
		return computer, fmt.Errorf("BLOCKED_BY_ENVIRONMENT: %w", err)
	}
	computer.RuntimeID = rt.ID
	computer.Status = Provisioning
	if err := m.save(ctx, computer); err != nil {
		return ProjectComputer{}, err
	}
	m.publish(computer, OpCreate, "project_computer.created", "Project Computer created")
	return computer, nil
}
func (m *Manager) Start(ctx context.Context, id ID) (ProjectComputer, error) {
	return m.transition(ctx, id, OpStart, func(c ProjectComputer) error {
		if c.RuntimeID == "" {
			return errors.New("runtime is not allocated")
		}
		return m.provider().Start(ctx, c.RuntimeID)
	}, Ready, "project_computer.ready")
}
func (m *Manager) Stop(ctx context.Context, id ID) (ProjectComputer, error) {
	return m.transition(ctx, id, OpStop, func(c ProjectComputer) error { return m.provider().Stop(ctx, c.RuntimeID) }, Stopped, "project_computer.stopped")
}
func (m *Manager) Restart(ctx context.Context, id ID) (ProjectComputer, error) {
	return m.transition(ctx, id, OpRestart, func(c ProjectComputer) error { return m.provider().Restart(ctx, c.RuntimeID) }, Ready, "project_computer.ready")
}
func (m *Manager) Destroy(ctx context.Context, id ID) (ProjectComputer, error) {
	return m.transition(ctx, id, OpDestroy, func(c ProjectComputer) error { return m.provider().Destroy(ctx, c.RuntimeID) }, Destroyed, "project_computer.destroyed")
}
func (m *Manager) Verify(ctx context.Context, id ID, checks []Capability) (ProjectComputer, error) {
	c, err := m.get(ctx, id)
	if err != nil {
		return ProjectComputer{}, err
	}
	if m.Provider == nil {
		return m.fail(ctx, c, OpVerify, errors.New("BLOCKED_BY_ENVIRONMENT: RuntimeProvider unavailable"))
	}
	c.Capabilities = []CapabilityCheck{}
	for _, cap := range checks {
		status, evidence := m.check(ctx, c, cap)
		c.Capabilities = append(c.Capabilities, CapabilityCheck{Capability: cap, Status: status, Evidence: evidence, CheckedAt: m.Now().UTC()})
	}
	c.Readiness = readiness(c.Capabilities)
	if hasBlocked(c.Capabilities) {
		c.Status = Degraded
	} else if hasFailure(c.Capabilities) {
		c.Status = Failed
	} else {
		c.Status = Ready
	}
	c.UpdatedAt = m.Now().UTC()
	if err := m.save(ctx, c); err != nil {
		return ProjectComputer{}, err
	}
	m.publish(c, OpVerify, "project_computer.ready", "Project Computer verification completed")
	return c, nil
}
func (m *Manager) transition(ctx context.Context, id ID, op Operation, action func(ProjectComputer) error, success Status, event string) (ProjectComputer, error) {
	c, err := m.get(ctx, id)
	if err != nil {
		return ProjectComputer{}, err
	}
	if c.Status == Destroyed {
		return c, errors.New("project computer is destroyed")
	}
	if m.Provider == nil {
		return m.fail(ctx, c, op, errors.New("BLOCKED_BY_ENVIRONMENT: RuntimeProvider unavailable"))
	}
	if err := action(c); err != nil {
		return m.fail(ctx, c, op, err)
	}
	c.Status = success
	c.UpdatedAt = m.Now().UTC()
	if err := m.save(ctx, c); err != nil {
		return ProjectComputer{}, err
	}
	m.publish(c, op, event, "Project Computer lifecycle operation completed")
	return c, nil
}
func (m *Manager) provider() runtime.Provider {
	if m.Provider == nil {
		return unavailableProvider{}
	}
	return m.Provider
}
func (m *Manager) Get(ctx context.Context, id ID) (ProjectComputer, error) { return m.get(ctx, id) }
func (m *Manager) List(ctx context.Context, projectID string) ([]ProjectComputer, error) {
	if m.Store != nil {
		return m.Store.List(ctx, projectID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []ProjectComputer{}
	for _, c := range m.computers {
		if projectID == "" || c.ProjectID == projectID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *Manager) get(ctx context.Context, id ID) (ProjectComputer, error) {
	m.mu.RLock()
	c, ok := m.computers[id]
	m.mu.RUnlock()
	if ok {
		return c, nil
	}
	if m.Store != nil {
		return m.Store.Get(ctx, id)
	}
	return ProjectComputer{}, errors.New("project computer not found")
}
func (m *Manager) save(ctx context.Context, c ProjectComputer) error {
	m.mu.Lock()
	m.computers[c.ID] = c
	m.mu.Unlock()
	if m.Store != nil {
		return m.Store.Save(ctx, c)
	}
	return nil
}
func (m *Manager) fail(ctx context.Context, c ProjectComputer, op Operation, err error) (ProjectComputer, error) {
	c.Status = Failed
	if stringsHasBlocked(err.Error()) {
		c.Status = BlockedByEnvironment
	}
	c.Metadata = ensureMeta(c.Metadata)
	c.Metadata["error"] = err.Error()
	c.UpdatedAt = m.Now().UTC()
	_ = m.save(ctx, c)
	m.publish(c, op, "project_computer.failed", err.Error())
	return c, err
}
func (m *Manager) publish(c ProjectComputer, op Operation, typ, msg string) {
	if m.Events != nil {
		m.Events.Publish(Event{Type: typ, ProjectID: c.ProjectID, ComputerID: c.ID, Operation: op, Status: c.Status, Message: msg, At: m.Now().UTC()})
	}
}
func (m *Manager) check(ctx context.Context, c ProjectComputer, cap Capability) (CapabilityStatus, string) {
	cmd := runtime.Command{Argv: []string{"sh", "-lc", "true"}, WorkingDir: "/workspace", TimeoutSeconds: 30}
	switch cap {
	case Filesystem:
		cmd.Argv = []string{"sh", "-lc", "test -d /workspace"}
	case Process:
		cmd.Argv = []string{"sh", "-lc", "ps >/dev/null 2>&1"}
	case Git:
		cmd.Argv = []string{"sh", "-lc", "command -v git"}
	case Browser:
		cmd.Argv = []string{"sh", "-lc", "command -v chromium || command -v chromium-browser"}
	case Network:
		cmd.Argv = []string{"sh", "-lc", "getent hosts example.com >/dev/null 2>&1"}
	case Ports, PTY, Containers, GPU, Snapshots:
		return Blocked, "capability requires runtime-specific verification"
	}
	out, err := m.Provider.Exec(ctx, c.RuntimeID, cmd)
	if err != nil {
		return CapabilityFailed, err.Error()
	}
	if out.ExitCode != 0 {
		return CapabilityFailed, out.Stderr
	}
	return Supported, out.Stdout
}
func checkResources(required environment.ResourceRequirement, available ResourceSnapshot) error {
	if required.CPU > available.CPU || required.MemoryBytes > available.MemoryBytes || required.DiskBytes > available.DiskBytes {
		return fmt.Errorf("INSUFFICIENT_RESOURCES: required cpu=%d memory=%d disk=%d, available cpu=%d memory=%d disk=%d", required.CPU, required.MemoryBytes, required.DiskBytes, available.CPU, available.MemoryBytes, available.DiskBytes)
	}
	if required.GPU && (!available.GPU || required.GPUMemoryBytes > available.GPUMemoryBytes) {
		return errors.New("INSUFFICIENT_RESOURCES: required GPU resources are unavailable")
	}
	return nil
}
func readiness(checks []CapabilityCheck) []Readiness {
	out := []Readiness{}
	has := func(c Capability) bool {
		for _, x := range checks {
			if x.Capability == c && x.Status == Supported {
				return true
			}
		}
		return false
	}
	if has(Filesystem) && has(Process) && has(Git) {
		out = append(out, CoreReady)
	}
	if has(Git) {
		out = append(out, AgentReady)
	}
	if has(Browser) {
		out = append(out, BrowserReady)
	}
	if has(Filesystem) {
		out = append(out, BuildReady)
	}
	if len(out) >= 4 {
		out = append(out, DeployReady)
	}
	if len(out) == 5 {
		out = append(out, FullyReady)
	}
	return out
}
func hasBlocked(v []CapabilityCheck) bool {
	for _, c := range v {
		if c.Status == Blocked {
			return true
		}
	}
	return false
}
func hasFailure(v []CapabilityCheck) bool {
	for _, c := range v {
		if c.Status == CapabilityFailed || c.Status == Unsupported {
			return true
		}
	}
	return false
}
func stringsHasBlocked(s string) bool { return len(s) >= 24 && contains(s, "BLOCKED_BY_ENVIRONMENT") }
func contains(s, v string) bool {
	for i := 0; i+len(v) <= len(s); i++ {
		if s[i:i+len(v)] == v {
			return true
		}
	}
	return false
}
func ensureMeta(v map[string]string) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	return v
}

type unavailableProvider struct{}

func (unavailableProvider) Name() string { return "unavailable" }
func (unavailableProvider) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	return runtime.Runtime{}, errors.New("runtime unavailable")
}
func (unavailableProvider) Start(context.Context, string) error {
	return errors.New("runtime unavailable")
}
func (unavailableProvider) Stop(context.Context, string) error {
	return errors.New("runtime unavailable")
}
func (unavailableProvider) Restart(context.Context, string) error {
	return errors.New("runtime unavailable")
}
func (unavailableProvider) Destroy(context.Context, string) error {
	return errors.New("runtime unavailable")
}
func (unavailableProvider) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusUnknown, errors.New("runtime unavailable")
}
func (unavailableProvider) Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error) {
	return runtime.ExecResult{}, errors.New("runtime unavailable")
}
