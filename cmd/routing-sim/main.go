package main

import (
	"encoding/json"
	"fmt"
	"os"

	"suda-forge/internal/routing"
)

func main() {
	models := []routing.ModelProfile{
		{ModelID: "local-code", ProviderID: "ollama", DisplayName: "Local Code", Local: true, ContextWindow: 32000, Availability: routing.Available, Capabilities: map[routing.Capability]bool{routing.Coding: true, routing.Backend: true, routing.Reasoning: true, routing.ToolUse: true}, Performance: routing.PerformanceProfile{CodingScore: .7, ReliabilityScore: .8, LatencyClass: "fast"}},
		{ModelID: "cloud-best", ProviderID: "openai", DisplayName: "Cloud Best", Remote: true, ContextWindow: 128000, Availability: routing.Available, Capabilities: map[routing.Capability]bool{routing.Coding: true, routing.Architecture: true, routing.Reasoning: true, routing.ToolUse: true}, Performance: routing.PerformanceProfile{CodingScore: 1, ReasoningScore: 1, ReliabilityScore: .95, LatencyClass: "slow"}},
	}
	request := routing.RoutingRequest{ProjectID: "simulator", AgentID: "codex", Policy: routing.Balanced, PrivacyLimit: routing.Public, LocalPolicy: routing.LocalFirst, AvailableRuntime: true, Task: routing.TaskProfile{TaskType: routing.TaskRefactor, ContextRequired: 1000, ReasoningRequired: true, Privacy: routing.Public}, Models: models}
	decision, err := routing.NewRouter(nil).Decide(request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(decision)
}
