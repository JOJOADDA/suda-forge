package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"suda-forge/internal/agent"
	"suda-forge/internal/aifabric"
	"suda-forge/internal/constitution"
	"suda-forge/internal/deployment"
	"suda-forge/internal/designintelligence"
	"suda-forge/internal/environment"
	"suda-forge/internal/events"
	"suda-forge/internal/knowledge"
	"suda-forge/internal/model"
	"suda-forge/internal/orchestration"
	"suda-forge/internal/productexperience"
	"suda-forge/internal/projectcomputer"
	"suda-forge/internal/projectintelligence"
	"suda-forge/internal/provisioning"
	"suda-forge/internal/routing"
	"suda-forge/internal/sharedinfra"
	"suda-forge/internal/verification"

	"suda-forge/domain/project"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/postgres"
	"suda-forge/internal/runtime"
)

type Server struct {
	Projects            postgres.Projects
	Lifecycle           lifecycle.Service
	Events              *events.Bus
	AgentService        *agent.Service
	AgentRegistry       *agent.Registry
	ModelRegistry       *model.Registry
	Router              *routing.Router
	RoutingModels       []routing.ModelProfile
	RoutingStore        *routing.Store
	Orchestrator        *orchestration.Orchestrator
	WorkflowStore       *orchestration.PostgresStore
	VerificationStore   *verification.Store
	VerificationEngine  *verification.Engine
	RepairLoop          *verification.RepairLoop
	RuntimeProvider     runtime.Provider
	AIManager           *aifabric.Manager
	AIStore             *aifabric.Store
	DeploymentManager   *deployment.Manager
	DeploymentStore     *deployment.Store
	ServiceDiscovery    deployment.ServiceDiscovery
	PortRegistry        deployment.PortRegistry
	ProxyProvider       deployment.ProxyProvider
	Infrastructure      *deployment.Catalog
	Intelligence        *projectintelligence.Engine
	IntelligenceStore   *projectintelligence.Store
	EnvironmentStore    *environment.Store
	Provisioning        *provisioning.Manager
	ProjectComputers    *projectcomputer.Manager
	ToolRegistry        *sharedinfra.Registry
	GlobalCache         *sharedinfra.Cache
	EnvironmentResolver sharedinfra.Resolver
	DesignIntelligence  *designintelligence.Engine
	DesignStore         *designintelligence.PostgresStore
	DesignSystems       map[string]designintelligence.DesignSystem
	KnowledgeStore      knowledge.Store
	ProductExperience   *productexperience.Service
	ProductStore        *productexperience.PostgresStore
	Constitutions       map[string]constitution.Constitution
	ConstitutionStore   *constitution.PostgresStore
	ActivityLog         *productexperience.ActivityLog
	VisualQA            productexperience.VisualQABoundary
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
	mux.HandleFunc("GET /api/models/{id}", s.getModel)
	mux.HandleFunc("GET /api/providers/{id}", s.getProvider)
	mux.HandleFunc("POST /api/model-routing/decide", s.decideModel)
	mux.HandleFunc("POST /api/projects/{project}/intelligence/analyze", s.analyzeProject)
	mux.HandleFunc("POST /api/projects/{project}/environment/manifests", s.createEnvironmentManifest)
	mux.HandleFunc("POST /api/projects/{project}/provisioning/plans", s.planProvisioning)
	mux.HandleFunc("POST /api/provisioning/{run}/start", s.startProvisioning)
	mux.HandleFunc("GET /api/provisioning/{run}", s.getProvisioning)
	mux.HandleFunc("POST /api/provisioning/{run}/cancel", s.cancelProvisioning)
	mux.HandleFunc("POST /api/provisioning/{run}/resume", s.resumeProvisioning)
	mux.HandleFunc("POST /api/provisioning/{run}/cleanup", s.cleanupProvisioning)
	mux.HandleFunc("GET /api/project-computers", s.listProjectComputers)
	mux.HandleFunc("POST /api/project-computers", s.createProjectComputer)
	mux.HandleFunc("GET /api/project-computers/{id}", s.getProjectComputer)
	mux.HandleFunc("POST /api/project-computers/{id}/start", s.startProjectComputer)
	mux.HandleFunc("POST /api/project-computers/{id}/stop", s.stopProjectComputer)
	mux.HandleFunc("POST /api/project-computers/{id}/restart", s.restartProjectComputer)
	mux.HandleFunc("POST /api/project-computers/{id}/verify", s.verifyProjectComputer)
	mux.HandleFunc("POST /api/project-computers/{id}/rebuild", s.rebuildProjectComputer)
	mux.HandleFunc("DELETE /api/project-computers/{id}", s.destroyProjectComputer)
	mux.HandleFunc("GET /api/tools", s.listSharedTools)
	mux.HandleFunc("GET /api/tools/{id}", s.getSharedTool)
	mux.HandleFunc("GET /api/tools/{id}/versions", s.listSharedToolVersions)
	mux.HandleFunc("GET /api/cache", s.cacheOverview)
	mux.HandleFunc("GET /api/cache/artifacts", s.cacheOverview)
	mux.HandleFunc("GET /api/cache/stats", s.cacheStats)
	mux.HandleFunc("POST /api/projects/{project}/environment/resolve", s.resolveEnvironment)
	mux.HandleFunc("POST /api/projects/{project}/environment/verify", s.verifyProjectEnvironment)
	mux.HandleFunc("POST /api/projects/{project}/environment/repair", s.repairProjectEnvironment)
	mux.HandleFunc("POST /api/projects/{project}/design/analyze", s.analyzeDesign)
	mux.HandleFunc("GET /api/projects/{project}/design", s.getDesign)
	mux.HandleFunc("GET /api/projects/{project}/knowledge", s.getKnowledge)
	mux.HandleFunc("GET /api/projects/{project}/knowledge/graph", s.getKnowledge)
	mux.HandleFunc("POST /api/projects/{project}/knowledge/nodes", s.upsertKnowledgeNode)
	mux.HandleFunc("POST /api/projects/{project}/knowledge/edges", s.upsertKnowledgeEdge)
	mux.HandleFunc("POST /api/projects/{project}/impact/analyze", s.analyzeImpact)
	mux.HandleFunc("GET /api/projects/{project}/agent-context", s.getAgentContext)
	mux.HandleFunc("POST /api/projects/{project}/autonomous-loop/plan", s.planAutonomousLoop)
	mux.HandleFunc("POST /api/projects/{project}/governance/evaluate", s.evaluateGovernance)
	mux.HandleFunc("POST /api/projects/{project}/constitutions", s.createConstitution)
	mux.HandleFunc("GET /api/projects/{project}/constitutions/{agent}", s.getConstitution)

	mux.HandleFunc("GET /api/projects/{project}/activity", s.getProjectActivity)
	mux.HandleFunc("GET /api/projects/{project}/activity/stream", s.projectActivityStream)
	mux.HandleFunc("POST /api/projects/{project}/visual-qa", s.runVisualQA)

	mux.HandleFunc("POST /api/projects/{project}/plans", s.createPlan)
	mux.HandleFunc("POST /api/projects/{project}/workflows", s.createWorkflow)
	mux.HandleFunc("GET /api/projects/{project}/workflows/{workflow}", s.getWorkflow)
	mux.HandleFunc("POST /api/projects/{project}/workflows/{workflow}/cancel", s.cancelWorkflow)
	mux.HandleFunc("POST /api/verifications", s.createVerification)
	mux.HandleFunc("GET /api/verifications/{id}", s.getVerification)
	mux.HandleFunc("POST /api/verifications/{id}/cancel", s.cancelVerification)
	mux.HandleFunc("POST /api/verifications/{id}/repair", s.repairVerification)
	mux.HandleFunc("GET /api/verifications/{id}/artifacts", s.verificationArtifacts)
	mux.HandleFunc("GET /api/ai/runtimes", s.aiRuntimes)
	mux.HandleFunc("GET /api/ai/runtimes/{id}", s.aiRuntime)
	mux.HandleFunc("POST /api/ai/runtimes/{id}/start", s.aiRuntimeStart)
	mux.HandleFunc("POST /api/ai/runtimes/{id}/stop", s.aiRuntimeStop)
	mux.HandleFunc("POST /api/ai/runtimes/{id}/health", s.aiRuntimeHealth)
	mux.HandleFunc("GET /api/ai/models", s.aiModels)
	mux.HandleFunc("GET /api/ai/models/{id}", s.aiModel)
	mux.HandleFunc("POST /api/ai/models/discover", s.aiModelsDiscover)
	mux.HandleFunc("POST /api/ai/models/install", s.aiModelInstall)
	mux.HandleFunc("POST /api/ai/models/load", s.aiModelLoad)
	mux.HandleFunc("POST /api/ai/models/unload", s.aiModelUnload)
	mux.HandleFunc("GET /api/ai/hardware", s.aiHardware)
	mux.HandleFunc("GET /api/ai/gpus", s.aiGPUs)
	mux.HandleFunc("GET /api/ai/health", s.aiHealth)
	mux.HandleFunc("POST /api/ai/inference", s.aiInference)
	mux.HandleFunc("POST /api/ai/inference/stream", s.aiInferenceStream)

	mux.HandleFunc("GET /api/projects/{project}/ai-settings", s.aiProjectSettings)
	mux.HandleFunc("GET /api/projects/{project}/services", s.deploymentServices)
	mux.HandleFunc("POST /api/projects/{project}/services/discover", s.deploymentDiscoverServices)

	mux.HandleFunc("GET /api/projects/{project}/ports", s.deploymentPorts)
	mux.HandleFunc("POST /api/projects/{project}/ports", s.deploymentReservePort)

	mux.HandleFunc("GET /api/projects/{project}/deployments", s.deploymentList)
	mux.HandleFunc("POST /api/projects/{project}/deployments", s.deploymentCreate)
	mux.HandleFunc("GET /api/projects/{project}/deployments/{deployment}", s.deploymentGet)
	mux.HandleFunc("POST /api/projects/{project}/deployments/{deployment}/rollback", s.deploymentRollback)
	mux.HandleFunc("GET /api/projects/{project}/releases", s.releaseList)
	mux.HandleFunc("POST /api/projects/{project}/health-checks", s.healthCheckRun)

	mux.HandleFunc("POST /api/projects/{project}/previews", s.previewCreate)
	mux.HandleFunc("DELETE /api/projects/{project}/previews/{preview}", s.previewDelete)
	mux.HandleFunc("GET /api/projects/{project}/previews", s.previewList)
	mux.HandleFunc("POST /api/projects/{project}/domains", s.domainCreate)
	mux.HandleFunc("GET /api/projects/{project}/domains", s.domainList)
	mux.HandleFunc("POST /api/projects/{project}/domains/{domain}/certificate", s.certificateIssue)

	mux.HandleFunc("PUT /api/projects/{project}/ai-settings", s.aiUpdateProjectSettings)

	mux.HandleFunc("GET /api/tasks/{id}/verifications", s.taskVerifications)

	mux.HandleFunc("POST /api/projects/{project}/workflows/{workflow}/tasks/{task}/approvals", s.requestApproval)
	mux.HandleFunc("POST /api/projects/{project}/workflows/{workflow}/approvals/{approval}/resolve", s.resolveApproval)
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

func (s Server) getProvider(w http.ResponseWriter, r *http.Request) {
	if s.ModelRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("model registry unavailable"))
		return
	}
	provider, ok := s.ModelRegistry.Provider(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("provider not found"))
		return
	}
	writeJSON(w, http.StatusOK, provider)
}
func (s Server) getModel(w http.ResponseWriter, r *http.Request) {
	if s.ModelRegistry == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("model registry unavailable"))
		return
	}
	item, ok := s.ModelRegistry.Model(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("model not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s Server) decideModel(w http.ResponseWriter, r *http.Request) {
	if s.Router == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("routing engine unavailable"))
		return
	}
	var request routing.RoutingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(request.Models) == 0 {
		request.Models = append([]routing.ModelProfile{}, s.RoutingModels...)
		if s.AIManager != nil {
			request.Models = append(request.Models, aifabric.RoutingProfiles(s.AIManager.Registry.Models(), s.AIManager.CachedHealth())...)
		}
	}
	if s.AIStore != nil && request.ProjectID != "" {
		if settings, settingsErr := s.AIStore.ProjectSettings(r.Context(), request.ProjectID); settingsErr == nil {
			if request.Policy == "" {
				request.Policy = settings.RoutingPolicy
			}
			if request.PrivacyLimit == "" {
				request.PrivacyLimit = settings.PrivacyPolicy
			}
			if settings.LocalOnly {
				request.LocalPolicy = routing.LocalOnly
			}
			if request.Budget == 0 {
				request.Budget = settings.Budget
			}
			request.Models = aifabric.ApplyProjectPolicy(settings, request.Models)
		}
	}
	decision, err := s.Router.DecideWithFallbacks(request)
	if s.RoutingStore != nil {
		_ = s.RoutingStore.SaveDecision(r.Context(), routing.DecisionID(time.Now().UTC()), request, decision)
	}
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "decision": decision})
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func (s Server) createPlan(w http.ResponseWriter, r *http.Request) {
	if s.Orchestrator == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("orchestrator unavailable"))
		return
	}
	var input orchestration.PlannerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.Intent.ProjectID = r.PathValue("project")
	plan, err := s.Orchestrator.Planner.Plan(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err = orchestration.ValidatePlan(plan); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}
