package sharedinfra

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) SaveTool(ctx context.Context, t Tool) error {
	if s.DB == nil {
		return errors.New("shared infrastructure database unavailable")
	}
	platforms, _ := json.Marshal(t.Platforms)
	deps, _ := json.Marshal(t.Dependencies)
	caps, _ := json.Marshal(t.Capabilities)
	meta, _ := json.Marshal(t.Metadata)
	if _, err := s.DB.Exec(ctx, `INSERT INTO shared_tools(id,name,category,platforms,dependencies,capabilities,install_strategy,verification_strategy,artifact_identity,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,category=EXCLUDED.category,platforms=EXCLUDED.platforms,dependencies=EXCLUDED.dependencies,capabilities=EXCLUDED.capabilities`, t.ID, t.Name, string(t.Category), platforms, deps, caps, t.InstallStrategy, t.VerificationStrategy, t.ArtifactIdentity, meta); err != nil {
		return err
	}
	for _, v := range t.Versions {
		if err := s.SaveVersion(ctx, t.ID, v); err != nil {
			return err
		}
	}
	return nil
}
func (s PostgresStore) SaveVersion(ctx context.Context, toolID string, v ToolVersion) error {
	deps, _ := json.Marshal(v.Dependencies)
	compat, _ := json.Marshal(v.Compatibility)
	_, err := s.DB.Exec(ctx, `INSERT INTO shared_tool_versions(tool_id,version,platform,architecture,dependencies,artifact_id,installation_method,verification_method,compatibility) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(tool_id,version,platform,architecture) DO UPDATE SET artifact_id=EXCLUDED.artifact_id,compatibility=EXCLUDED.compatibility`, toolID, v.Version, v.Platform, v.Architecture, deps, v.Artifact.ID, v.InstallationMethod, v.VerificationMethod, compat)
	return err
}
func (s PostgresStore) SaveArtifact(ctx context.Context, a Artifact) error {
	if s.DB == nil {
		return errors.New("shared infrastructure database unavailable")
	}
	meta, _ := json.Marshal(a.Metadata)
	_, err := s.DB.Exec(ctx, `INSERT INTO shared_artifacts(id,name,version,platform,architecture,size,checksum,source,storage_location,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO UPDATE SET checksum=EXCLUDED.checksum,storage_location=EXCLUDED.storage_location`, a.ID, a.Name, a.Version, a.Platform, a.Architecture, a.Size, a.Checksum, a.Source, a.StorageLocation, meta)
	return err
}
func (s PostgresStore) SaveCacheEntry(ctx context.Context, e CacheEntry) error {
	if s.DB == nil {
		return errors.New("shared infrastructure database unavailable")
	}
	return s.SaveCacheEntryRaw(ctx, e)
}
func (s PostgresStore) SaveCacheEntryRaw(ctx context.Context, e CacheEntry) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO global_cache_entries(artifact_id,status,verified_at,last_used_at,ref_count,storage_location) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(artifact_id) DO UPDATE SET status=EXCLUDED.status,verified_at=EXCLUDED.verified_at,last_used_at=EXCLUDED.last_used_at,ref_count=EXCLUDED.ref_count`, e.Artifact.ID, string(e.Status), e.VerifiedAt, e.LastUsedAt, e.RefCount, e.Artifact.StorageLocation)
	return err
}

func (s PostgresStore) LoadTools(ctx context.Context) ([]Tool, error) {
	if s.DB == nil {
		return nil, errors.New("shared infrastructure database unavailable")
	}
	rows, err := s.DB.Query(ctx, `SELECT id,name,category,platforms,dependencies,capabilities,install_strategy,verification_strategy,artifact_identity,metadata FROM shared_tools ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tool{}
	for rows.Next() {
		var t Tool
		var platforms, deps, caps, meta []byte
		var category string
		if err := rows.Scan(&t.ID, &t.Name, &category, &platforms, &deps, &caps, &t.InstallStrategy, &t.VerificationStrategy, &t.ArtifactIdentity, &meta); err != nil {
			return nil, err
		}
		t.Category = ToolCategory(category)
		_ = json.Unmarshal(platforms, &t.Platforms)
		_ = json.Unmarshal(deps, &t.Dependencies)
		_ = json.Unmarshal(caps, &t.Capabilities)
		_ = json.Unmarshal(meta, &t.Metadata)
		vrows, err := s.DB.Query(ctx, `SELECT version,platform,architecture,dependencies,artifact_id,installation_method,verification_method,compatibility FROM shared_tool_versions WHERE tool_id=$1 ORDER BY version`, t.ID)
		if err != nil {
			return nil, err
		}
		for vrows.Next() {
			var v ToolVersion
			var vd, compat []byte
			var artifactID string
			if err := vrows.Scan(&v.Version, &v.Platform, &v.Architecture, &vd, &artifactID, &v.InstallationMethod, &v.VerificationMethod, &compat); err != nil {
				vrows.Close()
				return nil, err
			}
			_ = json.Unmarshal(vd, &v.Dependencies)
			_ = json.Unmarshal(compat, &v.Compatibility)
			if artifact, ok := loadArtifact(ctx, s.DB, artifactID); ok {
				v.Artifact = artifact
			}
			t.Versions = append(t.Versions, v)
		}
		vrows.Close()
		out = append(out, t)
	}
	return out, rows.Err()
}
func loadArtifact(ctx context.Context, db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (Artifact, bool) {
	if id == "" {
		return Artifact{}, false
	}
	var a Artifact
	var meta []byte
	if err := db.QueryRow(ctx, `SELECT id,name,version,platform,architecture,size,checksum,source,storage_location,metadata FROM shared_artifacts WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.Version, &a.Platform, &a.Architecture, &a.Size, &a.Checksum, &a.Source, &a.StorageLocation, &meta); err != nil {
		return Artifact{}, false
	}
	_ = json.Unmarshal(meta, &a.Metadata)
	return a, true
}
