package sharedinfra

import "time"

type ToolCategory string

const (
	Language       ToolCategory = "LANGUAGE"
	Framework      ToolCategory = "FRAMEWORK"
	SDK            ToolCategory = "SDK"
	BuildTool      ToolCategory = "BUILD_TOOL"
	PackageManager ToolCategory = "PACKAGE_MANAGER"
	Browser        ToolCategory = "BROWSER"
	Testing        ToolCategory = "TESTING"
	Database       ToolCategory = "DATABASE"
	DevOps         ToolCategory = "DEVOPS"
	AIAgent        ToolCategory = "AI_AGENT"
	CLI            ToolCategory = "CLI"
	Utility        ToolCategory = "UTILITY"
)

type Tool struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Category             ToolCategory      `json:"category"`
	Versions             []ToolVersion     `json:"versions"`
	Platforms            []string          `json:"platforms"`
	Dependencies         []string          `json:"dependencies"`
	Capabilities         []string          `json:"capabilities"`
	InstallStrategy      string            `json:"install_strategy"`
	VerificationStrategy string            `json:"verification_strategy"`
	ArtifactIdentity     string            `json:"artifact_identity"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}
type ToolVersion struct {
	Version            string   `json:"version"`
	Platform           string   `json:"platform"`
	Architecture       string   `json:"architecture"`
	Dependencies       []string `json:"dependencies"`
	Artifact           Artifact `json:"artifact"`
	InstallationMethod string   `json:"installation_method"`
	VerificationMethod string   `json:"verification_method"`
	Compatibility      []string `json:"compatibility"`
}
type Artifact struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Platform        string            `json:"platform"`
	Architecture    string            `json:"architecture"`
	Size            int64             `json:"size"`
	Checksum        string            `json:"checksum"`
	Source          string            `json:"source"`
	StorageLocation string            `json:"storage_location"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}
type CacheStatus string

const (
	CacheHit     CacheStatus = "HIT"
	CacheMiss    CacheStatus = "MISS"
	CacheCorrupt CacheStatus = "CORRUPT"
	CacheInvalid CacheStatus = "INVALID"
)

type CacheEntry struct {
	Artifact   Artifact    `json:"artifact"`
	Status     CacheStatus `json:"status"`
	VerifiedAt *time.Time  `json:"verified_at,omitempty"`
	LastUsedAt *time.Time  `json:"last_used_at,omitempty"`
	RefCount   int         `json:"ref_count"`
}
type Resolution struct {
	Requirement          string      `json:"requirement"`
	ToolID               string      `json:"tool_id"`
	Version              ToolVersion `json:"version"`
	Cache                CacheStatus `json:"cache_status"`
	Reason               string      `json:"reason"`
	VerificationRequired bool        `json:"verification_required"`
}
type PlanStep struct {
	ID                string           `json:"id"`
	Type              string           `json:"type"`
	Name              string           `json:"name"`
	Dependencies      []string         `json:"dependencies"`
	Status            string           `json:"status"`
	EstimatedResource map[string]int64 `json:"estimated_resource,omitempty"`
	CacheStatus       CacheStatus      `json:"cache_status"`
	RetryPolicy       map[string]int   `json:"retry_policy,omitempty"`
	Verification      string           `json:"verification"`
}