func (s Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	if s.Orchestrator == nil || s.WorkflowStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("orchestration persistence unavailable"))
		return
	}
	var input orchestration.PlannerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input.Intent.ProjectID = r.PathValue("project")
	workflow, err := s.Orchestrator.Plan(input)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if err = s.WorkflowStore.SaveWorkflow(r.Context(), workflow); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "workflow.created", ProjectID: workflow.ProjectID, Data: workflow})
	}
	writeJSON(w, http.StatusCreated, workflow)

}
func (s Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	if s.WorkflowStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("workflow store unavailable"))
		return
	}
	workflow, err := s.WorkflowStore.GetWorkflow(r.Context(), r.PathValue("project"), orchestration.ID(r.PathValue("workflow")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}
func (s Server) cancelWorkflow(w http.ResponseWriter, r *http.Request) {
	if s.WorkflowStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("workflow store unavailable"))
		return
	}
	workflow, err := s.WorkflowStore.GetWorkflow(r.Context(), r.PathValue("project"), orchestration.ID(r.PathValue("workflow")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	workflow.Status = orchestration.WorkflowCancelled
	workflow.UpdatedAt = time.Now().UTC()
	if err = s.WorkflowStore.SaveWorkflow(r.Context(), workflow); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "workflow.cancelled", ProjectID: workflow.ProjectID, Data: map[string]any{"workflow_id": workflow.ID}})
	}
	writeJSON(w, http.StatusOK, workflow)

}

