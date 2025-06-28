package simulation

type CDNLogic struct{}

func (c CDNLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// TTL (time-to-live) determines how long cached content stays fresh
	ttl := int(AsFloat64(props["ttlMs"]))
	cache, ok := props["_cache"].(map[string]int)
	if !ok {
		cache = make(map[string]int)
		props["_cache"] = cache
	}

	output := []*Request{}

	for _, req := range queue {
		path := req.ID // using request ID as a stand-in for "path"
		lastCached, ok := cache[path]
		if !ok || tick*1000-lastCached > ttl {
			// Cache miss or stale
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, "miss")
			reqCopy.LatencyMS += 50 // simulate extra latency for cache miss
			output = append(output, &reqCopy)
			cache[path] = tick * 1000
		} else {
			// Cache hit — suppress forwarding
			continue
		}
	}

	return output, true
}
