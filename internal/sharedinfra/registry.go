package sharedinfra

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry { return &Registry{tools: map[string]Tool{}} }
func (r *Registry) Register(t Tool) error {
	if t.ID == "" || t.Name == "" {
		return errors.New("tool id and name are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.ID] = t
	return nil
}
func (r *Registry) Get(id string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[id]
	return t, ok
}
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}
func (r *Registry) Resolve(toolID, requiredVersion, platform, architecture string) (ToolVersion, error) {
	t, ok := r.Get(toolID)
	if !ok {
		return ToolVersion{}, fmt.Errorf("tool %s not registered", toolID)
	}
	for _, v := range t.Versions {
		if versionMatches(v.Version, requiredVersion) && (platform == "" || v.Platform == platform) && (architecture == "" || v.Architecture == architecture) {
			return v, nil
		}
	}
	return ToolVersion{}, fmt.Errorf("no compatible version for %s %s on %s/%s", toolID, requiredVersion, platform, architecture)
}
func versionMatches(actual, required string) bool {
	if required == "" || required == "latest-compatible" || required == "latest" {
		return true
	}
	if actual == required {
		return true
	}
	if strings.HasSuffix(required, ".x") {
		return strings.HasPrefix(actual, strings.TrimSuffix(required, ".x"))
	}
	return false
}

type Cache struct {
	mu      sync.Mutex
	entries map[string]CacheEntry
	blobs   map[string][]byte
	now     func() time.Time
}

func NewCache(now func() time.Time) *Cache {
	if now == nil {
		now = time.Now
	}
	return &Cache{entries: map[string]CacheEntry{}, blobs: map[string][]byte{}, now: now}
}
func artifactKey(a Artifact) string {
	return strings.Join([]string{a.ID, a.Name, a.Version, a.Platform, a.Architecture, a.Checksum}, "|")
}
func (c *Cache) Lookup(a Artifact) (CacheEntry, CacheStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[artifactKey(a)]
	if !ok {
		return CacheEntry{Artifact: a, Status: CacheMiss}, CacheMiss
	}
	blob := c.blobs[artifactKey(a)]
	if !verifyChecksum(blob, a.Checksum) {
		e.Status = CacheCorrupt
		c.entries[artifactKey(a)] = e
		return e, CacheCorrupt
	}
	now := c.now().UTC()
	e.Status = CacheHit
	e.LastUsedAt = &now
	e.RefCount++
	c.entries[artifactKey(a)] = e
	return e, CacheHit
}
func (c *Cache) Put(a Artifact, data []byte) (CacheEntry, error) {
	if a.Checksum == "" {
		return CacheEntry{}, errors.New("artifact checksum is required")
	}
	if !verifyChecksum(data, a.Checksum) {
		return CacheEntry{}, fmt.Errorf("artifact checksum mismatch for %s", a.ID)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := artifactKey(a)
	if existing, ok := c.entries[key]; ok {
		now := c.now().UTC()
		existing.Status = CacheHit
		existing.VerifiedAt = &now
		existing.LastUsedAt = &now
		existing.RefCount++
		c.entries[key] = existing
		return existing, nil
	}
	stored := append([]byte(nil), data...)
	c.blobs[key] = stored
	now := c.now().UTC()
	e := CacheEntry{Artifact: a, Status: CacheHit, VerifiedAt: &now, LastUsedAt: &now, RefCount: 1}
	c.entries[key] = e
	return e, nil
}
func (c *Cache) Invalidate(a Artifact) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := artifactKey(a)
	if e, ok := c.entries[key]; ok {
		e.Status = CacheInvalid
		c.entries[key] = e
		delete(c.blobs, key)
	}
}
func (c *Cache) Stats() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]int{"entries": len(c.entries), "bytes": 0, "hits": 0}
	for k, e := range c.entries {
		out["bytes"] += len(c.blobs[k])
		if e.Status == CacheHit {
			out["hits"]++
		}
	}
	return out
}
func verifyChecksum(data []byte, expected string) bool {
	if expected == "" {
		return false
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	expected = strings.TrimPrefix(expected, "sha256:")
	return strings.EqualFold(actual, expected)
}

func (r *Registry) Load(tools []Tool) error {
	for _, t := range tools {
		if err := r.Register(t); err != nil {
			return err
		}
	}
	return nil
}
