package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"suda-forge/internal/auth"
	"suda-forge/internal/environment"
	"suda-forge/internal/events"
	"suda-forge/internal/projectintelligence"
	"suda-forge/internal/provisioning"
)

type provisioningBusSink struct{ bus *events.Bus }

func (s provisioningBusSink) Publish(e provisioning.Event) {
	if s.bus != nil {
		s.bus.Publish(events.Event{Type: e.Type, ProjectID: e.ProjectID, Data: e})
	}
}
func (s Server) analyzeProject(w http.ResponseWriter, r *http.Request) {
	if s.Intelligence == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project intelligence unavailable"})
		return
	}
	var input projectintelligence.ProjectIntent
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	input.ProjectID = r.PathValue("project")
	analysis, err := s.Intelligence.Analyze(input, r.URL.Query().Get("override"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if s.IntelligenceStore != nil {
		if err := s.IntelligenceStore.SaveAnalysis(r.Context(), analysis); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, analysis)
}
func (s Server) createEnvironmentManifest(w http.ResponseWriter, r *http.Request) {
	if s.Intelligence == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "project intelligence unavailable"})
		return
	}
	var input struct {
		Intent   projectintelligence.ProjectIntent `json:"intent"`
		Override string                            `json:"override"`
		Profile  environment.Profile               `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	input.Intent.ProjectID = r.PathValue("project")
	analysis, err := s.Intelligence.Analyze(input.Intent, input.Override)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	m, err := environment.FromDecision(analysis.Decision, input.Profile, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	if s.IntelligenceStore != nil {
		if err := s.IntelligenceStore.SaveAnalysis(r.Context(), analysis); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if s.EnvironmentStore != nil {
		if err := s.EnvironmentStore.SaveManifest(r.Context(), m); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"analysis": analysis, "manifest": m})
}
func (s Server) planProvisioning(w http.ResponseWriter, r *http.Request) {
	if s.Provisioning == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "provisioning unavailable"})
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var envelope struct {
		Manifest  environment.Manifest `json:"manifest"`
		RuntimeID string               `json:"runtime_id"`
	}
	_ = json.Unmarshal(raw, &envelope)
	m := envelope.Manifest
	if m.ID == "" {
		if err := json.Unmarshal(raw, &m); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	run, err := s.Provisioning.PlanWithRuntime(r.PathValue("project"), m, envelope.RuntimeID)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, run)
}
func (s Server) startProvisioning(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProvisioningAccess(w, r, auth.PermissionRun); !ok {
		return
	}
	run, err := s.Provisioning.Provision(r.Context(), provisioning.ID(r.PathValue("run")))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "run": run})
		return
	}
	writeJSON(w, http.StatusOK, run)
}
func (s Server) getProvisioning(w http.ResponseWriter, r *http.Request) {
	run, ok := s.requireProvisioningAccess(w, r, auth.PermissionRead)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, run)
}
func (s Server) cancelProvisioning(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProvisioningAccess(w, r, auth.PermissionRun); !ok {
		return
	}
	run, err := s.Provisioning.RequestCancel(r.Context(), provisioning.ID(r.PathValue("run")))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}
func (s Server) resumeProvisioning(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProvisioningAccess(w, r, auth.PermissionRun); !ok {
		return
	}
	run, err := s.Provisioning.Resume(r.Context(), provisioning.ID(r.PathValue("run")))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "run": run})
		return
	}
	writeJSON(w, http.StatusOK, run)
}
func (s Server) cleanupProvisioning(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireProvisioningAccess(w, r, auth.PermissionRun); !ok {
		return
	}
	if err := s.Provisioning.Cleanup(r.Context(), provisioning.ID(r.PathValue("run"))); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "CLEANUP_REQUESTED"})
}
