package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

type memorySessionRepository struct {
	sessions map[string]Session
	hashes   map[string]string
	touched  string
	revoked  map[string]bool
}

func newMemorySessionRepository() *memorySessionRepository {
	return &memorySessionRepository{
		sessions: make(map[string]Session),
		hashes:   make(map[string]string),
		revoked:  make(map[string]bool),
	}
}

func (r *memorySessionRepository) CreateSession(_ context.Context, session Session, tokenHash []byte) error {
	r.sessions[session.ID] = session
	r.hashes[string(tokenHash)] = session.ID
	return nil
}

func (r *memorySessionRepository) FindActiveSession(_ context.Context, tokenHash []byte, now time.Time) (Session, error) {
	id, ok := r.hashes[string(tokenHash)]
	if !ok || r.revoked[id] {
		return Session{}, ErrSessionNotFound
	}
	session := r.sessions[id]
	if !now.Before(session.ExpiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (r *memorySessionRepository) TouchSession(_ context.Context, sessionID string, now time.Time) error {
	r.touched = sessionID
	session := r.sessions[sessionID]
	session.LastSeenAt = &now
	r.sessions[sessionID] = session
	return nil
}

func (r *memorySessionRepository) RevokeSession(_ context.Context, tokenHash []byte, _ time.Time) error {
	id, ok := r.hashes[string(tokenHash)]
	if !ok {
		return ErrSessionNotFound
	}
	r.revoked[id] = true
	return nil
}

func (r *memorySessionRepository) RevokeAllUserSessions(_ context.Context, userID string, _ time.Time) error {
	for id, session := range r.sessions {
		if session.UserID == userID {
			r.revoked[id] = true
		}
	}
	return nil
}

func (r *memorySessionRepository) DeleteExpiredSessions(_ context.Context, before time.Time) error {
	for id, session := range r.sessions {
		if !session.ExpiresAt.After(before) || r.revoked[id] {
			delete(r.sessions, id)
		}
	}
	return nil
}

func TestSessionServiceCreateAndAuthenticate(t *testing.T) {
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	repo := newMemorySessionRepository()
	service := SessionService{Repository: repo, Now: func() time.Time { return now }, TTL: time.Hour}

	session, token, err := service.Create(context.Background(), SessionInput{UserID: "user-1", UserAgent: "test", IPAddress: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if token == "" || session.ID == "" {
		t.Fatal("Create() returned empty token or session id")
	}
	if len(hashToken(token)) != 32 {
		t.Fatalf("hash length = %d, want 32", len(hashToken(token)))
	}
	if _, ok := repo.hashes[token]; ok {
		t.Fatal("repository must not store the raw bearer token")
	}

	got, err := service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != session.ID || got.UserID != "user-1" {
		t.Fatalf("Authenticate() = %+v, want session %s/user-1", got, session.ID)
	}
}

func TestSessionServiceRejectsExpiredAndRevokedTokens(t *testing.T) {
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	repo := newMemorySessionRepository()
	service := SessionService{Repository: repo, Now: func() time.Time { return now }, TTL: time.Minute}

	_, token, err := service.Create(context.Background(), SessionInput{UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Revoke(context.Background(), token); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("Authenticate(revoked) error = %v, want ErrInvalidSession", err)
	}

	repo = newMemorySessionRepository()
	service.Repository = repo
	_, expiredToken, err := service.Create(context.Background(), SessionInput{UserID: "user-1", TTL: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	service.Now = func() time.Time { return now.Add(2 * time.Second) }
	if _, err := service.Authenticate(context.Background(), expiredToken); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("Authenticate(expired) error = %v, want ErrInvalidSession", err)
	}
}

func TestSessionServiceTouchAndRevokeAll(t *testing.T) {
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	repo := newMemorySessionRepository()
	service := SessionService{Repository: repo, Now: func() time.Time { return now }, TTL: time.Hour}

	session, token, err := service.Create(context.Background(), SessionInput{UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Touch(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if repo.touched != session.ID {
		t.Fatalf("touched session = %s, want %s", repo.touched, session.ID)
	}
	if err := service.RevokeAll(context.Background(), "user-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("Authenticate(after revoke all) error = %v, want ErrInvalidSession", err)
	}
}

func TestRandomTokenIsURLSafeAndHasExpectedEntropy(t *testing.T) {
	first, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two random session tokens are equal")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("token is not raw URL-safe base64: %v", err)
	}
	if len(decoded) != SessionTokenBytes {
		t.Fatalf("decoded token bytes = %d, want %d", len(decoded), SessionTokenBytes)
	}
}
