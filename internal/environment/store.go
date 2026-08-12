package environment

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) SaveManifest(ctx context.Context, m Manifest) error {
	if s.DB == nil {
		return errors.New("environment database unavailable")
	}
	raw, _ := json.Marshal(m)
	_, err := s.DB.Exec(ctx, `INSERT INTO environment_manifests(id,project_id,version,base_image,os_name,architecture,profile,decision_id,manifest,fingerprint,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO UPDATE SET manifest=EXCLUDED.manifest,fingerprint=EXCLUDED.fingerprint`, m.ID, m.ProjectID, m.Version, m.BaseImage, m.OS, m.Architecture, m.Profile, m.DecisionID, raw, m.Fingerprint, m.CreatedAt)
	return err
}
