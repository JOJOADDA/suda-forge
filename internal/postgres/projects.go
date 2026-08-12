package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/domain/project"
)

var ErrNotFound = errors.New("project not found")

type Projects struct{ DB *pgxpool.Pool }

func (r Projects) Create(ctx context.Context, p project.Project) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO projects (id, name, slug, status, runtime_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.Name, p.Slug, p.Status, p.RuntimeID, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r Projects) Update(ctx context.Context, p project.Project) error {
	_, err := r.DB.Exec(ctx, `UPDATE projects SET name=$2, slug=$3, status=$4, runtime_id=$5, updated_at=$6 WHERE id=$1`, p.ID, p.Name, p.Slug, p.Status, p.RuntimeID, p.UpdatedAt)
	return err
}

func (r Projects) List(ctx context.Context) ([]project.Project, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, name, slug, status, COALESCE(runtime_id,''), created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []project.Project
	for rows.Next() {
		var p project.Project
		var id string
		var status string
		if err := rows.Scan(&id, &p.Name, &p.Slug, &status, &p.RuntimeID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ID = project.ID(id)
		p.Status = project.Status(status)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r Projects) Get(ctx context.Context, id project.ID) (project.Project, error) {
	var p project.Project
	var rawID, status string
	err := r.DB.QueryRow(ctx, `SELECT id, name, slug, status, COALESCE(runtime_id,''), created_at, updated_at FROM projects WHERE id=$1`, id).Scan(&rawID, &p.Name, &p.Slug, &status, &p.RuntimeID, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return project.Project{}, ErrNotFound
	}
	if err != nil {
		return project.Project{}, err
	}
	p.ID = project.ID(rawID)
	p.Status = project.Status(status)
	return p, nil
}

func RuntimeID(p project.ID, now time.Time) string {
	return string(p) + "_runtime_" + now.UTC().Format("20060102150405")
}
