//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/domain/project"
)

func TestProjectsCRUDIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	repo := Projects{DB: db}
	id := project.ID("integration_" + time.Now().UTC().Format("20060102150405.000000000"))
	now := time.Now().UTC().Truncate(time.Microsecond)
	p := project.Project{ID: id, Name: "Integration", Slug: "integration", Status: project.StatusCreating, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(context.Background(), "DELETE FROM projects WHERE id=$1", id) })
	got, err := repo.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != project.StatusCreating {
		t.Fatalf("status = %s", got.Status)
	}
	got.Status = project.StatusFailed
	got.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, project.ID("missing-integration")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}
