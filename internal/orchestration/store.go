package orchestration

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

var ErrWorkflowNotFound = errors.New("workflow not found")

func (s PostgresStore) SaveWorkflow(ctx context.Context, workflow Workflow) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO workflows (id,project_id,goal,status,execution_strategy,failure_policy,max_parallel_tasks,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,updated_at=EXCLUDED.updated_at`, workflow.ID, workflow.ProjectID, workflow.Goal, workflow.Status, workflow.Strategy, workflow.FailurePolicy, workflow.MaxParallel, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return err
	}
	for _, id := range SortedTaskIDs(workflow.Graph) {
		task := workflow.Graph.Tasks[id]
		rawRouting, _ := json.Marshal(task.RoutingRequest)
		rawConstraints, _ := json.Marshal(task.Constraints)
		rawRetry, _ := json.Marshal(task.Retry)
		_, err = s.DB.Exec(ctx, `INSERT INTO orchestration_tasks (id,workflow_id,project_id,parent_task_id,title,description,task_type,priority,status,assigned_agent,routing_request,constraints,retry_policy,deadline,created_at,started_at,completed_at) VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,NULLIF($10,''),$11,$12,$13,$14,$15,$16,$17) ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,assigned_agent=EXCLUDED.assigned_agent,started_at=EXCLUDED.started_at,completed_at=EXCLUDED.completed_at`, task.ID, workflow.ID, task.ProjectID, task.ParentTaskID, task.Title, task.Description, task.TaskType, task.Priority, task.Status, task.AssignedAgent, rawRouting, rawConstraints, rawRetry, task.Deadline, task.CreatedAt, task.StartedAt, task.CompletedAt)
		if err != nil {
			return err
		}
		for _, dep := range workflow.Graph.Dependencies[id] {
			if _, err = s.DB.Exec(ctx, `INSERT INTO task_dependencies (task_id,depends_on_task_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, dep); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s PostgresStore) GetWorkflow(ctx context.Context, projectID string, id ID) (Workflow, error) {
	var w Workflow
	var status, strategy, failure string
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,goal,status,execution_strategy,failure_policy,max_parallel_tasks,created_at,updated_at FROM workflows WHERE project_id=$1 AND id=$2`, projectID, id).Scan(&w.ID, &w.ProjectID, &w.Goal, &status, &strategy, &failure, &w.MaxParallel, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workflow{}, ErrWorkflowNotFound
	}
	if err != nil {
		return Workflow{}, err
	}
	w.Status = WorkflowStatus(status)
	w.Strategy = ExecutionStrategy(strategy)
	w.FailurePolicy = FailurePolicy(failure)
	w.Graph = TaskGraph{Tasks: map[ID]Task{}, Dependencies: map[ID][]ID{}}
	rows, err := s.DB.Query(ctx, `SELECT id,workflow_id,project_id,COALESCE(parent_task_id,''),title,description,task_type,priority,status,COALESCE(assigned_agent,''),deadline,created_at,started_at,completed_at FROM orchestration_tasks WHERE workflow_id=$1 ORDER BY id`, id)
	if err != nil {
		return Workflow{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var task Task
		var status string
		if err := rows.Scan(&task.ID, &task.WorkflowID, &task.ProjectID, &task.ParentTaskID, &task.Title, &task.Description, &task.TaskType, &task.Priority, &status, &task.AssignedAgent, &task.Deadline, &task.CreatedAt, &task.StartedAt, &task.CompletedAt); err != nil {
			return Workflow{}, err
		}
		task.Status = TaskStatus(status)
		w.Graph.Tasks[task.ID] = task
		w.Graph.Dependencies[task.ID] = []ID{}
	}
	if err := rows.Err(); err != nil {
		return Workflow{}, err
	}
	depRows, err := s.DB.Query(ctx, `SELECT task_id, depends_on_task_id FROM task_dependencies WHERE task_id IN (SELECT id FROM orchestration_tasks WHERE workflow_id=$1) ORDER BY task_id, depends_on_task_id`, id)
	if err != nil {
		return Workflow{}, err
	}
	defer depRows.Close()
	for depRows.Next() {
		var taskID, dependency ID
		if err := depRows.Scan(&taskID, &dependency); err != nil {
			return Workflow{}, err
		}
		w.Graph.Dependencies[taskID] = append(w.Graph.Dependencies[taskID], dependency)
		task := w.Graph.Tasks[taskID]
		task.Dependencies = append(task.Dependencies, dependency)
		w.Graph.Tasks[taskID] = task

	}
	return w, depRows.Err()
}
