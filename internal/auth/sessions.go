package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidSession = errors.New("invalid or expired session")
	ErrInvalidTTL     = errors.New("session ttl must be positive")
)

const (
	DefaultSessionTTL = 24 * time.Hour
	SessionTokenBytes = 32
)

type SessionService struct {
	Repository SessionRepository
	Now        func() time.Time
	TTL        time.Duration
}

func (s SessionService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s SessionService) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return DefaultSessionTTL
}

// Create issues a bearer token only once. The repository receives only its
// SHA-256 digest, so a database read does not expose a reusable session token.
func (s SessionService) Create(ctx context.Context, input SessionInput) (Session, string, error) {
	if s.Repository == nil {
		return Session{}, "", errors.New("session repository is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return Session{}, "", errors.New("user id is required")
	}
	ttl := input.TTL
	if ttl == 0 {
		ttl = s.ttl()
	}
	if ttl <= 0 {
		return Session{}, "", ErrInvalidTTL
	}

	token, err := randomToken()
	if err != nil {
		return Session{}, "", err
	}
	now := s.now()
	sessionID, err := randomID()
	if err != nil {
		return Session{}, "", err
	}
	session := Session{
		ID:        sessionID,
		UserID:    input.UserID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		UserAgent: input.UserAgent,
		IPAddress: input.IPAddress,
	}
	if err := s.Repository.CreateSession(ctx, session, hashToken(token)); err != nil {
		return Session{}, "", err
	}
	return session, token, nil
}

func (s SessionService) Authenticate(ctx context.Context, token string) (Session, error) {
	if s.Repository == nil || strings.TrimSpace(token) == "" {
		return Session{}, ErrInvalidSession
	}
	session, err := s.Repository.FindActiveSession(ctx, hashToken(token), s.now())
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	return session, nil
}

func (s SessionService) Touch(ctx context.Context, sessionID string) error {
	if s.Repository == nil || strings.TrimSpace(sessionID) == "" {
		return ErrInvalidSession
	}
	return s.Repository.TouchSession(ctx, sessionID, s.now())
}

func (s SessionService) Revoke(ctx context.Context, token string) error {
	if s.Repository == nil || strings.TrimSpace(token) == "" {
		return ErrInvalidSession
	}
	return s.Repository.RevokeSession(ctx, hashToken(token), s.now())
}

func (s SessionService) RevokeAll(ctx context.Context, userID string) error {
	if s.Repository == nil || strings.TrimSpace(userID) == "" {
		return ErrInvalidSession
	}
	return s.Repository.RevokeAllUserSessions(ctx, userID, s.now())
}

func (s SessionService) Cleanup(ctx context.Context) error {
	if s.Repository == nil {
		return errors.New("session repository is required")
	}
	return s.Repository.DeleteExpiredSessions(ctx, s.now())
}

func randomToken() (string, error) {
	buf := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
