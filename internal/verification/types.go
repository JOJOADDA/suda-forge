package verification

import (
	"context"
	"time"

	"suda-forge/internal/runtime"
)

type ID string

type Status string

const (
	Pending   Status = "PENDING"
	Running   Status = "RUNNING"
	Passed    Status = "PASSED"
	Failed    Status = "FAILED"
	Cancelled Status = "CANCELLED"
	TimedOut  Status = "TIMED_OUT"
)

type CheckType string

const (
	Build           CheckType = "BUILD"
	Typecheck       CheckType = "TYPECHECK"
	Lint            CheckType = "LINT"
	Format          CheckType = "FORMAT"
	UnitTest        CheckType = "UNIT_TEST"
	IntegrationTest CheckType = "INTEGRATION_TEST"
	E2ETest         CheckType = "E2E_TEST"
	RuntimeHealth   CheckType = "RUNTIME_HEALTH"
	HTTPHealth      CheckType = "HTTP_HEALTH"
	DatabaseHealth  CheckType = "DATABASE_HEALTH"
	Browser         CheckType = "BROWSER"
	Security        CheckType = "SECURITY"
	Dependency      CheckType = "DEPENDENCY"
	Custom          CheckType = "CUSTOM"
)

type FailureType string

const (
	BuildFailure       FailureType = "BUILD_FAILURE"
	TestFailure        FailureType = "TEST_FAILURE"
	TypeFailure        FailureType = "TYPE_FAILURE"
	LintFailure        FailureType = "LINT_FAILURE"
	RuntimeFailure     FailureType = "RUNTIME_FAILURE"
	BrowserFailure     FailureType = "BROWSER_FAILURE"
	DatabaseFailure    FailureType = "DATABASE_FAILURE"
	SecurityFailure    FailureType = "SECURITY_FAILURE"
	Timeout            FailureType = "TIMEOUT"
	EnvironmentFailure FailureType = "ENVIRONMENT_FAILURE"
	UnknownFailure     FailureType = "UNKNOWN"
)

type Policy string

const (
	Strict       Policy = "STRICT"
	Standard     Policy = "STANDARD"
	Fast         Policy = "FAST"
	CustomPolicy Policy = "CUSTOM"
)

type VerificationRun struct {
	ID          ID                     `json:"id"`
	ProjectID   string                 `json:"project_id"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	TaskID      string                 `json:"task_id,omitempty"`
	TaskRunID   string                 `json:"task_run_id,omitempty"`
	Status      Status                 `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Summary     Summary                `json:"summary"`
	Profile     VerificationProfile    `json:"profile"`
	Checks      []VerificationCheck    `json:"checks"`
	Results     []VerificationResult   `json:"results"`
	Failures    []FailureReport        `json:"failures"`
	Repairs     []RepairAttempt        `json:"repairs"`
	Artifacts   []VerificationArtifact `json:"artifacts"`
	State       ProjectState           `json:"state"`
}

type Summary struct {
	RequiredPassed int `json:"required_passed"`
	RequiredFailed int `json:"required_failed"`
	OptionalPassed int `json:"optional_passed"`
	OptionalFailed int `json:"optional_failed"`
	Blocked        int `json:"blocked"`
}
type ProjectState struct {
	CommitSHA    string   `json:"commit_sha,omitempty"`
	Worktree     string   `json:"worktree"`
	ChangedFiles []string `json:"changed_files,omitempty"`
}

