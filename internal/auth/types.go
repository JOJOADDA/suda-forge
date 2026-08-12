package auth

import (
	"context"
	"time"
)

type GlobalRole string

const (
	RoleAdmin    GlobalRole = "admin"
	RoleOperator GlobalRole = "operator"
	RoleMember   GlobalRole = "member"
	RoleViewer   GlobalRole = "viewer"
)

type User struct {
	ID          string
	Email       string
	DisplayName string
	Role        GlobalRole
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

type Session struct {
	ID         string
	UserID     string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastSeenAt *time.Time
	RevokedAt  *time.Time
	UserAgent  string
	IPAddress  string
}

type SessionInput struct {
	UserID    string
	TTL       time.Duration
	UserAgent string
	IPAddress string
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session Session, tokenHash []byte) error
	FindActiveSession(ctx context.Context, tokenHash []byte, now time.Time) (Session, error)
	TouchSession(ctx context.Context, sessionID string, now time.Time) error
	RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error
	RevokeAllUserSessions(ctx context.Context, userID string, now time.Time) error
	DeleteExpiredSessions(ctx context.Context, before time.Time) error
}
