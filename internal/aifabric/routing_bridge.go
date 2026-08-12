package aifabric

import (
	"suda-forge/internal/routing"
	"time"
)

func RoutingProfiles(models []ModelDescriptor, health map[RuntimeID]RuntimeHealth) []routing.ModelProfile {
	out := make([]routing.ModelProfile, 0, len(models))
	for _, model := range models {
		caps := map[routing.Capability]bool{}
		for _, pair := range []struct {
			a Capability
			b routing.Capability
		}{{CapabilityCoding, routing.Coding}, {CapabilityReasoning, routing.Reasoning}, {CapabilityArchitecture, routing.Architecture}, {CapabilityVision, routing.Vision}, {CapabilityToolUse, routing.ToolUse}, {CapabilityStructuredOutput, routing.StructuredOutput}, {CapabilityLongContext, routing.LongContext}, {CapabilityFast, routing.FastResponse}} {
			caps[pair.b] = model.Capabilities[pair.a]
		}
		healthy := model.Status != ModelFailed
		if h, ok := health[model.RuntimeID]; ok {
			healthy = h.Status == RuntimeOnline || h.Status == RuntimeDegraded
		}
		availability := routing.Available
		if !healthy {
			availability = routing.Unavailable
		}
		out = append(out, routing.ModelProfile{ModelID: string(model.ID), ProviderID: model.ProviderID, RuntimeID: string(model.RuntimeID), DisplayName: model.DisplayName, Capabilities: caps, Performance: routing.PerformanceProfile{LatencyClass: latencyClass(model.Latency)}, Pricing: routing.Pricing{Currency: "USD", PricingUnit: "1M tokens"}, Availability: availability, Local: model.Local, Remote: model.Remote, ContextWindow: model.ContextWindow, PrivacyLevel: model.PrivacyLevel, RuntimeHealthy: healthy, Resources: routing.ModelResourceRequirement{MemoryBytes: model.ResourceRequirement.MemoryBytes, VRAMBytes: model.ResourceRequirement.VRAMBytes, GPURequired: model.ResourceRequirement.GPURequired}})
	}
	return out
}
func latencyClass(d time.Duration) string {
	if d > 0 && d < 500*time.Millisecond {
		return "fast"
	}
	return "normal"
}
