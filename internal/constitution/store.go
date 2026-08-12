package constitution

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) Save(ctx context.Context, c Constitution) error {
	if s.DB == nil {
		return errors.New("constitution database unavailable")
	}
	authority, _ := json.Marshal(c.Authority)
	restrictions, _ := json.Marshal(c.Restrictions)
	decisions, _ := json.Marshal(c.DecisionRules)
	tools, _ := json.Marshal(c.ToolRules)
	verification, _ := json.Marshal(c.VerificationRules)
	security, _ := json.Marshal(c.SecurityRules)
	collaboration, _ := json.Marshal(c.CollaborationRules)
	policy, _ := json.Marshal(c.Policy)
	_, err := s.DB.Exec(ctx, `INSERT INTO agent_constitutions(id,project_id,agent_id,identity,mission,authority,restrictions,decision_rules,tool_rules,verification_rules,security_rules,collaboration_rules,policy) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(project_id,agent_id) DO UPDATE SET identity=EXCLUDED.identity,mission=EXCLUDED.mission,authority=EXCLUDED.authority,restrictions=EXCLUDED.restrictions,decision_rules=EXCLUDED.decision_rules,tool_rules=EXCLUDED.tool_rules,verification_rules=EXCLUDED.verification_rules,security_rules=EXCLUDED.security_rules,collaboration_rules=EXCLUDED.collaboration_rules,policy=EXCLUDED.policy,updated_at=now()`, string(c.ID), c.ProjectID, string(c.AgentID), c.Identity, c.Mission, authority, restrictions, decisions, tools, verification, security, collaboration, policy)
	return err
}
func (s PostgresStore) Get(ctx context.Context, projectID, agentID string) (Constitution, error) {
	if s.DB == nil {
		return Constitution{}, errors.New("constitution database unavailable")
	}
	var c Constitution
	var authority, restrictions, decisions, tools, verification, security, collaboration, policy []byte
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,agent_id,identity,mission,authority,restrictions,decision_rules,tool_rules,verification_rules,security_rules,collaboration_rules,policy FROM agent_constitutions WHERE project_id=$1 AND agent_id=$2`, projectID, agentID).Scan(&c.ID, &c.ProjectID, &c.AgentID, &c.Identity, &c.Mission, &authority, &restrictions, &decisions, &tools, &verification, &security, &collaboration, &policy)
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal(authority, &c.Authority)
	_ = json.Unmarshal(restrictions, &c.Restrictions)
	_ = json.Unmarshal(decisions, &c.DecisionRules)
	_ = json.Unmarshal(tools, &c.ToolRules)
	_ = json.Unmarshal(verification, &c.VerificationRules)
	_ = json.Unmarshal(security, &c.SecurityRules)
	_ = json.Unmarshal(collaboration, &c.CollaborationRules)
	_ = json.Unmarshal(policy, &c.Policy)
	return c, nil
}
