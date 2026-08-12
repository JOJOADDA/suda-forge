package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"suda-forge/internal/auth"

	"suda-forge/internal/environment"
	"suda-forge/internal/events"
	"suda-forge/internal/projectcomputer"
)

func (s Server) listProjectComputers(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" || !s.requireProjectPermission(w, r, projectID, auth.PermissionRead) {
		if projectID == "" {
			writeError(w, http.StatusBadRequest, errors.New("project_id is required"))
		}
		return
	}
	items, err := s.ProjectComputers.List(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) createProjectComputer(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	var input struct {
		ProjectID string                           `json:"project_id"`
		Manifest  environment.Manifest             `json:"manifest"`
		Available projectcomputer.ResourceSnapshot `json:"available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !s.requireProjectPermission(w, r, input.ProjectID, auth.PermissionEdit) {
		return
	}
	if input.Available.CPU == 0 {
		input.Available = projectcomputer.ResourceSnapshot{CPU: 2, MemoryBytes: 4 << 30, DiskBytes: 20 << 30, Network: true}
	}
	computer, err := s.ProjectComputers.Create(r.Context(), input.ProjectID, input.Manifest, input.Available)
	if err != nil {
		code := http.StatusConflict
		if strings.Contains(err.Error(), "INSUFFICIENT_RESOURCES") {
			code = http.StatusUnprocessableEntity
		}
		writeJSON(w, code, map[string]any{"error": err.Error(), "project_computer": computer})
		return
	}
	writeJSON(w, http.StatusCreated, computer)
}
func (s Server) getProjectComputer(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	c, err := s.ProjectComputers.Get(r.Context(), projectcomputer.ID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if !s.requireProjectPermission(w, r, c.ProjectID, auth.PermissionRead) {
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s Server) startProjectComputer(w http.ResponseWriter, r *http.Request) {
	s.lifecycleComputer(w, r, auth.PermissionRun, func(id projectcomputer.ID) (projectcomputer.ProjectComputer, error) {
		return s.ProjectComputers.Start(r.Context(), id)
	})
}
func (s Server) stopProjectComputer(w http.ResponseWriter, r *http.Request) {
	s.lifecycleComputer(w, r, auth.PermissionRun, func(id projectcomputer.ID) (projectcomputer.ProjectComputer, error) {
		return s.ProjectComputers.Stop(r.Context(), id)
	})
}
func (s Server) restartProjectComputer(w http.ResponseWriter, r *http.Request) {
	s.lifecycleComputer(w, r, auth.PermissionRun, func(id projectcomputer.ID) (projectcomputer.ProjectComputer, error) {
		return s.ProjectComputers.Restart(r.Context(), id)
	})
}
func (s Server) destroyProjectComputer(w http.ResponseWriter, r *http.Request) {
	s.lifecycleComputer(w, r, auth.PermissionEdit, func(id projectcomputer.ID) (projectcomputer.ProjectComputer, error) {
		return s.ProjectComputers.Destroy(r.Context(), id)
	})
}
func (s Server) lifecycleComputer(w http.ResponseWriter, r *http.Request, permission auth.ProjectPermission, fn func(projectcomputer.ID) (projectcomputer.ProjectComputer, error)) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	computer, err := s.ProjectComputers.Get(r.Context(), projectcomputer.ID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if !s.requireProjectPermission(w, r, computer.ProjectID, permission) {
		return
	}
	c, err := fn(projectcomputer.ID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "project_computer": c})
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s Server) verifyProjectComputer(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	var input struct {
		Capabilities []projectcomputer.Capability `json:"capabilities"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if len(input.Capabilities) == 0 {
		input.Capabilities = []projectcomputer.Capability{projectcomputer.Filesystem, projectcomputer.Process, projectcomputer.Git}
	}
	computer, err := s.ProjectComputers.Get(r.Context(), projectcomputer.ID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if !s.requireProjectPermission(w, r, computer.ProjectID, auth.PermissionRun) {
		return
	}
	c, err := s.ProjectComputers.Verify(r.Context(), projectcomputer.ID(r.PathValue("id")), input.Capabilities)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "project_computer": c})
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s Server) rebuildProjectComputer(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	id := projectcomputer.ID(r.PathValue("id"))
	computer, err := s.ProjectComputers.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if !s.requireProjectPermission(w, r, computer.ProjectID, auth.PermissionEdit) {
		return
	}
	if _, err := s.ProjectComputers.Destroy(r.Context(), id); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "REBUILD_REQUIRES_MANIFEST", "project_computer_id": string(id)})
}
func (s Server) listSharedTools(w http.ResponseWriter, r *http.Request) {
	if s.ToolRegistry == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "tool registry unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, s.ToolRegistry.List())
}
func (s Server) getSharedTool(w http.ResponseWriter, r *http.Request) {
	if s.ToolRegistry == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "tool registry unavailable"})
		return
	}
	t, ok := s.ToolRegistry.Get(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tool not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}
