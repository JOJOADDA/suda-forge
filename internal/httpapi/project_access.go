package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"suda-forge/internal/auth"
	"suda-forge/internal/provisioning"
)

func (s Server) requireProjectPermission(w http.ResponseWriter, r *http.Request, projectID string, permission auth.ProjectPermission) bool {
	principal, ok := authPrincipal(r.Context())
	if !ok || s.Auth == nil {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return false
	}
	if err := s.Auth.RequireProjectAccess(r.Context(), principal.User, projectID, permission); err != nil {
		if errors.Is(err, auth.ErrProjectPermissionDenied) || errors.Is(err, auth.ErrMembershipNotFound) {
			writeError(w, http.StatusForbidden, errors.New("project permission denied"))
			return false
		}
		writeError(w, http.StatusInternalServerError, err)
		return false
	}
	return true
}

func (s Server) requireProvisioningAccess(w http.ResponseWriter, r *http.Request, permission auth.ProjectPermission) (provisioning.Run, bool) {
	if s.Provisioning == nil || s.Provisioning.Store == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("provisioning persistence unavailable"))
		return provisioning.Run{}, false
	}
	run, err := s.Provisioning.Store.Get(r.Context(), provisioning.ID(r.PathValue("run")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return provisioning.Run{}, false
	}
	if !s.requireProjectPermission(w, r, run.ProjectID, permission) {
		return provisioning.Run{}, false
	}
	return run, true
}

func (s Server) projectAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, ok := projectIDFromRequest(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if !s.requireProjectPermission(w, r, projectID, projectPermissionForRequest(r)) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func projectIDFromRequest(r *http.Request) (string, bool) {
	if id := r.PathValue("project"); id != "" {
		return id, true
	}
	if id := r.PathValue("id"); id != "" && strings.HasPrefix(r.URL.Path, "/api/v1/projects/") {
		return id, true
	}
	return "", false
}

func projectPermissionForRequest(r *http.Request) auth.ProjectPermission {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return auth.PermissionRead
	}
	path := r.URL.Path
	if strings.HasSuffix(path, "/start") || strings.HasSuffix(path, "/stop") || strings.HasSuffix(path, "/restart") {
		return auth.PermissionRun
	}
	if strings.Contains(path, "/deployments") || strings.Contains(path, "/previews") || strings.Contains(path, "/domains") || strings.Contains(path, "/health-checks") {
		return auth.PermissionDeploy
	}
	if strings.Contains(path, "/agent-sessions") || strings.Contains(path, "/autonomous-loop") || strings.Contains(path, "/visual-qa") {
		return auth.PermissionRun
	}
	return auth.PermissionEdit
}
