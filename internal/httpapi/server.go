package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"suda-forge/domain/project"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/postgres"
)

type Server struct {
	Projects  postgres.Projects
	Lifecycle lifecycle.Service
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/projects", s.listProjects)
	mux.HandleFunc("POST /api/v1/projects", s.createProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", s.getProject)
	mux.HandleFunc("POST /api/v1/projects/{id}/start", s.startProject)
	mux.HandleFunc("POST /api/v1/projects/{id}/stop", s.stopProject)
	return withJSON(mux)
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
