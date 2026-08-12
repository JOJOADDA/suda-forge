package orchestration

import (
	"context"
	"testing"
)

type eventExec struct{}

func (eventExec) Execute(context.Context, Task, TaskRun) (TaskResult, error) {
	return TaskResult{Status: TaskSucceeded, Summary: "done"}, nil
}
func (eventExec) Cancel(context.Context, TaskRun) error { return nil }
func TestObservedExecutorPublishesNormalizedEvents(t *testing.T) {
	sink := &MemoryEventSink{}
	e := ObservedExecutor{Inner: eventExec{}, Sink: sink}
	_, err := e.Execute(context.Background(), Task{ID: "t1", WorkflowID: "w1"}, TaskRun{ID: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	items := sink.Events()
	if len(items) != 2 || items[0].Type != "task.started" || items[1].Type != "task.completed" {
		t.Fatalf("events=%+v", items)
	}
}
