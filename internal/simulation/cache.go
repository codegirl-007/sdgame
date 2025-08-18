package simulation

import (
	"time"
)

type CacheLogic struct{}

type CacheEntry struct {
	Data        string
	Timestamp   int
	AccessTime  int
	AccessCount int
	InsertOrder int
}

func (c CacheLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract cache properties
	cacheTTL := int(AsFloat64(props["cacheTTL"]))
	if cacheTTL == 0 {
		cacheTTL = 300000 // default 5 minutes in ms
	}

	maxEntries := int(AsFloat64(props["maxEntries"]))
	if maxEntries == 0 {
		maxEntries = 1000 // default max entries
	}

	evictionPolicy := AsString(props["evictionPolicy"])
	if evictionPolicy == "" {
		evictionPolicy = "LRU" // default eviction policy
	}

	// Initialize cache data structures in props
	cacheData, ok := props["_cacheData"].(map[string]*CacheEntry)
	if !ok {
		cacheData = make(map[string]*CacheEntry)
		props["_cacheData"] = cacheData
	}

	insertCounter, ok := props["_insertCounter"].(int)
	if !ok {
		insertCounter = 0
	}

	// Current timestamp for this tick
	currentTime := tick * 100 // assuming 100ms per tick

	// Clean up expired entries first
	c.cleanExpiredEntries(cacheData, currentTime, cacheTTL)

	output := []*Request{}

	for _, req := range queue {
		cacheKey := req.ID + "-" + req.Type // Use request ID and type as cache key

		// Check for cache hit
		entry, hit := cacheData[cacheKey]
		if hit && !c.isExpired(entry, currentTime, cacheTTL) {
			// Cache hit - return immediately with minimal latency
			reqCopy := *req
			reqCopy.LatencyMS += 1 // 1ms for in-memory access
			reqCopy.Path = append(reqCopy.Path, "cache-hit")

			// Update access tracking for eviction policies
			entry.AccessTime = currentTime
			entry.AccessCount++

			output = append(output, &reqCopy)
		} else {
			// Cache miss - forward request downstream
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, "cache-miss")

			// For simulation purposes, we'll cache the "response" immediately
			// In a real system, this would happen when the response comes back
			insertCounter++
			newEntry := &CacheEntry{
				Data:        "cached-data", // In real implementation, this would be the response data
				Timestamp:   currentTime,
				AccessTime:  currentTime,
				AccessCount: 1,
				InsertOrder: insertCounter,
			}

			// First check if we need to evict before adding
			if len(cacheData) >= maxEntries {
				c.evictEntry(cacheData, evictionPolicy)
			}

			// Now add the new entry
			cacheData[cacheKey] = newEntry

			output = append(output, &reqCopy)
		}
	}

	// Update insert counter in props
	props["_insertCounter"] = insertCounter

	return output, true
}

func (c CacheLogic) cleanExpiredEntries(cacheData map[string]*CacheEntry, currentTime, ttl int) {
	for key, entry := range cacheData {
		if c.isExpired(entry, currentTime, ttl) {
			delete(cacheData, key)
		}
	}
}

func (c CacheLogic) isExpired(entry *CacheEntry, currentTime, ttl int) bool {
	return (currentTime - entry.Timestamp) > ttl
}

func (c CacheLogic) evictEntry(cacheData map[string]*CacheEntry, policy string) {
	if len(cacheData) == 0 {
		return
	}

	var keyToEvict string

	switch policy {
	case "LRU":
		// Evict least recently used
		oldestTime := int(^uint(0) >> 1) // Max int
		for key, entry := range cacheData {
			if entry.AccessTime < oldestTime {
				oldestTime = entry.AccessTime
				keyToEvict = key
			}
		}

	case "LFU":
		// Evict least frequently used
		minCount := int(^uint(0) >> 1) // Max int
		for key, entry := range cacheData {
			if entry.AccessCount < minCount {
				minCount = entry.AccessCount
				keyToEvict = key
			}
		}

	case "FIFO":
		// Evict first in (oldest insert order)
		minOrder := int(^uint(0) >> 1) // Max int
		for key, entry := range cacheData {
			if entry.InsertOrder < minOrder {
				minOrder = entry.InsertOrder
				keyToEvict = key
			}
		}

	case "random":
		// Evict random entry
		keys := make([]string, 0, len(cacheData))
		for key := range cacheData {
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			// Use timestamp as pseudo-random seed
			seed := time.Now().UnixNano()
			keyToEvict = keys[seed%int64(len(keys))]
		}

	default:
		// Default to LRU
		oldestTime := int(^uint(0) >> 1)
		for key, entry := range cacheData {
			if entry.AccessTime < oldestTime {
				oldestTime = entry.AccessTime
				keyToEvict = key
			}
		}
	}

	if keyToEvict != "" {
		delete(cacheData, keyToEvict)
	}
}
