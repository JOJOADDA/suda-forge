package constitution

import (
	"suda-forge/internal/agent"
	"testing"
)

func base() Constitution {
	return Constitution{ProjectID: "p1", AgentID: "codex", Identity: "implementation agent", Mission: "build safely", Authority: []string{"read workspace"}, Restrictions: []string{"no production changes"}, VerificationRules: []string{"build and test"}, Policy: agent.PermissionPolicy{ProjectID: "p1", AgentID: "codex", Allowed: []agent.Permission{agent.PermissionFilesystemRead}, ApprovalRequired: []agent.Permission{agent.PermissionFilesystemWrite}}}
}
func TestPolicyEvaluation(t *testing.T) {
	c := base()
	e := PolicyEvaluator{}
	if got := e.Evaluate(c, Action{ProjectID: "p1", AgentID: "codex", Permission: agent.PermissionFilesystemRead, Risk: RiskRead}).Effect; got != Allow {
		t.Fatalf("expected allow, got %s", got)
	}
	if got := e.Evaluate(c, Action{ProjectID: "p1", AgentID: "codex", Permission: agent.PermissionFilesystemWrite, Risk: RiskWrite}).Effect; got != ApprovalRequired {
		t.Fatalf("expected approval, got %s", got)
	}
	if got := e.Evaluate(c, Action{ProjectID: "p1", AgentID: "codex", Permission: agent.PermissionGitPush, Risk: RiskProduction}).Effect; got != Deny {
		t.Fatalf("expected deny, got %s", got)
	}
}
func TestValidate(t *testing.T) {
	if err := Validate(base()); err != nil {
		t.Fatal(err)
	}
	if err := Validate(Constitution{ProjectID: "p1"}); err == nil {
		t.Fatal("expected invalid constitution")
	}
}
