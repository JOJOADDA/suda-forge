package productexperience

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"suda-forge/internal/constitution"
	"suda-forge/internal/designintelligence"
	"suda-forge/internal/knowledge"
)

type Service struct {
	Knowledge     knowledge.Store
	DesignSystems map[string]designintelligence.DesignSystem
	Constitutions map[string]constitution.Constitution
	Now           func() time.Time
}

func NewService(now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{Knowledge: knowledge.NewMemoryStore(now), DesignSystems: map[string]designintelligence.DesignSystem{}, Constitutions: map[string]constitution.Constitution{}, Now: now}
}
func (s *Service) Assemble(in ContextRequest) (AgentContext, error) {
	if in.ProjectID == "" || in.Task == "" {
		return AgentContext{}, errors.New("project_id and task are required")
	}
	c, ok := s.Constitutions[in.ProjectID+":"+in.AgentID]
	if !ok {
		return AgentContext{}, errors.New("agent constitution not found")
	}
	g, err := s.Knowledge.Graph(in.ProjectID)
	if err != nil {
		return AgentContext{}, err
	}
	var design *designintelligence.DesignSystem
	if d, ok := s.DesignSystems[in.ProjectID]; ok {
		copy := d
		design = &copy
	}
	return AgentContext{CoreConstitution: "SUDA FORGE agents act only within project policy, use available tools truthfully, and verify meaningful changes.", SecurityPolicy: c.SecurityRules, Role: in.Role, ProjectPolicy: c, Task: in.Task, Knowledge: g, DesignSystem: design, CurrentState: in.CurrentState, AvailableTools: in.AvailableTools, RuntimeCapabilities: in.RuntimeCapabilities, VerificationRequirements: in.VerificationRequirements, AssembledAt: s.Now().UTC()}, nil
}
func (s *Service) Recover(projectID string, in ContextRequest) (SessionRecovery, error) {
	ctx, err := s.Assemble(in)
	if err != nil {
		return SessionRecovery{}, err
	}
	decisions, tasks, bugs := []string{}, []string{}, []string{}
	for _, n := range ctx.Knowledge.Nodes {
		switch n.Type {
		case knowledge.Decision:
			decisions = append(decisions, n.Name)
		case knowledge.Task:
			tasks = append(tasks, n.Name)
		case knowledge.Bug:
			bugs = append(bugs, n.Name)
		}
	}
	summary := fmt.Sprintf("Project %s has %d knowledge nodes, %d decisions, %d tasks, and %d bugs.", projectID, len(ctx.Knowledge.Nodes), len(decisions), len(tasks), len(bugs))
	return SessionRecovery{ProjectID: projectID, CurrentState: in.CurrentState, Knowledge: ctx.Knowledge, Decisions: decisions, Tasks: tasks, Bugs: bugs, DesignSystem: ctx.DesignSystem, GitState: map[string]string{}, VerificationState: map[string]string{}, Summary: summary}, nil
}
func (s *Service) Impact(projectID string, root knowledge.NodeID) (ImpactAnalysis, error) {
	g, err := s.Knowledge.Graph(projectID)
	if err != nil {
		return ImpactAnalysis{}, err
	}
	seen := map[knowledge.NodeID]bool{root: true}
	queue := []knowledge.NodeID{root}
	out := []knowledge.Node{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, e := range g.Edges {
			if e.From != id || seen[e.To] {
				continue
			}
			seen[e.To] = true
			queue = append(queue, e.To)
		}
	}
	for _, n := range g.Nodes {
		if seen[n.ID] && n.ID != root {
			out = append(out, n)
		}
	}
	result := ImpactAnalysis{ProjectID: projectID, RootNode: root, Risk: "LOW", Nodes: out, Files: []string{}, APIs: []string{}, Components: []string{}, Tests: []string{}, Reasons: []string{}}
	for _, n := range out {
		switch n.Type {
		case knowledge.File:
			result.Files = append(result.Files, n.Path)
		case knowledge.API:
			result.APIs = append(result.APIs, n.Name)
		case knowledge.Component:
			result.Components = append(result.Components, n.Name)
		case knowledge.Test:
			result.Tests = append(result.Tests, n.Name)
		case knowledge.Database, knowledge.Table:
			result.Risk = "HIGH"
			result.ApprovalRequired = true
			result.Reasons = append(result.Reasons, "database impact requires approval")
		}
	}
	if len(result.Files)+len(result.APIs)+len(result.Components)+len(result.Tests) > 8 && result.Risk == "LOW" {
		result.Risk = "MEDIUM"
	}
	if len(result.Reasons) == 0 {
		result.Reasons = append(result.Reasons, "impact is derived from structured graph relationships")
	}
	return result, nil
}
func (s *Service) PlanLoop(projectID string, blocked map[LoopStage]string) (LoopPlan, error) {
	if projectID == "" {
		return LoopPlan{}, errors.New("project_id is required")
	}
	stages := []LoopStage{Intent, Plan, Architect, Design, Implement, Build, Test, Verify, VisualQA, Security, Fix, Commit, Deploy, PostDeployVerify}
	delegates := map[LoopStage]string{Intent: "projectintelligence.Engine", Plan: "orchestration.Orchestrator", Architect: "projectintelligence.Engine", Design: "designintelligence.Engine", Implement: "orchestration.Orchestrator", Build: "verification.Engine", Test: "verification.Engine", Verify: "verification.Engine", VisualQA: "verification.Engine + Project Computer browser capability", Security: "verification.Engine", Fix: "verification.RepairLoop", Commit: "orchestration.WorktreeManager", Deploy: "deployment.Manager", PostDeployVerify: "verification.Engine + deployment health"}
	blockedStages := []LoopStage{}
	for stage := range blocked {
		blockedStages = append(blockedStages, stage)
	}
	status := "READY"
	if len(blockedStages) > 0 {
		status = "PARTIALLY_BLOCKED"
	}
	return LoopPlan{ID: "loop_" + projectID, ProjectID: projectID, Stages: stages, Delegates: delegates, Status: status, BlockedStages: blockedStages, CreatedAt: s.Now().UTC()}, nil
}
func NormalizeActionText(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
