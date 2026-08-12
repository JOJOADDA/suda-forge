package routing

import (
	"sync"
	"time"
)

type MemoryHealthCache struct {
	mu    sync.RWMutex
	items map[string]ProviderHealth
}

func NewMemoryHealthCache() *MemoryHealthCache {
	return &MemoryHealthCache{items: map[string]ProviderHealth{}}
}
func (c *MemoryHealthCache) Get(id string) (ProviderHealth, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	h, ok := c.items[id]
	return h, ok
}
func (c *MemoryHealthCache) Put(h ProviderHealth) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[h.ProviderID] = h
}
func (c *MemoryHealthCache) Fresh(id string, maxAge time.Duration, now time.Time) (ProviderHealth, bool) {
	h, ok := c.Get(id)
	if !ok || now.Sub(h.CheckedAt) > maxAge {
		return ProviderHealth{}, false
	}
	return h, true
}

type CostEstimator interface {
	Estimate(ModelProfile, int, int) float64
}
type TokenCostEstimator struct{}

func (TokenCostEstimator) Estimate(model ModelProfile, input, output int) float64 {
	return float64(input)/1000000*model.Pricing.InputCost + float64(output)/1000000*model.Pricing.OutputCost
}
