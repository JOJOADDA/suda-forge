package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type PostgresSessionRepository struct {
	DB *pgxpool.Pool
}

func (r PostgresSessionRepository) CreateSession(ctx context.Context, session Session, tokenHash []byte) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet)
	`, session.ID, session.UserID, tokenHash, session.CreatedAt, session.ExpiresAt, session.UserAgent, session.IPAddress)
	return err
}

func (r PostgresSessionRepository) FindActiveSession(ctx context.Context, tokenHash []byte, now time.Time) (Session, error) {
	if r.DB == nil {
		return Session{}, errors.New("database pool is required")
	}
	var session Session
	var ip *string
	err := r.DB.QueryRow(ctx, `
		SELECT id, user_id, created_at, expires_at, last_seen_at, revoked_at, user_agent, host(ip_address)
		FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2
	`, tokenHash, now).Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastSeenAt,
		&session.RevokedAt,
		&session.UserAgent,
		&ip,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if ip != nil {
		session.IPAddress = *ip
	}
	return session, nil
}

func (r PostgresSessionRepository) TouchSession(ctx context.Context, sessionID string, now time.Time) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		UPDATE sessions
		SET last_seen_at = $2
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > $2
	`, sessionID, now)
	return err
}

func (r PostgresSessionRepository) RevokeSession(ctx context.Context, tokenHash []byte, now time.Time) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash, now)
	return err
}

func (r PostgresSessionRepository) RevokeAllUserSessions(ctx context.Context, userID string, now time.Time) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, now)
	return err
}

func (r PostgresSessionRepository) DeleteExpiredSessions(ctx context.Context, before time.Time) error {
	if r.DB == nil {
		return errors.New("database pool is required")
	}
	_, err := r.DB.Exec(ctx, `
		DELETE FROM sessions
		WHERE expires_at <= $1 OR (revoked_at IS NOT NULL AND revoked_at <= $1)
	`, before)
	return err
}
