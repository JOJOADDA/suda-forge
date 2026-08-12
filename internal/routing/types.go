package routing

import "time"

type Capability string

const (
	Coding           Capability = "coding"
	Reasoning        Capability = "reasoning"
	Architecture     Capability = "architecture"
	Debugging        Capability = "debugging"
	Refactoring      Capability = "refactoring"
	Frontend         Capability = "frontend"
	Backend          Capability = "backend"
	Database         Capability = "database"
	DevOps           Capability = "devops"
	Security         Capability = "security"
	Testing          Capability = "testing"
	Documentation    Capability = "documentation"
	Vision           Capability = "vision"
	ToolUse          Capability = "tool_use"
	StructuredOutput Capability = "structured_output"
	LongContext      Capability = "long_context"
	FastResponse     Capability = "fast_response"
)

type Availability string

const (
	Available   Availability = "AVAILABLE"
	Degraded    Availability = "DEGRADED"
	Unavailable Availability = "UNAVAILABLE"
	Disabled    Availability = "DISABLED"
	Unknown     Availability = "UNKNOWN"
)

type PrivacyClass string

const (
	Public       PrivacyClass = "PUBLIC"
	Internal     PrivacyClass = "INTERNAL"
	Confidential PrivacyClass = "CONFIDENTIAL"
	Private      PrivacyClass = "PRIVATE"
)

type LocalPolicy string

const (
	LocalPolicyAny LocalPolicy = "REMOTE_ALLOWED"
	LocalFirst     LocalPolicy = "LOCAL_FIRST"
	LocalOnly      LocalPolicy = "LOCAL_ONLY"
)

type RoutingPolicy string

const (
	Best         RoutingPolicy = "BEST"
	Cheapest     RoutingPolicy = "CHEAPEST"
	Fastest      RoutingPolicy = "FASTEST"
	PrivacyFirst RoutingPolicy = "PRIVACY_FIRST"
	Balanced     RoutingPolicy = "BALANCED"
	CloudFirst   RoutingPolicy = "CLOUD_FIRST"
	Custom       RoutingPolicy = "CUSTOM"
)

type TaskType string

const (
	TaskCode          TaskType = "CODE"
	TaskRefactor      TaskType = "REFACTOR"
	TaskDebug         TaskType = "DEBUG"
	TaskArchitecture  TaskType = "ARCHITECTURE"
	TaskUI            TaskType = "UI"
	TaskDatabase      TaskType = "DATABASE"
	TaskDevOps        TaskType = "DEVOPS"
	TaskSecurity      TaskType = "SECURITY"
	TaskTest          TaskType = "TEST"
	TaskDocumentation TaskType = "DOCUMENTATION"
	TaskGeneral       TaskType = "GENERAL"
)

