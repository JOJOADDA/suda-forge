package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"suda-forge/internal/aifabric"
	"suda-forge/internal/auth"
)

func (s Server) aiRuntimes(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	out := []map[string]any{}
	for _, runtime := range s.AIManager.Runtimes() {
		health, _ := s.AIManager.Health(r.Context(), runtime.Spec().ID)
		out = append(out, map[string]any{"spec": runtime.Spec(), "capabilities": runtime.Capabilities(), "health": health})
	}
	writeJSON(w, http.StatusOK, out)
}
func (s Server) aiRuntime(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	runtime, err := s.AIManager.Runtime(aifabric.RuntimeID(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	health, _ := s.AIManager.Health(r.Context(), runtime.Spec().ID)
	writeJSON(w, http.StatusOK, map[string]any{"spec": runtime.Spec(), "capabilities": runtime.Capabilities(), "health": health})
}
func (s Server) aiRuntimeStart(w http.ResponseWriter, r *http.Request) {
	s.aiRuntimeLifecycle(w, r, "start")
}
func (s Server) aiRuntimeStop(w http.ResponseWriter, r *http.Request) {
	s.aiRuntimeLifecycle(w, r, "stop")
}
func (s Server) aiRuntimeLifecycle(w http.ResponseWriter, r *http.Request, op string) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var err error
	id := aifabric.RuntimeID(r.PathValue("id"))
	if op == "start" {
		err = s.AIManager.Start(r.Context(), id)
	} else {
		err = s.AIManager.Stop(r.Context(), id)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": op + " requested", "runtime_id": string(id)})
}
func (s Server) aiRuntimeHealth(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	health, err := s.AIManager.Health(r.Context(), aifabric.RuntimeID(r.PathValue("id")))
	if err != nil {
		writeJSON(w, http.StatusOK, health)
		return
	}
	if s.AIStore != nil {
		if runtime, ok := s.AIManager.Registry.Runtime(aifabric.RuntimeID(r.PathValue("id"))); ok {
			_ = s.AIStore.SaveRuntime(r.Context(), runtime, health)
		}
	}
	writeJSON(w, http.StatusOK, health)
}
func (s Server) aiModels(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.AIManager.Registry.Models())
}
func (s Server) aiModel(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	model, ok := s.AIManager.Registry.Model(aifabric.ModelID(r.PathValue("id")))
	if !ok {
		writeError(w, http.StatusNotFound, aifabric.ErrModelNotFound)
		return
	}
	writeJSON(w, http.StatusOK, model)
}
func (s Server) aiModelsDiscover(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	models, err := s.AIManager.Discover(r.Context())
	payload := map[string]any{"models": models}
	if err != nil {
		payload["status"] = "PARTIAL_OR_BLOCKED_BY_ENVIRONMENT"
		payload["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, payload)
}
func (s Server) aiModelInstall(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var req aifabric.ModelInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	model, err := s.AIManager.Install(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, model)
}
func (s Server) aiModelLoad(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var req aifabric.ModelLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.AIManager.Load(r.Context(), req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"model_id": req.ModelID, "status": "READY"})
}
func (s Server) aiModelUnload(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var req aifabric.ModelLoadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.AIManager.Unload(r.Context(), req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"model_id": req.ModelID, "status": "INSTALLED"})
}
func (s Server) aiHardware(w http.ResponseWriter, r *http.Request) {
	resources, err := aifabric.DiscoverHostResources(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	if s.AIStore != nil {
		_ = s.AIStore.SaveHardware(r.Context(), resources)
	}
	writeJSON(w, http.StatusOK, resources)
}
func (s Server) aiGPUs(w http.ResponseWriter, r *http.Request) {
	resources, err := aifabric.DiscoverHostResources(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(w, http.StatusOK, resources.GPUs)
}
func (s Server) aiHealth(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	health := s.AIManager.HealthAll(context.Background())
	writeJSON(w, http.StatusOK, map[string]any{"runtimes": health, "models": s.AIManager.Registry.Models()})
}

func (s Server) aiProjectSettings(w http.ResponseWriter, r *http.Request) {
	if s.AIStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric store unavailable"))
		return
	}
	settings, err := s.AIStore.ProjectSettings(r.Context(), r.PathValue("project"))
	if err != nil {
		settings = aifabric.ProjectAISettings{ProjectID: r.PathValue("project"), RoutingPolicy: "BALANCED", PrivacyPolicy: "PUBLIC"}
	}
	writeJSON(w, http.StatusOK, settings)
}
func (s Server) aiUpdateProjectSettings(w http.ResponseWriter, r *http.Request) {
	if s.AIStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric store unavailable"))
		return
	}
	var settings aifabric.ProjectAISettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	settings.ProjectID = r.PathValue("project")
	if settings.RoutingPolicy == "" {
		settings.RoutingPolicy = "BALANCED"
	}
	if settings.PrivacyPolicy == "" {
		settings.PrivacyPolicy = "PUBLIC"
	}
	if err := s.AIStore.SaveProjectSettings(r.Context(), settings); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s Server) aiInference(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var req aifabric.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProjectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("project_id is required"))
		return
	}
	if !s.requireProjectPermission(w, r, req.ProjectID, auth.PermissionRun) {
		return
	}
	response, err := s.AIManager.Generate(r.Context(), req)
	if s.AIStore != nil {
		status := "COMPLETED"
		if err != nil {
			status = "FAILED"
		}
		_ = s.AIStore.SaveUsage(r.Context(), req, response, status)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
func (s Server) aiInferenceStream(w http.ResponseWriter, r *http.Request) {
	if s.AIManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("ai fabric unavailable"))
		return
	}
	var req aifabric.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProjectID == "" {
		writeError(w, http.StatusBadRequest, errors.New("project_id is required"))
		return
	}
	if !s.requireProjectPermission(w, r, req.ProjectID, auth.PermissionRun) {
		return
	}
	stream, err := s.AIManager.Stream(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}
	encoder := json.NewEncoder(w)
	for event := range stream {
		_, _ = w.Write([]byte("data: "))
		_ = encoder.Encode(event)
		_, _ = w.Write([]byte("\n"))
		flusher.Flush()
	}
}
