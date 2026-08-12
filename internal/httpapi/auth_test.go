package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"suda-forge/internal/auth"
)

type testUsers struct {
	users  map[string]auth.User
	hashes map[string]string
}

func (r *testUsers) CountUsers(context.Context) (int, error) { return len(r.users), nil }
func (r *testUsers) CreateAdmin(_ context.Context, user auth.User, passwordHash string) error {
	r.users[user.Email] = user
	r.hashes[user.Email] = passwordHash
	return nil
}
func (r *testUsers) FindByEmail(_ context.Context, email string) (auth.User, string, error) {
	user, ok := r.users[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return auth.User{}, "", auth.ErrUserNotFound
	}
	return user, r.hashes[user.Email], nil
}
func (r *testUsers) FindByID(_ context.Context, id string) (auth.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return auth.User{}, auth.ErrUserNotFound
}
func (r *testUsers) TouchLogin(context.Context, string, time.Time) error { return nil }

type testSessions struct {
	byHash  map[string]auth.Session
	revoked map[string]bool
}

func (r *testSessions) CreateSession(_ context.Context, session auth.Session, hash []byte) error {
	r.byHash[string(hash)] = session
	return nil
}
func (r *testSessions) FindActiveSession(_ context.Context, hash []byte, now time.Time) (auth.Session, error) {
	session, ok := r.byHash[string(hash)]
	if !ok || r.revoked[session.ID] || !now.Before(session.ExpiresAt) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return session, nil
}
func (r *testSessions) TouchSession(context.Context, string, time.Time) error { return nil }
func (r *testSessions) RevokeSession(_ context.Context, hash []byte, _ time.Time) error {
	if session, ok := r.byHash[string(hash)]; ok {
		r.revoked[session.ID] = true
	}
	return nil
}
func (r *testSessions) RevokeAllUserSessions(_ context.Context, userID string, _ time.Time) error {
	for _, session := range r.byHash {
		if session.UserID == userID {
			r.revoked[session.ID] = true
		}
	}
	return nil
}
func (r *testSessions) DeleteExpiredSessions(context.Context, time.Time) error { return nil }

func testAuthServer() http.Handler {
	users := &testUsers{users: map[string]auth.User{}, hashes: map[string]string{}}
	sessions := &testSessions{byHash: map[string]auth.Session{}, revoked: map[string]bool{}}
	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	service := &auth.AuthService{
		Users:    users,
		Sessions: auth.SessionService{Repository: sessions, Now: func() time.Time { return now }, TTL: time.Hour},
		Now:      func() time.Time { return now },
	}
	return (Server{Auth: service, AuthCookieSecure: false}).Handler()
}

func TestAuthMiddlewareAndSessionEndpoints(t *testing.T) {
	handler := testAuthServer()

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized API status = %d, want 401", unauthorized.Code)
	}

	bootstrap := httptest.NewRecorder()
	bootstrapRequest := httptest.NewRequest(http.MethodPost, "/auth/bootstrap", strings.NewReader(`{"email":"admin@example.com","display_name":"Admin","password":"a-very-strong-password"}`))
	bootstrapRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(bootstrap, bootstrapRequest)
	if bootstrap.Code != http.StatusCreated {
		t.Fatalf("bootstrap status = %d, want 201: %s", bootstrap.Code, bootstrap.Body.String())
	}
	cookie := bootstrap.Result().Cookies()[0]
	if cookie.Name != sessionCookieName || cookie.Value == "" || !cookie.HttpOnly || cookie.Secure {
		t.Fatalf("bootstrap cookie = %+v", cookie)
	}

	me := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meRequest.AddCookie(cookie)
	handler.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), "admin@example.com") {
		t.Fatalf("me status/body = %d/%s", me.Code, me.Body.String())
	}

	logout := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRequest.AddCookie(cookie)
	handler.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status = %d", logout.Code)
	}

	afterLogout := httptest.NewRecorder()
	afterLogoutRequest := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	afterLogoutRequest.AddCookie(cookie)
	handler.ServeHTTP(afterLogout, afterLogoutRequest)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401", afterLogout.Code)
	}
}
