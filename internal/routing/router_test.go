package routing

import "testing"

func testModels() []ModelProfile {
	return []ModelProfile{
		{ModelID: "local-code", ProviderID: "ollama", DisplayName: "Local Code", Local: true, Remote: false, ContextWindow: 32000, Availability: Available, Capabilities: map[Capability]bool{Coding: true, Backend: true, Reasoning: true, ToolUse: true}, Performance: PerformanceProfile{CodingScore: .7, ReliabilityScore: .8, LatencyClass: "fast"}, Pricing: Pricing{Currency: "USD", PricingUnit: "1M tokens"}},
		{ModelID: "cloud-best", ProviderID: "openai", DisplayName: "Cloud Best", Remote: true, ContextWindow: 128000, Availability: Available, Capabilities: map[Capability]bool{Coding: true, Architecture: true, Reasoning: true, ToolUse: true, LongContext: true}, Performance: PerformanceProfile{CodingScore: 1, ReasoningScore: 1, ReliabilityScore: .95, LatencyClass: "slow"}, Pricing: Pricing{InputCost: 10, OutputCost: 30, Currency: "USD", PricingUnit: "1M tokens"}},
		{ModelID: "down", ProviderID: "openai", DisplayName: "Down", Remote: true, ContextWindow: 128000, Availability: Unavailable, Capabilities: map[Capability]bool{Coding: true}},
	}
}
func TestRouterBalancedCodingIsDeterministic(t *testing.T) {
	req := RoutingRequest{ProjectID: "p1", AgentID: "codex", Task: TaskProfile{TaskType: TaskCode, ContextRequired: 1000}, Policy: Balanced, PrivacyLimit: Public, AvailableRuntime: true, Models: testModels()}
	r := NewRouter(nil)
	a, e := r.Decide(req)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := r.Decide(req)
	if a.Selected != b.Selected || a.Reason != b.Reason {
		t.Fatalf("non deterministic: %+v / %+v", a, b)
	}
	if a.Selected.ModelID != "local-code" {
		t.Fatalf("selected %s", a.Selected.ModelID)
	}
}
func TestRouterHardConstraints(t *testing.T) {
	r := NewRouter(nil)
	cases := []struct {
		name string
		req  RoutingRequest
	}{
		{"private", RoutingRequest{Task: TaskProfile{Privacy: Private}, PrivacyLimit: Private, AvailableRuntime: true, Models: []ModelProfile{testModels()[1]}}},
		{"local only", RoutingRequest{LocalPolicy: LocalOnly, AvailableRuntime: true, Models: []ModelProfile{testModels()[1]}}},
		{"budget", RoutingRequest{Budget: .000001, AvailableRuntime: true, Models: []ModelProfile{testModels()[1]}}},
		{"context", RoutingRequest{Task: TaskProfile{ContextRequired: 999999}, AvailableRuntime: true, Models: testModels()}},
		{"runtime", RoutingRequest{AvailableRuntime: false, Models: testModels()}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := r.Decide(tc.req); err != ErrNoCompatibleModel {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
func TestRouterPolicyAndOverride(t *testing.T) {
	r := NewRouter(nil)
	req := RoutingRequest{AgentID: "codex", Task: TaskProfile{TaskType: TaskArchitecture}, Policy: Best, AvailableRuntime: true, Models: testModels()}
	d, err := r.Decide(req)
	if err != nil {
		t.Fatal(err)
	}
	if d.Selected.ModelID != "cloud-best" {
		t.Fatalf("best selected %s", d.Selected.ModelID)
	}
	req.UserOverride = &ModelReference{ProviderID: "ollama", ModelID: "local-code"}
	d, err = r.Decide(req)
	if err != nil {
		t.Fatal(err)
	}
	if d.Selected.ModelID != "local-code" {
		t.Fatalf("override selected %s", d.Selected.ModelID)
	}
}
