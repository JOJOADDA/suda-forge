package productexperience

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) SaveActivity(ctx context.Context, a Activity) error {
	if s.DB == nil {
		return errors.New("product experience database unavailable")
	}
	data, _ := json.Marshal(a.Data)
	_, err := s.DB.Exec(ctx, `INSERT INTO product_activity(id,project_id,event_type,state,message,progress,data,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET state=EXCLUDED.state,message=EXCLUDED.message,progress=EXCLUDED.progress,data=EXCLUDED.data`, a.ID, a.ProjectID, a.Type, string(a.State), a.Message, a.Progress, data, a.Timestamp)
	return err
}
func (s PostgresStore) SaveContext(ctx context.Context, id string, c AgentContext) error {
	if s.DB == nil {
		return errors.New("product experience database unavailable")
	}
	raw, _ := json.Marshal(c)
	_, err := s.DB.Exec(ctx, `INSERT INTO product_context_snapshots(id,project_id,agent_id,task,context) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET context=EXCLUDED.context`, id, c.ProjectPolicy.ProjectID, string(c.ProjectPolicy.AgentID), c.Task, raw)
	return err
}
func (s PostgresStore) SaveImpact(ctx context.Context, id string, a ImpactAnalysis) error {
	if s.DB == nil {
		return errors.New("product experience database unavailable")
	}
	files, _ := json.Marshal(a.Files)
	apis, _ := json.Marshal(a.APIs)
	components, _ := json.Marshal(a.Components)
	tests, _ := json.Marshal(a.Tests)
	nodes, _ := json.Marshal(a.Nodes)
	reasons, _ := json.Marshal(a.Reasons)
	_, err := s.DB.Exec(ctx, `INSERT INTO impact_analyses(id,project_id,root_node,risk,files,apis,components,tests,affected_nodes,approval_required,reasons) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, a.ProjectID, string(a.RootNode), a.Risk, files, apis, components, tests, nodes, a.ApprovalRequired, reasons)
	return err
}
func (s PostgresStore) SaveLoop(ctx context.Context, l LoopPlan) error {
	if s.DB == nil {
		return errors.New("product experience database unavailable")
	}
	stages, _ := json.Marshal(l.Stages)
	delegates, _ := json.Marshal(l.Delegates)
	blocked, _ := json.Marshal(l.BlockedStages)
	_, err := s.DB.Exec(ctx, `INSERT INTO autonomous_loop_plans(id,project_id,stages,delegates,status,blocked_stages,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO UPDATE SET stages=EXCLUDED.stages,delegates=EXCLUDED.delegates,status=EXCLUDED.status,blocked_stages=EXCLUDED.blocked_stages`, l.ID, l.ProjectID, stages, delegates, l.Status, blocked, l.CreatedAt)
	return err
}
