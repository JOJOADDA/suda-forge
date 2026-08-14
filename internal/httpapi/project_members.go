package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"suda-forge/internal/auth"
)

var validProjectMemberRoles = map[string]bool{
	"owner":  true,
	"editor": true,
	"runner": true,
	"viewer": true,
}

func (s Server) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	if !s.requireProjectPermission(w, r, r.PathValue("project"), auth.PermissionRead) {
		return
	}
	if s.Auth == nil || s.Auth.Memberships == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("membership service unavailable"))
		return
	}
	members, err := s.Auth.Memberships.ListProjectMembers(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if members == nil {
		members = []auth.ProjectMember{}
	}
	writeJSON(w, http.StatusOK, members)
}

func (s Server) setProjectMember(w http.ResponseWriter, r *http.Request) {
	if !s.requireProjectPermission(w, r, r.PathValue("project"), auth.PermissionDeploy) {
		return
	}
	if s.Auth == nil || s.Auth.Memberships == nil || s.Auth.Users == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("membership service unavailable"))
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	role := strings.ToLower(strings.TrimSpace(input.Role))
	if !validProjectMemberRoles[role] {
		writeError(w, http.StatusBadRequest, errors.New("role must be owner, editor, runner, or viewer"))
		return
	}
	userID := r.PathValue("user")
	if _, err := s.Auth.Users.FindByID(r.Context(), userID); err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, errors.New("user not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC()
	if err := s.Auth.Memberships.SetMembership(r.Context(), auth.ProjectMembership{ProjectID: r.PathValue("project"), UserID: userID, Role: role, CreatedAt: now, UpdatedAt: now}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	members, err := s.Auth.Memberships.ListProjectMembers(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, member := range members {
		if member.UserID == userID {
			writeJSON(w, http.StatusOK, member)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, errors.New("membership was not returned after update"))
}

func (s Server) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	if !s.requireProjectPermission(w, r, r.PathValue("project"), auth.PermissionDeploy) {
		return
	}
	if s.Auth == nil || s.Auth.Memberships == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("membership service unavailable"))
		return
	}
	if err := s.Auth.Memberships.RemoveMembership(r.Context(), r.PathValue("project"), r.PathValue("user")); err != nil {
		if errors.Is(err, auth.ErrMembershipNotFound) {
			writeError(w, http.StatusConflict, errors.New("member not found or cannot remove the last owner"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}
