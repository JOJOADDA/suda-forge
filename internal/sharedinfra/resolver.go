package sharedinfra

import (
	"fmt"
	"strings"

	"suda-forge/internal/environment"
)

type Resolver struct {
	Registry     *Registry
	Cache        *Cache
	Platform     string
	Architecture string
}

func (r Resolver) ResolveManifest(m environment.Manifest) ([]Resolution, error) {
	if r.Registry == nil {
		return nil, fmt.Errorf("tool registry unavailable")
	}
	out := []Resolution{}
	resolve := func(req, name string) error {
		v, err := r.Registry.Resolve(name, req, r.Platform, r.Architecture)
		if err != nil {
			return err
		}
		status := CacheMiss
		if r.Cache != nil {
			_, status = r.Cache.Lookup(v.Artifact)
		}
		out = append(out, Resolution{Requirement: name, ToolID: name, Version: v, Cache: status, Reason: reason(status), VerificationRequired: true})
		return nil
	}
	for _, x := range m.Languages {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.PackageManagers {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.Frameworks {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.BuildTools {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.TestTools {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.SDKs {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.Browsers {
		if err := resolve(x.Version, normalizeTool(x.Name)); err != nil {
			return nil, err
		}
	}
	for _, x := range m.AgentCLIs {
		if err := resolve(x.Version, normalizeTool(x.AgentID)); err != nil {
			return nil, err
		}
	}
	return dedup(out), nil
}
func reason(s CacheStatus) string {
	switch s {
	case CacheHit:
		return "artifact verified and reusable from global cache"
	case CacheMiss:
		return "artifact is not present in global cache"
	case CacheCorrupt:
		return "cached artifact failed integrity verification"
	default:
		return "cache status requires resolution"
	}
}
func normalizeTool(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "node", "nodejs":
		return "node"
	case "pnpm", "npm", "yarn":
		return v
	case "git":
		return "git"
	case "chromium", "chrome":
		return "chromium"
	case "claude_code":
		return "claude_code"
	default:
		return v
	}
}
func dedup(in []Resolution) []Resolution {
	seen := map[string]bool{}
	out := []Resolution{}
	for _, v := range in {
		key := v.ToolID + "@" + v.Version.Version
		if !seen[key] {
			seen[key] = true
			out = append(out, v)
		}
	}
	return out
}
func DefaultRegistry() *Registry {
	r := NewRegistry()
	add := func(id string, cat ToolCategory, versions ...string) {
		list := []ToolVersion{}
		for _, v := range versions {
			list = append(list, ToolVersion{Version: v, Platform: "linux", Architecture: "amd64", Artifact: Artifact{ID: id + "@" + v, Name: id, Version: v, Platform: "linux", Architecture: "amd64"}, InstallationMethod: "runtime-scoped", VerificationMethod: "binary-version-and-capability"})
		}
		_ = r.Register(Tool{ID: id, Name: id, Category: cat, Versions: list, Platforms: []string{"linux"}, InstallStrategy: "runtime-scoped", VerificationStrategy: "binary-version-and-capability", ArtifactIdentity: id})
	}
	add("node", Language, "20.11.0", "22.1.0")
	add("pnpm", PackageManager, "9.15.0", "10.0.0")
	add("git", CLI, "2.43.0")
	add("chromium", Browser, "stable")
	add("codex", AIAgent, "latest-compatible")
	add("claude_code", AIAgent, "latest-compatible")
	add("kimi", AIAgent, "latest-compatible")
	add("playwright", Testing, "latest-compatible")
	add("react", Framework, "latest-compatible")
	add("react-native", Framework, "latest-compatible")
	add("go", Language, "1.23.0")
	return r
}
