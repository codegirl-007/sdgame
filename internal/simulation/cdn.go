package simulation

type CDNLogic struct{}

func (c CDNLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {

	// read the ttl for cached content
	ttl := int(AsFloat64(props["ttl"]))

	// retrieve the cdn's cache from props
	cache, ok := props["_cache"].(map[string]int)
	if !ok {
		cache = make(map[string]int)
		props["_cache"] = cache
	}

	// prepare a list to collect output requests (those that are forwarded past the cdn)
	output := []*Request{}

	// iterate over each request in the queue
	for _, req := range queue {
		path := req.ID
		lastCached, ok := cache[path]
		// check if it has been more than ttl seconds since this content was last cached?
		if !ok || (req.Timestamp-lastCached) > ttl {
			// Cache miss or stale
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, "miss")
			reqCopy.LatencyMS += 50
			output = append(output, &reqCopy)
			cache[path] = req.Timestamp
		}
		// else cache hit, suppressed
	}

	return output, true
}
