package runtime

import (
	"context"
	"testing"
)

type contractProvider struct{ runtimes map[string]Status }

func (p *contractProvider) Create(_ context.Context, spec Spec) (Runtime, error) {
	p.runtimes[spec.Name] = StatusStopped
	return Runtime{ID: spec.Name, Provider: "contract", Status: StatusStopped}, nil
}
func (p *contractProvider) Start(_ context.Context, id string) error {
	p.runtimes[id] = StatusRunning
	return nil
}
func (p *contractProvider) Stop(_ context.Context, id string) error {
	p.runtimes[id] = StatusStopped
	return nil
}
func (p *contractProvider) Restart(ctx context.Context, id string) error {
	_ = p.Stop(ctx, id)
	return p.Start(ctx, id)
}
func (p *contractProvider) Destroy(_ context.Context, id string) error {
	delete(p.runtimes, id)
	return nil
}
func (p *contractProvider) Status(_ context.Context, id string) (Status, error) {
	return p.runtimes[id], nil
}
func (p *contractProvider) Exec(_ context.Context, id string, cmd Command) (ExecResult, error) {
	return ExecResult{ExitCode: 0, Stdout: id + ":" + cmd.Argv[0]}, nil
}

func TestRuntimeProviderContractHarness(t *testing.T) {
	ctx := context.Background()
	p := &contractProvider{runtimes: map[string]Status{}}
	rt, err := p.Create(ctx, Spec{Name: "contract-project"})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Status(ctx, rt.ID); got != StatusStopped {
		t.Fatalf("created status = %s", got)
	}
	if err := p.Start(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := p.Status(ctx, rt.ID); got != StatusRunning {
		t.Fatalf("started status = %s", got)
	}
	result, err := p.Exec(ctx, rt.ID, Command{Argv: []string{"pwd"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exec exit = %d", result.ExitCode)
	}
	if err := p.Restart(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
	if err := p.Stop(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
	if err := p.Destroy(ctx, rt.ID); err != nil {
		t.Fatal(err)
	}
}
