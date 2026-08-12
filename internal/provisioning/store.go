package provisioning

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) Save(ctx context.Context, r Run) error {
	if s.DB == nil {
		return errors.New("provisioning database unavailable")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO provisioning_runs(id,project_id,runtime_id,manifest_id,status,last_successful_step,error,started_at,completed_at,cancel_requested,run) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO UPDATE SET runtime_id=EXCLUDED.runtime_id,status=EXCLUDED.status,last_successful_step=EXCLUDED.last_successful_step,error=EXCLUDED.error,completed_at=EXCLUDED.completed_at,cancel_requested=EXCLUDED.cancel_requested,run=EXCLUDED.run`, string(r.ID), r.ProjectID, r.RuntimeID, r.Manifest.ID, string(r.Status), r.LastSuccessfulStep, r.Error, r.StartedAt, r.CompletedAt, r.CancelRequested, raw)
	return err
}
func (s PostgresStore) Get(ctx context.Context, id ID) (Run, error) {
	if s.DB == nil {
		return Run{}, errors.New("provisioning database unavailable")
	}
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT run FROM provisioning_runs WHERE id=$1`, string(id)).Scan(&raw)
	if err != nil {
		return Run{}, err
	}
	var r Run
	err = json.Unmarshal(raw, &r)
	return r, err
}
func (s PostgresStore) List(ctx context.Context, projectID string) ([]Run, error) {
	if s.DB == nil {
		return nil, errors.New("provisioning database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT run FROM provisioning_runs WHERE project_id=$1 ORDER BY started_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r Run
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
