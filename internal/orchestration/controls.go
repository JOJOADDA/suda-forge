package orchestration

import (
	"context"
	"errors"
	"sync"
	"time"
)

type ResourceRequest struct {
	CPU        int    `json:"cpu"`
	MemoryMB   int    `json:"memory_mb"`
	DiskMB     int    `json:"disk_mb"`
	ProviderID string `json:"provider_id,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
}
type ResourceLease interface{ Release(context.Context) error }
type ResourceManager interface {
	Reserve(context.Context, ResourceRequest) (ResourceLease, error)
}
type MemoryResourceManager struct {
	mu       sync.Mutex
	Capacity ResourceRequest
	Used     ResourceRequest
}

func (m *MemoryResourceManager) Reserve(_ context.Context, req ResourceRequest) (ResourceLease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Used.CPU+req.CPU > m.Capacity.CPU || m.Used.MemoryMB+req.MemoryMB > m.Capacity.MemoryMB || m.Used.DiskMB+req.DiskMB > m.Capacity.DiskMB {
		return nil, errors.New("resources unavailable")
	}
	m.Used.CPU += req.CPU
	m.Used.MemoryMB += req.MemoryMB
	m.Used.DiskMB += req.DiskMB
	return resourceLease{release: func(context.Context) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.Used.CPU -= req.CPU
		m.Used.MemoryMB -= req.MemoryMB
		m.Used.DiskMB -= req.DiskMB
		return nil
	}}, nil
}

type resourceLease struct{ release func(context.Context) error }

func (l resourceLease) Release(ctx context.Context) error { return l.release(ctx) }

type ApprovalGate interface {
	Request(context.Context, Approval) (Approval, error)
	Resolve(context.Context, ID, bool) (Approval, error)
}
type InMemoryApprovalGate struct {
	mu    sync.Mutex
	items map[ID]Approval
}

func NewInMemoryApprovalGate() *InMemoryApprovalGate {
	return &InMemoryApprovalGate{items: map[ID]Approval{}}
}
func (g *InMemoryApprovalGate) Request(_ context.Context, a Approval) (Approval, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a.ID == "" {
		a.ID = ID("approval_" + time.Now().UTC().Format("20060102150405.000000000"))
	}
	a.Status = "WAITING_FOR_APPROVAL"
	g.items[a.ID] = a
	return a, nil
}
func (g *InMemoryApprovalGate) Resolve(_ context.Context, id ID, approved bool) (Approval, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, ok := g.items[id]
	if !ok {
		return Approval{}, errors.New("approval not found")
	}
	if approved {
		a.Status = "APPROVED"
	} else {
		a.Status = "REJECTED"
	}
	now := time.Now().UTC()
	a.ResolvedAt = &now
	g.items[id] = a
	return a, nil
}

type RecoveryAction string

const (
	Resume    RecoveryAction = "RESUME"
	Reconcile RecoveryAction = "RECONCILE"
	Retry     RecoveryAction = "RETRY"
	Fail      RecoveryAction = "FAIL"
)

type RecoveryInspector interface {
	AgentExists(context.Context, string) bool
	SessionExists(context.Context, string) bool
	RuntimeExists(context.Context, string) bool
}

func RecoverStaleTask(ctx context.Context, task Task, run TaskRun, inspector RecoveryInspector) RecoveryAction {
	if inspector == nil {
		return Fail
	}
	if inspector.AgentExists(ctx, run.AgentID) && inspector.SessionExists(ctx, string(run.ID)) && inspector.RuntimeExists(ctx, task.ProjectID) {
		return Reconcile
	}
	if task.Retry.MaxAttempts > run.Attempt {
		return Retry
	}
	return Fail
}

type WorktreeManager interface {
	Create(context.Context, string, string) (string, error)
	Integrate(context.Context, string, []string) (string, error)
	Cleanup(context.Context, string) error
}
type IntegrationResult struct {
	Branch    string   `json:"branch"`
	Commit    string   `json:"commit,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
	Status    string   `json:"status"`
}
