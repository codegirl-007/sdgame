package simulation

import (
	"math/rand"
	"sort"
)

type CacheEntry struct {
	Value       interface{}
	Timestamp   int
	ExpireAt    int
	AccessCount int
}

type CacheNode struct {
	ID             string
	Label          string
	CacheTTL       int
	MaxEntries     int
	EvictionPolicy string
	CurrentLoad    int
	Queue          []*Request
	Cache          map[string]CacheEntry
	Alive          bool
	Targets        []string
	Output         []*Request
}

func (c *CacheNode) GetID() string {
	return c.ID
}

func (c *CacheNode) Type() string {
	return "cache"
}

func (c *CacheNode) IsAlive() bool {
	return c.Alive
}

func (c *CacheNode) Tick(tick int, currentTimeMs int) {
	for key, entry := range c.Cache {
		if currentTimeMs > entry.ExpireAt {
			delete(c.Cache, key)
		}
	}

	if len(c.Cache) > c.MaxEntries {
		evictCount := len(c.Cache) - c.MaxEntries
		keys := make([]string, 0, len(c.Cache))
		for k := range c.Cache {
			keys = append(keys, k)
		}

		switch c.EvictionPolicy {
		case "Random":
			rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
		case "LRU":
			sort.Slice(keys, func(i, j int) bool {
				return c.Cache[keys[i]].Timestamp < c.Cache[keys[j]].Timestamp
			})
		case "LFU":
			sort.Slice(keys, func(i, j int) bool {
				return c.Cache[keys[i]].AccessCount < c.Cache[keys[j]].AccessCount
			})
		}

		for i := 0; i < evictCount && i < len(keys); i++ {
			delete(c.Cache, keys[i])
		}
	}

	toProcess := min(len(c.Queue), 10)
	for i := 0; i < toProcess; i++ {
		req := c.Queue[i]
		if entry, found := c.Cache[req.ID]; found && currentTimeMs <= entry.ExpireAt {
			// Cache hit
			req.LatencyMS += 2
			req.Path = append(req.Path, c.ID+"(hit)")
		} else {
			// Cache miss
			req.LatencyMS += 5
			req.Path = append(req.Path, c.ID+"(miss)")
			c.Cache[req.ID] = CacheEntry{
				Value:     req,
				Timestamp: currentTimeMs,
				ExpireAt:  currentTimeMs + c.CacheTTL*1000,
			}
			c.Output = append(c.Output, req)
		}
	}
	c.Queue = c.Queue[toProcess:]
}

func (c *CacheNode) Receive(req *Request) {
	c.Queue = append(c.Queue, req)
}

func (c *CacheNode) Emit() []*Request {
	out := append([]*Request(nil), c.Output...)
	c.Output = c.Output[:0]
	return out
}

func (c *CacheNode) AddTarget(targetID string) {
	c.Targets = append(c.Targets, targetID)
}

func (c *CacheNode) GetTargets() []string {
	return c.Targets
}
