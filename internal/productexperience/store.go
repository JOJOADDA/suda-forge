package productexperience

import (
	"context"
	"encoding/json"
	"errors"
	"time"

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
	execution := LoopExecution{LoopPlan: l, Results: map[LoopStage]LoopStageResult{}, UpdatedAt: l.CreatedAt}
	return s.SaveLoopExecution(ctx, execution)
}
func (s PostgresStore) SaveLoopExecution(ctx context.Context, l LoopExecution) error {
	if s.DB == nil {
		return errors.New("product experience database unavailable")
	}
	stages, _ := json.Marshal(l.Stages)
	delegates, _ := json.Marshal(l.Delegates)
	blocked, _ := json.Marshal(l.BlockedStages)
	results, _ := json.Marshal(l.Results)
	created := l.CreatedAt
	if created.IsZero() {
		created = timeNowUTC()
	}
	updated := l.UpdatedAt
	if updated.IsZero() {
		updated = created
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO autonomous_loop_plans(id,project_id,goal,stages,delegates,status,blocked_stages,created_at,current_stage,results,error,updated_at,worker_id,lease_until) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(id) DO UPDATE SET stages=EXCLUDED.stages,delegates=EXCLUDED.delegates,status=EXCLUDED.status,blocked_stages=EXCLUDED.blocked_stages,current_stage=EXCLUDED.current_stage,results=EXCLUDED.results,error=EXCLUDED.error,updated_at=EXCLUDED.updated_at,worker_id=EXCLUDED.worker_id,lease_until=EXCLUDED.lease_until`, l.ID, l.ProjectID, l.Goal, stages, delegates, l.Status, blocked, created, string(l.CurrentStage), results, l.Error, updated, "", nil)
	return err
}
func (s PostgresStore) GetLoopExecution(ctx context.Context, id string) (LoopExecution, error) {
	if s.DB == nil {
		return LoopExecution{}, errors.New("product experience database unavailable")
	}
	var l LoopExecution
	var stages, delegates, blocked, results []byte
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,goal,stages,delegates,status,blocked_stages,created_at,current_stage,results,error,updated_at FROM autonomous_loop_plans WHERE id=$1`, id).Scan(&l.ID, &l.ProjectID, &l.Goal, &stages, &delegates, &l.Status, &blocked, &l.CreatedAt, &l.CurrentStage, &results, &l.Error, &l.UpdatedAt)
	if err != nil {
		return LoopExecution{}, err
	}
	if err := json.Unmarshal(stages, &l.Stages); err != nil {
		return LoopExecution{}, err
	}
	if err := json.Unmarshal(delegates, &l.Delegates); err != nil {
		return LoopExecution{}, err
	}
	if err := json.Unmarshal(blocked, &l.BlockedStages); err != nil {
		return LoopExecution{}, err
	}
	if err := json.Unmarshal(results, &l.Results); err != nil {
		return LoopExecution{}, err
	}
	if l.Results == nil {
		l.Results = map[LoopStage]LoopStageResult{}
	}
	return l, nil
}
func (s PostgresStore) ListRunnableLoopExecutions(ctx context.Context) ([]LoopExecution, error) {
	if s.DB == nil {
		return nil, errors.New("product experience database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id FROM autonomous_loop_plans WHERE status IN ('RUNNING','BLOCKED') ORDER BY updated_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LoopExecution
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		l, err := s.GetLoopExecution(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func timeNowUTC() time.Time { return time.Now().UTC() }