func (s Server) requestApproval(w http.ResponseWriter, r *http.Request) {
	if s.WorkflowStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("workflow store unavailable"))
		return
	}
	var body struct {
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	approval := orchestration.Approval{ID: orchestration.ID("approval_" + time.Now().UTC().Format("20060102150405.000000000")), WorkflowID: orchestration.ID(r.PathValue("workflow")), TaskID: orchestration.ID(r.PathValue("task")), Permission: body.Permission, Status: "WAITING_FOR_APPROVAL", RequestedAt: time.Now().UTC()}
	if err := s.WorkflowStore.SaveApproval(r.Context(), approval); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.Events != nil {
		s.Events.Publish(events.Event{Type: "task.approval_required", ProjectID: r.PathValue("project"), Data: approval})
	}
	writeJSON(w, http.StatusCreated, approval)
}
func (s Server) resolveApproval(w http.ResponseWriter, r *http.Request) {
	if s.WorkflowStore == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("workflow store unavailable"))
		return
	}
	var body struct {
		Approved bool `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	approval, err := s.WorkflowStore.GetApproval(r.Context(), orchestration.ID(r.PathValue("approval")))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if approval.WorkflowID != orchestration.ID(r.PathValue("workflow")) {
		writeError(w, http.StatusNotFound, errors.New("approval does not belong to workflow"))
		return
	}
	if body.Approved {
		approval.Status = "APPROVED"
	} else {
		approval.Status = "REJECTED"
	}
	now := time.Now().UTC()
	approval.ResolvedAt = &now
	if err := s.WorkflowStore.SaveApproval(r.Context(), approval); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, approval)
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
