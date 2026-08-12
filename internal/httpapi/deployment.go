package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"suda-forge/internal/deployment"
	"suda-forge/internal/events"
)

func (s Server) deploymentServices(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment store unavailable"))
		return
	}
	items, err := s.DeploymentStore.Services(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) deploymentPorts(w http.ResponseWriter, r *http.Request) {
	if s.PortRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("port registry unavailable"))
		return
	}
	items, err := s.PortRegistry.List(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) deploymentReservePort(w http.ResponseWriter, r *http.Request) {
	if s.PortRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("port registry unavailable"))
		return
	}
	var binding deployment.PortBinding
	if err := json.NewDecoder(r.Body).Decode(&binding); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	binding.ProjectID = r.PathValue("project")
	if binding.Protocol == "" {
		binding.Protocol = "tcp"
	}
	if binding.Exposure == "" {
		binding.Exposure = deployment.Internal
	}
	item, err := s.PortRegistry.Reserve(r.Context(), binding)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if s.DeploymentStore != nil {
		_ = s.DeploymentStore.SavePort(r.Context(), item)
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s Server) deploymentDiscoverServices(w http.ResponseWriter, r *http.Request) {
	if s.ServiceDiscovery == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("service discovery unavailable"))
		return
	}
	var input struct {
		RuntimeID string `json:"runtime_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	if input.RuntimeID == "" {
		writeError(w, http.StatusBadRequest, errors.New("runtime_id is required"))
		return
	}
	items, err := s.ServiceDiscovery.Discover(r.Context(), input.RuntimeID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	for i := range items {
		items[i].ProjectID = r.PathValue("project")
		items[i].ID = deployment.ID(items[i].ProjectID + "/" + items[i].RuntimeID + "/" + fmt.Sprint(items[i].Port))
		if s.DeploymentStore != nil {
			_ = s.DeploymentStore.SaveService(r.Context(), items[i])
		}
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "service.discovered", ProjectID: r.PathValue("project"), Data: items})
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) deploymentList(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment store unavailable"))
		return
	}
	items, err := s.DeploymentStore.Deployments(r.Context(), r.PathValue("project"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) deploymentGet(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment manager unavailable"))
		return
	}
	item, err := s.DeploymentManager.Deployment(r.Context(), deployment.ID(r.PathValue("deployment")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if item.ProjectID != r.PathValue("project") {
		writeError(w, http.StatusNotFound, errors.New("deployment not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s Server) deploymentCreate(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment manager unavailable"))
		return
	}
	var input struct {
		Environment    deployment.Environment        `json:"environment"`
		Version        string                        `json:"version"`
		SourceRevision string                        `json:"source_revision"`
		RuntimeTarget  string                        `json:"runtime_target"`
		ReleaseID      deployment.ID                 `json:"release_id"`
		Strategy       deployment.DeploymentStrategy `json:"strategy"`
		Metadata       map[string]any                `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Environment == "" {
		input.Environment = deployment.Development
	}
	if input.Strategy == "" {
		input.Strategy = deployment.StrategyRecreate
	}
	if input.SourceRevision == "" {
		writeError(w, http.StatusBadRequest, errors.New("source_revision is required"))
		return
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if _, ok := input.Metadata["verification_run_id"]; !ok {
		writeError(w, http.StatusBadRequest, errors.New("verification_run_id is required"))
		return
	}
	release, releaseErr := s.DeploymentManager.CreateRelease(r.Context(), deployment.Release{ProjectID: r.PathValue("project"), GitRevision: input.SourceRevision, Environment: input.Environment, Status: deployment.ReleaseCreated, BuildMetadata: map[string]any{"version": input.Version}})
	if releaseErr != nil {
		writeError(w, http.StatusBadRequest, releaseErr)
		return
	}
	input.ReleaseID = release.ID
	item, err := s.DeploymentManager.CreateDeployment(r.Context(), deployment.Deployment{ProjectID: r.PathValue("project"), Environment: input.Environment, Version: input.Version, SourceRevision: input.SourceRevision, RuntimeTarget: input.RuntimeTarget, ReleaseID: input.ReleaseID, Strategy: input.Strategy, Metadata: input.Metadata})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if s.DeploymentStore != nil {
		_ = s.DeploymentStore.SaveRelease(r.Context(), release)
		_ = s.DeploymentStore.SaveDeployment(r.Context(), item)
	}
	go func(created deployment.Deployment, rel deployment.Release) {
		final, _ := s.DeploymentManager.Deploy(context.Background(), created.ID, rel, deployment.EnvironmentConfig{ProjectID: created.ProjectID, Environment: created.Environment})
		if s.DeploymentStore != nil {
			_ = s.DeploymentStore.SaveDeployment(context.Background(), final)
		}
	}(item, release)
	writeJSON(w, http.StatusAccepted, item)
}
func (s Server) deploymentRollback(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment manager unavailable"))
		return
	}
	item, err := s.DeploymentManager.Rollback(r.Context(), r.PathValue("project"), deployment.ID(r.PathValue("deployment")))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if s.DeploymentStore != nil {
		_ = s.DeploymentStore.SaveDeployment(r.Context(), item)
	}
	writeJSON(w, http.StatusOK, item)
}
func (s Server) releaseList(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentManager == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("deployment manager unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.DeploymentManager.Releases(r.Context(), r.PathValue("project")))
}
func (s Server) previewCreate(w http.ResponseWriter, r *http.Request) {
	var input deployment.Preview
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.ProjectID = r.PathValue("project")
	if err := deployment.ValidateHostname(input.Hostname); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if input.Environment == "" {
		input.Environment = deployment.PreviewEnv
	}
	if input.Status == "" {
		input.Status = "PENDING"
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	input.UpdatedAt = input.CreatedAt
	if s.ProxyProvider == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("proxy provider unavailable"))
		return
	}
	if err := s.ProxyProvider.CreateRoute(r.Context(), input); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	input.URL = s.ProxyProvider.URL(input)
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "preview.created", ProjectID: input.ProjectID, Data: input})
	}
	writeJSON(w, http.StatusCreated, input)
}
func (s Server) previewDelete(w http.ResponseWriter, r *http.Request) {
	if s.ProxyProvider == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("proxy provider unavailable"))
		return
	}
	if err := s.ProxyProvider.DeleteRoute(r.Context(), deployment.ID(r.PathValue("preview"))); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "preview.deleted", ProjectID: r.PathValue("project"), Data: map[string]string{"id": r.PathValue("preview")}})
	}
	w.WriteHeader(http.StatusNoContent)
}
func normalizeHostname(hostname string) string { return strings.ToLower(strings.TrimSpace(hostname)) }

