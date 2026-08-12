package deployment

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"suda-forge/internal/runtime"
)

var ErrDeploymentNotFound = errors.New("deployment not found")
var ErrReleaseNotFound = errors.New("release not found")
var ErrInvalidDeploymentTransition = errors.New("invalid deployment transition")

type VerificationGate interface {
	Verify(context.Context, Deployment) error
}
type Manager struct {
	mu          sync.RWMutex
	releases    map[ID]Release
	deployments map[ID]Deployment
	active      map[string]ID
	Runtime     runtime.Provider
	Deployer    DeploymentProvider
	Health      HealthChecker
	Proxy       ProxyProvider
	Events      AuditSink
	Verify      VerificationGate
	Now         func() time.Time
}

func NewManager(now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{releases: map[ID]Release{}, deployments: map[ID]Deployment{}, active: map[string]ID{}, Now: now}
}
func (s *Manager) CreateRelease(_ context.Context, release Release) (Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if release.ID == "" {
		release.ID = ID(fmt.Sprintf("release_%d", s.Now().UnixNano()))
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = s.Now().UTC()
	}
	if release.Status == "" {
		release.Status = ReleaseCreated
	}
	s.releases[release.ID] = release
	return release, nil
}
func (s *Manager) Releases(_ context.Context, projectID string) []Release {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Release{}
	for _, release := range s.releases {
		if release.ProjectID == projectID {
			out = append(out, release)
		}
	}
	return out
}
func (s *Manager) CreateDeployment(_ context.Context, deployment Deployment) (Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if deployment.ID == "" {
		deployment.ID = ID(fmt.Sprintf("deploy_%d", s.Now().UnixNano()))
	}
	if deployment.Status == "" {
		deployment.Status = DeploymentPending
	}
	if deployment.Strategy == "" {
		deployment.Strategy = StrategyRecreate
	}
	deployment.CreatedAt = s.Now().UTC()
	s.deployments[deployment.ID] = deployment
	s.emit("deployment.created", deployment)
	return deployment, nil
}
func (s *Manager) Deployment(_ context.Context, id ID) (Deployment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deployment, ok := s.deployments[id]
	if !ok {
		return Deployment{}, ErrDeploymentNotFound
	}
	return deployment, nil
}
func (s *Manager) Deploy(ctx context.Context, id ID, release Release, environment EnvironmentConfig) (Deployment, error) {
	s.mu.Lock()
	deployment, ok := s.deployments[id]
	if !ok {
		s.mu.Unlock()
		return Deployment{}, ErrDeploymentNotFound
	}
	if deployment.ReleaseID == "" {
		deployment.ReleaseID = release.ID
	}
	s.mu.Unlock()
	steps := []struct {
		status DeploymentStatus
		run    func() error
	}{{DeploymentPreparing, func() error { return s.prepare(ctx, deployment, environment) }}, {DeploymentBuilding, func() error { return s.execConfigured(ctx, deployment, "build") }},
		{DeploymentTesting, func() error { return s.execConfigured(ctx, deployment, "test") }},
		{DeploymentVerifying, func() error {
			if s.Verify != nil {
				return s.Verify.Verify(ctx, deployment)
			}
			return nil
		}}, {DeploymentDeploying, func() error {
			if s.Deployer == nil {
				return errors.New("deployment provider unavailable")
			}
			return s.Deployer.Deploy(ctx, deployment, release, environment)
		}}, {DeploymentHealthCheck, func() error {
			if s.Health == nil {
				return errors.New("health checker unavailable")
			}
			target, ok := deployment.Metadata["health_target"].(string)
			if !ok || target == "" {
				return errors.New("health_target is required before activation")
			}
			check := HealthCheck{ID: ID(string(deployment.ID) + "/health"), ProjectID: deployment.ProjectID, RuntimeID: deployment.RuntimeTarget, Type: HealthHTTP, Target: target, Timeout: 10 * time.Second, Retries: 2, FailureThreshold: 1, SuccessThreshold: 1}
			_, err := s.Health.Check(ctx, check)
			return err
		}},
		{DeploymentActive, func() error { return s.activate(deployment) }}}
	for _, step := range steps {
		if err := s.transition(id, step.status); err != nil {
			return s.fail(id, err)
		}
		if err := step.run(); err != nil {
			return s.fail(id, err)
		}
	}
	return s.Deployment(ctx, id)
}
func (s *Manager) Rollback(ctx context.Context, projectID string, target ID) (Deployment, error) {
	s.mu.Lock()
	previous, ok := s.deployments[target]
	s.mu.Unlock()
	if !ok {
		return Deployment{}, ErrDeploymentNotFound
	}
	if s.Deployer == nil {
		return Deployment{}, errors.New("deployment provider unavailable")
	}
	if err := s.Deployer.Deploy(ctx, previous, Release{ID: previous.ReleaseID, ProjectID: projectID, GitRevision: previous.SourceRevision}, EnvironmentConfig{ProjectID: projectID, Environment: previous.Environment}); err != nil {
		return Deployment{}, err
	}
	_ = s.transition(target, DeploymentRolledBack)
	s.emit("deployment.rolled_back", previous)
	return s.Deployment(ctx, target)
}
func (s *Manager) prepare(ctx context.Context, deployment Deployment, environment EnvironmentConfig) error {
	if environment.CPU < 0 || environment.MemoryBytes < 0 || environment.DiskBytes < 0 {
		return errors.New("invalid resource policy")
	}
	return nil
}
func (s *Manager) execConfigured(ctx context.Context, deployment Deployment, name string) error {
	raw, ok := deployment.Metadata[name+"_argv"]
	if !ok {
		return fmt.Errorf("%s_argv is required", name)
	}
	values, ok := raw.([]any)
	if !ok {
		if typed, typedOK := raw.([]string); typedOK {
			return s.exec(ctx, deployment, name, typed)
		}
		return fmt.Errorf("%s_argv must be an array", name)
	}
	argv := make([]string, 0, len(values))
	for _, item := range values {
		value, ok := item.(string)
		if !ok || value == "" {
			return fmt.Errorf("%s_argv contains invalid argument", name)
		}
		argv = append(argv, value)
	}
	return s.exec(ctx, deployment, name, argv)
}
func (s *Manager) exec(ctx context.Context, deployment Deployment, name string, argv []string) error {
	if s.Runtime == nil {
		return errors.New("runtime provider unavailable")
	}
	result, err := s.Runtime.Exec(ctx, deployment.RuntimeTarget, runtime.Command{Argv: argv, WorkingDir: "/workspace", TimeoutSeconds: 900})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("%s failed: %s", name, result.Stderr)
	}
	return nil
}
func (s *Manager) activate(deployment Deployment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.deployments[deployment.ID]; ok {
		deployment = current
	}
	deployment.Status = DeploymentActive
	s.active[deployment.ProjectID] = deployment.ID
	now := s.Now().UTC()

	deployment.CompletedAt = &now
	deployment.HealthStatus = "PASSED"
	s.deployments[deployment.ID] = deployment
	return nil
}
func (s *Manager) transition(id ID, status DeploymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment, ok := s.deployments[id]
	if !ok {
		return ErrDeploymentNotFound
	}
	if !validTransition(deployment.Status, status) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidDeploymentTransition, deployment.Status, status)
	}
	deployment.Status = status
	if deployment.StartedAt == nil && status != DeploymentPending {
		now := s.Now().UTC()
		deployment.StartedAt = &now
	}
	s.deployments[id] = deployment
	s.emit("deployment."+string(status), deployment)
	return nil
}
func (s *Manager) fail(id ID, err error) (Deployment, error) {
	s.mu.Lock()
	deployment := s.deployments[id]
	deployment.Status = DeploymentFailed
	deployment.FailureReason = err.Error()
	now := s.Now().UTC()
	deployment.CompletedAt = &now
	s.deployments[id] = deployment
	s.mu.Unlock()
	s.emit("deployment.failed", deployment)
	return deployment, err
}
func validTransition(from, to DeploymentStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case DeploymentPending:
		return to == DeploymentPreparing || to == DeploymentCancelled
	case DeploymentPreparing:
		return to == DeploymentBuilding || to == DeploymentFailed
	case DeploymentBuilding:
		return to == DeploymentTesting || to == DeploymentFailed
	case DeploymentTesting:
		return to == DeploymentVerifying || to == DeploymentFailed
	case DeploymentVerifying:
		return to == DeploymentDeploying || to == DeploymentFailed
	case DeploymentDeploying:
		return to == DeploymentHealthCheck || to == DeploymentFailed
	case DeploymentHealthCheck:
		return to == DeploymentActive || to == DeploymentFailed
	case DeploymentActive:
		return to == DeploymentRolledBack
	default:
		return false
	}
}
func (s *Manager) emit(kind string, data any) {
	if s.Events != nil {
		s.Events.Publish(kind, "", data)
	}
}
