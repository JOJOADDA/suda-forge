package routing

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrNoCompatibleModel = errors.New("NO_COMPATIBLE_MODEL")

type Router struct{ Estimator CostEstimator }

func NewRouter(estimator CostEstimator) Router {
	if estimator == nil {
		estimator = TokenCostEstimator{}
	}
	return Router{Estimator: estimator}
}
func (r Router) DecideWithFallbacks(req RoutingRequest) (RoutingDecision, error) {
	decision, err := r.Decide(req)
	if err == nil {
		return decision, nil
	}
	for _, fallback := range req.Fallbacks {
		for _, model := range req.Models {
			if model.ProviderID == fallback.ProviderID && model.ModelID == fallback.ModelID {
				candidateReq := req
				candidateReq.Models = []ModelProfile{model}
				candidateReq.UserOverride = nil
				if fallbackDecision, fallbackErr := r.Decide(candidateReq); fallbackErr == nil {
					fallbackDecision.Reason = "Fallback selected after primary incompatibility: " + fallbackDecision.Reason
					return fallbackDecision, nil
				}
			}
		}
	}
	return decision, err
}
func (r Router) Decide(req RoutingRequest) (RoutingDecision, error) {
	policy := effectivePolicy(req)
	if req.PrivacyLimit == "" {
		req.PrivacyLimit = Public
	}
	var accepted, rejected []CandidateResult
	for _, m := range req.Models {
		candidate := r.evaluate(req, policy, m)
		if candidate.Accepted {
			accepted = append(accepted, candidate)
		} else {
			rejected = append(rejected, candidate)
		}
	}
	if req.UserOverride != nil {
		accepted, rejected = applyOverride(req.UserOverride, accepted, rejected)
	}
	if len(accepted) == 0 {
		return RoutingDecision{Rejected: rejected, ConstraintsApplied: []string{"hard_constraints"}}, ErrNoCompatibleModel
	}
	sort.SliceStable(accepted, func(i, j int) bool {
		if accepted[i].Score != accepted[j].Score {
			return accepted[i].Score > accepted[j].Score
		}
		if accepted[i].Model.ProviderID != accepted[j].Model.ProviderID {
			return accepted[i].Model.ProviderID < accepted[j].Model.ProviderID
		}
		return accepted[i].Model.ModelID < accepted[j].Model.ModelID
	})
	selected := accepted[0]
	alternatives := accepted[1:]
	decision := RoutingDecision{Selected: selected.Model, ProviderID: selected.Model.ProviderID, Reason: fmt.Sprintf("Selected %s because %s.", selected.Model.ModelID, strings.Join(selected.Reasons, ", ")), Alternatives: alternatives, Rejected: rejected, ConstraintsApplied: []string{"privacy", "availability", "capability", "agent_compatibility", "runtime", "runtime_health", "resources", "offline", "budget"},
		EstimatedCost: selected.EstimatedCost, Confidence: confidence(selected.Score)}
	return decision, nil
}
func effectivePolicy(req RoutingRequest) RoutingPolicy {
	if req.Policy != "" {
		return req.Policy
	}
	if req.ProjectPolicy != nil {
		return *req.ProjectPolicy
	}
	if req.OrganizationPolicy != nil {
		return *req.OrganizationPolicy
	}
	if req.GlobalPolicy != nil {
		return *req.GlobalPolicy
	}
	return Balanced
}
func (r Router) evaluate(req RoutingRequest, policy RoutingPolicy, model ModelProfile) CandidateResult {
	c := CandidateResult{Model: ModelReference{ProviderID: model.ProviderID, ModelID: model.ModelID}, EstimatedCost: r.Estimator.Estimate(model, 1000, 1000)}
	fail := func(reason string) CandidateResult { c.Rejections = append(c.Rejections, reason); return c }
	if model.Availability != Available && model.Availability != Degraded {
		return fail("model is not available")
	}
	if !req.AvailableRuntime {
		return fail("runtime is unavailable")
	}
	if req.Offline && model.Remote {
		return fail("offline mode excludes remote model")
	}
	if model.RuntimeID != "" && !model.RuntimeHealthy {
		return fail("runtime health is not available")
	}
	if model.Resources.MemoryBytes > 0 && req.AvailableMemory > 0 && model.Resources.MemoryBytes > req.AvailableMemory {
		return fail("insufficient memory")
	}
	if model.Resources.VRAMBytes > 0 && req.AvailableVRAM > 0 && model.Resources.VRAMBytes > req.AvailableVRAM {
		return fail("insufficient GPU memory")
	}
	if model.Resources.GPURequired && !req.GPUAvailable {
		return fail("GPU is required")
	}
	if req.LocalPolicy == LocalOnly && !model.Local {
		return fail("local-only policy")
	}
	if req.PrivacyLimit == Private && !model.Local {
		return fail("private project cannot use remote provider")
	}
	if req.Task.Privacy == Private && !model.Local {
		return fail("private task cannot use remote provider")
	}
	if req.Task.ContextRequired > model.ContextWindow {
		return fail("insufficient context window")
	}
	if req.Task.VisionRequired && !model.Capabilities[Vision] {
		return fail("vision capability missing")
	}
	if req.Task.ToolUseRequired && !model.Capabilities[ToolUse] {
		return fail("tool use capability missing")
	}
	if len(model.SupportedAgents) > 0 && !contains(model.SupportedAgents, req.AgentID) {
		return fail("agent incompatible with model")
	}
	if req.Budget > 0 && c.EstimatedCost > req.Budget {
		return fail("budget hard limit")
	}
	c.Accepted = true
	capability := capabilityScore(req.Task, model)
	c.Score = capability + model.Performance.ReliabilityScore
	c.Reasons = append(c.Reasons, capabilityReasons(req.Task, model)...)
	switch policy {
	case Cheapest:
		c.Score += 1 / (1 + c.EstimatedCost)
		c.Reasons = append(c.Reasons, "cost policy prefers lower estimate")
	case Fastest:
		if model.Performance.LatencyClass == "fast" {
			c.Score += 2
			c.Reasons = append(c.Reasons, "fast latency class")
		}
	case Best:
		c.Score += model.Performance.ReasoningScore + model.Performance.CodingScore
		c.Reasons = append(c.Reasons, "best policy prefers performance")
	case PrivacyFirst:
		if model.Local {
			c.Score += 3
			c.Reasons = append(c.Reasons, "privacy policy prefers local model")
		}
	case Balanced:
		if model.Local {
			c.Score += 0.5
		}
		c.Reasons = append(c.Reasons, "balanced policy")
	case CloudFirst:
		if model.Remote {
			c.Score += 2
			c.Reasons = append(c.Reasons, "cloud-first preference")
		}
	}
	if req.LocalPolicy == LocalFirst && model.Local {
		c.Score += 2
		c.Reasons = append(c.Reasons, "local-first preference")
	}
	return c
}
func applyOverride(override *ModelReference, accepted, rejected []CandidateResult) ([]CandidateResult, []CandidateResult) {
	for i, c := range accepted {
		if c.Model == *override {
			remaining := append([]CandidateResult{}, rejected...)
			remaining = append(remaining, accepted[:i]...)
			remaining = append(remaining, accepted[i+1:]...)
			return []CandidateResult{c}, remaining
		}
	}
	return nil, append(rejected, CandidateResult{Model: *override, Rejections: []string{"user override violates hard constraints"}})
}
func capabilityScore(task TaskProfile, model ModelProfile) float64 {
	score := 0.0
	for _, pair := range []struct {
		need   bool
		cap    Capability
		weight float64
	}{{task.ReasoningRequired, Reasoning, 3}, {task.ToolUseRequired, ToolUse, 2}, {task.VisionRequired, Vision, 3}} {
		if pair.need && model.Capabilities[pair.cap] {
			score += pair.weight
		}
	}
	switch task.TaskType {
	case TaskCode, TaskRefactor, TaskDebug:
		if model.Capabilities[Coding] {
			score += 3
		}
	case TaskArchitecture:
		if model.Capabilities[Architecture] {
			score += 3
		}
	case TaskUI:
		if model.Capabilities[Frontend] {
			score += 3
		}
	case TaskDatabase:
		if model.Capabilities[Database] {
			score += 3
		}
	case TaskDevOps:
		if model.Capabilities[DevOps] {
			score += 3
		}
	case TaskSecurity:
		if model.Capabilities[Security] {
			score += 3
		}
	case TaskTest:
		if model.Capabilities[Testing] {
			score += 3
		}
	case TaskDocumentation:
		if model.Capabilities[Documentation] {
			score += 3
		}
	}
	return score
}
func capabilityReasons(task TaskProfile, model ModelProfile) []string {
	reasons := []string{}
	if task.TaskType != "" {
		reasons = append(reasons, fmt.Sprintf("%s capability match", strings.ToLower(string(task.TaskType))))
	}
	if task.ContextRequired > 0 {
		reasons = append(reasons, "required context fits")
	}
	return reasons
}
func contains(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}
func confidence(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
