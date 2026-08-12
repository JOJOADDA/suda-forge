package orchestration

import "testing"

func TestPlannerProducesValidatedDependencyPlan(t *testing.T) {
	plan, err := DeterministicPlanner{}.Plan(PlannerInput{Intent: UserIntent{ProjectID: "p1", Goal: "build feature"}})
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ValidatePlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Tasks) != 3 {
		t.Fatalf("tasks=%d", len(graph.Tasks))
	}
	if got := ReadyTasks(graph); len(got) != 1 || got[0] != "analyze" {
		t.Fatalf("ready=%v", got)
	}
}
func TestPlanRejectsUnknownDependency(t *testing.T) {
	_, err := ValidatePlan(TaskPlan{Goal: "x", Tasks: []Task{{ID: "a"}}, Dependencies: map[ID][]ID{"a": {"missing"}}})
	if err != ErrMissingDependency {
		t.Fatalf("error=%v", err)
	}
}
