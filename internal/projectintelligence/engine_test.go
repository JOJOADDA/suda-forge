package projectintelligence

import "testing"

func TestAnalyzeEducationalMobileGame(t *testing.T) {
	a, err := (Engine{}).Analyze(ProjectIntent{ProjectID: "p", UserPrompt: "I want an educational game for kindergarten children", Platforms: []string{"mobile"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Classification.PrimaryType != "mobile" {
		t.Fatalf("expected mobile, got %s", a.Classification.PrimaryType)
	}
	if a.Decision.Selected.Framework != "react-native" {
		t.Fatalf("expected react-native, got %s", a.Decision.Selected.Framework)
	}
	if len(a.Requirements) < 3 {
		t.Fatalf("expected requirements, got %d", len(a.Requirements))
	}
}
func TestOverrideMustBeCompatible(t *testing.T) {
	if _, err := (Engine{}).Analyze(ProjectIntent{ProjectID: "p", UserPrompt: "mobile game", Platforms: []string{"mobile"}}, "flutter"); err == nil {
		t.Fatal("expected incompatible override")
	}
	a, err := (Engine{}).Analyze(ProjectIntent{ProjectID: "p", UserPrompt: "mobile game", Platforms: []string{"mobile"}}, "react-native")
	if err != nil || a.Decision.Status != "OVERRIDE_ACCEPTED" {
		t.Fatalf("expected accepted override, got %v", err)
	}
}
