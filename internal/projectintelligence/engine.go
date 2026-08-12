package projectintelligence

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Engine struct{ Now func() time.Time }

var ErrOverrideIncompatible = errors.New("requested technology override is incompatible with project constraints")

func (e Engine) Analyze(intent ProjectIntent, override string) (Analysis, error) {
	if strings.TrimSpace(intent.ProjectID) == "" || strings.TrimSpace(intent.UserPrompt) == "" {
		return Analysis{}, errors.New("project_id and user_prompt are required")
	}
	if e.Now == nil {
		e.Now = time.Now
	}
	requirements := extractRequirements(intent)
	classification := classify(intent)
	candidates := candidatesFor(classification, intent)
	decision, err := selectArchitecture(intent, classification, candidates, override, e.Now())
	if err != nil {
		return Analysis{}, err
	}
	return Analysis{Intent: intent, Requirements: requirements, Classification: classification, Decision: decision}, nil
}
func extractRequirements(i ProjectIntent) []ProjectRequirement {
	out := []ProjectRequirement{}
	add := func(cat, desc string, p Priority, req bool, source string, confidence float64) {
		out = append(out, ProjectRequirement{ID: fmt.Sprintf("req_%d", len(out)+1), ProjectID: i.ProjectID, Category: cat, Description: desc, Priority: p, Required: req, Source: source, Confidence: confidence})
	}
	prompt := strings.ToLower(i.UserPrompt)
	add("functional", i.UserPrompt, PriorityRequired, true, "user_prompt", 0.98)
	if strings.Contains(prompt, "game") {
		add("media", "Interactive media and game loop are required", PriorityHigh, true, "keyword:game", 0.9)
	}
	if strings.Contains(prompt, "mobile") || contains(i.Platforms, "mobile") {
		add("platform", "Mobile platform support is required", PriorityRequired, true, "platform", 0.96)
	}
	if strings.Contains(prompt, "offline") || contains(i.Constraints, "offline") {
		add("offline", "The project must work without network access", PriorityHigh, true, "constraint", 0.94)
	}
	if strings.Contains(prompt, "children") || strings.Contains(strings.ToLower(i.TargetAudience), "child") {
		add("accessibility", "Child-friendly accessible interaction is required", PriorityHigh, true, "audience", 0.9)
	}
	add("testing", "Automated unit and end-to-end verification is required", PriorityHigh, true, "platform_defaults", 0.85)
	add("deployment", "The project must have a reproducible deployment target", PriorityMedium, false, "platform_defaults", 0.8)
	return out
}
func classify(i ProjectIntent) Classification {
	p := strings.ToLower(i.UserPrompt)
	c := Classification{PrimaryType: "web", Confidence: .62, Evidence: []string{"default web-compatible project"}}
	if containsAny(p, "game", "educational game") {
		c.PrimaryType = "game"
		c.SecondaryTypes = []string{"educational", "multimedia"}
		c.Confidence = .91
		c.Evidence = []string{"game keyword", "interactive/media intent"}
	}
	if contains(i.Platforms, "mobile") || strings.Contains(p, "mobile") {
		c.PrimaryType = "mobile"
		c.SecondaryTypes = append(c.SecondaryTypes, "cross-platform")
		c.Confidence = .94
		c.Evidence = append(c.Evidence, "mobile platform requirement")
	}
	if containsAny(p, "api", "backend", "service") {
		c.PrimaryType = "backend"
		c.SecondaryTypes = append(c.SecondaryTypes, "API")
	}
	if containsAny(p, "AI", "assistant", "model") {
		c.SecondaryTypes = append(c.SecondaryTypes, "AI application")
	}
	return c
}
func candidatesFor(c Classification, i ProjectIntent) []ArchitectureCandidate {
	react := TechnologyStack{Language: "typescript", Framework: "react", Runtime: "node", PackageManager: "pnpm", BuildSystem: "vite", TestFramework: []string{"vitest"}, E2EFramework: []string{"playwright"}, Infrastructure: []string{"Project Computer", "Deployment Fabric"}}
	mobile := TechnologyStack{Language: "typescript", Framework: "react-native", Runtime: "node", PackageManager: "pnpm", BuildSystem: "expo", TestFramework: []string{"vitest"}, E2EFramework: []string{"playwright"}, Infrastructure: []string{"Project Computer", "Deployment Fabric"}}
	goapi := TechnologyStack{Language: "go", Framework: "net/http", Runtime: "go", PackageManager: "go modules", BuildSystem: "go build", TestFramework: []string{"go test"}, Infrastructure: []string{"Project Computer", "Deployment Fabric"}}
	if c.PrimaryType == "mobile" || c.PrimaryType == "game" {
		return []ArchitectureCandidate{{ID: "react-native-expo", Stack: mobile, Advantages: []string{"cross-platform", "fast iteration", "existing agent compatibility"}, Disadvantages: []string{"native SDK setup required"}, Compatibility: .94, Complexity: .62, Performance: .82, Maintainability: .84, Ecosystem: .9, Confidence: .91}, {ID: "react-web", Stack: react, Advantages: []string{"mature web tooling", "simple preview"}, Disadvantages: []string{"not native mobile"}, Compatibility: .71, Complexity: .38, Performance: .8, Maintainability: .88, Ecosystem: .93, Confidence: .75}}
	}
	if c.PrimaryType == "backend" {
		return []ArchitectureCandidate{{ID: "go-api", Stack: goapi, Advantages: []string{"small runtime", "strong server tooling"}, Disadvantages: []string{"less UI support"}, Compatibility: .92, Complexity: .55, Performance: .93, Maintainability: .9, Ecosystem: .78, Confidence: .89}, {ID: "react-api", Stack: react, Advantages: []string{"shared TypeScript ecosystem"}, Disadvantages: []string{"larger runtime surface"}, Compatibility: .72, Complexity: .64, Performance: .7, Maintainability: .8, Ecosystem: .9, Confidence: .76}}
	}
	return []ArchitectureCandidate{{ID: "react-web", Stack: react, Advantages: []string{"existing web support", "fast preview"}, Disadvantages: []string{"browser-only"}, Compatibility: .9, Complexity: .4, Performance: .8, Maintainability: .88, Ecosystem: .93, Confidence: .88}}
}
func selectArchitecture(i ProjectIntent, c Classification, candidates []ArchitectureCandidate, override string, now time.Time) (ArchitectureDecision, error) {
	d := ArchitectureDecision{ID: fmt.Sprintf("adr_%d", now.UnixNano()), ProjectID: i.ProjectID, Intent: i, Classification: c, Candidates: candidates, Status: "PROPOSED", CreatedAt: now, Rejected: map[string]string{}}
	if override != "" {
		for _, candidate := range candidates {
			if strings.Contains(strings.ToLower(candidate.Stack.Framework), strings.ToLower(override)) || strings.Contains(strings.ToLower(candidate.Stack.Language), strings.ToLower(override)) || strings.EqualFold(candidate.ID, override) {
				d.SelectedID = candidate.ID
				d.Selected = candidate.Stack
				d.Override = override
				d.Status = "OVERRIDE_ACCEPTED"
				d.Reasons = []string{"explicit user override", "validated against project classification and available candidates"}
				return d, nil
			}
		}
		return d, fmt.Errorf("%w: %s", ErrOverrideIncompatible, override)
	}
	best := -1.0
	for _, candidate := range candidates {
		score := candidate.Compatibility*.30 + (1-candidate.Complexity)*.15 + candidate.Performance*.2 + candidate.Maintainability*.15 + candidate.Ecosystem*.1 + candidate.Confidence*.1
		if score > best {
			best = score
			d.SelectedID = candidate.ID
			d.Selected = candidate.Stack
			d.Reasons = []string{"highest deterministic compatibility score", "fit to classified project type", "existing runtime and agent compatibility"}
		}
		if candidate.ID != d.SelectedID {
			d.Rejected[candidate.ID] = "lower deterministic score"
		}
	}
	d.Status = "PROPOSED"
	return d, nil
}
func contains(items []string, value string) bool {
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}
func containsAny(s string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(s, strings.ToLower(v)) {
			return true
		}
	}
	return false
}
