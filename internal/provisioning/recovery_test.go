package provisioning

import (
	"context"
	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
	"sync"
	"testing"
)

type recoveryStore struct {
	mu   sync.Mutex
	runs map[ID]Run
}

func (s *recoveryStore) Save(_ context.Context, r Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.ID] = r
	return nil
}
func (s *recoveryStore) Get(_ context.Context, id ID) (Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return Run{}, context.Canceled
	}
	return r, nil
}
func (s *recoveryStore) List(_ context.Context, _ string) ([]Run, error) { return nil, nil }

type recoveryRuntime struct{}

func (recoveryRuntime) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	return runtime.Runtime{ID: "rt-recovered", Status: runtime.StatusRunning}, nil
}
func (recoveryRuntime) Start(context.Context, string) error   { return nil }
func (recoveryRuntime) Stop(context.Context, string) error    { return nil }
func (recoveryRuntime) Destroy(context.Context, string) error { return nil }
func (recoveryRuntime) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusRunning, nil
}
func (recoveryRuntime) Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error) {
	return runtime.ExecResult{ExitCode: 0}, nil
}
func TestManagerRecoversPersistedPlan(t *testing.T) {
	store := &recoveryStore{runs: map[ID]Run{}}
	m1 := NewManager(nil)
	m1.Store = store
	m1.Runtime = recoveryRuntime{}
	manifest := environment.Manifest{ID: "m1", ProjectID: "p1", Resources: environment.ResourceRequirement{CPU: 1, MemoryBytes: 1, DiskBytes: 1}, Languages: []environment.RuntimeRequirement{{Name: "go"}}}
	planned, err := m1.Plan("p1", manifest)
	if err != nil {
		t.Fatal(err)
	}
	m2 := NewManager(nil)
	m2.Store = store
	m2.Runtime = recoveryRuntime{}
	run, err := m2.Provision(context.Background(), planned.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != Ready {
		t.Fatalf("expected recovered ready run, got %s", run.Status)
	}
}
