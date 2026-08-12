package model

import (
	"errors"
	"sync"

	"suda-forge/internal/agent"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]agent.Provider
	models    map[string]agent.Model
}

func NewRegistry() *Registry {
	return &Registry{providers: map[string]agent.Provider{}, models: map[string]agent.Model{}}
}
func (r *Registry) RegisterProvider(p agent.Provider) error {
	if p.ID == "" || p.Name == "" {
		return errors.New("provider id and name are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[p.ID]; ok {
		return errors.New("provider already registered")
	}
	r.providers[p.ID] = p
	return nil
}
func (r *Registry) RegisterModel(m agent.Model) error {
	if m.ID == "" || m.ProviderID == "" || m.ModelID == "" {
		return errors.New("model id, provider id, and model id are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[m.ProviderID]; !ok {
		return errors.New("provider not registered")
	}
	if _, ok := r.models[m.ID]; ok {
		return errors.New("model already registered")
	}
	r.models[m.ID] = m
	return nil
}
func (r *Registry) Providers() []agent.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]agent.Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}
func (r *Registry) Models() []agent.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]agent.Model, 0, len(r.models))
	for _, m := range r.models {
		out = append(out, m)
	}
	return out
}
func (r *Registry) Provider(id string) (agent.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}
func (r *Registry) Model(id string) (agent.Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[id]
	return m, ok
}
func (r *Registry) ModelsForProvider(providerID string) []agent.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []agent.Model{}
	for _, m := range r.models {
		if m.ProviderID == providerID {
			out = append(out, m)
		}
	}
	return out
}

type AgentConfiguration struct {
	ID                    string                 `json:"id"`
	AgentID               agent.ID               `json:"agent_id"`
	Name                  string                 `json:"name"`
	Models                []agent.ModelReference `json:"models"`
	DefaultModel          *agent.ModelReference  `json:"default_model,omitempty"`
	CredentialReferenceID string                 `json:"credential_reference_id,omitempty"`
	Permissions           agent.PermissionPolicy `json:"permissions"`
}
