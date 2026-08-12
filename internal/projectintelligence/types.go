package projectintelligence

import "time"

type Priority string

const (
	PriorityRequired Priority = "REQUIRED"
	PriorityHigh     Priority = "HIGH"
	PriorityMedium   Priority = "MEDIUM"
	PriorityLow      Priority = "LOW"
)

type BudgetPolicy struct {
	MaxMonthlyCost float64 `json:"max_monthly_cost"`
	PreferLocal    bool    `json:"prefer_local"`
	OfflineOnly    bool    `json:"offline_only"`
}
type ProjectIntent struct {
	ProjectID      string       `json:"project_id"`
	UserPrompt     string       `json:"user_prompt"`
	TargetAudience string       `json:"target_audience"`
	Platforms      []string     `json:"platforms"`
	Constraints    []string     `json:"constraints"`
	Preferences    []string     `json:"preferences"`
	Budget         BudgetPolicy `json:"budget"`
}
type ProjectRequirement struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"project_id"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Required    bool     `json:"required"`
	Source      string   `json:"source"`
	Confidence  float64  `json:"confidence"`
}
type Classification struct {
	PrimaryType    string   `json:"primary_type"`
	SecondaryTypes []string `json:"secondary_types"`
	Confidence     float64  `json:"confidence"`
	Evidence       []string `json:"evidence"`
}
type TechnologyStack struct {
	Language       string   `json:"language"`
	Framework      string   `json:"framework"`
	Runtime        string   `json:"runtime"`
	PackageManager string   `json:"package_manager"`
	BuildSystem    string   `json:"build_system"`
	TestFramework  []string `json:"test_framework"`
	E2EFramework   []string `json:"e2e_framework"`
	Database       string   `json:"database"`
	Infrastructure []string `json:"infrastructure"`
}
type ArchitectureCandidate struct {
	ID              string          `json:"id"`
	Stack           TechnologyStack `json:"stack"`
	Advantages      []string        `json:"advantages"`
	Disadvantages   []string        `json:"disadvantages"`
	Compatibility   float64         `json:"compatibility"`
	Complexity      float64         `json:"complexity"`
	Performance     float64         `json:"performance"`
	Maintainability float64         `json:"maintainability"`
	Ecosystem       float64         `json:"ecosystem"`
	Confidence      float64         `json:"confidence"`
	RejectedReason  string          `json:"rejected_reason,omitempty"`
}
type ArchitectureDecision struct {
	ID             string                  `json:"id"`
	ProjectID      string                  `json:"project_id"`
	Intent         ProjectIntent           `json:"intent"`
	Classification Classification          `json:"classification"`
	Candidates     []ArchitectureCandidate `json:"candidates"`
	SelectedID     string                  `json:"selected_id"`
	Selected       TechnologyStack         `json:"selected"`
	Reasons        []string                `json:"reasons"`
	Rejected       map[string]string       `json:"rejected"`
	Override       string                  `json:"override,omitempty"`
	Status         string                  `json:"status"`
	CreatedAt      time.Time               `json:"created_at"`
}
type Analysis struct {
	Intent         ProjectIntent        `json:"intent"`
	Requirements   []ProjectRequirement `json:"requirements"`
	Classification Classification       `json:"classification"`
	Decision       ArchitectureDecision `json:"decision"`
}
