package deployment

import "suda-forge/internal/events"

type BusSink struct{ Bus *events.Bus }

func (s BusSink) Publish(kind, projectID string, data any) {
	if s.Bus != nil {
		s.Bus.Publish(events.Event{Type: kind, ProjectID: projectID, Data: data})
	}
}
