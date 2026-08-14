package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"suda-forge/internal/auth"
)

func (s Server) listProjectAudit(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project")
	if !s.requireProjectPermission(w, r, projectID, auth.PermissionRead) {
		return
	}
	if s.AuditStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("audit store unavailable"))
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be between 1 and 200"))
			return
		}
		limit = parsed
	}
	items, err := s.AuditStore.ListProject(r.Context(), projectID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
