package aifabric

import (
	"context"
	"time"
)

type BenchmarkResult struct {
	ModelID           ModelID       `json:"model_id"`
	RuntimeID         RuntimeID     `json:"runtime_id"`
	Latency           time.Duration `json:"latency"`
	FirstTokenLatency time.Duration `json:"first_token_latency"`
	TokensPerSecond   float64       `json:"tokens_per_second"`
	Success           bool          `json:"success"`
	Error             string        `json:"error,omitempty"`
}
type CapabilityVerificationStatus string

const (
	CapabilityVerified   CapabilityVerificationStatus = "VERIFIED"
	CapabilityUnverified CapabilityVerificationStatus = "UNVERIFIED"
	CapabilityFailed     CapabilityVerificationStatus = "FAILED"
)

type CapabilityVerification struct {
	ModelID    ModelID                      `json:"model_id"`
	RuntimeID  RuntimeID                    `json:"runtime_id"`
	Capability Capability                   `json:"capability"`
	Status     CapabilityVerificationStatus `json:"status"`
	Evidence   map[string]any               `json:"evidence,omitempty"`
	CheckedAt  time.Time                    `json:"checked_at"`
}

func Benchmark(ctx context.Context, runtime AIRuntime, request InferenceRequest) BenchmarkResult {
	started := time.Now()
	response, err := runtime.Generate(ctx, request)
	result := BenchmarkResult{ModelID: request.ModelID, RuntimeID: request.RuntimeID, Latency: time.Since(started), Success: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if response.Usage.TokensPerSecond > 0 {
		result.TokensPerSecond = response.Usage.TokensPerSecond
	}
	return result
}
func VerifyCapability(ctx context.Context, runtime AIRuntime, request InferenceRequest, capability Capability) CapabilityVerification {
	result := CapabilityVerification{ModelID: request.ModelID, RuntimeID: request.RuntimeID, Capability: capability, Status: CapabilityUnverified, CheckedAt: time.Now().UTC()}
	if capability == CapabilityToolUse {
		if len(request.Tools) == 0 {
			result.Evidence = map[string]any{"reason": "tool definition required"}
			return result
		}
		response, err := runtime.Generate(ctx, request)
		if err != nil {
			result.Status = CapabilityFailed
			result.Evidence = map[string]any{"error": err.Error()}
			return result
		}
		if len(response.ToolCalls) > 0 {
			result.Status = CapabilityVerified
		} else {
			result.Evidence = map[string]any{"reason": "no tool call returned"}
		}
		return result
	}
	return result
}
