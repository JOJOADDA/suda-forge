package audit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID           string         `json:"id"`
	ActorUserID  *string        `json:"actor_user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	ProjectID    *string        `json:"project_id,omitempty"`
	Outcome      string         `json:"outcome"`
	Metadata     map[string]any `json:"metadata"`
	IPAddress    *string        `json:"ip_address,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Store interface {
	ListProject(ctx context.Context, projectID string, limit int) ([]Event, error)
}

type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) ListProject(ctx context.Context, projectID string, limit int) ([]Event, error) {
	if s.DB == nil {
		return nil, errors.New("audit database unavailable")
	}
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.DB.Query(ctx, `
		SELECT id, actor_user_id, action, resource_type, resource_id, project_id, outcome, metadata, ip_address, created_at
		FROM audit_events WHERE project_id=$1 ORDER BY created_at DESC LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Event, 0)
	for rows.Next() {
		var item Event
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ActorUserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.ProjectID, &item.Outcome, &metadata, &item.IPAddress, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
