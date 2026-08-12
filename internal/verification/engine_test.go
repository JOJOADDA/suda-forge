package verification

import (
	"context"
	"errors"
	"testing"
	"time"

	"suda-forge/internal/runtime"
)

type fakeRuntime struct {
	exit int
	err  error
}

func (f fakeRuntime) Name() string { return "fake" }
func (f fakeRuntime) Create(context.Context, runtime.Spec) (runtime.Runtime, error) {
	return runtime.Runtime{}, nil
}
func (f fakeRuntime) Start(context.Context, string) error   { return nil }
func (f fakeRuntime) Stop(context.Context, string) error    { return nil }
func (f fakeRuntime) Restart(context.Context, string) error { return nil }
func (f fakeRuntime) Destroy(context.Context, string) error { return nil }
func (f fakeRuntime) Status(context.Context, string) (runtime.Status, error) {
	return runtime.StatusRunning, nil
}
func (f fakeRuntime) Exec(context.Context, string, runtime.Command) (runtime.ExecResult, error) {
	return runtime.ExecResult{ExitCode: f.exit, Stdout: "ok", Stderr: "bad"}, f.err
}

type sink struct{ events []string }

func (s *sink) Publish(k, p string, d any) { s.events = append(s.events, k) }

func check(id ID, t CheckType, required bool, command string) VerificationCheck {
	return VerificationCheck{ID: id, Type: t, Name: string(id), Required: required, Timeout: time.Second, RetryPolicy: RetryPolicy{MaxAttempts: 1}, Configuration: map[string]any{"argv": []string{"tool", command}}}
}
func TestEngineBuildPassAndCommitEvidence(t *testing.T) {
	e := &Engine{Registry: NewAdapterRegistry()}
	e.Registry.Register(Build, CommandAdapter{})
	run, err := e.Run(context.Background(), CheckRequest{ProjectID: "p", Checks: []VerificationCheck{check("build", Build, true, "build")}}, ProjectContext{ProjectID: "p", RuntimeID: "r", RuntimeProvider: fakeRuntime{}, CommitSHA: "abc", Worktree: "clean"})
	if err != nil || run.Status != Passed {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	if run.Results[0].Evidence.CommitSHA != "abc" {
		t.Fatal("commit evidence missing")
	}
}
func TestEngineRequiredFailureAndOptionalFailure(t *testing.T) {
	e := &Engine{Registry: NewAdapterRegistry()}
	e.Registry.Register(Build, CommandAdapter{})
	run, err := e.Run(context.Background(), CheckRequest{ProjectID: "p", Checks: []VerificationCheck{check("required", Build, true, "fail"), check("optional", Build, false, "fail")}}, ProjectContext{ProjectID: "p", RuntimeID: "r", RuntimeProvider: fakeRuntime{exit: 1}})
	if err != nil || run.Status != Failed {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	if run.Summary.RequiredFailed != 1 || run.Summary.OptionalFailed != 1 {
		t.Fatalf("summary=%+v", run.Summary)
	}
}
func TestEngineBlocksDependentCheck(t *testing.T) {
	e := &Engine{Registry: NewAdapterRegistry()}
	e.Registry.Register(Build, CommandAdapter{})
	run, err := e.Run(context.Background(), CheckRequest{ProjectID: "p", Checks: []VerificationCheck{check("build", Build, true, "fail"), {ID: "runtime", Type: RuntimeHealth, Name: "runtime", Required: true, Dependencies: []ID{"build"}}}}, ProjectContext{ProjectID: "p", RuntimeID: "r", RuntimeProvider: fakeRuntime{exit: 1}})
	if err != nil || len(run.Results) != 2 {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	if run.Results[1].Failure == nil || run.Results[1].Failure.SuspectedCause != "dependency check did not pass" {
		t.Fatalf("blocked result=%+v", run.Results[1])
	}
}
func TestRepairLoopBounded(t *testing.T) {
	engine := &Engine{Registry: NewAdapterRegistry()}
	engine.Registry.Register(Build, CommandAdapter{})
	calls := 0
	loop := &RepairLoop{Engine: engine, MaxAttempts: 2, Analyzer: DeterministicFailureAnalyzer{}, Executor: repairFunc(func(context.Context, RepairPlan, ProjectContext, TaskContext) error { calls++; return nil })}
	run, err := loop.Run(context.Background(), CheckRequest{ProjectID: "p", Checks: []VerificationCheck{check("build", Build, true, "fail")}}, ProjectContext{ProjectID: "p", RuntimeID: "r", RuntimeProvider: fakeRuntime{exit: 1}}, TaskContext{TaskID: "t"})
	if err != nil || calls != 2 || len(run.Repairs) != 2 {
		t.Fatalf("run=%+v calls=%d err=%v", run, calls, err)
	}
}

type repairFunc func(context.Context, RepairPlan, ProjectContext, TaskContext) error

func (f repairFunc) ExecuteRepair(c context.Context, p RepairPlan, x ProjectContext, t TaskContext) error {
	return f(c, p, x, t)
}

var _ = errors.New

func TestUnsupportedBrowserIsExplicitFailure(t *testing.T) {
	e := &Engine{Registry: DefaultRegistry()}
	run, err := e.Run(context.Background(), CheckRequest{ProjectID: "p", Checks: []VerificationCheck{{ID: "browser", Type: Browser, Name: "Browser", Required: false}}}, ProjectContext{ProjectID: "p"})
	if err != nil || run.Status != Passed || run.Summary.OptionalFailed != 1 {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	if run.Results[0].Failure == nil || run.Results[0].Failure.FailureType != EnvironmentFailure {
		t.Fatalf("result=%+v", run.Results[0])
	}
}
func TestCancelledRunDoesNotPass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &Engine{Registry: NewAdapterRegistry()}
	e.Registry.Register(Build, CommandAdapter{})
	run, err := e.Run(ctx, CheckRequest{ProjectID: "p", Checks: []VerificationCheck{check("build", Build, true, "build")}}, ProjectContext{ProjectID: "p", RuntimeID: "r", RuntimeProvider: fakeRuntime{}})
	if err != nil || run.Status != Cancelled {
		t.Fatalf("run=%+v err=%v", run, err)
	}
}
