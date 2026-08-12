package projectcomputer

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) Save(ctx context.Context, c ProjectComputer) error {
	if s.DB == nil {
		return errors.New("project computer database unavailable")
	}
	raw, _ := json.Marshal(c)
	_, err := s.DB.Exec(ctx, `INSERT INTO project_computers(id,project_id,runtime_provider,runtime_id,image,image_version,status,resources,environment_fingerprint,readiness,capabilities,created_at,updated_at,metadata) VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(id) DO UPDATE SET runtime_id=EXCLUDED.runtime_id,status=EXCLUDED.status,environment_fingerprint=EXCLUDED.environment_fingerprint,readiness=EXCLUDED.readiness,capabilities=EXCLUDED.capabilities,updated_at=EXCLUDED.updated_at,metadata=EXCLUDED.metadata`, string(c.ID), c.ProjectID, c.RuntimeProvider, c.RuntimeID, c.Image, c.ImageVersion, string(c.Status), raw, c.EnvironmentFingerprint, c.Readiness, c.Capabilities, c.CreatedAt, c.UpdatedAt, c.Metadata)
	return err
}
func (s PostgresStore) Get(ctx context.Context, id ID) (ProjectComputer, error) {
	if s.DB == nil {
		return ProjectComputer{}, errors.New("project computer database unavailable")
	}
	var c ProjectComputer
	var resources, readiness, capabilities, metadata []byte
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,runtime_provider,COALESCE(runtime_id,''),image,image_version,status,resources,environment_fingerprint,readiness,capabilities,created_at,updated_at,metadata FROM project_computers WHERE id=$1`, string(id)).Scan(&c.ID, &c.ProjectID, &c.RuntimeProvider, &c.RuntimeID, &c.Image, &c.ImageVersion, &c.Status, &resources, &c.EnvironmentFingerprint, &readiness, &capabilities, &c.CreatedAt, &c.UpdatedAt, &metadata)
	if err != nil {
		return ProjectComputer{}, err
	}
	_ = json.Unmarshal(resources, &c.Resources)
	_ = json.Unmarshal(readiness, &c.Readiness)
	_ = json.Unmarshal(capabilities, &c.Capabilities)
	_ = json.Unmarshal(metadata, &c.Metadata)
	return c, nil
}
func (s PostgresStore) List(ctx context.Context, projectID string) ([]ProjectComputer, error) {
	if s.DB == nil {
		return nil, errors.New("project computer database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,runtime_provider,COALESCE(runtime_id,''),image,image_version,status,resources,environment_fingerprint,readiness,capabilities,created_at,updated_at,metadata FROM project_computers WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ProjectComputer{}
	for rows.Next() {
		var c ProjectComputer
		var resources, readiness, capabilities, metadata []byte
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.RuntimeProvider, &c.RuntimeID, &c.Image, &c.ImageVersion, &c.Status, &resources, &c.EnvironmentFingerprint, &readiness, &capabilities, &c.CreatedAt, &c.UpdatedAt, &metadata); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(resources, &c.Resources)
		_ = json.Unmarshal(readiness, &c.Readiness)
		_ = json.Unmarshal(capabilities, &c.Capabilities)
		_ = json.Unmarshal(metadata, &c.Metadata)
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s PostgresStore) SaveOperation(ctx context.Context, o OperationRecord) error {
	if s.DB == nil {
		return errors.New("project computer database unavailable")
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO project_computer_operations(id,computer_id,project_id,operation,status,run_id,error,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,error=EXCLUDED.error,completed_at=EXCLUDED.completed_at`, o.ID, string(o.ComputerID), o.ProjectID, string(o.Operation), o.Status, o.RunID, o.Error, o.StartedAt, o.CompletedAt)
	return err
}