type VerificationCheck struct {
	ID                      ID             `json:"id"`
	Type                    CheckType      `json:"type"`
	Name                    string         `json:"name"`
	Description             string         `json:"description,omitempty"`
	Required                bool           `json:"required"`
	Timeout                 time.Duration  `json:"timeout"`
	RetryPolicy             RetryPolicy    `json:"retry_policy"`
	Configuration           map[string]any `json:"configuration,omitempty"`
	EnvironmentRequirements []string       `json:"environment_requirements,omitempty"`
	Dependencies            []ID           `json:"dependencies,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}
type VerificationResult struct {
	CheckID     ID             `json:"check_id"`
	Status      Status         `json:"status"`
	Evidence    Evidence       `json:"evidence"`
	Failure     *FailureReport `json:"failure,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt time.Time      `json:"completed_at"`
}
type Evidence struct {
	ExitCode     *int           `json:"exit_code,omitempty"`
	Stdout       string         `json:"stdout,omitempty"`
	Stderr       string         `json:"stderr,omitempty"`
	Duration     time.Duration  `json:"duration"`
	TestCount    *int           `json:"test_count,omitempty"`
	PassedTests  *int           `json:"passed_tests,omitempty"`
	FailedTests  *int           `json:"failed_tests,omitempty"`
	SkippedTests *int           `json:"skipped_tests,omitempty"`
	HTTPStatus   *int           `json:"http_status,omitempty"`
	URL          string         `json:"url,omitempty"`
	ArtifactPath string         `json:"artifact_path,omitempty"`
	CommitSHA    string         `json:"commit_sha,omitempty"`
	Structured   map[string]any `json:"structured,omitempty"`
}
type FailureReport struct {
	ID                         ID          `json:"id"`
	FailureType                FailureType `json:"failure_type"`
	CheckID                    ID          `json:"check_id"`
	TaskID                     string      `json:"task_id,omitempty"`
	Command                    []string    `json:"command,omitempty"`
	ExitCode                   *int        `json:"exit_code,omitempty"`
	Stdout, Stderr, StackTrace string      `json:"stdout,omitempty" json:"stderr,omitempty" json:"stack_trace,omitempty"`
	AffectedFiles              []string    `json:"affected_files,omitempty"`
	SuspectedCause             string      `json:"suspected_cause"`
	Severity                   string      `json:"severity"`
	Retryable                  bool        `json:"retryable"`
}
type VerificationArtifact struct {
	ID        ID             `json:"id"`
	ProjectID string         `json:"project_id"`
	TaskID    string         `json:"task_id,omitempty"`
	RunID     ID             `json:"verification_run_id"`
	CheckID   ID             `json:"check_id"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
type RepairPlan struct {
	Problem             string   `json:"problem"`
	RootCauseHypothesis string   `json:"root_cause_hypothesis"`
	AffectedFiles       []string `json:"affected_files"`
	RecommendedChanges  []string `json:"recommended_changes"`
	VerificationTargets []ID     `json:"verification_targets"`
	Risk                string   `json:"risk"`
	Confidence          float64  `json:"confidence"`
}
type RepairAttempt struct {
	ID          ID         `json:"id"`
	RunID       ID         `json:"verification_run_id"`
	Attempt     int        `json:"attempt"`
	Status      Status     `json:"status"`
	Plan        RepairPlan `json:"plan"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
type VerificationProfile struct {
	Name   string              `json:"name"`
	Policy Policy              `json:"policy"`
	Checks []VerificationCheck `json:"checks"`
}
type ProjectContext struct {
	ProjectID       string
	RuntimeID       string
	RuntimeProvider runtime.Provider
	Workspace       string
	CommitSHA       string
	Worktree        string
	ChangedFiles    []string
}
type TaskContext struct {
	WorkflowID  string
	TaskID      string
	TaskRunID   string
	Description string
}

type CheckRequest struct {
	ProjectID  string              `json:"project_id"`
	WorkflowID string              `json:"workflow_id,omitempty"`
	TaskID     string              `json:"task_id,omitempty"`
	TaskRunID  string              `json:"task_run_id,omitempty"`
	RuntimeID  string              `json:"runtime_id,omitempty"`
	Workspace  string              `json:"workspace,omitempty"`
	Profile    string              `json:"profile,omitempty"`
	Policy     Policy              `json:"policy,omitempty"`
	Checks     []VerificationCheck `json:"checks,omitempty"`
}
type Adapter interface {
	Discover(context.Context, ProjectContext) error
	Validate(context.Context, VerificationCheck, ProjectContext) error
	Execute(context.Context, VerificationCheck, ProjectContext) (Evidence, error)
	CollectEvidence(context.Context, VerificationCheck, ProjectContext, Evidence) (Evidence, error)
	NormalizeResult(VerificationCheck, Evidence, error) VerificationResult
}
type AdapterRegistry struct{ adapters map[CheckType]Adapter }

func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{adapters: map[CheckType]Adapter{}}
}
func (r *AdapterRegistry) Register(t CheckType, a Adapter) { r.adapters[t] = a }
func (r *AdapterRegistry) Get(t CheckType) (Adapter, bool) { a, ok := r.adapters[t]; return a, ok }
