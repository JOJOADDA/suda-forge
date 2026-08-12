package agent

import (
	"context"
	"testing"

	"suda-forge/internal/runtime"
)

type fakeProcess struct {
	started bool
	wrote   string
	stopped bool
}

func (p *fakeProcess) Start(context.Context, runtime.Spec, runtime.Command) (string, error) {
	p.started = true
	return "process-1", nil
}
func (p *fakeProcess) Write(_ context.Context, _ string, data []byte) error {
	p.wrote = string(data)
	return nil
}
func (p *fakeProcess) Stop(context.Context, string) error             { p.stopped = true; return nil }
func (p *fakeProcess) Status(context.Context, string) (string, error) { return "running", nil }
func (p *fakeProcess) Output(context.Context, string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "provider line"
	close(ch)
	return ch, nil
}

func TestCLIAdaptersUseRuntimeProcessContract(t *testing.T) {
	for _, adapter := range []*CLIAdapter{NewCodexAdapter(&fakeProcess{}), NewClaudeCodeAdapter(&fakeProcess{}), NewKimiAdapter(&fakeProcess{})} {
		process := adapter.Process.(*fakeProcess)
		session := Session{ID: SessionID(adapter.ID() + "-session"), RuntimeID: "runtime-1", WorkingDirectory: "/workspace"}
		if err := adapter.Start(context.Background(), session); err != nil {
			t.Fatal(adapter.ID(), err)
		}
		if !process.started {
			t.Fatal(adapter.ID(), "did not start through process manager")
		}
		if err := adapter.SendMessage(context.Background(), session.ID, "hello"); err != nil {
			t.Fatal(adapter.ID(), err)
		}
		if process.wrote != "hello\n" {
			t.Fatalf("%s wrote %q", adapter.ID(), process.wrote)
		}
		stream, err := adapter.StreamEvents(context.Background(), session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got := <-stream; got.Type != EventMessage || got.Raw["provider"] != adapter.ID() || got.Raw["line"] != "provider line" {
			t.Fatalf("unexpected normalized event: %+v", got)
		}
		if err := adapter.Cancel(context.Background(), session.ID); err != nil {
			t.Fatal(err)
		}
		if !process.stopped {
			t.Fatal(adapter.ID(), "did not stop through process manager")
		}
	}
}
