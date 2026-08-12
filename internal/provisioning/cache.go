package provisioning

import (
	"context"
	"sync"
)

type MemoryCache struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewMemoryCache() *MemoryCache { return &MemoryCache{values: map[string]string{}} }
func (c *MemoryCache) Get(_ context.Context, key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.values[key]
	return v, ok
}
func (c *MemoryCache) Put(_ context.Context, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}
