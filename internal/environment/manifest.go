package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"suda-forge/internal/projectintelligence"
)

func FromDecision(decision projectintelligence.ArchitectureDecision, profile Profile, now time.Time) (Manifest, error) {
	if decision.ProjectID == "" || decision.Selected.Framework == "" {
		return Manifest{}, errors.New("architecture decision is incomplete")
	}
	if profile == "" {
		profile = Standard
	}
	stack := decision.Selected
	m := Manifest{ID: fmt.Sprintf("manifest_%s_%d", decision.ProjectID, now.UnixNano()), ProjectID: decision.ProjectID, Version: "1", BaseImage: baseImage(stack, profile), OS: "ubuntu", Architecture: "amd64", Profile: profile, DecisionID: decision.ID, CreatedAt: now}
	m.Languages = []RuntimeRequirement{{Name: stack.Runtime, Version: runtimeVersion(stack.Runtime), Required: true, Source: "architecture"}}
	if stack.PackageManager != "" {
		m.PackageManagers = []ToolRequirement{{Name: stack.PackageManager, Version: "latest-compatible", Required: true, Source: "architecture"}}
	}
	if stack.Framework != "" {
		m.Frameworks = []ToolRequirement{{Name: stack.Framework, Version: "latest-compatible", Required: true, Source: "architecture"}}
	}
	if stack.BuildSystem != "" {
		m.BuildTools = []ToolRequirement{{Name: stack.BuildSystem, Version: "latest-compatible", Required: true, Source: "architecture"}}
	}
	for _, tool := range stack.TestFramework {
		m.TestTools = append(m.TestTools, ToolRequirement{Name: tool, Version: "latest-compatible", Required: true, Source: "architecture"})
	}
	for _, tool := range stack.E2EFramework {
		m.Browsers = append(m.Browsers, BrowserRequirement{Name: "chromium", Engine: "chromium", Automation: tool, Version: "latest-compatible", Required: false})
	}
	m.AgentCLIs = []AgentRequirement{{AgentID: "codex", Required: true}, {AgentID: "claude_code", Required: false}, {AgentID: "kimi", Required: false}}
	m.Resources = profileResources(profile)
	m.Ports = []PortRequirement{{Name: "app", Port: 3000, Protocol: "tcp", Required: true}}
	m.Fingerprint = FingerprintFor(m)
	return m, nil
}
func FingerprintFor(m Manifest) string {
	raw, _ := json.Marshal(struct {
		Version, BaseImage, OS, Architecture string
		Languages                            []RuntimeRequirement
		PackageManagers                      []ToolRequirement
		Frameworks                           []ToolRequirement
		BuildTools                           []ToolRequirement
		TestTools                            []ToolRequirement
		Browsers                             []BrowserRequirement
		Agents                               []AgentRequirement
		Resources                            ResourceRequirement
	}{m.Version, m.BaseImage, m.OS, m.Architecture, m.Languages, m.PackageManagers, m.Frameworks, m.BuildTools, m.TestTools, m.Browsers, m.AgentCLIs, m.Resources})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func BuildFingerprint(m Manifest, now time.Time) Fingerprint {
	f := Fingerprint{Value: FingerprintFor(m), OS: m.OS, Image: m.BaseImage, Languages: map[string]string{}, Frameworks: map[string]string{}, Tools: map[string]string{}, Agents: map[string]string{}, SystemPackages: []string{}, CreatedAt: now}
	for _, v := range m.Languages {
		f.Languages[v.Name] = v.Version
	}
	for _, v := range m.Frameworks {
		f.Frameworks[v.Name] = v.Version
	}
	for _, v := range m.PackageManagers {
		f.Tools[v.Name] = v.Version
	}
	for _, v := range m.BuildTools {
		f.Tools[v.Name] = v.Version
	}
	for _, v := range m.TestTools {
		f.Tools[v.Name] = v.Version
	}
	for _, v := range m.AgentCLIs {
		f.Agents[v.AgentID] = v.Version
	}
	if len(m.Browsers) > 0 {
		f.Browser = m.Browsers[0].Name
	}
	return f
}
func baseImage(s projectintelligence.TechnologyStack, p Profile) string {
	if s.Framework == "react-native" {
		return "suda/mobile-node"
	}
	if s.Runtime == "go" {
		return "suda-go"
	}
	if strings.Contains(s.Runtime, "python") {
		return "suda-python"
	}
	if p == Full {
		return "suda-base"
	}
	return "suda/node"
}
func runtimeVersion(runtime string) string {
	switch runtime {
	case "node":
		return "22"
	case "go":
		return "1.23"
	case "python":
		return "3.12"
	default:
		return "managed"
	}
}
func profileResources(p Profile) ResourceRequirement {
	switch p {
	case Minimal:
		return ResourceRequirement{CPU: 1, MemoryBytes: 1073741824, DiskBytes: 10737418240}
	case Full:
		return ResourceRequirement{CPU: 4, MemoryBytes: 8589934592, DiskBytes: 32212254720}
	default:
		return ResourceRequirement{CPU: 2, MemoryBytes: 4294967296, DiskBytes: 21474836480}
	}
}
