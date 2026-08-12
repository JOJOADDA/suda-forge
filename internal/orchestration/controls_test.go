package orchestration

import (
	"context"
	"testing"
)

type inspector struct{ agent, session, runtime bool }

func (i inspector) AgentExists(context.Context, string) bool   { return i.agent }
func (i inspector) SessionExists(context.Context, string) bool { return i.session }
func (i inspector) RuntimeExists(context.Context, string) bool { return i.runtime }
func TestResourceManagerLimitsAndRelease(t *testing.T) {
	m := &MemoryResourceManager{Capacity: ResourceRequest{CPU: 2, MemoryMB: 1000, DiskMB: 1000}}
	lease, err := m.Reserve(context.Background(), ResourceRequest{CPU: 2, MemoryMB: 500})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Reserve(context.Background(), ResourceRequest{CPU: 1}); err == nil {
		t.Fatal("expected limit")
	}
	if err = lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Reserve(context.Background(), ResourceRequest{CPU: 2}); err != nil {
		t.Fatal(err)
	}
}
func TestApprovalAndRecovery(t *testing.T) {
	g := NewInMemoryApprovalGate()
	a, _ := g.Request(context.Background(), Approval{TaskID: "t1"})
	if a.Status != "WAITING_FOR_APPROVAL" {
		t.Fatal(a.Status)
	}
	a, _ = g.Resolve(context.Background(), a.ID, true)
	if a.Status != "APPROVED" {
		t.Fatal(a.Status)
	}
	task := Task{ProjectID: "p1", Retry: RetryPolicy{MaxAttempts: 2}}
	run := TaskRun{ID: "r1", Attempt: 1, AgentID: "a"}
	if got := RecoverStaleTask(context.Background(), task, run, inspector{true, true, true}); got != Reconcile {
		t.Fatal(got)
	}
	if got := RecoverStaleTask(context.Background(), task, run, inspector{}); got != Retry {
		t.Fatal(got)
	}
}
