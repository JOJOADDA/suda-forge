package deployment

import (
	"context"
	"errors"
	"testing"

	"suda-forge/internal/runtime"
)

type testRuntime struct{ calls int }

func (r *testRuntime) Name() string { return "test" }
func (r *testRuntime) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	return runtime.Runtime{ID: "rt", Status: runtime.StatusRunning}, nil
}
func (r *testRuntime) Start(context.Context, string) error   { return nil }
func (r *testRuntime) Stop(context.Context, string) error    { return nil }
func (r *testRuntime) Restart(context.Context, string) error { return nil }
func (r *testRuntime) Destroy(context.Context, string) error { return nil }
func (r *testRuntime) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusRunning, nil
}
func (r *testRuntime) Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error) {
	r.calls++
	return runtime.ExecResult{ExitCode: 0}, nil
}

type testHealth struct{}

func (testHealth) Check(_ context.Context, c HealthCheck) (HealthCheck, error) {
	c.Status = "PASSED"
	return c, nil
}

type testDeployer struct{}

func (testDeployer) Deploy(context.Context, Deployment, Release, EnvironmentConfig) error { return nil }
func (testDeployer) Stop(context.Context, Deployment) error                               { return nil }
func (testDeployer) Status(context.Context, Deployment) (DeploymentStatus, error) {
	return DeploymentActive, nil
}
func TestPortRegistryConflicts(t *testing.T) {
	r := NewMemoryPortRegistry()
	_, err := r.Reserve(context.Background(), PortBinding{ProjectID: "a", Protocol: "tcp", ExternalPort: 8080})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Reserve(context.Background(), PortBinding{ProjectID: "b", Protocol: "tcp", ExternalPort: 8080})
	if !errors.Is(err, ErrPortConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
func TestRouteValidationRejectsPrivateTargets(t *testing.T) {
	if err := ValidateTargetURL("http://127.0.0.1:8080"); !errors.Is(err, ErrInvalidRouteTarget) {
		t.Fatalf("expected SSRF rejection, got %v", err)
	}
	if err := ValidateHostname("bad/host"); err == nil {
		t.Fatal("expected hostname rejection")
	}
}
func TestDeploymentRequiresVerificationAndHealth(t *testing.T) {
	manager := NewManager(nil)
	manager.Runtime = &testRuntime{}
	manager.Deployer = testDeployer{}
	manager.Health = testHealth{}
	manager.Verify = VerificationAdapter{Check: func(context.Context, string) error { return nil }}
	created, _ := manager.CreateDeployment(context.Background(), Deployment{ProjectID: "p", RuntimeTarget: "rt", SourceRevision: "abc", Metadata: map[string]any{"verification_run_id": "run", "health_target": "curl -fsS http://service/health", "build_argv": []string{"true"}, "test_argv": []string{"true"}}})
	release, _ := manager.CreateRelease(context.Background(), Release{ProjectID: "p", GitRevision: "abc"})
	final, err := manager.Deploy(context.Background(), created.ID, release, EnvironmentConfig{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != DeploymentActive {
		t.Fatalf("expected active, got %s", final.Status)
	}
}
func TestDeploymentFailsWithoutVerification(t *testing.T) {
	manager := NewManager(nil)
	manager.Runtime = &testRuntime{}
	manager.Deployer = testDeployer{}
	manager.Health = testHealth{}
	manager.Verify = VerificationAdapter{Check: func(context.Context, string) error { return errors.New("not passed") }}
	created, _ := manager.CreateDeployment(context.Background(), Deployment{ProjectID: "p", RuntimeTarget: "rt", SourceRevision: "abc", Metadata: map[string]any{"verification_run_id": "run", "health_target": "curl -fsS http://service/health", "build_argv": []string{"true"}, "test_argv": []string{"true"}}})
	release, _ := manager.CreateRelease(context.Background(), Release{ProjectID: "p", GitRevision: "abc"})
	final, err := manager.Deploy(context.Background(), created.ID, release, EnvironmentConfig{ProjectID: "p"})
	if err == nil || final.Status != DeploymentFailed {
		t.Fatalf("expected failed deployment, got %s/%v", final.Status, err)
	}
}
