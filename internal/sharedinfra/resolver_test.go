package sharedinfra

import (
	"suda-forge/internal/environment"
	"testing"
)

func TestResolveManifestUsesRegistryAndDeduplicates(t *testing.T) {
	r := DefaultRegistry()
	c := NewCache(nil)
	m := environment.Manifest{ProjectID: "p", Languages: []environment.RuntimeRequirement{{Name: "node", Version: "22.x", Required: true}}, PackageManagers: []environment.ToolRequirement{{Name: "pnpm", Version: "10.x", Required: true}}, BuildTools: []environment.ToolRequirement{{Name: "node", Version: "22.x", Required: true}}}
	res, err := (Resolver{Registry: r, Cache: c, Platform: "linux", Architecture: "amd64"}).ResolveManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("expected deduplicated node/pnpm resolutions, got %d", len(res))
	}
	if res[0].Cache != CacheMiss {
		t.Fatalf("expected cache miss, got %s", res[0].Cache)
	}
}
