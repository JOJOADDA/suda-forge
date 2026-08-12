package sharedinfra

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func TestCacheHitMissAndDedup(t *testing.T) {
	c := NewCache(nil)
	data := []byte("node-22")
	a := Artifact{ID: "node-22", Name: "node", Version: "22.x", Platform: "linux", Architecture: "amd64", Checksum: checksum(data)}
	if _, status := c.Lookup(a); status != CacheMiss {
		t.Fatalf("expected miss, got %s", status)
	}
	if _, err := c.Put(a, data); err != nil {
		t.Fatal(err)
	}
	first, status := c.Lookup(a)
	if status != CacheHit || first.RefCount != 2 {
		t.Fatalf("expected hit with refcount 2, got %s %d", status, first.RefCount)
	}
	second, err := c.Put(a, data)
	if err != nil || second.RefCount != 3 {
		t.Fatalf("expected dedup reuse, got %#v %v", second, err)
	}
}
func TestCacheDetectsCorruptionAfterStore(t *testing.T) {
	c := NewCache(nil)
	data := []byte("verified")
	a := Artifact{ID: "tool", Checksum: checksum(data)}
	if _, err := c.Put(a, data); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.blobs[artifactKey(a)] = []byte("tampered")
	c.mu.Unlock()
	if _, status := c.Lookup(a); status != CacheCorrupt {
		t.Fatalf("expected corrupt cache status, got %s", status)
	}
}
func TestCacheRejectsChecksumMismatch(t *testing.T) {
	c := NewCache(nil)
	a := Artifact{ID: "git", Checksum: checksum([]byte("expected"))}
	if _, err := c.Put(a, []byte("tampered")); err == nil {
		t.Fatal("expected checksum failure")
	}
}
func TestRegistryVersionResolution(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(Tool{ID: "node", Name: "Node", Versions: []ToolVersion{{Version: "20.11.0", Platform: "linux", Architecture: "amd64"}, {Version: "22.1.0", Platform: "linux", Architecture: "amd64"}}})
	v, err := r.Resolve("node", "22.x", "linux", "amd64")
	if err != nil || v.Version != "22.1.0" {
		t.Fatalf("unexpected resolution %#v %v", v, err)
	}
	if _, err := r.Resolve("node", "24.x", "linux", "amd64"); err == nil {
		t.Fatal("expected version mismatch")
	}
}
