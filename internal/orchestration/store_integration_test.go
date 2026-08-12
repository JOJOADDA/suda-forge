//go:build integration

package orchestration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkflowStoreRoundTrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var project string
	if err = db.QueryRow(context.Background(), `SELECT id FROM projects ORDER BY created_at DESC LIMIT 1`).Scan(&project); err != nil {
		t.Fatal(err)
	}
	w, err := Orchestrator{Planner: DeterministicPlanner{}, Now: time.Now}.Plan(PlannerInput{Intent: UserIntent{ProjectID: project, Goal: "integration workflow"}})
	if err != nil {
		t.Fatal(err)
	}
	store := PostgresStore{DB: db}
	if err = store.SaveWorkflow(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetWorkflow(context.Background(), project, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Graph.Tasks) != 3 || len(loaded.Graph.Dependencies) == 0 {
		t.Fatalf("loaded graph=%+v", loaded.Graph)
	}
}
