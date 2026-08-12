package deployment

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) SaveService(ctx context.Context, service Service) error {
	if s.DB == nil {
		return errors.New("deployment database unavailable")
	}
	meta, _ := json.Marshal(service.Metadata)
	_, err := s.DB.Exec(ctx, `INSERT INTO deployment_services(id,project_id,name,runtime_id,process_identity,protocol,host,port,health_endpoint,status,environment,metadata,created_at,updated_at) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,NULLIF($9,''),$10,$11,$12,$13,$14) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,port=EXCLUDED.port,health_endpoint=EXCLUDED.health_endpoint,metadata=EXCLUDED.metadata,updated_at=EXCLUDED.updated_at`, service.ID, service.ProjectID, service.Name, service.RuntimeID, service.ProcessIdentity, service.Protocol, service.Host, service.Port, service.HealthEndpoint, service.Status, service.Environment, meta, service.CreatedAt, service.UpdatedAt)
	return err
}
func (s Store) Services(ctx context.Context, projectID string) ([]Service, error) {
	if s.DB == nil {
		return nil, errors.New("deployment database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,name,runtime_id,COALESCE(process_identity,''),protocol,host,port,COALESCE(health_endpoint,''),status,environment,metadata,created_at,updated_at FROM deployment_services WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Service{}
	for rows.Next() {
		var item Service
		var meta []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.RuntimeID, &item.ProcessIdentity, &item.Protocol, &item.Host, &item.Port, &item.HealthEndpoint, &item.Status, &item.Environment, &meta, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &item.Metadata)
		out = append(out, item)
	}
	return out, rows.Err()
}
func (s Store) SavePort(ctx context.Context, binding PortBinding) error {
	if s.DB == nil {
		return errors.New("deployment database unavailable")
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO deployment_ports(id,project_id,runtime_id,service_id,internal_port,external_port,protocol,exposure_mode,status,health,created_at,updated_at) VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,health=EXCLUDED.health,updated_at=EXCLUDED.updated_at`, binding.ID, binding.ProjectID, binding.RuntimeID, binding.ServiceID, binding.InternalPort, binding.ExternalPort, binding.Protocol, binding.Exposure, binding.Status, binding.Health, binding.CreatedAt, binding.UpdatedAt)
	return err
}
func (s Store) SaveRelease(ctx context.Context, release Release) error {
	if s.DB == nil {
		return errors.New("deployment database unavailable")
	}
	artifacts, _ := json.Marshal(release.ArtifactReferences)
	metadata, _ := json.Marshal(release.BuildMetadata)
	_, err := s.DB.Exec(ctx, `INSERT INTO deployment_releases(id,project_id,git_revision,artifact_references,build_metadata,environment,configuration_version,verification_run_id,status,created_at) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),$9,$10) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,build_metadata=EXCLUDED.build_metadata`, release.ID, release.ProjectID, release.GitRevision, artifacts, metadata, release.Environment, release.ConfigurationVersion, release.VerificationRunID, release.Status, release.CreatedAt)
	return err
}
func (s Store) SaveDeployment(ctx context.Context, d Deployment) error {
	if s.DB == nil {
		return errors.New("deployment database unavailable")
	}
	metadata, _ := json.Marshal(d.Metadata)
	_, err := s.DB.Exec(ctx, `INSERT INTO deployments(id,project_id,environment,version,source_revision,runtime_target,release_id,strategy,status,health_status,failure_reason,metadata,created_at,started_at,completed_at) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,NULLIF($10,''),NULLIF($11,''),$12,$13,$14,$15) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,health_status=EXCLUDED.health_status,failure_reason=EXCLUDED.failure_reason,metadata=EXCLUDED.metadata,started_at=EXCLUDED.started_at,completed_at=EXCLUDED.completed_at`, d.ID, d.ProjectID, d.Environment, d.Version, d.SourceRevision, d.RuntimeTarget, d.ReleaseID, d.Strategy, d.Status, d.HealthStatus, d.FailureReason, metadata, d.CreatedAt, d.StartedAt, d.CompletedAt)
	return err
}
func (s Store) Deployments(ctx context.Context, projectID string) ([]Deployment, error) {
	if s.DB == nil {
		return nil, errors.New("deployment database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,environment,version,source_revision,runtime_target,COALESCE(release_id,''),strategy,status,COALESCE(health_status,''),COALESCE(failure_reason,''),metadata,created_at,started_at,completed_at FROM deployments WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Deployment{}
	for rows.Next() {
		var d Deployment
		var metadata []byte
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Environment, &d.Version, &d.SourceRevision, &d.RuntimeTarget, &d.ReleaseID, &d.Strategy, &d.Status, &d.HealthStatus, &d.FailureReason, &metadata, &d.CreatedAt, &d.StartedAt, &d.CompletedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &d.Metadata)
		out = append(out, d)
	}
	return out, rows.Err()
}
