package agent

import (
	"errors"
	"fmt"
	"time"
)

type ID string
type SessionID string
type ModelReference struct {
	ProviderID      string `json:"provider_id"`
	ModelID         string `json:"model_id"`
	ConfigurationID string `json:"configuration_id,omitempty"`
}
type Capability string

const (
	CapabilityTerminal         Capability = "terminal"
	CapabilityFilesystem       Capability = "filesystem"
	CapabilityGit              Capability = "git"
	CapabilityBrowser          Capability = "browser"
	CapabilityWeb              Capability = "web"
	CapabilityToolCalling      Capability = "tool_calling"
	CapabilityInteractive      Capability = "interactive"
	CapabilityStreaming        Capability = "streaming"
	CapabilityStructuredOutput Capability = "structured_output"
	CapabilityVision           Capability = "vision"
	CapabilityParallelTasks    Capability = "parallel_tasks"
	CapabilityLongRunning      Capability = "long_running"
	CapabilityHumanHandoff     Capability = "human_handoff"
)

type AgentDefinition struct {
	ID                   ID                `json:"id"`
	Name                 string            `json:"name"`
	DisplayName          string            `json:"display_name"`
	Adapter              string            `json:"adapter"`
	Version              string            `json:"version"`
	Description          string            `json:"description"`
	Capabilities         []Capability      `json:"capabilities"`
	RuntimeRequirements  map[string]string `json:"runtime_requirements,omitempty"`
	AuthenticationMethod string            `json:"authentication_method"`
	Configuration        map[string]string `json:"configuration,omitempty"`
	Status               string            `json:"status"`
}
type Provider struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Type               string            `json:"type"`
	BaseURL            string            `json:"base_url,omitempty"`
	AuthenticationType string            `json:"authentication_type"`
	Status             string            `json:"status"`
	Configuration      map[string]string `json:"configuration,omitempty"`
}
type Model struct {
	ID                                                                  string `json:"id"`
	ProviderID                                                          string `json:"provider_id"`
	ModelID                                                             string `json:"model_id"`
	DisplayName                                                         string `json:"display_name"`
	ContextWindow                                                       int    `json:"context_window"`
	Reasoning, Coding, Vision, ToolUse, StructuredOutput, Local, Remote bool
	InputCost, OutputCost                                               float64
	LatencyClass, Availability                                          string
	Metadata                                                            map[string]string `json:"metadata,omitempty"`
}
type Permission string

const (
	PermissionFilesystemRead    Permission = "filesystem.read"
	PermissionFilesystemWrite   Permission = "filesystem.write"
	PermissionTerminalExecute   Permission = "terminal.execute"
	PermissionGitCommit         Permission = "git.commit"
	PermissionGitPush           Permission = "git.push"
	PermissionBrowserNavigate   Permission = "browser.navigate"
	PermissionNetworkAccess     Permission = "network.access"
	PermissionSecretRead        Permission = "secret.read"
	PermissionDeploymentExecute Permission = "deployment.execute"
)

type PermissionPolicy struct {
	ProjectID        string       `json:"project_id"`
	AgentID          ID           `json:"agent_id"`
	Allowed          []Permission `json:"allowed"`
	ApprovalRequired []Permission `json:"approval_required,omitempty"`
}
type CredentialReference struct {
	ID         string            `json:"id"`
	ProjectID  string            `json:"project_id"`
	ProviderID string            `json:"provider_id"`
	Kind       string            `json:"kind"`
	SecretName string            `json:"secret_name"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
type SessionStatus string

const (
	SessionCreated    SessionStatus = "CREATED"
	SessionStarting   SessionStatus = "STARTING"
	SessionRunning    SessionStatus = "RUNNING"
	SessionPaused     SessionStatus = "PAUSED"
	SessionWaiting    SessionStatus = "WAITING"
	SessionCompleting SessionStatus = "COMPLETING"
	SessionCompleted  SessionStatus = "COMPLETED"
	SessionFailed     SessionStatus = "FAILED"
	SessionCancelled  SessionStatus = "CANCELLED"
	SessionRecovering SessionStatus = "RECOVERING"
)

type Session struct {
	ID                   SessionID      `json:"id"`
	ProjectID            string         `json:"project_id"`
	AgentID              ID             `json:"agent_id"`
	Model                ModelReference `json:"model_reference"`
	RuntimeID            string         `json:"runtime_id"`
	WorkingDirectory     string         `json:"working_directory"`
	Status               SessionStatus  `json:"status"`
	CreatedAt, UpdatedAt time.Time      `json:"created_at"`
}

var ErrInvalidSessionTransition = errors.New("invalid agent session transition")

func (s *Session) Transition(next SessionStatus, now time.Time) error {
	if !validTransition(s.Status, next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidSessionTransition, s.Status, next)
	}
	s.Status = next
	s.UpdatedAt = now
	return nil
}
func validTransition(from, to SessionStatus) bool {
	switch from {
	case SessionCreated:
		return to == SessionStarting || to == SessionCancelled
	case SessionStarting:
		return to == SessionRunning || to == SessionFailed || to == SessionCancelled
	case SessionRunning:
		return to == SessionPaused || to == SessionWaiting || to == SessionCompleting || to == SessionFailed || to == SessionCancelled
	case SessionPaused:
		return to == SessionRunning || to == SessionCancelled || to == SessionFailed
	case SessionWaiting:
		return to == SessionRunning || to == SessionCancelled || to == SessionFailed
	case SessionCompleting:
		return to == SessionCompleted || to == SessionFailed
	case SessionFailed:
		return to == SessionRecovering || to == SessionCancelled
	case SessionRecovering:
		return to == SessionStarting || to == SessionFailed || to == SessionCancelled
	default:
		return false
	}
}

type EventType string

const (
	EventSessionCreated     EventType = "agent.session.created"
	EventSessionStarted     EventType = "agent.session.started"
	EventMessage            EventType = "agent.message"
	EventThinking           EventType = "agent.thinking"
	EventToolStarted        EventType = "agent.tool.started"
	EventToolOutput         EventType = "agent.tool.output"
	EventFileChanged        EventType = "agent.file.changed"
	EventCommandStarted     EventType = "agent.command.started"
	EventCommandOutput      EventType = "agent.command.output"
	EventCommandCompleted   EventType = "agent.command.completed"
	EventError              EventType = "agent.error"
	EventPermissionRequired EventType = "agent.permission.required"
	EventUserInputRequired  EventType = "agent.user.input.required"
	EventSessionCompleted   EventType = "agent.session.completed"
)

type Usage struct {
	Provider      string  `json:"provider,omitempty"`
	Model         string  `json:"model,omitempty"`
	InputTokens   int     `json:"input_tokens,omitempty"`
	OutputTokens  int     `json:"output_tokens,omitempty"`
	DurationMS    int64   `json:"duration_ms,omitempty"`
	EstimatedCost float64 `json:"estimated_cost,omitempty"`
}
type Event struct {
	ID               string         `json:"id"`
	SessionID        SessionID      `json:"session_id"`
	Type             EventType      `json:"type"`
	Timestamp        time.Time      `json:"timestamp"`
	Normalized       map[string]any `json:"normalized,omitempty"`
	Raw              map[string]any `json:"raw,omitempty"`
	Usage            *Usage         `json:"usage,omitempty"`
	RequiresApproval bool           `json:"requires_approval,omitempty"`
}