func (s Server) listSharedToolVersions(w http.ResponseWriter, r *http.Request) {
	if s.ToolRegistry == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "tool registry unavailable"})
		return
	}
	t, ok := s.ToolRegistry.Get(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tool not found"})
		return
	}
	writeJSON(w, http.StatusOK, t.Versions)
}
func (s Server) cacheOverview(w http.ResponseWriter, r *http.Request) {
	if s.GlobalCache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "global cache unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scope": "SUDA_FORGE", "stats": s.GlobalCache.Stats()})
}
func (s Server) cacheStats(w http.ResponseWriter, r *http.Request) { s.cacheOverview(w, r) }
func (s Server) resolveEnvironment(w http.ResponseWriter, r *http.Request) {
	var m environment.Manifest
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	m.ProjectID = r.PathValue("project")
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "environment.resolve.started", ProjectID: m.ProjectID, Data: map[string]string{"manifest_id": m.ID}})
	}
	res, err := s.EnvironmentResolver.ResolveManifest(m)
	if err != nil {
		if s.Events != nil {
			s.Events.Publish(events.Event{Type: "environment.resolve.completed", ProjectID: m.ProjectID, Data: map[string]string{"status": "FAILED", "error": err.Error()}})
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "environment.resolve.completed", ProjectID: m.ProjectID, Data: res})
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": m.ProjectID, "resolutions": res})
}

func (s Server) verifyProjectEnvironment(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	var input struct {
		ComputerID   string                       `json:"computer_id"`
		Capabilities []projectcomputer.Capability `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if input.ComputerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "computer_id is required"})
		return
	}
	if len(input.Capabilities) == 0 {
		input.Capabilities = []projectcomputer.Capability{projectcomputer.Filesystem, projectcomputer.Process, projectcomputer.Git, projectcomputer.Browser}
	}
	computer, err := s.ProjectComputers.Verify(r.Context(), projectcomputer.ID(input.ComputerID), input.Capabilities)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "project_computer": computer})
		return
	}
	writeJSON(w, http.StatusOK, computer)
}

func (s Server) repairProjectEnvironment(w http.ResponseWriter, r *http.Request) {
	if s.ProjectComputers == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project computer service unavailable"})
		return
	}
	var input struct {
		ComputerID   string                       `json:"computer_id"`
		Capabilities []projectcomputer.Capability `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if input.ComputerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "computer_id is required"})
		return
	}
	if len(input.Capabilities) == 0 {
		input.Capabilities = []projectcomputer.Capability{projectcomputer.Filesystem, projectcomputer.Process, projectcomputer.Git}
	}
	computer, err := s.ProjectComputers.Verify(r.Context(), projectcomputer.ID(input.ComputerID), input.Capabilities)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "project_computer": computer})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "REPAIR_VERIFICATION_REQUESTED", "project_computer": computer})
}
