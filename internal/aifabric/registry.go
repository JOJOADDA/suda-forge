package aifabric

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var ErrRuntimeNotFound = errors.New("ai runtime not found")
var ErrModelNotFound = errors.New("ai model not found")

type RuntimeRegistry struct {
	mu       sync.RWMutex
	runtimes map[RuntimeID]AIRuntime
	models   map[ModelID]ModelDescriptor
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{runtimes: map[RuntimeID]AIRuntime{}, models: map[ModelID]ModelDescriptor{}}
}
func (r *RuntimeRegistry) RegisterRuntime(runtime AIRuntime) error {
	if runtime == nil || runtime.Spec().ID == "" {
		return errors.New("runtime and runtime id are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runtimes[runtime.Spec().ID] = runtime
	return nil
}
func (r *RuntimeRegistry) Runtime(id RuntimeID) (AIRuntime, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.runtimes[id]
	return v, ok
}
func (r *RuntimeRegistry) Runtimes() []AIRuntime {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]RuntimeID, 0, len(r.runtimes))
	for id := range r.runtimes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]AIRuntime, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.runtimes[id])
	}
	return out
}
func (r *RuntimeRegistry) RegisterModel(model ModelDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.ID] = model
}
func (r *RuntimeRegistry) Model(id ModelID) (ModelDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.models[id]
	return v, ok
}
func (r *RuntimeRegistry) Models() []ModelDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]ModelID, 0, len(r.models))
	for id := range r.models {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]ModelDescriptor, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.models[id])
	}
	return out
}
func (r *RuntimeRegistry) DiscoverAll(ctx context.Context) ([]ModelDescriptor, error) {
	var first error
	out := []ModelDescriptor{}
	for _, runtime := range r.Runtimes() {
		models, err := runtime.Discover(ctx)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		for _, model := range models {
			r.RegisterModel(model)
			out = append(out, model)
		}
	}
	return out, first
}
