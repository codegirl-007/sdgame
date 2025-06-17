package simulation

type CacheNode struct {
	ID             string
	Label          string
	CacheTTL       int
	MaxEntries     int
	EvictionPolicy string
	CurrentLoad    int
	Queue          []Request
	Cache          map[string]interface{}
	Alive          bool
	Targets        []string
}

func (c *CacheNode) GetID() string {
	return c.ID
}

func (c *CacheNode) Type() string {
	return "cache"
}

func (c *CacheNode) GetQueue() []Request {
	return c.Queue
}
func (c *CacheNode) Tick(tick int) {
	if len(c.Cache) > c.MaxEntries {
		evictCount := len(c.Cache) - c.MaxEntries

		i := 0

		for k := range c.Cache {
			delete(c.Cache, k)
			i++
			if i >= evictCount {
				break
			}
		}
	}
}

func (c *CacheNode) Receive(req *Request) {
	c.Queue = append(c.Queue, *req)
}

func (c *CacheNode) Emit() []*Request {
	var out []*Request

	// emit everything in a single tick.
	for _, req := range c.Queue {
		// check cache
		if _, ok := c.Cache[req.ID]; !ok {
			c.Cache[req.ID] = struct{}{}
			out = append(out, &req)
		}
	}
	c.Queue = nil
	return out
}

func (c *CacheNode) IsAlive() bool { return c.Alive }
