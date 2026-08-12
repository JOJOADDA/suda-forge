package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectPermission string

const (
	PermissionRead   ProjectPermission = "project.read"
	PermissionEdit   ProjectPermission = "project.edit"
	PermissionRun    ProjectPermission = "project.run"
	PermissionDeploy ProjectPermission = "project.deploy"
)

type ProjectMembership struct {
	ProjectID string
	UserID    string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MembershipRepository interface {
	GetMembership(ctx context.Context, projectID, userID string) (ProjectMembership, error)
	SetMembership(ctx context.Context, membership ProjectMembership) error
	ListProjectIDs(ctx context.Context, userID string) ([]string, error)
}

var ErrMembershipNotFound = errors.New("project membership not found")
var ErrProjectPermissionDenied = errors.New("project permission denied")

type PostgresMembershipRepository struct {
	DB *pgxpool.Pool
}

func (r PostgresMembershipRepository) GetMembership(ctx context.Context, projectID, userID string) (ProjectMembership, error) {
	if r.DB == nil {
		return ProjectMembership{}, errors.New("database pool is required")
	}
	var membership ProjectMembership
	err := r.DB.QueryRow(ctx, `
		SELECT project_id, user_id, role, created_at, updated_at
		FROM project_memberships WHERE project_id=$1 AND user_id=$2
	`, projectID, userID).Scan(&membership.ProjectID, &membership.UserID, &membership.Role, &membership.CreatedAt, &membership.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProjectMembership{}, ErrMembershipNotFound
	}
	return membership, err
}

func (r PostgresMembershipRepository) SetMembership(ctx context.Context, membership ProjectMembership) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		INSERT INTO project_memberships (project_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (project_id, user_id) DO UPDATE SET role=$3, updated_at=$4
	`, membership.ProjectID, membership.UserID, membership.Role, membership.CreatedAt)
	return err
}

func (r PostgresMembershipRepository) ListProjectIDs(ctx context.Context, userID string) ([]string, error) {
	if r.DB == nil {
		return nil, errors.New("database pool is required")
	}
	rows, err := r.DB.Query(ctx, `SELECT project_id FROM project_memberships WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s AuthService) RequireProjectAccess(ctx context.Context, user User, projectID string, permission ProjectPermission) error {
	if user.Role == RoleAdmin || user.Role == RoleOperator {
		return nil
	}
	if s.Memberships == nil {
		return errors.New("membership repository is required")
	}
	membership, err := s.Memberships.GetMembership(ctx, projectID, user.ID)
	if err != nil {
		if errors.Is(err, ErrMembershipNotFound) {
			return ErrProjectPermissionDenied
		}
		return err
	}
	if projectPermissionAllowed(membership.Role, permission) {
		return nil
	}
	return ErrProjectPermissionDenied
}

func projectPermissionAllowed(role string, permission ProjectPermission) bool {
	switch permission {
	case PermissionRead:
		return role == "owner" || role == "editor" || role == "runner" || role == "viewer"
	case PermissionEdit:
		return role == "owner" || role == "editor"
	case PermissionRun:
		return role == "owner" || role == "editor" || role == "runner"
	case PermissionDeploy:
		return role == "owner"
	default:
		return false
	}
}
