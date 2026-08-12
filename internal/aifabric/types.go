package aifabric

import (
	"context"
	"time"
)

type RuntimeID string
type ModelID string
type RequestID string

type RuntimeStatus string

const (
	RuntimeOnline   RuntimeStatus = "ONLINE"
	RuntimeDegraded RuntimeStatus = "DEGRADED"
	RuntimeOffline  RuntimeStatus = "OFFLINE"
	RuntimeStarting RuntimeStatus = "STARTING"
	RuntimeStopping RuntimeStatus = "STOPPING"
	RuntimeError    RuntimeStatus = "ERROR"
)

type ModelLifecycle string

const (
	ModelDiscovered ModelLifecycle = "DISCOVERED"
	ModelAvailable  ModelLifecycle = "AVAILABLE"
	ModelInstalling ModelLifecycle = "INSTALLING"
	ModelInstalled  ModelLifecycle = "INSTALLED"
	ModelLoading    ModelLifecycle = "LOADING"
	ModelReady      ModelLifecycle = "READY"
	ModelUnloading  ModelLifecycle = "UNLOADING"
	ModelRemoving   ModelLifecycle = "REMOVING"
	ModelFailed     ModelLifecycle = "FAILED"
)

type Capability string

const (
	CapabilityCoding           Capability = "CODING"
	CapabilityReasoning        Capability = "REASONING"
	CapabilityArchitecture     Capability = "ARCHITECTURE"
	CapabilityFast             Capability = "FAST"
	CapabilityVision           Capability = "VISION"
	CapabilityLongContext      Capability = "LONG_CONTEXT"
	CapabilityAgentic          Capability = "AGENTIC"
	CapabilityPrivate          Capability = "PRIVATE"
	CapabilityLocal            Capability = "LOCAL"
	CapabilityCheap            Capability = "CHEAP"
	CapabilityGeneral          Capability = "GENERAL"
	CapabilityToolUse          Capability = "TOOL_USE"
	CapabilityStructuredOutput Capability = "STRUCTURED_OUTPUT"
	CapabilityEmbedding        Capability = "EMBEDDING"
)

type RuntimeCapabilities struct {
	Generate         bool `json:"generate"`
	Stream           bool `json:"stream"`
	Embeddings       bool `json:"embeddings"`
	Install          bool `json:"install"`
	Load             bool `json:"load"`
	Unload           bool `json:"unload"`
	Vision           bool `json:"vision"`
	StructuredOutput bool `json:"structured_output"`
	ToolCalling      bool `json:"tool_calling"`
	ModelDiscovery   bool `json:"model_discovery"`
	OpenAICompatible bool `json:"openai_compatible"`
}
type RuntimeSpec struct {
	ID            RuntimeID         `json:"id"`
	Kind          string            `json:"kind"`
	Endpoint      string            `json:"endpoint"`
	Local         bool              `json:"local"`
	AutoStart     bool              `json:"auto_start"`
	Configuration map[string]string `json:"configuration,omitempty"`
}
type RuntimeHealth struct {
	RuntimeID       RuntimeID     `json:"runtime_id"`
	Status          RuntimeStatus `json:"status"`
	Version         string        `json:"version,omitempty"`
	Endpoint        string        `json:"endpoint"`
	Latency         time.Duration `json:"latency"`
	CPU             float64       `json:"cpu,omitempty"`
	Memory          uint64        `json:"memory,omitempty"`
	GPU             int           `json:"gpu_count,omitempty"`
	GPUMemory       uint64        `json:"gpu_memory,omitempty"`
	LoadedModels    []string      `json:"loaded_models,omitempty"`
	AvailableModels []string      `json:"available_models,omitempty"`
	LastChecked     time.Time     `json:"last_checked"`
	Error           string        `json:"error,omitempty"`
}
type ModelDescriptor struct {
	ID                  ModelID             `json:"id"`
	ProviderID          string              `json:"provider_id"`
	RuntimeID           RuntimeID           `json:"runtime_id"`
	DisplayName         string              `json:"display_name"`
	Version             string              `json:"version,omitempty"`
	ContextWindow       int                 `json:"context_window"`
	MaxOutputTokens     int                 `json:"max_output_tokens,omitempty"`
	Capabilities        map[Capability]bool `json:"capabilities"`
	Local               bool                `json:"local"`
	Remote              bool                `json:"remote"`
	PrivacyLevel        string              `json:"privacy_level,omitempty"`
	Availability        string              `json:"availability"`
	Status              ModelLifecycle      `json:"status"`
	Latency             time.Duration       `json:"latency,omitempty"`
	ResourceRequirement ResourceRequirement `json:"resource_requirement,omitempty"`
}
type ResourceRequirement struct {
	MemoryBytes uint64 `json:"memory_bytes,omitempty"`
	VRAMBytes   uint64 `json:"vram_bytes,omitempty"`
	GPURequired bool   `json:"gpu_required,omitempty"`
}
type ModelInstallRequest struct {
	ModelID              ModelID             `json:"model_id"`
	RuntimeID            RuntimeID           `json:"runtime_id"`
	Source               string              `json:"source"`
	Quantization         string              `json:"quantization,omitempty"`
	Revision             string              `json:"revision,omitempty"`
	ResourceRequirements ResourceRequirement `json:"resource_requirements,omitempty"`
}
type ModelLoadRequest struct {
	ModelID   ModelID   `json:"model_id"`
	RuntimeID RuntimeID `json:"runtime_id"`
	Warm      bool      `json:"warm"`
}
type RuntimeHealthRequest struct {
	RuntimeID RuntimeID `json:"runtime_id"`
}

