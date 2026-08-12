package agent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("agent session not found")

type Store interface {
	CreateSession(context.Context, Session) error
	GetSession(context.Context, string, SessionID) (Session, error)
	ListSessions(context.Context, string) ([]Session, error)
	UpdateSession(context.Context, Session) error
	AppendEvent(context.Context, Event) error
	ListEvents(context.Context, string, SessionID) ([]Event, error)
}
type PostgresStore struct{ DB *pgxpool.Pool }

func (s PostgresStore) CreateSession(ctx context.Context, session Session) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO agent_sessions (id,project_id,agent_id,configuration_id,provider_id,model_id,runtime_id,working_directory,status,created_at,updated_at) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7,$8,$9,$10,$11)`, session.ID, session.ProjectID, session.AgentID, session.Model.ConfigurationID, session.Model.ProviderID, session.Model.ModelID, session.RuntimeID, session.WorkingDirectory, session.Status, session.CreatedAt, session.UpdatedAt)
	return err
}
func (s PostgresStore) GetSession(ctx context.Context, projectID string, id SessionID) (Session, error) {
	var out Session
	var status string
	var agentID, providerID, modelID, configurationID string
	err := s.DB.QueryRow(ctx, `SELECT id,project_id,agent_id,COALESCE(configuration_id,''),COALESCE(provider_id,''),COALESCE(model_id,''),runtime_id,working_directory,status,created_at,updated_at FROM agent_sessions WHERE project_id=$1 AND id=$2`, projectID, id).Scan(&out.ID, &out.ProjectID, &agentID, &configurationID, &providerID, &modelID, &out.RuntimeID, &out.WorkingDirectory, &status, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}
	out.AgentID = ID(agentID)
	out.Model = ModelReference{ConfigurationID: configurationID, ProviderID: providerID, ModelID: modelID}
	out.Status = SessionStatus(status)
	return out, nil
}
func (s PostgresStore) ListSessions(ctx context.Context, projectID string) ([]Session, error) {
	rows, err := s.DB.Query(ctx, `SELECT id,project_id,agent_id,COALESCE(configuration_id,''),COALESCE(provider_id,''),COALESCE(model_id,''),runtime_id,working_directory,status,created_at,updated_at FROM agent_sessions WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var item Session
		var status, agentID, configurationID, providerID, modelID string
		if err := rows.Scan(&item.ID, &item.ProjectID, &agentID, &configurationID, &providerID, &modelID, &item.RuntimeID, &item.WorkingDirectory, &status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.AgentID = ID(agentID)
		item.Model = ModelReference{ConfigurationID: configurationID, ProviderID: providerID, ModelID: modelID}
		item.Status = SessionStatus(status)
		out = append(out, item)
	}
	return out, rows.Err()
}
func (s PostgresStore) UpdateSession(ctx context.Context, session Session) error {
	tag, err := s.DB.Exec(ctx, `UPDATE agent_sessions SET status=$3,updated_at=$4 WHERE project_id=$1 AND id=$2`, session.ProjectID, session.ID, session.Status, session.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}
func (s PostgresStore) AppendEvent(ctx context.Context, event Event) error {
	normalized, _ := json.Marshal(event.Normalized)
	raw, _ := json.Marshal(event.Raw)
	usage, _ := json.Marshal(event.Usage)
	_, err := s.DB.Exec(ctx, `INSERT INTO agent_session_events (id,session_id,event_type,normalized,raw,usage,requires_approval,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, event.ID, event.SessionID, event.Type, normalized, raw, usage, event.RequiresApproval, event.Timestamp)
	return err
}
func (s PostgresStore) ListEvents(ctx context.Context, projectID string, sessionID SessionID) ([]Event, error) {
	rows, err := s.DB.Query(ctx, `SELECT e.id,e.session_id,e.event_type,e.normalized,e.raw,e.usage,e.requires_approval,e.created_at FROM agent_session_events e JOIN agent_sessions s ON s.id=e.session_id WHERE s.project_id=$1 AND e.session_id=$2 ORDER BY e.created_at ASC`, projectID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var typ string
		var normalized, raw, usage []byte
		if err := rows.Scan(&e.ID, &e.SessionID, &typ, &normalized, &raw, &usage, &e.RequiresApproval, &e.Timestamp); err != nil {
			return nil, err
		}
		e.Type = EventType(typ)
		_ = json.Unmarshal(normalized, &e.Normalized)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &e.Raw)
		}
		if string(usage) != "null" && len(usage) > 0 {
			e.Usage = &Usage{}
			_ = json.Unmarshal(usage, e.Usage)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func NewSession(projectID string, agentID ID, model ModelReference, runtimeID, workingDirectory string, now time.Time) Session {
	return Session{ID: SessionID("sess_" + now.UTC().Format("20060102150405.000000000")), ProjectID: projectID, AgentID: agentID, Model: model, RuntimeID: runtimeID, WorkingDirectory: workingDirectory, Status: SessionCreated, CreatedAt: now, UpdatedAt: now}
}
