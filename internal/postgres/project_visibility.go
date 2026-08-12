package postgres

import (
	"context"
	"errors"

	"suda-forge/domain/project"
)

// ListVisible returns only projects the user may see. Administrators retain
// global visibility; other users are filtered through project_memberships.
func (r Projects) ListVisible(ctx context.Context, userID string, globalRole string) ([]project.Project, error) {
	if r.DB == nil {
		return nil, errors.New("database pool is required")
	}
	query := `
		SELECT p.id, p.name, p.slug, p.status, COALESCE(p.runtime_id,''), p.created_at, p.updated_at
		FROM projects p
		WHERE ($2 IN ('admin', 'operator') OR EXISTS (
			SELECT 1 FROM project_memberships pm
			WHERE pm.project_id = p.id AND pm.user_id = $1
		))
		ORDER BY p.created_at DESC
	`
	rows, err := r.DB.Query(ctx, query, userID, globalRole)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []project.Project
	for rows.Next() {
		var p project.Project
		var id, status string
		if err := rows.Scan(&id, &p.Name, &p.Slug, &status, &p.RuntimeID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ID = project.ID(id)
		p.Status = project.Status(status)
		items = append(items, p)
	}
	return items, rows.Err()
}
