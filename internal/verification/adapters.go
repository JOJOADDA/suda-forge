package verification

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"suda-forge/internal/runtime"
)

type CommandAdapter struct{}

func (CommandAdapter) Discover(context.Context, ProjectContext) error { return nil }
func (CommandAdapter) Validate(_ context.Context, check VerificationCheck, _ ProjectContext) error {
	if len(commandFor(check)) == 0 {
		return errors.New("verification command is not configured")
	}
	return nil
}
func (CommandAdapter) Execute(ctx context.Context, check VerificationCheck, project ProjectContext) (Evidence, error) {
	if project.RuntimeProvider == nil {
		return Evidence{}, errors.New("project runtime provider unavailable")
	}
	argv := commandFor(check)
	if len(argv) == 0 {
		return Evidence{}, errors.New("verification command is not configured")
	}
	timeout := check.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	result, err := project.RuntimeProvider.Exec(ctx, project.RuntimeID, runtime.Command{Argv: argv, WorkingDir: project.Workspace, Environment: envFor(check), TimeoutSeconds: int(timeout.Seconds())})
	duration := time.Since(started)
	exit := result.ExitCode
	evidence := Evidence{ExitCode: &exit, Stdout: sanitize(result.Stdout), Stderr: sanitize(result.Stderr), Duration: duration, CommitSHA: project.CommitSHA}
	if err != nil {
		return evidence, err
	}
	if result.ExitCode != 0 {
		return evidence, fmt.Errorf("verification command exited with code %d", result.ExitCode)
	}
	return evidence, nil
}
func (CommandAdapter) CollectEvidence(_ context.Context, _ VerificationCheck, _ ProjectContext, evidence Evidence) (Evidence, error) {
	return evidence, nil
}
func (CommandAdapter) NormalizeResult(check VerificationCheck, evidence Evidence, err error) VerificationResult {
	result := VerificationResult{CheckID: check.ID, Evidence: evidence, StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()}
	if err == nil {
		if isTestCheck(check.Type) && strings.Contains(strings.ToLower(evidence.Stdout+" "+evidence.Stderr), "no test files") {
			result.Status = Failed
			result.Failure = &FailureReport{ID: ID("failure_" + string(check.ID)), FailureType: TestFailure, CheckID: check.ID, SuspectedCause: "NO_TESTS_FOUND", Severity: severity(check.Required), Retryable: false}
			return result
		}
		result.Status = Passed
		return result
	}
	result.Status = Failed
	failureType := classify(check.Type)
	if errors.Is(err, context.DeadlineExceeded) {
		failureType = Timeout
	}
	result.Failure = &FailureReport{ID: ID("failure_" + string(check.ID)), FailureType: failureType, CheckID: check.ID, Command: commandFor(check), ExitCode: evidence.ExitCode, Stdout: evidence.Stdout, Stderr: evidence.Stderr, SuspectedCause: err.Error(), Severity: severity(check.Required), Retryable: check.RetryPolicy.MaxAttempts > 1}
	return result
}

type UnsupportedAdapter struct{}

func (UnsupportedAdapter) Discover(context.Context, ProjectContext) error { return nil }
func (UnsupportedAdapter) Validate(context.Context, VerificationCheck, ProjectContext) error {
	return nil
}
func (UnsupportedAdapter) Execute(context.Context, VerificationCheck, ProjectContext) (Evidence, error) {
	return Evidence{}, errors.New("verification adapter is unavailable in this environment")
}
func (UnsupportedAdapter) CollectEvidence(_ context.Context, _ VerificationCheck, _ ProjectContext, evidence Evidence) (Evidence, error) {
	return evidence, nil
}
func (UnsupportedAdapter) NormalizeResult(check VerificationCheck, evidence Evidence, err error) VerificationResult {
	return VerificationResult{CheckID: check.ID, Status: Failed, Evidence: evidence, Failure: &FailureReport{ID: ID("failure_" + string(check.ID)), FailureType: EnvironmentFailure, CheckID: check.ID, SuspectedCause: err.Error(), Severity: severity(check.Required), Retryable: false}, StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()}
}

func DefaultRegistry() *AdapterRegistry {
	r := NewAdapterRegistry()
	commandTypes := []CheckType{Build, Typecheck, Lint, Format, UnitTest, IntegrationTest, E2ETest, Custom, Dependency, Security}
	for _, t := range commandTypes {
		r.Register(t, CommandAdapter{})
	}
	for _, t := range []CheckType{RuntimeHealth, HTTPHealth, DatabaseHealth, Browser} {
		r.Register(t, UnsupportedAdapter{})
	}
	return r
}
func commandFor(check VerificationCheck) []string {
	if v, ok := check.Configuration["argv"].([]string); ok {
		return append([]string(nil), v...)
	}
	if v, ok := check.Configuration["argv"].([]any); ok {
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if v, ok := check.Configuration["command"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.Fields(v)
	}
	return nil
}
func envFor(check VerificationCheck) []string {
	if raw, ok := check.Configuration["environment"].([]string); ok {
		return append([]string(nil), raw...)
	}
	if raw, ok := check.Configuration["environment"].([]any); ok {
		out := []string{}
		for _, v := range raw {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
func sanitize(s string) string {
	for _, secret := range []string{"API_KEY", "TOKEN", "PASSWORD", "SECRET"} {
		if strings.Contains(strings.ToUpper(s), secret) {
			return "[REDACTED]"
		}
	}
	return s
}
func classify(t CheckType) FailureType {
	switch t {
	case Build:
		return BuildFailure
	case Typecheck:
		return TypeFailure
	case Lint, Format:
		return LintFailure
	case UnitTest, IntegrationTest, E2ETest:
		return TestFailure
	case RuntimeHealth, HTTPHealth:
		return RuntimeFailure
	case DatabaseHealth:
		return DatabaseFailure
	case Browser:
		return BrowserFailure
	case Security, Dependency:
		return SecurityFailure
	default:
		return UnknownFailure
	}
}
func isTestCheck(t CheckType) bool { return t == UnitTest || t == IntegrationTest || t == E2ETest }
func severity(required bool) string {
	if required {
		return "HIGH"
	}
	return "MEDIUM"
}
