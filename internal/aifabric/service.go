package aifabric

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	Registry *RuntimeRegistry
	Events   EventSink
	mu       sync.RWMutex
	health   map[RuntimeID]RuntimeHealth
}

func NewManager(registry *RuntimeRegistry, events EventSink) *Manager {
	if registry == nil {
		registry = NewRuntimeRegistry()
	}
	return &Manager{Registry: registry, Events: events, health: map[RuntimeID]RuntimeHealth{}}
}
func (m *Manager) Runtimes() []AIRuntime { return m.Registry.Runtimes() }
func (m *Manager) Runtime(id RuntimeID) (AIRuntime, error) {
	r, ok := m.Registry.Runtime(id)
	if !ok {
		return nil, ErrRuntimeNotFound
	}
	return r, nil
}
func (m *Manager) Discover(ctx context.Context) ([]ModelDescriptor, error) {
	models, first := m.Registry.DiscoverAll(ctx)
	for _, model := range models {
		if m.Events != nil {
			m.Events.Publish("ai.model.discovered", string(model.RuntimeID), model)
		}
	}
	return models, first
}
func (m *Manager) Health(ctx context.Context, id RuntimeID) (RuntimeHealth, error) {
	r, err := m.Runtime(id)
	if err != nil {
		return RuntimeHealth{}, err
	}
	health, err := r.Health(ctx)
	m.mu.Lock()
	m.health[id] = health
	m.mu.Unlock()
	if m.Events != nil {
		m.Events.Publish("ai.runtime.health_changed", string(id), health)
	}
	return health, err
}
func (m *Manager) HealthAll(ctx context.Context) map[RuntimeID]RuntimeHealth {
	out := map[RuntimeID]RuntimeHealth{}
	for _, r := range m.Runtimes() {
		health, _ := m.Health(ctx, r.Spec().ID)
		out[r.Spec().ID] = health
	}
	return out
}
func (m *Manager) CachedHealth() map[RuntimeID]RuntimeHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[RuntimeID]RuntimeHealth{}
	for k, v := range m.health {
		out[k] = v
	}
	return out
}
func (m *Manager) Start(ctx context.Context, id RuntimeID) error {
	return m.lifecycle(ctx, id, "started", func(r AIRuntime) error { return r.Start(ctx) })
}
func (m *Manager) Stop(ctx context.Context, id RuntimeID) error {
	return m.lifecycle(ctx, id, "stopped", func(r AIRuntime) error { return r.Stop(ctx) })
}
func (m *Manager) Restart(ctx context.Context, id RuntimeID) error {
	return m.lifecycle(ctx, id, "restarted", func(r AIRuntime) error { return r.Restart(ctx) })
}
func (m *Manager) lifecycle(ctx context.Context, id RuntimeID, event string, fn func(AIRuntime) error) error {
	r, err := m.Runtime(id)
	if err != nil {
		return err
	}
	if err = fn(r); err != nil {
		return err
	}
	if m.Events != nil {
		m.Events.Publish("ai.runtime."+event, string(id), r.Spec())
	}
	return nil
}
func (m *Manager) Install(ctx context.Context, req ModelInstallRequest) (ModelDescriptor, error) {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return ModelDescriptor{}, err
	}
	if !r.Capabilities().Install {
		return ModelDescriptor{}, errors.New("runtime does not support installation")
	}
	if m.Events != nil {
		m.Events.Publish("ai.model.install_started", string(req.RuntimeID), req)
	}
	model, err := r.Install(ctx, req)
	if err != nil {
		if m.Events != nil {
			m.Events.Publish("ai.model.failed", string(req.RuntimeID), map[string]any{"request": req, "error": err.Error()})
		}
		return ModelDescriptor{}, err
	}
	model.Status = ModelInstalled
	m.Registry.RegisterModel(model)
	if m.Events != nil {
		m.Events.Publish("ai.model.install_completed", string(req.RuntimeID), model)
	}
	return model, nil
}
func (m *Manager) Load(ctx context.Context, req ModelLoadRequest) error {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return err
	}
	if !r.Capabilities().Load {
		return errors.New("runtime does not support loading")
	}
	if m.Events != nil {
		m.Events.Publish("ai.model.load_started", string(req.RuntimeID), req)
	}
	if err = r.LoadModel(ctx, req); err != nil {
		return err
	}
	model, ok := m.Registry.Model(req.ModelID)
	if ok {
		model.Status = ModelReady
		m.Registry.RegisterModel(model)
		if m.Events != nil {
			m.Events.Publish("ai.model.ready", string(req.RuntimeID), model)
		}
	}
	return nil
}
func (m *Manager) Unload(ctx context.Context, req ModelLoadRequest) error {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return err
	}
	if err = r.UnloadModel(ctx, req); err != nil {
		return err
	}
	if model, ok := m.Registry.Model(req.ModelID); ok {
		model.Status = ModelInstalled
		m.Registry.RegisterModel(model)
	}
	if m.Events != nil {
		m.Events.Publish("ai.model.unloaded", string(req.RuntimeID), req)
	}
	return nil
}
func (m *Manager) Generate(ctx context.Context, req InferenceRequest) (InferenceResponse, error) {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return InferenceResponse{}, err
	}
	if m.Events != nil {
		m.Events.Publish("ai.request.started", req.ProjectID, req)
	}
	started := time.Now()
	response, err := r.Generate(ctx, req)
	response.Latency = time.Since(started)
	if err != nil {
		if m.Events != nil {
			m.Events.Publish("ai.request.failed", req.ProjectID, map[string]any{"request_id": req.RequestID, "error": err.Error()})
		}
		return InferenceResponse{}, err
	}
	if m.Events != nil {
		m.Events.Publish("ai.request.completed", req.ProjectID, response)
	}
	return response, nil
}
func (m *Manager) Stream(ctx context.Context, req InferenceRequest) (<-chan StreamEvent, error) {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return nil, err
	}
	ch, err := r.Stream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan StreamEvent, 16)
	go func() {
		defer close(out)
		for event := range ch {
			event.RequestID = req.RequestID
			if m.Events != nil && event.Type == "token" {
				m.Events.Publish("ai.request.token", req.ProjectID, event)
			}
			out <- event
		}
		if m.Events != nil {
			m.Events.Publish("ai.request.completed", req.ProjectID, map[string]any{"request_id": req.RequestID})
		}
	}()
	return out, nil
}
func (m *Manager) Embeddings(ctx context.Context, req InferenceRequest) ([][]float32, error) {
	r, err := m.Runtime(req.RuntimeID)
	if err != nil {
		return nil, err
	}
	if !r.Capabilities().Embeddings {
		return nil, fmt.Errorf("runtime does not support embeddings")
	}
	return r.Embeddings(ctx, req)
}
