package orchestration

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (s PostgresStore) GetApproval(ctx context.Context, id ID) (Approval, error) {
	var a Approval
	var status string
	err := s.DB.QueryRow(ctx, `SELECT id,workflow_id,task_id,permission,status,requested_at,resolved_at FROM task_approvals WHERE id=$1`, id).Scan(&a.ID, &a.WorkflowID, &a.TaskID, &a.Permission, &status, &a.RequestedAt, &a.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Approval{}, errors.New("approval not found")
	}
	if err != nil {
		return Approval{}, err
	}
	a.Status = status
	return a, nil
}
