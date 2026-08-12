package main

import (
	"suda-forge/internal/events"
	"suda-forge/internal/projectcomputer"
)

type projectComputerEventSink struct{ Bus *events.Bus }

func (s projectComputerEventSink) Publish(e projectcomputer.Event) {
	if s.Bus != nil {
		s.Bus.Publish(events.Event{Type: e.Type, ProjectID: e.ProjectID, Data: e})
	}
}
