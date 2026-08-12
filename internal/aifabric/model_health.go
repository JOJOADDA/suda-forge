package aifabric

import (
	"context"
	"time"
)

type ModelHealthStatus string

const (
	ModelHealthAvailable   ModelHealthStatus = "AVAILABLE"
	ModelHealthUnavailable ModelHealthStatus = "UNAVAILABLE"
	ModelHealthLoading     ModelHealthStatus = "LOADING"
	ModelHealthLoaded      ModelHealthStatus = "LOADED"
	ModelHealthError       ModelHealthStatus = "ERROR"
	ModelHealthUnknown     ModelHealthStatus = "UNKNOWN"
)

type ModelHealth struct {
	RuntimeID           RuntimeID           `json:"runtime_id"`
	ModelID             ModelID             `json:"model_id"`
	Status              ModelHealthStatus   `json:"status"`
	LastChecked         time.Time           `json:"last_checked"`
	Latency             time.Duration       `json:"latency"`
	ErrorCount          int                 `json:"error_count"`
	SuccessfulRequests  int                 `json:"successful_requests"`
	TokensPerSecond     float64             `json:"tokens_per_second,omitempty"`
	ContextLimit        int                 `json:"context_limit,omitempty"`
	ResourceRequirement ResourceRequirement `json:"resource_requirement,omitempty"`
}

func (m *Manager) ModelHealth(ctx context.Context) []ModelHealth {
	snapshots := []ModelHealth{}
	for _, model := range m.Registry.Models() {
		health, ok := m.CachedHealth()[model.RuntimeID]
		status := ModelHealthUnknown
		if ok {
			status = ModelHealthUnavailable
			if health.Status == RuntimeOnline || health.Status == RuntimeDegraded {
				status = ModelHealthAvailable
			}
			if health.Status == RuntimeError {
				status = ModelHealthError
			}
		}
		snapshots = append(snapshots, ModelHealth{RuntimeID: model.RuntimeID, ModelID: model.ID, Status: status, LastChecked: health.LastChecked, Latency: health.Latency, ContextLimit: model.ContextWindow, ResourceRequirement: model.ResourceRequirement})
	}
	return snapshots
}
