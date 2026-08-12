package main

import (
	"context"
	"fmt"
	"suda-forge/internal/agent"
	"suda-forge/internal/constitution"
)

type constitutionGuard struct {
	Constitutions map[string]constitution.Constitution
	Store         constitution.PostgresStore
}

func (g constitutionGuard) Authorize(ctx context.Context, session agent.Session, permission agent.Permission) error {
	key := session.ProjectID + ":" + string(session.AgentID)
	c, ok := g.Constitutions[key]
	if !ok {
		loaded, err := g.Store.Get(ctx, session.ProjectID, string(session.AgentID))
		if err != nil {
			return fmt.Errorf("governance constitution unavailable: %w", err)
		}
		c = loaded
		g.Constitutions[key] = c
	}
	risk := constitution.RiskExecute
	if permission == agent.PermissionFilesystemRead {
		risk = constitution.RiskRead
	}
	d := constitution.PolicyEvaluator{}.Evaluate(c, constitution.Action{ProjectID: session.ProjectID, AgentID: session.AgentID, Permission: permission, Risk: risk, Description: "start agent execution"})
	if d.Effect != constitution.Allow {
		return fmt.Errorf("governance %s: %s", d.Effect, d.Reason)
	}
	return nil
}
