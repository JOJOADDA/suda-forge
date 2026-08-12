//go:build integration

package agent

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAgentSessionAndEventsIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := PostgresStore{DB: db}
	now := time.Now().UTC().Truncate(time.Microsecond)
	session := NewSession("proj_1786535793573152374", "codex", ModelReference{}, "runtime-integration", "/workspace", now)
	session.ID = SessionID("integration_" + now.Format("20060102150405.000000000"))
	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(context.Background(), "DELETE FROM agent_sessions WHERE id=$1", session.ID) })
	if err := store.AppendEvent(ctx, Event{ID: "evt_" + string(session.ID), SessionID: session.ID, Type: EventMessage, Timestamp: now, Normalized: map[string]any{"text": "hello"}, Raw: map[string]any{"provider": "test"}}); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetSession(ctx, session.ProjectID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != SessionCreated {
		t.Fatalf("status = %s", got.Status)
	}
	events, err := store.ListEvents(ctx, session.ProjectID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != EventMessage {
		t.Fatalf("events = %+v", events)
	}
}
