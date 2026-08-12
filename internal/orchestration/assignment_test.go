package orchestration

import (
	"testing"

	"suda-forge/internal/agent"
	"suda-forge/internal/routing"
)

func TestAssignmentUsesRouterThenAgentSelector(t *testing.T) {
	service := AssignmentService{Router: func(routing.RoutingRequest) (routing.RoutingDecision, error) {
		return routing.RoutingDecision{Selected: routing.ModelReference{ProviderID: "ollama", ModelID: "local-code"}}, nil
	}, Selector: CapabilityAgentSelector{}}
	assignment, decision, err := service.Assign(Task{ID: "t1"}, []agent.AgentDefinition{{ID: "codex", DisplayName: "Codex", Status: "AVAILABLE"}}, func(Task) routing.RoutingRequest { return routing.RoutingRequest{} })
	if err != nil {
		t.Fatal(err)
	}
	if assignment.AgentID != "codex" || decision.Selected.ModelID != "local-code" {
		t.Fatalf("assignment=%+v decision=%+v", assignment, decision)
	}
}
