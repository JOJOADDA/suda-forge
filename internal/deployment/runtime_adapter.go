package deployment

import (
	"context"
	"errors"
	"fmt"

	"suda-forge/internal/runtime"
)

type RuntimeDeploymentProvider struct{ Runtime runtime.Provider }

func (p RuntimeDeploymentProvider) Deploy(ctx context.Context, d Deployment, r Release, e EnvironmentConfig) error {
	if p.Runtime == nil {
		return errors.New("runtime provider unavailable")
	}
	path := "/workspace/.suda-forge/releases/" + string(r.ID)
	result, err := p.Runtime.Exec(ctx, d.RuntimeTarget, runtime.Command{Argv: []string{"sh", "-lc", "mkdir -p -- '" + path + "' && printf '%s\n' '" + r.GitRevision + "' > '" + path + "/REVISION'"}, WorkingDir: "/workspace", TimeoutSeconds: 60})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("runtime deployment failed: %s", result.Stderr)
	}
	return nil
}
func (p RuntimeDeploymentProvider) Status(ctx context.Context, d Deployment) (DeploymentStatus, error) {
	if p.Runtime == nil {
		return DeploymentFailed, errors.New("runtime provider unavailable")
	}
	status, err := p.Runtime.Status(ctx, d.RuntimeTarget)
	if err != nil {
		return DeploymentFailed, err
	}
	if status == runtime.StatusRunning {
		return DeploymentActive, nil
	}
	if status == runtime.StatusFailed {
		return DeploymentFailed, nil
	}
	return DeploymentPending, nil
}
func (p RuntimeDeploymentProvider) Stop(ctx context.Context, d Deployment) error {
	if p.Runtime == nil {
		return errors.New("runtime provider unavailable")
	}
	_, err := p.Runtime.Exec(ctx, d.RuntimeTarget, runtime.Command{Argv: []string{"sh", "-lc", "true"}, WorkingDir: "/workspace", TimeoutSeconds: 30})
	return err
}

type VerificationAdapter struct {
	Check func(context.Context, string) error
}

func (v VerificationAdapter) Verify(ctx context.Context, d Deployment) error {
	if v.Check == nil || d.Metadata == nil {
		return errors.New("deployment requires authoritative verification evidence")
	}
	runID, ok := d.Metadata["verification_run_id"].(string)
	if !ok || runID == "" {
		return errors.New("deployment requires verification_run_id")
	}
	return v.Check(ctx, runID)
}
