package deployment

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/internal/events"
)

type CompositeAuditSink struct {
	Bus *events.Bus
	DB  *pgxpool.Pool
}

func (s CompositeAuditSink) Publish(kind, projectID string, data any) {
	if s.Bus != nil {
		s.Bus.Publish(events.Event{Type: kind, ProjectID: projectID, Data: data})
	}
	if s.DB != nil {
		payload, _ := json.Marshal(data)
		_, _ = s.DB.Exec(context.Background(), `INSERT INTO deployment_events(project_id,type,payload) VALUES($1,$2,$3)`, projectID, kind, payload)
	}
}
