package deployment

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type PostgresPortRegistry struct{ DB *pgxpool.Pool }

func (r PostgresPortRegistry) Reserve(ctx context.Context, b PortBinding) (PortBinding, error) {
	if r.DB == nil {
		return PortBinding{}, errors.New("deployment port database unavailable")
	}
	var owner string
	if err := r.DB.QueryRow(ctx, `SELECT id FROM deployment_port_bindings WHERE protocol=$1 AND external_port=$2`, b.Protocol, b.ExternalPort).Scan(&owner); err == nil && owner != "" && ID(owner) != b.ID {
		return PortBinding{}, ErrPortConflict
	}
	now := time.Now().UTC()
	if b.ID == "" {
		b.ID = ID(fmt.Sprintf("port_%s_%d", b.ProjectID, b.ExternalPort))
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	b.Status = "RESERVED"
	_, err := r.DB.Exec(ctx, `INSERT INTO deployment_port_bindings(id,project_id,runtime_id,service_id,internal_port,external_port,protocol,exposure,status,health,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(protocol,external_port) DO UPDATE SET project_id=EXCLUDED.project_id,runtime_id=EXCLUDED.runtime_id,service_id=EXCLUDED.service_id,internal_port=EXCLUDED.internal_port,exposure=EXCLUDED.exposure,status=EXCLUDED.status,health=EXCLUDED.health,updated_at=EXCLUDED.updated_at WHERE deployment_port_bindings.id=EXCLUDED.id`, string(b.ID), b.ProjectID, b.RuntimeID, b.ServiceID, b.InternalPort, b.ExternalPort, b.Protocol, string(b.Exposure), b.Status, b.Health, b.CreatedAt, b.UpdatedAt)
	if err != nil {
		return PortBinding{}, fmt.Errorf("reserve port: %w", err)
	}
	return b, nil
}
func (r PostgresPortRegistry) Release(ctx context.Context, id ID) error {
	if r.DB == nil {
		return errors.New("deployment port database unavailable")
	}
	tag, err := r.DB.Exec(ctx, `DELETE FROM deployment_port_bindings WHERE id=$1`, string(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("port binding not found")
	}
	return nil
}
func (r PostgresPortRegistry) List(ctx context.Context, projectID string) ([]PortBinding, error) {
	if r.DB == nil {
		return nil, errors.New("deployment port database unavailable")
	}
	rows, err := r.DB.Query(ctx, `SELECT id,project_id,runtime_id,service_id,internal_port,external_port,protocol,exposure,status,health,created_at,updated_at FROM deployment_port_bindings WHERE project_id=$1 ORDER BY external_port`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PortBinding{}
	for rows.Next() {
		var b PortBinding
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.RuntimeID, &b.ServiceID, &b.InternalPort, &b.ExternalPort, &b.Protocol, &b.Exposure, &b.Status, &b.Health, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

var _ PortRegistry = PostgresPortRegistry{}
