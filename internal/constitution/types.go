package constitution

import (
	"errors"
	"suda-forge/internal/agent"
)

type ID string
type Risk string

const (
	RiskRead       Risk = "READ"
	RiskWrite      Risk = "WRITE"
	RiskExecute    Risk = "EXECUTE"
	RiskNetwork    Risk = "NETWORK"
	RiskInstall    Risk = "INSTALL"
	RiskDelete     Risk = "DELETE"
	RiskDatabase   Risk = "DATABASE"
	RiskDeploy     Risk = "DEPLOY"
	RiskProduction Risk = "PRODUCTION"
	RiskCredential Risk = "CREDENTIAL"
)

type Effect string

const (
	Allow            Effect = "ALLOW"
	Deny             Effect = "DENY"
	ApprovalRequired Effect = "APPROVAL_REQUIRED"
)

type Action struct {
	ProjectID   string           `json:"project_id"`
	AgentID     agent.ID         `json:"agent_id"`
	Permission  agent.Permission `json:"permission"`
	Risk        Risk             `json:"risk"`
	Resource    string           `json:"resource,omitempty"`
	Description string           `json:"description,omitempty"`
}
type Decision struct {
	Effect     Effect `json:"effect"`
	Action     Action `json:"action"`
	Reason     string `json:"reason"`
	ApprovalID string `json:"approval_id,omitempty"`
}
type Constitution struct {
	ID                 ID                     `json:"id"`
	ProjectID          string                 `json:"project_id"`
	AgentID            agent.ID               `json:"agent_id"`
	Identity           string                 `json:"identity"`
	Mission            string                 `json:"mission"`
	Authority          []string               `json:"authority"`
	Restrictions       []string               `json:"restrictions"`
	DecisionRules      []string               `json:"decision_rules"`
	ToolRules          map[string]string      `json:"tool_rules"`
	VerificationRules  []string               `json:"verification_rules"`
	SecurityRules      []string               `json:"security_rules"`
	CollaborationRules []string               `json:"collaboration_rules"`
	Policy             agent.PermissionPolicy `json:"policy"`
}
type PolicyEvaluator struct{}

func (PolicyEvaluator) Evaluate(c Constitution, a Action) Decision {
	if c.ProjectID != a.ProjectID || c.AgentID != a.AgentID {
		return Decision{Effect: Deny, Action: a, Reason: "constitution ownership mismatch"}
	}
	if a.Risk == RiskProduction || a.Risk == RiskDatabase {
		return Decision{Effect: Deny, Action: a, Reason: "high-risk production/database action requires a specialized approval boundary"}
	}
	if !contains(c.Policy.Allowed, a.Permission) && !contains(c.Policy.ApprovalRequired, a.Permission) {
		return Decision{Effect: Deny, Action: a, Reason: "permission is not allowed by project policy"}
	}
	if contains(c.Policy.ApprovalRequired, a.Permission) {
		return Decision{Effect: ApprovalRequired, Action: a, Reason: "project policy requires approval for this action"}
	}
	return Decision{Effect: Allow, Action: a, Reason: "allowed by project policy and constitution"}
}
func contains(items []agent.Permission, want agent.Permission) bool {
	for _, v := range items {
		if v == want {
			return true
		}
	}
	return false
}

var ErrInvalidConstitution = errors.New("invalid agent constitution")

func Validate(c Constitution) error {
	if c.ProjectID == "" || c.AgentID == "" || c.Identity == "" || c.Mission == "" {
		return ErrInvalidConstitution
	}
	if len(c.Authority) == 0 || len(c.Restrictions) == 0 || len(c.VerificationRules) == 0 {
		return ErrInvalidConstitution
	}
	return nil
}
