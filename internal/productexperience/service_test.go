package productexperience

import (
	"suda-forge/internal/agent"
	"suda-forge/internal/constitution"
	"suda-forge/internal/knowledge"
	"testing"
)

func TestContextRecoveryAndImpact(t *testing.T) {
	s := NewService(nil)
	s.Constitutions["p1:codex"] = constitution.Constitution{ProjectID: "p1", AgentID: "codex", Identity: "builder", Mission: "build", Authority: []string{"workspace"}, Restrictions: []string{"no production"}, SecurityRules: []string{"approval for deploy"}, VerificationRules: []string{"build"}, Policy: agent.PermissionPolicy{ProjectID: "p1", AgentID: "codex", Allowed: []agent.Permission{agent.PermissionFilesystemRead}}}
	_, _ = s.Knowledge.UpsertNode(knowledge.Node{ID: "file.auth", ProjectID: "p1", Type: knowledge.File, Name: "auth.go", Path: "internal/auth.go"})
	_, _ = s.Knowledge.UpsertNode(knowledge.Node{ID: "api.login", ProjectID: "p1", Type: knowledge.API, Name: "login", Path: "/login"})
	_, _ = s.Knowledge.UpsertNode(knowledge.Node{ID: "table.users", ProjectID: "p1", Type: knowledge.Table, Name: "users"})
	_, _ = s.Knowledge.UpsertEdge(knowledge.Edge{ID: "file.api", ProjectID: "p1", From: "file.auth", To: "api.login", Type: knowledge.Implements})
	_, _ = s.Knowledge.UpsertEdge(knowledge.Edge{ID: "api.table", ProjectID: "p1", From: "api.login", To: "table.users", Type: knowledge.Writes})
	ctx, err := s.Assemble(ContextRequest{ProjectID: "p1", AgentID: "codex", Role: "implementer", Task: "implement login"})
	if err != nil || len(ctx.Knowledge.Nodes) != 3 {
		t.Fatalf("context assembly failed: %#v %v", ctx, err)
	}
	recovery, err := s.Recover("p1", ContextRequest{ProjectID: "p1", AgentID: "codex", Task: "resume"})
	if err != nil || len(recovery.Knowledge.Nodes) != 3 {
		t.Fatalf("recovery failed: %#v %v", recovery, err)
	}
	impact, err := s.Impact("p1", "file.auth")
	if err != nil || impact.Risk != "HIGH" || !impact.ApprovalRequired {
		t.Fatalf("expected high impact approval, got %#v %v", impact, err)
	}
}
func TestLoopPlanUsesExistingSubsystems(t *testing.T) {
	plan, err := NewService(nil).PlanLoop("p1", map[LoopStage]string{VisualQA: "browser capability unavailable"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "PARTIALLY_BLOCKED" || plan.Delegates[Fix] != "verification.RepairLoop" {
		t.Fatalf("unexpected loop plan: %#v", plan)
	}
}
