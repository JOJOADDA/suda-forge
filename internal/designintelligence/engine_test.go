package designintelligence

import "testing"

func TestAnalyzeProducesStructuredDesignSystem(t *testing.T) {
	a, err := NewEngine(nil).Analyze(AnalysisInput{ProjectID: "p1", ProductDescription: "Interactive learning app", Audience: "children", Platform: "mobile", Features: []string{"games"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.DesignSystem.Tokens) < 5 || len(a.DesignSystem.Components) < 3 || len(a.DesignSystem.AccessibilityRules) == 0 {
		t.Fatalf("incomplete design system: %#v", a.DesignSystem)
	}
	if a.DesignSystem.Components[0].Tokens == nil {
		t.Fatal("components must reference structured tokens")
	}
}
func TestAnalyzeRequiresDescription(t *testing.T) {
	if _, err := NewEngine(nil).Analyze(AnalysisInput{ProjectID: "p1"}); err == nil {
		t.Fatal("expected missing description error")
	}
}
