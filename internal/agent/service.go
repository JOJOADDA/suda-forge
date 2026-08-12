package agent

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Service struct {
	Store    Store
	Adapters *Registry
	Now      func() time.Time
}

func (s Service) CreateSession(ctx context.Context, session Session) error {
	if session.ProjectID == "" || session.RuntimeID == "" || session.WorkingDirectory == "" {
		return errors.New("project, runtime, and working directory are required")
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if session.Status == "" {
		session.Status = SessionCreated
	}
	if err := s.Store.CreateSession(ctx, session); err != nil {
		return err
	}
	return s.Store.AppendEvent(ctx, Event{ID: fmt.Sprintf("evt_%d", s.Now().UnixNano()), SessionID: session.ID, Type: EventSessionCreated, Timestamp: s.Now().UTC(), Normalized: map[string]any{"project_id": session.ProjectID, "agent_id": session.AgentID}})
}
func (s Service) Start(ctx context.Context, projectID string, id SessionID) (Session, error) {
	session, err := s.Store.GetSession(ctx, projectID, id)
	if err != nil {
		return Session{}, err
	}
	adapter, ok := s.Adapters.Get(string(session.AgentID))
	if !ok {
		return Session{}, fmt.Errorf("agent adapter not registered: %s", session.AgentID)
	}
	now := s.Now()
	if err := session.Transition(SessionStarting, now); err != nil {
		return Session{}, err
	}
	if err := s.Store.UpdateSession(ctx, session); err != nil {
		return Session{}, err
	}
	if err := adapter.Start(ctx, session); err != nil {
		_ = session.Transition(SessionFailed, s.Now())
		_ = s.Store.UpdateSession(ctx, session)
		return Session{}, err
	}
	if err := session.Transition(SessionRunning, s.Now()); err != nil {
		return Session{}, err
	}
	if err := s.Store.UpdateSession(ctx, session); err != nil {
		return Session{}, err
	}
	_ = s.Store.AppendEvent(ctx, Event{ID: fmt.Sprintf("evt_%d", s.Now().UnixNano()), SessionID: id, Type: EventSessionStarted, Timestamp: s.Now().UTC()})
	return session, nil
}
func (s Service) SendMessage(ctx context.Context, projectID string, id SessionID, message string) error {
	session, err := s.Store.GetSession(ctx, projectID, id)
	if err != nil {
		return err
	}
	adapter, ok := s.Adapters.Get(string(session.AgentID))
	if !ok {
		return errors.New("agent adapter not registered")
	}
	return adapter.SendMessage(ctx, id, message)
}
func (s Service) Cancel(ctx context.Context, projectID string, id SessionID) (Session, error) {
	session, err := s.Store.GetSession(ctx, projectID, id)
	if err != nil {
		return Session{}, err
	}
	adapter, ok := s.Adapters.Get(string(session.AgentID))
	if !ok {
		return Session{}, errors.New("agent adapter not registered")
	}
	if err := adapter.Cancel(ctx, id); err != nil {
		return Session{}, err
	}
	if err := session.Transition(SessionCancelled, s.Now()); err != nil {
		return Session{}, err
	}
	if err := s.Store.UpdateSession(ctx, session); err != nil {
		return Session{}, err
	}
	_ = s.Store.AppendEvent(ctx, Event{ID: fmt.Sprintf("evt_%d", s.Now().UnixNano()), SessionID: id, Type: EventSessionCompleted, Timestamp: s.Now().UTC(), Normalized: map[string]any{"status": SessionCancelled}})
	return session, nil
}
