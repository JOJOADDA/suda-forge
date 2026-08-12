package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/adapters/runtimes/lxc"
	"suda-forge/internal/agent"
	"suda-forge/internal/aifabric"
	"suda-forge/internal/auth"
	"suda-forge/internal/config"
	"suda-forge/internal/constitution"
	"suda-forge/internal/deployment"
	"suda-forge/internal/designintelligence"
	"suda-forge/internal/environment"
	"suda-forge/internal/events"
	"suda-forge/internal/httpapi"
	"suda-forge/internal/knowledge"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/model"
	"suda-forge/internal/orchestration"
	"suda-forge/internal/postgres"
	"suda-forge/internal/productexperience"
	"suda-forge/internal/projectcomputer"
	"suda-forge/internal/projectintelligence"
	"suda-forge/internal/provisioning"
	"suda-forge/internal/routing"
	"suda-forge/internal/sharedinfra"
	"suda-forge/internal/verification"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool creation failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	runtimeProvider := lxc.New()
	if cfg.LXCBinary != "" && cfg.LXCBinary != "lxc" {
		runtimeProvider.CreateBinary = cfg.LXCBinary
	}
	projects := postgres.Projects{DB: db}
	userRepository := auth.PostgresUserRepository{DB: db}
	sessionRepository := auth.PostgresSessionRepository{DB: db}
	membershipRepository := auth.PostgresMembershipRepository{DB: db}
	authService := &auth.AuthService{Users: userRepository, Sessions: auth.SessionService{Repository: sessionRepository, Now: time.Now}, Memberships: membershipRepository, Now: time.Now}
	lifecycleService := lifecycle.Service{Projects: projects, Runtime: runtimeProvider, Now: time.Now}
	agentRegistry := agent.NewRegistry()
	_ = agentRegistry.RegisterDefinition(agent.AgentDefinition{ID: "codex", Name: "codex", DisplayName: "Codex", Adapter: "codex", Status: "AVAILABLE"})
	_ = agentRegistry.RegisterDefinition(agent.AgentDefinition{ID: "claude-code", Name: "claude-code", DisplayName: "Claude Code", Adapter: "claude-code", Status: "AVAILABLE"})
	_ = agentRegistry.RegisterDefinition(agent.AgentDefinition{ID: "kimi", Name: "kimi", DisplayName: "Kimi", Adapter: "kimi", Status: "AVAILABLE"})
	_ = agentRegistry.Register(agent.NewCodexAdapter(nil))
	_ = agentRegistry.Register(agent.NewClaudeCodeAdapter(nil))
	_ = agentRegistry.Register(agent.NewKimiAdapter(nil))
	agentStore := agent.PostgresStore{DB: db}
	agentService := agent.Service{Store: agentStore, Adapters: agentRegistry, Now: time.Now}
	modelRegistry := model.NewRegistry()
	_ = modelRegistry.RegisterProvider(agent.Provider{ID: "custom", Name: "Custom", Type: "custom", Status: "AVAILABLE"})
	_ = modelRegistry.RegisterProvider(agent.Provider{ID: "openai", Name: "OpenAI", Type: "cloud", Status: "AVAILABLE"})
	_ = modelRegistry.RegisterProvider(agent.Provider{ID: "ollama", Name: "Ollama", Type: "local", Status: "AVAILABLE"})
	_ = modelRegistry.RegisterModel(agent.Model{ID: "cloud-best", ProviderID: "openai", ModelID: "cloud-best", DisplayName: "Cloud Best", ContextWindow: 128000, Reasoning: true, Coding: true, ToolUse: true, Remote: true})
	_ = modelRegistry.RegisterModel(agent.Model{ID: "local-code", ProviderID: "ollama", ModelID: "local-code", DisplayName: "Local Code", ContextWindow: 32000, Coding: true, ToolUse: true, Local: true})
	routingModels := []routing.ModelProfile{
		{ModelID: "cloud-best", ProviderID: "openai", DisplayName: "Cloud Best", Remote: true, ContextWindow: 128000, Availability: routing.Available, Capabilities: map[routing.Capability]bool{routing.Coding: true, routing.Architecture: true, routing.Reasoning: true, routing.ToolUse: true, routing.LongContext: true}, Performance: routing.PerformanceProfile{CodingScore: 1, ReasoningScore: 1, ReliabilityScore: .95, LatencyClass: "slow"}, Pricing: routing.Pricing{InputCost: 10, OutputCost: 30, Currency: "USD", PricingUnit: "1M tokens", EffectiveDate: time.Now().UTC()}},
		{ModelID: "local-code", ProviderID: "ollama", DisplayName: "Local Code", Local: true, ContextWindow: 32000, Availability: routing.Available, Capabilities: map[routing.Capability]bool{routing.Coding: true, routing.Backend: true, routing.Reasoning: true, routing.ToolUse: true}, Performance: routing.PerformanceProfile{CodingScore: .7, ReliabilityScore: .8, LatencyClass: "fast"}, Pricing: routing.Pricing{Currency: "USD", PricingUnit: "1M tokens", EffectiveDate: time.Now().UTC()}},
	}
	router := routing.NewRouter(nil)
	routingStore := routing.Store{DB: db}
	planner := orchestration.DeterministicPlanner{}
	orchestrator := orchestration.Orchestrator{Planner: planner, Now: time.Now}
	workflowStore := orchestration.PostgresStore{DB: db}
	eventBus := events.NewBus()
	aiRegistry := aifabric.NewRuntimeRegistry()
	if cfg.OllamaURL != "" {
		_ = aiRegistry.RegisterRuntime(aifabric.NewOllamaRuntime(aifabric.RuntimeSpec{ID: "ollama", Kind: "ollama", Endpoint: cfg.OllamaURL, Local: true}))
	}
	if cfg.VLLMURL != "" {
		_ = aiRegistry.RegisterRuntime(aifabric.NewVLLMRuntime(aifabric.RuntimeSpec{ID: "vllm", Kind: "vllm", Endpoint: cfg.VLLMURL, Local: true}))
	}
	if cfg.LlamaCPPURL != "" {
		_ = aiRegistry.RegisterRuntime(aifabric.NewLlamaCPPRuntime(aifabric.RuntimeSpec{ID: "llama.cpp", Kind: "llama.cpp", Endpoint: cfg.LlamaCPPURL, Local: true}))
	}
	aiManager := aifabric.NewManager(aiRegistry, aifabric.BusSink{Bus: eventBus})
	aiStore := aifabric.Store{DB: db}
	verificationStore := verification.Store{DB: db}
	deploymentStore := deployment.Store{DB: db}
	deploymentManager := deployment.NewManager(time.Now)
	serviceDiscovery := deployment.RuntimeServiceDiscovery{Runtime: runtimeProvider}
	portRegistry := deployment.PostgresPortRegistry{DB: db}
	infrastructureCatalog := deployment.NewCatalog()
	infrastructureCatalog.Certificates = deployment.CaddyCertificate{Proxy: deployment.CaddyProxy{AdminURL: cfg.CaddyAdminURL}}
	deploymentManager.Runtime = runtimeProvider
	deploymentManager.Events = deployment.CompositeAuditSink{Bus: eventBus, DB: db}
	deploymentManager.Proxy = deployment.CaddyProxy{AdminURL: cfg.CaddyAdminURL}
	deploymentManager.Health = deployment.RuntimeHealthChecker{Runtime: runtimeProvider}
	deploymentManager.Deployer = deployment.RuntimeDeploymentProvider{Runtime: runtimeProvider}
	deploymentManager.Verify = deployment.VerificationAdapter{Check: func(ctx context.Context, runID string) error {
		run, err := verificationStore.Get(ctx, verification.ID(runID))
		if err != nil {
			return err
		}
		if run.Status != verification.Passed {
			return fmt.Errorf("verification run %s is not passed", runID)
		}
		return nil
	}}
	_ = deployment.LocalStorage{Root: cfg.DeployStorageRoot}
	verificationEngine := &verification.Engine{Registry: verification.DefaultRegistry(), Events: verification.BusSink{Bus: eventBus}, Now: time.Now}
	repairLoop := &verification.RepairLoop{Engine: verificationEngine, Analyzer: verification.DeterministicFailureAnalyzer{}, Executor: verification.OrchestrationRepairExecutor{Executor: orchestration.RuntimeAgentExecutor{}}, MaxAttempts: 3, Events: verification.BusSink{Bus: eventBus}, Now: time.Now}
	intelligenceEngine := &projectintelligence.Engine{Now: time.Now}
	intelligenceStore := &projectintelligence.Store{DB: db}
	environmentStore := &environment.Store{DB: db}
	provisioningManager := provisioning.NewManager(time.Now)
	provisioningManager.Runtime = runtimeProvider
	provisioningManager.Store = provisioning.PostgresStore{DB: db}
	provisioningManager.Events = provisioningEventSink{Bus: eventBus}
	projectComputerManager := projectcomputer.NewManager(time.Now)
	projectComputerManager.Provider = runtimeProvider
	projectComputerStore := projectcomputer.PostgresStore{DB: db}
	projectComputerManager.Store = projectComputerStore
	projectComputerManager.Events = projectComputerEventSink{Bus: eventBus}
	toolRegistry := sharedinfra.DefaultRegistry()
	sharedStore := sharedinfra.PostgresStore{DB: db}
	loadedTools, loadErr := sharedStore.LoadTools(ctx)
	if loadErr != nil {
		logger.Error("shared infrastructure registry load failed", "error", loadErr)
		os.Exit(1)
	}
	if len(loadedTools) > 0 {
		if err := toolRegistry.Load(loadedTools); err != nil {
			logger.Error("shared infrastructure registry hydrate failed", "error", err)
			os.Exit(1)
		}
	} else {
		for _, tool := range toolRegistry.List() {
			if err := sharedStore.SaveTool(ctx, tool); err != nil {
				logger.Error("shared infrastructure registry seed failed", "tool", tool.ID, "error", err)
				os.Exit(1)
			}
		}
	}
	globalCache := sharedinfra.NewCache(time.Now)
	globalCache.Persistence = sharedStore
	if err := globalCache.Hydrate(ctx); err != nil {
		logger.Error("shared cache hydration failed", "error", err)
		os.Exit(1)
	}

	environmentResolver := sharedinfra.Resolver{Registry: toolRegistry, Cache: globalCache, Platform: "linux", Architecture: "amd64"}
	designEngine := designintelligence.NewEngine(time.Now)
	designStore := designintelligence.PostgresStore{DB: db}
	knowledgeStore := knowledge.PostgresStore{DB: db}
	constitutions := map[string]constitution.Constitution{}
	constitutionStore := constitution.PostgresStore{DB: db}
	agentService.Guard = constitutionGuard{Constitutions: constitutions, Store: constitutionStore}
	productExperience := productexperience.NewService(time.Now)
	productExperience.Knowledge = knowledgeStore
	productExperience.Constitutions = constitutions
	productExperience.DesignStore = &designStore
	productExperience.ConstitutionStore = constitutionStore
	designSystems := map[string]designintelligence.DesignSystem{}
	productExperience.DesignSystems = designSystems
	productStore := productexperience.PostgresStore{DB: db}
	loopExecutor := productexperience.DelegatedStageExecutor{
		Orchestrate: func(ctx context.Context, execution productexperience.LoopExecution) (map[string]any, error) {
			workflow, err := orchestrator.Plan(orchestration.PlannerInput{Intent: orchestration.UserIntent{ProjectID: execution.ProjectID, Goal: execution.Goal}})
			if err != nil {
				return nil, err
			}
			if err := workflowStore.SaveWorkflow(ctx, workflow); err != nil {
				return nil, err
			}
			return map[string]any{"workflow_id": workflow.ID, "status": workflow.Status}, nil
		},
		Verify: func(context.Context, productexperience.LoopExecution, productexperience.LoopStage) (map[string]any, error) {
			return nil, productexperience.ErrLoopBlocked
		},
		Now: time.Now,
	}
	loopCoordinator := productexperience.NewCoordinator(&productStore, loopExecutor, time.Now)
	go func() {
		if err := loopCoordinator.Recover(context.Background()); err != nil {
			logger.Error("autonomous_loop_recovery_failed", "error", err)
		}
	}()
	activityLog := productexperience.NewActivityLog(time.Now)
	activityLog.Store = productStore
	go forwardProductActivity(context.Background(), eventBus, activityLog)
	api := httpapi.Server{Auth: authService, AuthCookieSecure: cfg.AuthCookieSecure, Projects: projects, Lifecycle: lifecycleService, Events: eventBus, AgentService: &agentService, AgentRegistry: agentRegistry, ModelRegistry: modelRegistry, Router: &router, RoutingModels: routingModels, RoutingStore: &routingStore, Orchestrator: &orchestrator, WorkflowStore: &workflowStore, VerificationStore: &verificationStore, VerificationEngine: verificationEngine, RepairLoop: repairLoop, RuntimeProvider: runtimeProvider, AIManager: aiManager, AIStore: &aiStore, DeploymentManager: deploymentManager, DeploymentStore: &deploymentStore, ServiceDiscovery: serviceDiscovery, PortRegistry: portRegistry, ProxyProvider: deployment.CaddyProxy{AdminURL: cfg.CaddyAdminURL}, Infrastructure: infrastructureCatalog, Intelligence: intelligenceEngine, IntelligenceStore: intelligenceStore, EnvironmentStore: environmentStore, Provisioning: provisioningManager, ProjectComputers: projectComputerManager, ToolRegistry: toolRegistry, GlobalCache: globalCache, EnvironmentResolver: environmentResolver, DesignIntelligence: designEngine, DesignStore: &designStore, DesignSystems: designSystems, KnowledgeStore: knowledgeStore, ProductExperience: productExperience, ProductStore: &productStore, LoopCoordinator: loopCoordinator, Constitutions: constitutions, ConstitutionStore: &constitutionStore, ActivityLog: activityLog, VisualQA: productexperience.VisualQABoundary{Computers: projectComputerManager}}

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		logger.Info("server_started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	logger.Info("server_stopped")
}
