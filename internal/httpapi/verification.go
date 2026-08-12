package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"suda-forge/internal/auth"
	"suda-forge/internal/events"
	"suda-forge/internal/verification"
)

func (s Server) createVerification(w http.ResponseWriter, r *http.Request) {
	if s.VerificationEngine == nil || s.VerificationStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("verification service unavailable"))
		return
	}
	var req verification.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProjectID == "" {
		req.ProjectID = r.URL.Query().Get("project_id")
	}
	if req.ProjectID == "" || !s.requireProjectPermission(w, r, req.ProjectID, auth.PermissionRun) {
		return
	}
	project := verification.ProjectContext{ProjectID: req.ProjectID, RuntimeID: req.RuntimeID, Workspace: req.Workspace, RuntimeProvider: s.RuntimeProvider}
	run, err := s.VerificationEngine.Run(r.Context(), req, project)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err = s.VerificationStore.Save(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}
func (s Server) getVerification(w http.ResponseWriter, r *http.Request) {
	if s.VerificationStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("verification store unavailable"))
		return
	}
	run, err := s.VerificationStore.Get(r.Context(), verification.ID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !s.requireProjectPermission(w, r, run.ProjectID, auth.PermissionRead) {
		return
	}
	writeJSON(w, http.StatusOK, run)
}
func (s Server) taskVerifications(w http.ResponseWriter, r *http.Request) {
	if s.VerificationStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("verification store unavailable"))
		return
	}
	runs, err := s.VerificationStore.ListForTask(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	principal, ok := authPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("authentication required"))
		return
	}
	visible := make([]verification.VerificationRun, 0, len(runs))
	for _, run := range runs {
		if principal.User.Role == auth.RoleAdmin || s.Auth.RequireProjectAccess(r.Context(), principal.User, run.ProjectID, auth.PermissionRead) == nil {
			visible = append(visible, run)
		}
	}
	writeJSON(w, http.StatusOK, visible)
}
func (s Server) verificationArtifacts(w http.ResponseWriter, r *http.Request) {
	if s.VerificationStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("verification store unavailable"))
		return
	}
	run, err := s.VerificationStore.Get(r.Context(), verification.ID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !s.requireProjectPermission(w, r, run.ProjectID, auth.PermissionRead) {
		return
	}
	items, err := s.VerificationStore.Artifacts(r.Context(), verification.ID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) cancelVerification(w http.ResponseWriter, r *http.Request) {
	if s.VerificationStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("verification store unavailable"))
		return
	}
	run, err := s.VerificationStore.Get(r.Context(), verification.ID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !s.requireProjectPermission(w, r, run.ProjectID, auth.PermissionRun) {
		return
	}
	if run.Status == verification.Pending || run.Status == verification.Running {
		run.Status = verification.Cancelled
		now := time.Now().UTC()
		run.CompletedAt = &now
		if err := s.VerificationStore.Save(r.Context(), run); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if s.Events != nil {
			s.Events.Publish(events.Event{Type: "verification.cancelled", ProjectID: run.ProjectID, Data: run})
		}
	}
	writeJSON(w, http.StatusOK, run)
}
func (s Server) repairVerification(w http.ResponseWriter, r *http.Request) {
	if s.VerificationStore == nil || s.RepairLoop == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("repair service unavailable"))
		return
	}
	run, err := s.VerificationStore.Get(r.Context(), verification.ID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if !s.requireProjectPermission(w, r, run.ProjectID, auth.PermissionEdit) {
		return
	}
	if run.Status == verification.Passed {
		writeJSON(w, http.StatusOK, run)
		return
	}
	var body struct {
		RuntimeID, Workspace string
		MaxAttempts          int
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	req := verification.CheckRequest{ProjectID: run.ProjectID, WorkflowID: run.WorkflowID, TaskID: run.TaskID, TaskRunID: run.TaskRunID, RuntimeID: body.RuntimeID, Workspace: body.Workspace, Profile: run.Profile.Name, Policy: run.Profile.Policy, Checks: run.Checks}
	project := verification.ProjectContext{ProjectID: run.ProjectID, RuntimeID: body.RuntimeID, Workspace: body.Workspace, RuntimeProvider: s.RuntimeProvider}
	loop := *s.RepairLoop
	if body.MaxAttempts > 0 {
		loop.MaxAttempts = body.MaxAttempts
	}
	updated, err := loop.Run(r.Context(), req, project, verification.TaskContext{WorkflowID: run.WorkflowID, TaskID: run.TaskID, TaskRunID: run.TaskRunID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err = s.VerificationStore.Save(r.Context(), updated); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
