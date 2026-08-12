package main

import (
	"suda-forge/internal/events"
	"suda-forge/internal/provisioning"
)

type provisioningEventSink struct{ Bus *events.Bus }

func (s provisioningEventSink) Publish(e provisioning.Event) {
	if s.Bus != nil {
		s.Bus.Publish(events.Event{Type: e.Type, ProjectID: e.ProjectID, Data: e})
	}
}
