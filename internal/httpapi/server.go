package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"suda-forge/internal/agent"
	"suda-forge/internal/events"
	"suda-forge/internal/model"

	"suda-forge/domain/project"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/postgres"
	"suda-forge/internal/runtime"
)

type Server struct {
	Projects      postgres.Projects
	Lifecycle     lifecycle.Service
	Events        *events.Bus
	AgentService  *agent.Service
	AgentRegistry *agent.Registry
	ModelRegistry *model.Registry
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)

	mux.HandleFunc("GET /api/v1/events", s.events)
	mux.HandleFunc("GET /api/agents", s.listAgents)
	mux.HandleFunc("GET /api/providers", s.listProviders)
	mux.HandleFunc("GET /api/models", s.listModels)
	mux.HandleFunc("GET /api/projects/{project}/agents", s.listAgents)
	mux.HandleFunc("POST /api/projects/{project}/agent-sessions", s.createAgentSession)
	mux.HandleFunc("GET /api/projects/{project}/agent-sessions", s.listAgentSessions)
	mux.HandleFunc("POST /api/projects/{project}/agent-sessions/{session}/start", s.startAgentSession)
	mux.HandleFunc("POST /api/projects/{project}/agent-sessions/{session}/messages", s.sendAgentMessage)
	mux.HandleFunc("POST /api/projects/{project}/agent-sessions/{session}/cancel", s.cancelAgentSession)
	mux.HandleFunc("GET /api/projects/{project}/agent-sessions/{session}/events", s.listAgentEvents)
	mux.HandleFunc("GET /api/v1/projects", s.listProjects)
	mux.HandleFunc("POST /api/v1/projects", s.createProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", s.getProject)
	mux.HandleFunc("POST /api/v1/projects/{id}/start", s.startProject)
	mux.HandleFunc("POST /api/v1/projects/{id}/stop", s.stopProject)
	return withJSON(mux)
}

func (s Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, runtime.HostReadiness())
}
func (s Server) ready(w http.ResponseWriter, r *http.Request) {
	status := runtime.HostReadiness()
	if status.Runtime != "READY" {
		writeJSON(w, http.StatusServiceUnavailable, status)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s Server) events(w http.ResponseWriter, r *http.Request) {
	if s.Events == nil {
		http.Error(w, "event stream unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	for event := range s.Events.Subscribe(r.Context()) {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "event: %s\\ndata: %s\\n\\n", event.Type, data)
		flusher.Flush()
	}
}

func (s Server) listProviders(w http.ResponseWriter, r *http.Request) {
	if s.ModelRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("model registry unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.ModelRegistry.Providers())
}
func (s Server) listModels(w http.ResponseWriter, r *http.Request) {
	if s.ModelRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("model registry unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.ModelRegistry.Models())
}

func (s Server) listAgents(w http.ResponseWriter, r *http.Request) {
	if s.AgentRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent registry unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.AgentRegistry.Definitions())
}
func (s Server) createAgentSession(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	var input struct {
		AgentID          agent.ID `json:"agent_id"`
		RuntimeID        string   `json:"runtime_id"`
		WorkingDirectory string   `json:"working_directory"`
		ProviderID       string   `json:"provider_id"`
		ModelID          string   `json:"model_id"`
		ConfigurationID  string   `json:"configuration_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	now := time.Now().UTC()
	session := agent.NewSession(r.PathValue("project"), input.AgentID, agent.ModelReference{ProviderID: input.ProviderID, ModelID: input.ModelID, ConfigurationID: input.ConfigurationID}, input.RuntimeID, input.WorkingDirectory, now)
	if err := s.AgentService.CreateSession(r.Context(), session); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}
func (s Server) listAgentSessions(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil || s.AgentService.Store == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	sessions, err := s.AgentService.Store.ListSessions(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}
func (s Server) startAgentSession(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	session, err := s.AgentService.Start(r.Context(), r.PathValue("project"), agent.SessionID(r.PathValue("session")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}
func (s Server) sendAgentMessage(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	var input struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Message == "" {
		writeError(w, http.StatusBadRequest, errors.New("message is required"))
		return
	}
	if err := s.AgentService.SendMessage(r.Context(), r.PathValue("project"), agent.SessionID(r.PathValue("session")), input.Message); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
func (s Server) cancelAgentSession(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	session, err := s.AgentService.Cancel(r.Context(), r.PathValue("project"), agent.SessionID(r.PathValue("session")))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}
func (s Server) listAgentEvents(w http.ResponseWriter, r *http.Request) {
	if s.AgentService == nil || s.AgentService.Store == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("agent service unavailable"))
		return
	}
	events, err := s.AgentService.Store.ListEvents(r.Context(), r.PathValue("project"), agent.SessionID(r.PathValue("session")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.Projects.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) createProject(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	p, err := s.Lifecycle.Create(r.Context(), input.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s Server) getProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.Projects.Get(r.Context(), project.ID(r.PathValue("id")))
	if errors.Is(err, postgres.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s Server) startProject(w http.ResponseWriter, r *http.Request) {
	s.changeProjectStatus(w, r, project.StatusRunning)
}
func (s Server) stopProject(w http.ResponseWriter, r *http.Request) {
	s.changeProjectStatus(w, r, project.StatusStopped)
}

func (s Server) changeProjectStatus(w http.ResponseWriter, r *http.Request, status project.Status) {
	p, err := s.Lifecycle.ChangeStatus(r.Context(), project.ID(r.PathValue("id")), status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": http.StatusText(status), "message": err.Error()}})
}
