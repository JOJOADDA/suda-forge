package designintelligence

import "time"

type ID string

type DesignSystem struct {
	ID                 ID                    `json:"id"`
	ProjectID          string                `json:"project_id"`
	Name               string                `json:"name"`
	Version            string                `json:"version"`
	Tokens             []DesignToken         `json:"tokens"`
	Typography         Typography            `json:"typography"`
	ColorPalette       ColorPalette          `json:"color_palette"`
	Spacing            SpacingScale          `json:"spacing"`
	Radius             RadiusScale           `json:"radius"`
	Shadows            ShadowScale           `json:"shadows"`
	Breakpoints        []Breakpoint          `json:"breakpoints"`
	Components         []ComponentDefinition `json:"components"`
	Patterns           []Pattern             `json:"patterns"`
	LayoutRules        []LayoutRule          `json:"layout_rules"`
	MotionRules        []MotionRule          `json:"motion_rules"`
	AccessibilityRules []AccessibilityRule   `json:"accessibility_rules"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}
type DesignToken struct {
	ID          ID     `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Semantic    bool   `json:"semantic"`
}
type Typography struct {
	FontFamily string `json:"font_family"`
	Heading    string `json:"heading"`
	Body       string `json:"body"`
	Caption    string `json:"caption"`
	Weights    []int  `json:"weights"`
}
type ColorPalette struct {
	Primary    string `json:"primary"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	Muted      string `json:"muted"`
	Danger     string `json:"danger"`
	Success    string `json:"success"`
}
type SpacingScale map[string]string
type RadiusScale map[string]string
type ShadowScale map[string]string
type Breakpoint struct {
	Name     string `json:"name"`
	MinWidth int    `json:"min_width"`
}
type ComponentDefinition struct {
	ID           ID                 `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Tokens       []ID               `json:"tokens"`
	Variants     []ComponentVariant `json:"variants"`
	Dependencies []ID               `json:"dependencies"`
	UsedBy       []string           `json:"used_by"`
	Pages        []string           `json:"pages"`
	Tests        []string           `json:"tests"`
}
type ComponentVariant struct {
	Name   string            `json:"name"`
	Props  map[string]string `json:"props"`
	Tokens []ID              `json:"tokens"`
}
type Pattern struct {
	ID            ID       `json:"id"`
	Name          string   `json:"name"`
	Components    []ID     `json:"components"`
	Layout        string   `json:"layout"`
	Accessibility []string `json:"accessibility"`
}
type LayoutRule struct {
	ID        ID       `json:"id"`
	Name      string   `json:"name"`
	Rule      string   `json:"rule"`
	AppliesTo []string `json:"applies_to"`
}
type MotionRule struct {
	ID                    ID     `json:"id"`
	Name                  string `json:"name"`
	DurationMS            int    `json:"duration_ms"`
	Easing                string `json:"easing"`
	ReducedMotionBehavior string `json:"reduced_motion_behavior"`
}
type AccessibilityRule struct {
	ID          ID     `json:"id"`
	Name        string `json:"name"`
	Requirement string `json:"requirement"`
	Severity    string `json:"severity"`
}
type AnalysisInput struct {
	ProjectID          string   `json:"project_id"`
	ProductDescription string   `json:"product_description"`
	Audience           string   `json:"audience,omitempty"`
	Platform           string   `json:"platform,omitempty"`
	Features           []string `json:"features,omitempty"`
	ExistingComponents []string `json:"existing_components,omitempty"`
}
type Analysis struct {
	ProjectID    string       `json:"project_id"`
	DesignSystem DesignSystem `json:"design_system"`
	Decisions    []string     `json:"decisions"`
	Assumptions  []string     `json:"assumptions"`
	Confidence   float64      `json:"confidence"`
}
