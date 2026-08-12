package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"suda-forge/adapters/runtimes/lxc"
	"suda-forge/internal/agent"
	"suda-forge/internal/config"
	"suda-forge/internal/events"
	"suda-forge/internal/httpapi"
	"suda-forge/internal/lifecycle"
	"suda-forge/internal/model"
	"suda-forge/internal/orchestration"
	"suda-forge/internal/postgres"
	"suda-forge/internal/routing"
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
	verificationStore := verification.Store{DB: db}
	verificationEngine := &verification.Engine{Registry: verification.DefaultRegistry(), Events: verification.BusSink{Bus: eventBus}, Now: time.Now}
	repairLoop := &verification.RepairLoop{Engine: verificationEngine, Analyzer: verification.DeterministicFailureAnalyzer{}, Executor: verification.OrchestrationRepairExecutor{Executor: orchestration.RuntimeAgentExecutor{}}, MaxAttempts: 3, Events: verification.BusSink{Bus: eventBus}, Now: time.Now}
	api := httpapi.Server{Projects: projects, Lifecycle: lifecycleService, Events: eventBus, AgentService: &agentService, AgentRegistry: agentRegistry, ModelRegistry: modelRegistry, Router: &router, RoutingModels: routingModels, RoutingStore: &routingStore, Orchestrator: &orchestrator, WorkflowStore: &workflowStore, VerificationStore: &verificationStore, VerificationEngine: verificationEngine, RepairLoop: repairLoop, RuntimeProvider: runtimeProvider}

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
