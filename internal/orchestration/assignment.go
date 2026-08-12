package orchestration

import (
	"errors"

	"suda-forge/internal/agent"
	"suda-forge/internal/routing"
)

type RoutingRequestBuilder func(Task) routing.RoutingRequest
type AssignmentService struct {
	Router   func(routing.RoutingRequest) (routing.RoutingDecision, error)
	Selector AgentSelector
}

func (s AssignmentService) Assign(task Task, agents []agent.AgentDefinition, build RoutingRequestBuilder) (AgentAssignment, routing.RoutingDecision, error) {
	if s.Router == nil || s.Selector == nil || build == nil {
		return AgentAssignment{}, routing.RoutingDecision{}, errors.New("assignment dependencies are required")
	}
	decision, err := s.Router(build(task))
	if err != nil {
		return AgentAssignment{}, decision, err
	}
	assignment, err := s.Selector.Select(task, agents, decision)
	return assignment, decision, err
}
