package agent

import (
	"context"
	"errors"
	"sync"
	"time"

	"suda-forge/internal/runtime"
)

type Adapter interface {
	ID() string
	Start(context.Context, Session) error
	Stop(context.Context, SessionID) error
	Resume(context.Context, Session) error
	SendMessage(context.Context, SessionID, string) error
	StreamEvents(context.Context, SessionID) (<-chan Event, error)
	Cancel(context.Context, SessionID) error
	Status(context.Context, SessionID) (SessionStatus, error)
	Capabilities() []Capability
}
type Installer interface {
	Install(context.Context, string, string) error
	Uninstall(context.Context, string, string) error
	Detect(context.Context, string, string) (bool, error)
	Version(context.Context, string, string) (string, error)
	Health(context.Context, string, string) error
}
type CredentialResolver interface {
	Resolve(context.Context, CredentialReference) (map[string]string, error)
}
type ProcessManager interface {
	Start(context.Context, runtime.Spec, runtime.Command) (string, error)
	Write(context.Context, string, []byte) error
	Stop(context.Context, string) error
	Status(context.Context, string) (string, error)
	Output(context.Context, string) (<-chan string, error)
}
type Registry struct {
	mu          sync.RWMutex
	adapters    map[string]Adapter
	definitions map[ID]AgentDefinition
}

func NewRegistry() *Registry {
	return &Registry{adapters: map[string]Adapter{}, definitions: map[ID]AgentDefinition{}}
}
func (r *Registry) RegisterDefinition(definition AgentDefinition) error {
	if definition.ID == "" || definition.Adapter == "" {
		return errors.New("agent definition id and adapter are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[definition.ID]; exists {
		return errors.New("agent definition already registered")
	}
	r.definitions[definition.ID] = definition
	return nil
}
func (r *Registry) Definitions() []AgentDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AgentDefinition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		out = append(out, definition)
	}
	return out
}

func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil || adapter.ID() == "" {
		return errors.New("adapter id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[adapter.ID()]; exists {
		return errors.New("adapter already registered")
	}
	r.adapters[adapter.ID()] = adapter
	return nil
}
func (r *Registry) Get(id string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[id]
	return a, ok
}
func (r *Registry) List() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	return out
}

// MockAdapter is TEST ONLY. It never runs a host command or claims real CLI execution.
type MockAdapter struct {
	mu       sync.Mutex
	sessions map[SessionID]SessionStatus
	events   map[SessionID]chan Event
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{sessions: map[SessionID]SessionStatus{}, events: map[SessionID]chan Event{}}
}
func (m *MockAdapter) ID() string { return "mock-test-only" }
func (m *MockAdapter) Capabilities() []Capability {
	return []Capability{CapabilityStreaming, CapabilityInteractive, CapabilityToolCalling}
}
func (m *MockAdapter) Start(_ context.Context, s Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = SessionRunning
	ch := make(chan Event, 8)
	m.events[s.ID] = ch
	ch <- Event{SessionID: s.ID, Type: EventSessionStarted, Timestamp: time.Now().UTC()}
	ch <- Event{SessionID: s.ID, Type: EventMessage, Timestamp: time.Now().UTC(), Normalized: map[string]any{"text": "TEST ONLY mock agent started"}}
	return nil
}
func (m *MockAdapter) Stop(_ context.Context, id SessionID) error { return m.set(id, SessionCompleted) }
func (m *MockAdapter) Resume(_ context.Context, id Session) error {
	return m.set(id.ID, SessionRunning)
}
func (m *MockAdapter) SendMessage(_ context.Context, id SessionID, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch := m.events[id]; ch != nil {
		ch <- Event{SessionID: id, Type: EventMessage, Timestamp: time.Now().UTC(), Normalized: map[string]any{"text": message}}
	}
	return nil
}
func (m *MockAdapter) StreamEvents(_ context.Context, id SessionID) (<-chan Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.events[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	return ch, nil
}
func (m *MockAdapter) Cancel(_ context.Context, id SessionID) error {
	return m.set(id, SessionCancelled)
}
func (m *MockAdapter) Status(_ context.Context, id SessionID) (SessionStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status, ok := m.sessions[id]
	if !ok {
		return "", errors.New("session not found")
	}
	return status, nil
}
func (m *MockAdapter) set(id SessionID, status SessionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return errors.New("session not found")
	}
	m.sessions[id] = status
	if ch := m.events[id]; ch != nil {
		ch <- Event{SessionID: id, Type: EventSessionCompleted, Timestamp: time.Now().UTC()}
	}
	return nil
}
