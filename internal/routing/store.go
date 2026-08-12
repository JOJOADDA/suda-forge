package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) SaveDecision(ctx context.Context, id string, request RoutingRequest, decision RoutingDecision) error {
	task, _ := json.Marshal(request.Task)
	rawRequest, _ := json.Marshal(request)
	rawDecision, _ := json.Marshal(decision)
	_, err := s.DB.Exec(ctx, `INSERT INTO routing_decisions (id,project_id,agent_id,task_profile,request,selected_provider_id,selected_model_id,decision,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, request.ProjectID, request.AgentID, task, rawRequest, decision.ProviderID, decision.Selected.ModelID, rawDecision, time.Now().UTC())
	return err
}
func (s Store) SaveUsage(ctx context.Context, id, sessionID string, usage UsageEvent) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO model_usage_events (id,session_id,provider_id,model_id,event_type,input_tokens,output_tokens,duration_ms,estimated_cost,created_at) VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10)`, id, sessionID, usage.ProviderID, usage.ModelID, usage.Type, usage.InputTokens, usage.OutputTokens, usage.Duration.Milliseconds(), usage.EstimatedCost, time.Now().UTC())
	return err
}
func DecisionID(now time.Time) string { return fmt.Sprintf("routing_%d", now.UnixNano()) }
