package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"suda-forge/internal/auth"
)

const sessionCookieName = "suda_session"

type authContextKey struct{}

type AuthPrincipal struct {
	User    auth.User
	Session auth.Session
}

func withAuthPrincipal(ctx context.Context, principal AuthPrincipal) context.Context {
	return context.WithValue(ctx, authContextKey{}, principal)
}

func authPrincipal(ctx context.Context) (AuthPrincipal, bool) {
	principal, ok := ctx.Value(authContextKey{}).(AuthPrincipal)
	return principal, ok
}

func (s Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAuthentication(r) {
			next.ServeHTTP(w, r)
			return
		}
		if s.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, errors.New("authentication service unavailable"))
			return
		}
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		session, err := s.Auth.Sessions.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			clearSessionCookie(w, s.AuthCookieSecure)
			writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		user, err := s.Auth.CurrentUser(r.Context(), session)
		if err != nil || user.Status != "active" {
			clearSessionCookie(w, s.AuthCookieSecure)
			writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		_ = s.Auth.Sessions.Touch(r.Context(), session.ID)
		next.ServeHTTP(w, r.WithContext(withAuthPrincipal(r.Context(), AuthPrincipal{User: user, Session: session})))
	})
}

func requiresAuthentication(r *http.Request) bool {
	if r.URL.Path == "/healthz" || r.URL.Path == "/health" || r.URL.Path == "/ready" {
		return false
	}
	if r.URL.Path == "/auth/status" || r.URL.Path == "/auth/bootstrap" || r.URL.Path == "/auth/login" || r.URL.Path == "/auth/logout" {
		return false
	}
	return strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/auth/me" || r.URL.Path == "/auth/sessions/revoke-all"
}

func (s Server) authStatus(w http.ResponseWriter, r *http.Request) {
	if s.Auth == nil || s.Auth.Users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("authentication service unavailable"))
		return
	}
	count, err := s.Auth.Users.CountUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bootstrap_required": count == 0})
}

func (s Server) authBootstrap(w http.ResponseWriter, r *http.Request) {
	if s.Auth == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("authentication service unavailable"))
		return
	}
	var input struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, err := s.Auth.Bootstrap(r.Context(), input.Email, input.DisplayName, input.Password)
	if err != nil {
		if errors.Is(err, auth.ErrBootstrapComplete) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, token, err := s.Auth.Sessions.Create(r.Context(), auth.SessionInput{UserID: user.ID, UserAgent: r.UserAgent(), IPAddress: remoteIP(r)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	setSessionCookie(w, token, session.ExpiresAt, s.AuthCookieSecure)
	writeJSON(w, http.StatusCreated, publicAuthResponse(user, session))
}

func (s Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if s.Auth == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("authentication service unavailable"))
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, session, token, err := s.Auth.Login(r.Context(), input.Email, input.Password, r.UserAgent(), remoteIP(r))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	setSessionCookie(w, token, session.ExpiresAt, s.AuthCookieSecure)
	writeJSON(w, http.StatusOK, publicAuthResponse(user, session))
}

func (s Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if s.Auth != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			_ = s.Auth.Sessions.Revoke(r.Context(), cookie.Value)
		}
	}
	clearSessionCookie(w, s.AuthCookieSecure)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s Server) authMe(w http.ResponseWriter, r *http.Request) {
	principal, ok := authPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	writeJSON(w, http.StatusOK, publicAuthResponse(principal.User, principal.Session))
}

func (s Server) authRevokeAll(w http.ResponseWriter, r *http.Request) {
	principal, ok := authPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	if err := s.Auth.Sessions.RevokeAll(r.Context(), principal.User.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	clearSessionCookie(w, s.AuthCookieSecure)
	writeJSON(w, http.StatusOK, map[string]string{"status": "sessions_revoked"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expires, MaxAge: maxAge(expires), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func maxAge(expires time.Time) int {
	seconds := int(time.Until(expires).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func publicAuthResponse(user auth.User, session auth.Session) map[string]any {
	return map[string]any{"user": map[string]any{"id": user.ID, "email": user.Email, "display_name": user.DisplayName, "global_role": user.Role, "status": user.Status}, "session": map[string]any{"id": session.ID, "expires_at": session.ExpiresAt}}
}
