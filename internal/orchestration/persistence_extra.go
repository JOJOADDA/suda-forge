package orchestration

import (
	"context"
	"encoding/json"
	"time"
)

func (s PostgresStore) SaveTaskRun(ctx context.Context, run TaskRun) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO task_runs (id,task_id,attempt,status,agent_id,model_id,started_at,completed_at,error) VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7,$8,NULLIF($9,'')) ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,completed_at=EXCLUDED.completed_at,error=EXCLUDED.error`, run.ID, run.TaskID, run.Attempt, run.Status, run.AgentID, run.ModelID, run.StartedAt, run.CompletedAt, run.Error)
	return err
}
func (s PostgresStore) SaveArtifact(ctx context.Context, artifact TaskArtifact) error {
	meta, _ := json.Marshal(artifact.Metadata)
	_, err := s.DB.Exec(ctx, `INSERT INTO task_artifacts (id,project_id,task_id,task_run_id,kind,path,commit,metadata) VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),$8)`, artifact.ID, artifact.ProjectID, artifact.TaskID, artifact.TaskRunID, artifact.Kind, artifact.Path, artifact.Commit, meta)
	return err
}
func (s PostgresStore) SaveApproval(ctx context.Context, approval Approval) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO task_approvals (id,workflow_id,task_id,permission,status,requested_at,resolved_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,resolved_at=EXCLUDED.resolved_at`, approval.ID, approval.WorkflowID, approval.TaskID, approval.Permission, approval.Status, approval.RequestedAt, approval.ResolvedAt)
	return err
}
func (s PostgresStore) SaveEvent(ctx context.Context, event TaskEvent) error {
	data, _ := json.Marshal(event.Data)
	var task any
	if event.TaskID != "" {
		task = event.TaskID
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO task_events (id,workflow_id,task_id,event_type,data,created_at) VALUES ($1,$2,$3,$4,$5,$6)`, event.ID, event.WorkflowID, task, event.Type, data, event.CreatedAt)
	return err
}
func NewTaskEvent(workflow ID, task ID, typ string, data map[string]any) TaskEvent {
	return TaskEvent{ID: ID("evt_" + time.Now().UTC().Format("20060102150405.000000000")), WorkflowID: workflow, TaskID: task, Type: typ, Data: data, CreatedAt: time.Now().UTC()}
}
