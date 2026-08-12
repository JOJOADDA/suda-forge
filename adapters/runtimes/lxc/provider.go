package lxc

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"suda-forge/internal/runtime"
)

// Provider uses classic LXC userspace commands. The domain only sees runtime.Provider,
// so a future LXD, Docker, VM, or remote-node adapter does not alter application code.
type Provider struct {
	CreateBinary  string
	StartBinary   string
	StopBinary    string
	InfoBinary    string
	AttachBinary  string
	DestroyBinary string
}

func New() *Provider {
	return &Provider{CreateBinary: "lxc-create", StartBinary: "lxc-start", StopBinary: "lxc-stop", InfoBinary: "lxc-info", AttachBinary: "lxc-attach", DestroyBinary: "lxc-destroy"}
}
func (p *Provider) Name() string { return "lxc" }

func (p *Provider) Create(ctx context.Context, spec runtime.Spec) (runtime.Runtime, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return runtime.Runtime{}, errors.New("runtime name is required")
	}
	args := []string{"-n", spec.Name, "-t", "download", "--", "-d", "ubuntu", "-r", "jammy", "-a", "amd64"}
	if output, err := p.run(ctx, p.CreateBinary, args...); err != nil {
		return runtime.Runtime{}, commandError(spec.Name, "create", output, err)
	}
	if output, err := p.run(ctx, p.AttachBinary, "-n", spec.Name, "--", "mkdir", "-p", "/workspace"); err != nil {
		return runtime.Runtime{}, commandError(spec.Name, "workspace-init", output, err)
	}
	if spec.CPU > 0 {
		_, _ = p.run(ctx, "lxc-cgroup", "-n", spec.Name, "cpuset.cpus", fmt.Sprintf("0-%d", spec.CPU-1))
	}
	return runtime.Runtime{ID: spec.Name, Provider: p.Name(), Status: runtime.StatusStopped}, nil
}
func (p *Provider) Start(ctx context.Context, id string) error {
	_, err := p.run(ctx, p.StartBinary, "-n", id, "-d")
	return wrap(id, "start", err)
}
func (p *Provider) Stop(ctx context.Context, id string) error {
	_, err := p.run(ctx, p.StopBinary, "-n", id, "-k")
	return wrap(id, "stop", err)
}
func (p *Provider) Restart(ctx context.Context, id string) error {
	if err := p.Stop(ctx, id); err != nil {
		return err
	}
	return p.Start(ctx, id)
}
func (p *Provider) Destroy(ctx context.Context, id string) error {
	_, err := p.run(ctx, p.DestroyBinary, "-n", id)
	return wrap(id, "destroy", err)
}
func (p *Provider) Status(ctx context.Context, id string) (runtime.Status, error) {
	out, err := p.run(ctx, p.InfoBinary, "-n", id)
	if err != nil {
		return runtime.StatusUnknown, wrap(id, "status", err)
	}
	text := strings.ToLower(string(out))
	if strings.Contains(text, "state: running") {
		return runtime.StatusRunning, nil
	}
	if strings.Contains(text, "state: stopped") {
		return runtime.StatusStopped, nil
	}
	return runtime.StatusUnknown, nil
}
func (p *Provider) Exec(ctx context.Context, id string, cmd runtime.Command) (runtime.ExecResult, error) {
	if len(cmd.Argv) == 0 {
		return runtime.ExecResult{}, errors.New("command is required")
	}
	args := []string{"-n", id}
	if cmd.WorkingDir != "" {
		args = append(args, "--cwd", cmd.WorkingDir)
	}
	args = append(args, "--")
	args = append(args, cmd.Argv...)
	out, err := p.run(ctx, p.AttachBinary, args...)
	if err != nil {
		return runtime.ExecResult{Stderr: string(out), ExitCode: 1}, wrap(id, "exec", err)
	}
	return runtime.ExecResult{Stdout: string(out)}, nil
}
func (p *Provider) run(ctx context.Context, binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, args...).CombinedOutput()
}
func wrap(id, operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("lxc %s %s: %w", operation, id, err)
}

func commandError(id, operation string, output []byte, err error) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return wrap(id, operation, err)
	}
	return fmt.Errorf("lxc %s %s: %w: %s", operation, id, err, message)
}
