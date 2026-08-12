package projectcomputer

import (
	"context"
	"testing"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type fakeProvider struct {
	createErr error
	execExit  int
}

func (f fakeProvider) Name() string { return "fake" }
func (f fakeProvider) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	if f.createErr != nil {
		return runtime.Runtime{}, f.createErr
	}
	return runtime.Runtime{ID: "rt-1", Provider: "fake", Status: runtime.StatusRunning}, nil
}
func (f fakeProvider) Start(context.Context, string) error   { return nil }
func (f fakeProvider) Stop(context.Context, string) error    { return nil }
func (f fakeProvider) Restart(context.Context, string) error { return nil }
func (f fakeProvider) Destroy(context.Context, string) error { return nil }
func (f fakeProvider) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusRunning, nil
}
func (f fakeProvider) Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error) {
	return runtime.ExecResult{ExitCode: f.execExit, Stdout: "verified"}, nil
}
func manifest() environment.Manifest {
	return environment.Manifest{ID: "m1", ProjectID: "p1", BaseImage: "suda/node", Version: "1", Resources: environment.ResourceRequirement{CPU: 1, MemoryBytes: 100, DiskBytes: 100}}
}
func TestCreateBlockedWithoutProvider(t *testing.T) {
	m := NewManager(time.Now)
	c, err := m.Create(context.Background(), "p1", manifest(), ResourceSnapshot{CPU: 1, MemoryBytes: 100, DiskBytes: 100})
	if err == nil || c.Status != BlockedByEnvironment {
		t.Fatalf("expected blocked create, got %s %v", c.Status, err)
	}
}
func TestCreateRejectsInsufficientResources(t *testing.T) {
	m := NewManager(time.Now)
	_, err := m.Create(context.Background(), "p1", manifest(), ResourceSnapshot{CPU: 1, MemoryBytes: 1, DiskBytes: 100})
	if err == nil || !contains(err.Error(), "INSUFFICIENT_RESOURCES") {
		t.Fatalf("expected resource rejection, got %v", err)
	}
}
func TestVerifyProducesReadiness(t *testing.T) {
	m := NewManager(time.Now)
	m.Provider = fakeProvider{}
	c, err := m.Create(context.Background(), "p1", manifest(), ResourceSnapshot{CPU: 1, MemoryBytes: 100, DiskBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	c, err = m.Verify(context.Background(), c.ID, []Capability{Filesystem, Process, Git})
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != Ready || len(c.Readiness) == 0 {
		t.Fatalf("expected ready computer, got %s %#v", c.Status, c.Readiness)
	}
}
