package provisioning

import (
	"context"
	"testing"
	"time"

	"suda-forge/internal/environment"
	"suda-forge/internal/runtime"
)

type fakeRuntime struct{ execs int }

func (f *fakeRuntime) Name() string { return "fake-runtime" }
func (f *fakeRuntime) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	return runtime.Runtime{ID: "rt-p", Provider: "test", Status: runtime.StatusRunning}, nil
}
func (f *fakeRuntime) Start(context.Context, string) error   { return nil }
func (f *fakeRuntime) Stop(context.Context, string) error    { return nil }
func (f *fakeRuntime) Destroy(context.Context, string) error { return nil }
func (f *fakeRuntime) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusRunning, nil
}
func (f *fakeRuntime) Exec(_ context.Context, _ string, _ runtime.Command) (runtime.ExecResult, error) {
	f.execs++
	return runtime.ExecResult{ExitCode: 0, Stdout: "ok"}, nil
}

type eventSink struct{ events []Event }

func (e *eventSink) Publish(v Event) { e.events = append(e.events, v) }
func TestProvisioningReachesReadyWithRuntime(t *testing.T) {
	rt := &fakeRuntime{}
	sink := &eventSink{}
	m := NewManager(func() time.Time { return time.Unix(100, 0) })
	m.Runtime = rt
	m.Events = sink
	run, err := m.Plan("p", environment.Manifest{ProjectID: "p", Resources: environment.ResourceRequirement{CPU: 1}})
	if err != nil {
		t.Fatal(err)
	}
	final, err := m.Provision(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != Ready {
		t.Fatalf("expected ready, got %s", final.Status)
	}
	if rt.execs == 0 || len(sink.events) == 0 {
		t.Fatal("expected runtime execution and events")
	}
}
func TestProvisioningBlocksWithoutRuntime(t *testing.T) {
	m := NewManager(time.Now)
	run, _ := m.Plan("p", environment.Manifest{ProjectID: "p"})
	_, err := m.Provision(context.Background(), run.ID)
	if err == nil || run.ID == "" {
		t.Fatal("expected blocked provisioning error")
	}
}
func TestGraphRejectsCycle(t *testing.T) {
	if err := ValidateGraph([]Step{{ID: "a", Dependencies: []string{"b"}}, {ID: "b", Dependencies: []string{"a"}}}); err == nil {
		t.Fatal("expected cycle rejection")
	}
}