func (s Server) previewList(w http.ResponseWriter, r *http.Request) {
	if s.Infrastructure == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("infrastructure catalog unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.Infrastructure.Previews(r.Context(), r.PathValue("project")))
}
func (s Server) domainCreate(w http.ResponseWriter, r *http.Request) {
	if s.Infrastructure == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("infrastructure catalog unavailable"))
		return
	}
	var d deployment.Domain
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d.ProjectID = r.PathValue("project")
	d.Hostname = normalizeHostname(d.Hostname)
	saved, err := s.Infrastructure.SaveDomain(r.Context(), d)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "domain.created", ProjectID: d.ProjectID, Data: saved})
	}
	writeJSON(w, http.StatusCreated, saved)
}
func (s Server) domainList(w http.ResponseWriter, r *http.Request) {
	if s.Infrastructure == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("infrastructure catalog unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, s.Infrastructure.Domains(r.Context(), r.PathValue("project")))
}
func (s Server) certificateIssue(w http.ResponseWriter, r *http.Request) {
	if s.Infrastructure == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("infrastructure catalog unavailable"))
		return
	}
	var domain deployment.Domain
	for _, item := range s.Infrastructure.Domains(r.Context(), r.PathValue("project")) {
		if string(item.ID) == r.PathValue("domain") {
			domain = item
			break
		}
	}
	if domain.ID == "" {
		writeError(w, http.StatusNotFound, errors.New("domain not found"))
		return
	}
	certificate, err := s.Infrastructure.IssueCertificate(r.Context(), domain)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "domain.tls_ready", ProjectID: domain.ProjectID, Data: certificate})
	}
	writeJSON(w, http.StatusAccepted, certificate)
}

func (s Server) healthCheckRun(w http.ResponseWriter, r *http.Request) {
	if s.DeploymentManager == nil || s.DeploymentManager.Health == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("health checker unavailable"))
		return
	}
	var check deployment.HealthCheck
	if err := json.NewDecoder(r.Body).Decode(&check); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	check.ProjectID = r.PathValue("project")
	if check.RuntimeID == "" || check.Target == "" {
		writeError(w, http.StatusBadRequest, errors.New("runtime_id and target are required"))
		return
	}
	result, err := s.DeploymentManager.Health.Check(r.Context(), check)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, result)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "health.passed", ProjectID: check.ProjectID, RuntimeID: check.RuntimeID, Data: result})
	}
	writeJSON(w, http.StatusOK, result)
}
