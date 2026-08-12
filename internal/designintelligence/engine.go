package designintelligence

import (
	"errors"
	"strings"
	"time"
)

type Engine struct{ Now func() time.Time }

func NewEngine(now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{Now: now}
}
func (e *Engine) Analyze(in AnalysisInput) (Analysis, error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return Analysis{}, errors.New("project_id is required")
	}
	if strings.TrimSpace(in.ProductDescription) == "" {
		return Analysis{}, errors.New("product description is required")
	}
	now := e.Now().UTC()
	platform := strings.ToLower(in.Platform)
	if platform == "" {
		platform = "web"
	}
	system := DesignSystem{ID: ID("design_" + in.ProjectID), ProjectID: in.ProjectID, Name: "SUDA Product System", Version: "1.0.0", Typography: Typography{FontFamily: "Inter, system-ui, sans-serif", Heading: "1.75rem/1.15", Body: "1rem/1.6", Caption: "0.75rem/1.4", Weights: []int{400, 500, 600, 700}}, ColorPalette: ColorPalette{Primary: "#74d4b2", Background: "#0b1117", Surface: "#101a23", Text: "#e7edf3", Muted: "#8ca0ae", Danger: "#f26e6e", Success: "#68d391"}, Spacing: SpacingScale{"xs": "4px", "sm": "8px", "md": "16px", "lg": "24px", "xl": "40px"}, Radius: RadiusScale{"sm": "4px", "md": "6px", "lg": "12px"}, Shadows: ShadowScale{"sm": "0 2px 8px rgba(0,0,0,.18)", "md": "0 8px 24px rgba(0,0,0,.22)"}, Breakpoints: []Breakpoint{{Name: "mobile", MinWidth: 0}, {Name: "tablet", MinWidth: 768}, {Name: "desktop", MinWidth: 1200}}, CreatedAt: now, UpdatedAt: now}
	system.Tokens = []DesignToken{{ID: "color.primary", Name: "color.primary", Kind: "color", Value: system.ColorPalette.Primary, Semantic: true}, {ID: "color.background", Name: "color.background", Kind: "color", Value: system.ColorPalette.Background, Semantic: true}, {ID: "color.surface", Name: "color.surface", Kind: "color", Value: system.ColorPalette.Surface, Semantic: true}, {ID: "color.text", Name: "color.text", Kind: "color", Value: system.ColorPalette.Text, Semantic: true}, {ID: "spacing.md", Name: "spacing.md", Kind: "spacing", Value: system.Spacing["md"], Semantic: true}, {ID: "radius.md", Name: "radius.md", Kind: "radius", Value: system.Radius["md"], Semantic: true}, {ID: "typography.heading", Name: "typography.heading", Kind: "typography", Value: system.Typography.Heading, Semantic: true}, {ID: "typography.body", Name: "typography.body", Kind: "typography", Value: system.Typography.Body, Semantic: true}}
	components := []ComponentDefinition{{ID: "button", Name: "Button", Description: "Primary interactive action", Tokens: []ID{"color.primary", "spacing.md", "radius.md", "typography.body"}, Variants: []ComponentVariant{{Name: "primary", Props: map[string]string{"tone": "primary"}, Tokens: []ID{"color.primary"}}, {Name: "ghost", Props: map[string]string{"tone": "ghost"}, Tokens: []ID{"color.surface", "color.text"}}}}, {ID: "card", Name: "Card", Description: "Surface container for grouped content", Tokens: []ID{"color.surface", "spacing.md", "radius.md"}, Variants: []ComponentVariant{{Name: "default", Props: map[string]string{}, Tokens: []ID{"color.surface"}}}}, {ID: "input", Name: "Input", Description: "Accessible user input", Tokens: []ID{"color.surface", "color.text", "spacing.md", "radius.md"}, Variants: []ComponentVariant{{Name: "default", Props: map[string]string{"aria": "required"}, Tokens: []ID{"color.text"}}}}, {ID: "navbar", Name: "Navbar", Description: "Responsive project navigation", Tokens: []ID{"color.background", "spacing.md"}, Variants: []ComponentVariant{{Name: "responsive", Props: map[string]string{"mobile": "collapsible"}, Tokens: []ID{"spacing.md"}}}}}
	system.Components = components
	system.AccessibilityRules = []AccessibilityRule{{ID: "keyboard-navigation", Name: "Keyboard navigation", Requirement: "Interactive controls must be keyboard reachable and visibly focused", Severity: "REQUIRED"}, {ID: "contrast", Name: "Color contrast", Requirement: "Text and controls must meet the configured contrast threshold", Severity: "REQUIRED"}, {ID: "reduced-motion", Name: "Reduced motion", Requirement: "Animations must respect prefers-reduced-motion", Severity: "RECOMMENDED"}}
	system.LayoutRules = []LayoutRule{{ID: "responsive-shell", Name: "Responsive shell", Rule: "Use mobile, tablet, and desktop breakpoints with progressive disclosure", AppliesTo: []string{"page", "navbar", "sidebar"}}}
	system.MotionRules = []MotionRule{{ID: "standard-transition", Name: "Standard transition", DurationMS: 180, Easing: "ease-out", ReducedMotionBehavior: "disable"}}
	system.Patterns = []Pattern{{ID: "workspace-shell", Name: "Workspace shell", Components: []ID{"navbar", "card"}, Layout: "responsive two-column shell", Accessibility: []string{"keyboard-navigation", "contrast"}}}
	decisions := []string{"Use semantic tokens instead of raw visual values", "Prioritize responsive layout for " + platform, "Keep components reusable and traceable to tokens", "Treat accessibility rules as verification requirements"}
	assumptions := []string{}
	if len(in.Features) == 0 {
		assumptions = append(assumptions, "Feature list was not supplied; baseline product primitives were selected")
	}
	if in.Audience == "" {
		assumptions = append(assumptions, "Audience was not supplied; typography and contrast use general-purpose defaults")
	}
	return Analysis{ProjectID: in.ProjectID, DesignSystem: system, Decisions: decisions, Assumptions: assumptions, Confidence: 0.78}, nil
}