type HostResources struct {
	CPUCores           int           `json:"cpu_cores"`
	MemoryTotal        uint64        `json:"memory_total"`
	MemoryAvailable    uint64        `json:"memory_available"`
	DiskTotal          uint64        `json:"disk_total"`
	DiskAvailable      uint64        `json:"disk_available"`
	GPUCount           int           `json:"gpu_count"`
	GPUMemoryTotal     uint64        `json:"gpu_memory_total"`
	GPUMemoryAvailable uint64        `json:"gpu_memory_available"`
	GPUs               []GPUResource `json:"gpus,omitempty"`
	DetectedAt         time.Time     `json:"detected_at"`
}
type GPUResource struct {
	ID                string `json:"id"`
	Vendor            string `json:"vendor"`
	Model             string `json:"model"`
	MemoryTotal       uint64 `json:"memory_total"`
	MemoryAvailable   uint64 `json:"memory_available"`
	ComputeCapability string `json:"compute_capability,omitempty"`
	Driver            string `json:"driver,omitempty"`
	Runtime           string `json:"runtime,omitempty"`
}
type GPUAllocation struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	RuntimeID   RuntimeID `json:"runtime_id"`
	GPUIDs      []string  `json:"gpu_ids"`
	MemoryBytes uint64    `json:"memory_bytes"`
	Status      string    `json:"status"`
}

type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
}
type VisionInput struct {
	MIMEType string `json:"mime_type"`
	Data     []byte `json:"data,omitempty"`
	URL      string `json:"url,omitempty"`
}
type InferenceRequest struct {
	RequestID        RequestID        `json:"request_id"`
	ProjectID        string           `json:"project_id"`
	AgentID          string           `json:"agent_id,omitempty"`
	TaskID           string           `json:"task_id,omitempty"`
	ModelID          ModelID          `json:"model_id"`
	RuntimeID        RuntimeID        `json:"runtime_id"`
	Messages         []Message        `json:"messages"`
	SystemPrompt     string           `json:"system_prompt,omitempty"`
	Tools            []ToolDefinition `json:"tools,omitempty"`
	Temperature      *float64         `json:"temperature,omitempty"`
	MaxTokens        int              `json:"max_tokens,omitempty"`
	Context          map[string]any   `json:"context,omitempty"`
	StructuredOutput map[string]any   `json:"structured_output,omitempty"`
	VisionInputs     []VisionInput    `json:"vision_inputs,omitempty"`
	Stream           bool             `json:"stream"`
}
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
type Usage struct {
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedCost   float64 `json:"estimated_cost"`
	CPUMillis       int64   `json:"cpu_millis,omitempty"`
	GPUMillis       int64   `json:"gpu_millis,omitempty"`
	TokensPerSecond float64 `json:"tokens_per_second,omitempty"`
}
type InferenceResponse struct {
	RequestID    RequestID      `json:"request_id"`
	ModelID      ModelID        `json:"model_id"`
	RuntimeID    RuntimeID      `json:"runtime_id"`
	Content      string         `json:"content"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	Usage        Usage          `json:"usage"`
	Latency      time.Duration  `json:"latency"`
	FinishReason string         `json:"finish_reason"`
	RawMetadata  map[string]any `json:"raw_metadata,omitempty"`
}
type StreamEvent struct {
	RequestID RequestID `json:"request_id"`
	Type      string    `json:"type"`
	Delta     string    `json:"delta,omitempty"`
	Message   *Message  `json:"message,omitempty"`
	ToolCall  *ToolCall `json:"tool_call,omitempty"`
	Usage     *Usage    `json:"usage,omitempty"`
	Error     string    `json:"error,omitempty"`
	Done      bool      `json:"done,omitempty"`
}
type HealthSnapshot struct {
	Runtime RuntimeHealth     `json:"runtime"`
	Models  []ModelDescriptor `json:"models"`
}

type AIRuntime interface {
	Spec() RuntimeSpec
	Discover(context.Context) ([]ModelDescriptor, error)
	Install(context.Context, ModelInstallRequest) (ModelDescriptor, error)
	Remove(context.Context, ModelID) error
	Start(context.Context) error
	Stop(context.Context) error
	Restart(context.Context) error
	Health(context.Context) (RuntimeHealth, error)
	ListModels(context.Context) ([]ModelDescriptor, error)
	LoadModel(context.Context, ModelLoadRequest) error
	UnloadModel(context.Context, ModelLoadRequest) error
	Generate(context.Context, InferenceRequest) (InferenceResponse, error)
	Stream(context.Context, InferenceRequest) (<-chan StreamEvent, error)
	Embeddings(context.Context, InferenceRequest) ([][]float32, error)
	Capabilities() RuntimeCapabilities
	Resources(context.Context) (HostResources, error)
}

type EventSink interface{ Publish(string, string, any) }
