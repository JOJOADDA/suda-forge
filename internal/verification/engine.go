package verification

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"suda-forge/internal/runtime"
)

type EventSink interface{ Publish(string, string, any) }
type Engine struct {
	Registry *AdapterRegistry
	Events   EventSink
	Now      func() time.Time
}

func (e *Engine) Run(ctx context.Context, request CheckRequest, project ProjectContext) (VerificationRun, error) {
	if request.ProjectID == "" {
		return VerificationRun{}, errors.New("project_id is required")
	}
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now()
	}
	profile := Profile(request.Profile, request.Policy, request.Checks)
	if err := ValidateChecks(profile.Checks); err != nil {
		return VerificationRun{}, err
	}
	run := VerificationRun{ID: ID(fmt.Sprintf("vr_%d", now.UnixNano())), ProjectID: request.ProjectID, WorkflowID: request.WorkflowID, TaskID: request.TaskID, TaskRunID: request.TaskRunID, Status: Running, StartedAt: now, Profile: profile, Checks: profile.Checks, State: ProjectState{CommitSHA: project.CommitSHA, Worktree: project.Worktree, ChangedFiles: project.ChangedFiles}}
	if project.ProjectID == "" {
		project.ProjectID = request.ProjectID
	}
	project = captureProjectState(ctx, project)
	run.State = ProjectState{CommitSHA: project.CommitSHA, Worktree: project.Worktree, ChangedFiles: project.ChangedFiles}
	if e.Events != nil {
		e.Events.Publish("verification.started", request.ProjectID, run)
	}
	if e.Registry == nil {
		e.Registry = DefaultRegistry()
	}
	results := map[ID]VerificationResult{}
	checks := append([]VerificationCheck(nil), profile.Checks...)
	sort.Slice(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })
	for _, check := range checks {
		if err := ctx.Err(); err != nil {
			run.Status = Cancelled
			if errors.Is(err, context.DeadlineExceeded) {
				run.Status = TimedOut
			}
			break
		}
		blocked := false
		for _, dep := range check.Dependencies {
			if prior, ok := results[dep]; ok && prior.Status != Passed {
				blocked = true
			}
		}
		if blocked {
			result := VerificationResult{CheckID: check.ID, Status: Failed, Failure: &FailureReport{ID: ID("failure_" + string(check.ID)), FailureType: EnvironmentFailure, CheckID: check.ID, SuspectedCause: "dependency check did not pass", Severity: severity(check.Required), Retryable: false}, StartedAt: now, CompletedAt: time.Now().UTC()}
			results[check.ID] = result
			run.Results = append(run.Results, result)
			if e.Events != nil {
				e.Events.Publish("verification.check.failed", request.ProjectID, result)
			}
			continue
		}
		adapter, ok := e.Registry.Get(check.Type)
		if !ok {
			adapter = UnsupportedAdapter{}
		}
		if e.Events != nil {
			e.Events.Publish("verification.check.started", request.ProjectID, check)
		}
		if err := adapter.Discover(ctx, project); err != nil {
			results[check.ID] = adapter.NormalizeResult(check, Evidence{}, err)
			run.Results = append(run.Results, results[check.ID])
			continue
		}
		if err := adapter.Validate(ctx, check, project); err != nil {
			results[check.ID] = adapter.NormalizeResult(check, Evidence{}, err)
			run.Results = append(run.Results, results[check.ID])
			continue
		}
		var result VerificationResult
		attempts := check.RetryPolicy.MaxAttempts
		if attempts < 1 {
			attempts = 1
		}
		for attempt := 1; attempt <= attempts; attempt++ {
			evidence, err := adapter.Execute(ctx, check, project)
			if collected, collectErr := adapter.CollectEvidence(ctx, check, project, evidence); collectErr == nil {
				evidence = collected
			}
			result = adapter.NormalizeResult(check, evidence, err)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				result.Status = Cancelled
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					result.Status = TimedOut
				}
				break
			}
		}
		results[check.ID] = result
		run.Results = append(run.Results, result)
		if result.Evidence.ArtifactPath != "" {
			run.Artifacts = append(run.Artifacts, VerificationArtifact{ID: ID("artifact_" + string(run.ID) + "_" + string(check.ID)), ProjectID: request.ProjectID, TaskID: request.TaskID, RunID: run.ID, CheckID: check.ID, Kind: "evidence", Path: result.Evidence.ArtifactPath})
		}
		if e.Events != nil {
			if result.Status == Passed {
				e.Events.Publish("verification.check.passed", request.ProjectID, result)
			} else {
				e.Events.Publish("verification.check.failed", request.ProjectID, result)
			}
		}
		if ctx.Err() != nil {
			run.Status = Cancelled
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				run.Status = TimedOut
			}
			break
		}
	}
	if run.Status == Running {
		run.Status = Passed
	}
	for _, result := range run.Results {
		check := findCheck(run.Checks, result.CheckID)
		if result.Failure != nil && result.Failure.SuspectedCause == "dependency check did not pass" {
			run.Summary.Blocked++
		}
		if result.Status == Passed {
			if check.Required {
				run.Summary.RequiredPassed++
			} else {
				run.Summary.OptionalPassed++
			}
		} else if result.Status == Cancelled || result.Status == TimedOut {
			if check.Required {
				run.Summary.RequiredFailed++
			} else {
				run.Summary.OptionalFailed++
			}
		} else if check.Required {
			run.Summary.RequiredFailed++
		} else {
			run.Summary.OptionalFailed++
		}
		if result.Failure != nil {
			run.Failures = append(run.Failures, *result.Failure)
		}
	}
	if run.Summary.RequiredFailed > 0 {
		run.Status = Failed
	}
	completed := time.Now().UTC()
	run.CompletedAt = &completed
	if e.Events != nil {
		e.Events.Publish("verification.completed", request.ProjectID, run)
	}
	return run, nil
}
func Profile(name string, policy Policy, checks []VerificationCheck) VerificationProfile {
	if policy == "" {
		policy = Standard
	}
	if len(checks) > 0 {
		return VerificationProfile{Name: name, Policy: policy, Checks: checks}
	}
	required := func(id ID, t CheckType, n string, argv string) VerificationCheck {
		return VerificationCheck{ID: id, Type: t, Name: n, Required: true, Timeout: 10 * time.Minute, RetryPolicy: RetryPolicy{MaxAttempts: 1}, Configuration: map[string]any{"command": argv}}
	}
	switch policy {
	case Strict:
		return VerificationProfile{Name: name, Policy: policy, Checks: []VerificationCheck{required("build", Build, "Build", ""), required("tests", UnitTest, "Tests", ""), {ID: "security", Type: Security, Name: "Security", Required: true, Timeout: 5 * time.Minute, RetryPolicy: RetryPolicy{MaxAttempts: 1}, Configuration: map[string]any{}}}}
	case Fast:
		return VerificationProfile{Name: name, Policy: policy, Checks: []VerificationCheck{required("build", Build, "Build", ""), required("tests", UnitTest, "Essential tests", "")}}
	default:
		return VerificationProfile{Name: name, Policy: policy, Checks: []VerificationCheck{required("build", Build, "Build", ""), required("tests", UnitTest, "Tests", ""), {ID: "runtime", Type: RuntimeHealth, Name: "Runtime health", Required: true, Timeout: 2 * time.Minute, RetryPolicy: RetryPolicy{MaxAttempts: 1}, Dependencies: []ID{"build"}}}}
	}
}
func captureProjectState(ctx context.Context, project ProjectContext) ProjectContext {
	if project.RuntimeProvider == nil || project.RuntimeID == "" {
		if project.Worktree == "" {
			project.Worktree = "unknown"
		}
		return project
	}
	sha, _ := project.RuntimeProvider.Exec(ctx, project.RuntimeID, runtime.Command{Argv: []string{"git", "rev-parse", "HEAD"}, WorkingDir: project.Workspace, TimeoutSeconds: 10})
	if project.CommitSHA == "" && sha.ExitCode == 0 {
		project.CommitSHA = strings.TrimSpace(sha.Stdout)
	}
	status, _ := project.RuntimeProvider.Exec(ctx, project.RuntimeID, runtime.Command{Argv: []string{"git", "status", "--porcelain"}, WorkingDir: project.Workspace, TimeoutSeconds: 10})
	if project.Worktree == "" && status.ExitCode == 0 {
		project.Worktree = "clean"
		if strings.TrimSpace(status.Stdout) != "" {
			project.Worktree = "dirty"
			for _, line := range strings.Split(strings.TrimSpace(status.Stdout), "\\n") {
				if len(line) > 3 {
					project.ChangedFiles = append(project.ChangedFiles, strings.TrimSpace(line[3:]))
				}
			}
		}
	} else if project.Worktree == "" {
		project.Worktree = "unknown"
	}
	return project
}
func findCheck(checks []VerificationCheck, id ID) VerificationCheck {
	for _, c := range checks {
		if c.ID == id {
			return c
		}
	}
	return VerificationCheck{ID: id, Required: true}
}