type TaskProfile struct {
	TaskType           TaskType     `json:"task_type"`
	Language           string       `json:"language,omitempty"`
	Framework          string       `json:"framework,omitempty"`
	Complexity         int          `json:"complexity"`
	ReasoningRequired  bool         `json:"reasoning_required"`
	ContextRequired    int          `json:"context_required"`
	VisionRequired     bool         `json:"vision_required"`
	ToolUseRequired    bool         `json:"tool_use_required"`
	Privacy            PrivacyClass `json:"privacy_classification"`
	LatencyRequirement string       `json:"latency_requirement,omitempty"`
	BudgetLimit        float64      `json:"budget_limit,omitempty"`
	UserPreference     string       `json:"user_preference,omitempty"`
}
type PerformanceProfile struct {
	ReasoningScore, CodingScore, ToolUseScore, VisionScore, ContextScore, ReliabilityScore float64
	LatencyClass, CostClass                                                                string
	Source                                                                                 string `json:"source"`
}
type Pricing struct {
	InputCost     float64   `json:"input_cost"`
	OutputCost    float64   `json:"output_cost"`
	Currency      string    `json:"currency"`
	PricingUnit   string    `json:"pricing_unit"`
	EffectiveDate time.Time `json:"effective_date"`
}
type ModelResourceRequirement struct {
	MemoryBytes uint64 `json:"memory_bytes,omitempty"`
	VRAMBytes   uint64 `json:"vram_bytes,omitempty"`
	GPURequired bool   `json:"gpu_required,omitempty"`
}
type ModelProfile struct {
	ModelID         string                   `json:"model_id"`
	ProviderID      string                   `json:"provider_id"`
	RuntimeID       string                   `json:"runtime_id,omitempty"`
	DisplayName     string                   `json:"display_name"`
	Capabilities    map[Capability]bool      `json:"capabilities"`
	Performance     PerformanceProfile       `json:"performance"`
	Pricing         Pricing                  `json:"pricing"`
	Availability    Availability             `json:"availability"`
	Local           bool                     `json:"local"`
	Remote          bool                     `json:"remote"`
	ContextWindow   int                      `json:"context_window"`
	PrivacyLevel    string                   `json:"privacy_level,omitempty"`
	Resources       ModelResourceRequirement `json:"resources,omitempty"`
	RuntimeHealthy  bool                     `json:"runtime_healthy"`
	SupportedAgents []string                 `json:"supported_agents,omitempty"`
}
type ProviderHealth struct {
	ProviderID         string       `json:"provider_id"`
	Status             Availability `json:"status"`
	LatencyMS          int64        `json:"latency_ms"`
	RateLimitRemaining int          `json:"rate_limit_remaining,omitempty"`
	AuthenticationOK   bool         `json:"authentication_ok"`
	CheckedAt          time.Time    `json:"checked_at"`
}
type HealthCache interface {
	Get(string) (ProviderHealth, bool)
	Put(ProviderHealth)
}
type RoutingRequest struct {
	ProjectID          string           `json:"project_id"`
	AgentID            string           `json:"agent_id"`
	Task               TaskProfile      `json:"task_profile"`
	Policy             RoutingPolicy    `json:"routing_policy"`
	PrivacyLimit       PrivacyClass     `json:"privacy_policy"`
	LocalPolicy        LocalPolicy      `json:"local_policy"`
	Budget             float64          `json:"budget"`
	AvailableRuntime   bool             `json:"available_runtime"`
	RuntimeHealthy     bool             `json:"runtime_healthy"`
	Offline            bool             `json:"offline"`
	AvailableMemory    uint64           `json:"available_memory,omitempty"`
	AvailableVRAM      uint64           `json:"available_vram,omitempty"`
	GPUAvailable       bool             `json:"gpu_available,omitempty"`
	Models             []ModelProfile   `json:"available_models"`
	Fallbacks          []ModelReference `json:"fallbacks,omitempty"`
	UserOverride       *ModelReference  `json:"user_override,omitempty"`
	ProjectPolicy      *RoutingPolicy   `json:"project_policy,omitempty"`
	OrganizationPolicy *RoutingPolicy   `json:"organization_policy,omitempty"`
	GlobalPolicy       *RoutingPolicy   `json:"global_policy,omitempty"`
}
type ModelReference struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
}
type CandidateResult struct {
	Model         ModelReference `json:"model"`
	Accepted      bool           `json:"accepted"`
	Score         float64        `json:"score"`
	Reasons       []string       `json:"reasons"`
	Rejections    []string       `json:"rejections,omitempty"`
	EstimatedCost float64        `json:"estimated_cost"`
}
type RoutingDecision struct {
	Selected           ModelReference    `json:"selected_model"`
	ProviderID         string            `json:"provider"`
	Reason             string            `json:"reason"`
	Alternatives       []CandidateResult `json:"alternatives"`
	Rejected           []CandidateResult `json:"rejected_candidates,omitempty"`
	ConstraintsApplied []string          `json:"constraints_applied"`
	EstimatedCost      float64           `json:"estimated_cost"`
	Confidence         float64           `json:"confidence"`
}
type UsageEvent struct {
	Type          string        `json:"type"`
	ProviderID    string        `json:"provider"`
	ModelID       string        `json:"model"`
	InputTokens   int           `json:"input_tokens"`
	OutputTokens  int           `json:"output_tokens"`
	Duration      time.Duration `json:"duration"`
	EstimatedCost float64       `json:"estimated_cost"`
}
