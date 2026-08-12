package productexperience

import (
	"suda-forge/internal/constitution"
	"suda-forge/internal/designintelligence"
	"suda-forge/internal/knowledge"
	"time"
)

type ContextRequest struct {
	ProjectID                string            `json:"project_id"`
	AgentID                  string            `json:"agent_id"`
	Role                     string            `json:"role"`
	Task                     string            `json:"task"`
	CurrentState             map[string]string `json:"current_state"`
	AvailableTools           []string          `json:"available_tools"`
	RuntimeCapabilities      []string          `json:"runtime_capabilities"`
	VerificationRequirements []string          `json:"verification_requirements"`
}
type AgentContext struct {
	CoreConstitution         string                           `json:"core_constitution"`
	SecurityPolicy           []string                         `json:"security_policy"`
	Role                     string                           `json:"role"`
	ProjectPolicy            constitution.Constitution        `json:"project_policy"`
	Task                     string                           `json:"task"`
	Knowledge                knowledge.Graph                  `json:"knowledge"`
	DesignSystem             *designintelligence.DesignSystem `json:"design_system,omitempty"`
	CurrentState             map[string]string                `json:"current_state"`
	AvailableTools           []string                         `json:"available_tools"`
	RuntimeCapabilities      []string                         `json:"runtime_capabilities"`
	VerificationRequirements []string                         `json:"verification_requirements"`
	AssembledAt              time.Time                        `json:"assembled_at"`
}
type SessionRecovery struct {
	ProjectID         string                           `json:"project_id"`
	CurrentState      map[string]string                `json:"current_state"`
	Knowledge         knowledge.Graph                  `json:"knowledge"`
	Decisions         []string                         `json:"decisions"`
	Tasks             []string                         `json:"tasks"`
	Bugs              []string                         `json:"bugs"`
	DesignSystem      *designintelligence.DesignSystem `json:"design_system,omitempty"`
	GitState          map[string]string                `json:"git_state"`
	VerificationState map[string]string                `json:"verification_state"`
	Summary           string                           `json:"summary"`
}
type ImpactAnalysis struct {
	ProjectID        string           `json:"project_id"`
	RootNode         knowledge.NodeID `json:"root_node"`
	Risk             string           `json:"risk"`
	Files            []string         `json:"files"`
	APIs             []string         `json:"apis"`
	Components       []string         `json:"components"`
	Tests            []string         `json:"tests"`
	Nodes            []knowledge.Node `json:"nodes"`
	ApprovalRequired bool             `json:"approval_required"`
	Reasons          []string         `json:"reasons"`
}
type LoopStage string

const (
	Intent           LoopStage = "INTENT"
	Plan             LoopStage = "PLAN"
	Architect        LoopStage = "ARCHITECT"
	Design           LoopStage = "DESIGN"
	Implement        LoopStage = "IMPLEMENT"
	Build            LoopStage = "BUILD"
	Test             LoopStage = "TEST"
	Verify           LoopStage = "VERIFY"
	VisualQA         LoopStage = "VISUAL_QA"
	Security         LoopStage = "SECURITY"
	Fix              LoopStage = "FIX"
	Commit           LoopStage = "COMMIT"
	Deploy           LoopStage = "DEPLOY"
	PostDeployVerify LoopStage = "POST_DEPLOY_VERIFY"
)

type LoopPlan struct {
	ID            string               `json:"id"`
	ProjectID     string               `json:"project_id"`
	Stages        []LoopStage          `json:"stages"`
	Delegates     map[LoopStage]string `json:"delegates"`
	Status        string               `json:"status"`
	BlockedStages []LoopStage          `json:"blocked_stages,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}
type ActivityState string

const (
	Active    ActivityState = "ACTIVE"
	Waiting   ActivityState = "WAITING"
	Executing ActivityState = "EXECUTING"
	Verifying ActivityState = "VERIFYING"
	Blocked   ActivityState = "BLOCKED"
	Completed ActivityState = "COMPLETED"
	Failed    ActivityState = "FAILED"
)

type Activity struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	Type      string         `json:"type"`
	State     ActivityState  `json:"state"`
	Message   string         `json:"message"`
	Progress  *int           `json:"progress,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}
