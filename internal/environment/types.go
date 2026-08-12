package environment

import "time"

type Profile string

const (
	Minimal  Profile = "MINIMAL"
	Standard Profile = "STANDARD"
	Full     Profile = "FULL"
	Custom   Profile = "CUSTOM"
)

type Requirement struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
	Source   string `json:"source,omitempty"`
}
type RuntimeRequirement = Requirement
type ToolRequirement = Requirement
type BrowserRequirement struct {
	Name       string `json:"name"`
	Engine     string `json:"engine"`
	Automation string `json:"automation"`
	Version    string `json:"version"`
	Required   bool   `json:"required"`
}
type AgentRequirement struct {
	AgentID  string `json:"agent_id"`
	Version  string `json:"version"`
	Required bool   `json:"required"`
}
type EnvironmentVariable struct {
	Name      string `json:"name"`
	Value     string `json:"value,omitempty"`
	SecretRef string `json:"secret_ref,omitempty"`
	Required  bool   `json:"required"`
}
type PortRequirement struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Required bool   `json:"required"`
}
type ResourceRequirement struct {
	CPU            int   `json:"cpu"`
	MemoryBytes    int64 `json:"memory_bytes"`
	DiskBytes      int64 `json:"disk_bytes"`
	GPU            bool  `json:"gpu"`
	GPUMemoryBytes int64 `json:"gpu_memory_bytes"`
}
type Manifest struct {
	ID              string                `json:"id"`
	ProjectID       string                `json:"project_id"`
	Version         string                `json:"version"`
	BaseImage       string                `json:"base_image"`
	OS              string                `json:"os"`
	Architecture    string                `json:"architecture"`
	Profile         Profile               `json:"profile"`
	Languages       []RuntimeRequirement  `json:"languages"`
	PackageManagers []ToolRequirement     `json:"package_managers"`
	Frameworks      []ToolRequirement     `json:"frameworks"`
	BuildTools      []ToolRequirement     `json:"build_tools"`
	TestTools       []ToolRequirement     `json:"test_tools"`
	Browsers        []BrowserRequirement  `json:"browsers"`
	AgentCLIs       []AgentRequirement    `json:"agent_clis"`
	SDKs            []ToolRequirement     `json:"sdks"`
	SystemPackages  []ToolRequirement     `json:"system_packages"`
	EnvironmentVars []EnvironmentVariable `json:"environment_vars"`
	Ports           []PortRequirement     `json:"ports"`
	Resources       ResourceRequirement   `json:"resources"`
	DecisionID      string                `json:"decision_id"`
	Fingerprint     string                `json:"fingerprint,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
}
type Fingerprint struct {
	Value          string            `json:"value"`
	OS             string            `json:"os"`
	Image          string            `json:"image"`
	Languages      map[string]string `json:"languages"`
	Frameworks     map[string]string `json:"frameworks"`
	Tools          map[string]string `json:"tools"`
	Agents         map[string]string `json:"agents"`
	Browser        string            `json:"browser,omitempty"`
	SystemPackages []string          `json:"system_packages"`
	CreatedAt      time.Time         `json:"created_at"`
}
type PlanDecision string

const (
	Approved PlanDecision = "APPROVED"
	Degraded PlanDecision = "DEGRADED"
	Rejected PlanDecision = "REJECTED"
)

type ResourcePlan struct {
	Decision  PlanDecision        `json:"decision"`
	Reasons   []string            `json:"reasons"`
	Available ResourceRequirement `json:"available"`
	Required  ResourceRequirement `json:"required"`
}
type VerificationStatus string

const (
	VerificationReady    VerificationStatus = "READY"
	VerificationDegraded VerificationStatus = "DEGRADED"
	VerificationFailed   VerificationStatus = "FAILED"
	VerificationBlocked  VerificationStatus = "BLOCKED_BY_ENVIRONMENT"
)

type VerificationCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
type VerificationResult struct {
	Status      VerificationStatus  `json:"status"`
	Checks      []VerificationCheck `json:"checks"`
	Fingerprint Fingerprint         `json:"fingerprint"`
	RuntimeID   string              `json:"runtime_id"`
}
