package simulation

import (
	"testing"
)

func TestCacheLogic_CacheHitMiss(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000, // 10 seconds
		"maxEntries":     100,
		"evictionPolicy": "LRU",
	}

	// First request should be a miss
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0, Path: []string{"start"}}}
	output1, alive := cache.Tick(props, req1, 1)

	if !alive {
		t.Errorf("Cache should be alive")
	}

	if len(output1) != 1 {
		t.Errorf("Expected 1 output request, got %d", len(output1))
	}

	// Should be cache miss
	if output1[0].LatencyMS != 0 { // No latency added for miss
		t.Errorf("Expected 0ms latency for cache miss, got %dms", output1[0].LatencyMS)
	}

	// Check path contains cache-miss
	found := false
	for _, pathItem := range output1[0].Path {
		if pathItem == "cache-miss" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected cache-miss in path, got %v", output1[0].Path)
	}

	// Second identical request should be a hit
	req2 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0, Path: []string{"start"}}}
	output2, _ := cache.Tick(props, req2, 2)

	if len(output2) != 1 {
		t.Errorf("Expected 1 output request, got %d", len(output2))
	}

	// Should be cache hit with 1ms latency
	if output2[0].LatencyMS != 1 {
		t.Errorf("Expected 1ms latency for cache hit, got %dms", output2[0].LatencyMS)
	}

	// Check path contains cache-hit
	found = false
	for _, pathItem := range output2[0].Path {
		if pathItem == "cache-hit" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected cache-hit in path, got %v", output2[0].Path)
	}
}

func TestCacheLogic_TTLExpiration(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       1000, // 1 second
		"maxEntries":     100,
		"evictionPolicy": "LRU",
	}

	// First request - cache miss
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1, 1)

	// Second request within TTL - cache hit
	req2 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output2, _ := cache.Tick(props, req2, 5) // 5 * 100ms = 500ms later

	if output2[0].LatencyMS != 1 {
		t.Errorf("Expected cache hit (1ms), got %dms", output2[0].LatencyMS)
	}

	// Third request after TTL expiration - cache miss
	req3 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output3, _ := cache.Tick(props, req3, 15) // 15 * 100ms = 1500ms later (expired)

	if output3[0].LatencyMS != 0 {
		t.Errorf("Expected cache miss (0ms) after TTL expiration, got %dms", output3[0].LatencyMS)
	}
}

func TestCacheLogic_MaxEntriesEviction(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000,
		"maxEntries":     2, // Small cache size
		"evictionPolicy": "LRU",
	}

	// Add first entry
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1, 1)

	// Add second entry
	req2 := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req2, 2)

	// Verify both are cached
	req1Check := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1Check, _ := cache.Tick(props, req1Check, 3)
	if output1Check[0].LatencyMS != 1 {
		t.Errorf("Expected cache hit for req1, got %dms latency", output1Check[0].LatencyMS)
	}

	req2Check := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	output2Check, _ := cache.Tick(props, req2Check, 4)
	if output2Check[0].LatencyMS != 1 {
		t.Errorf("Expected cache hit for req2, got %dms latency", output2Check[0].LatencyMS)
	}

	// Add third entry (should evict LRU entry)
	req3 := []*Request{{ID: "req3", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req3, 5)

	// req1 was accessed at tick 3, req2 at tick 4, so req1 should be evicted
	req1CheckAgain := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1, _ := cache.Tick(props, req1CheckAgain, 6)
	if output1[0].LatencyMS != 0 {
		t.Errorf("Expected cache miss for LRU evicted entry, got %dms latency", output1[0].LatencyMS)
	}

	// After adding req1 back, the cache should be at capacity with different items
	// We don't test further to avoid complex cascading eviction scenarios
}

func TestCacheLogic_LRUEviction(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000,
		"maxEntries":     2,
		"evictionPolicy": "LRU",
	}

	// Add two entries
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1, 1)

	req2 := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req2, 2)

	// Access first entry (make it recently used)
	req1Access := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1Access, 3)

	// Add third entry (should evict req2, since req1 was more recently accessed)
	req3 := []*Request{{ID: "req3", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req3, 4)

	// Verify that req2 was evicted (should be cache miss)
	req2Check := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	output2, _ := cache.Tick(props, req2Check, 5)

	if output2[0].LatencyMS != 0 {
		t.Errorf("Expected cache miss for LRU evicted entry, got %dms latency", output2[0].LatencyMS)
	}

	// After adding req2 back, the cache should contain {req2, req1} or {req2, req3}
	// depending on LRU logic. We don't test further to avoid cascading evictions.
}

func TestCacheLogic_FIFOEviction(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000,
		"maxEntries":     2,
		"evictionPolicy": "FIFO",
	}

	// Add two entries
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1, 1)

	req2 := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req2, 2)

	// Access first entry multiple times (shouldn't matter for FIFO)
	req1Access := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1Access, 3)
	cache.Tick(props, req1Access, 4)

	// Add third entry (should evict req1, the first inserted)
	req3 := []*Request{{ID: "req3", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req3, 5)

	// Check that req1 was evicted (first in, first out)
	req1Check := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1, _ := cache.Tick(props, req1Check, 6)

	if output1[0].LatencyMS != 0 {
		t.Errorf("Expected cache miss for FIFO evicted entry, got %dms latency", output1[0].LatencyMS)
	}

	// After adding req1 back, the cache should contain {req2, req1} or {req3, req1}
	// depending on FIFO logic. We don't test further to avoid cascading evictions.
}

func TestCacheLogic_DefaultValues(t *testing.T) {
	cache := CacheLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	req := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output, _ := cache.Tick(props, req, 1)

	if len(output) != 1 {
		t.Errorf("Expected 1 output request")
	}

	// Should be cache miss with 0ms latency
	if output[0].LatencyMS != 0 {
		t.Errorf("Expected 0ms latency for cache miss with defaults, got %dms", output[0].LatencyMS)
	}

	// Second request should be cache hit
	req2 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output2, _ := cache.Tick(props, req2, 2)

	if output2[0].LatencyMS != 1 {
		t.Errorf("Expected 1ms latency for cache hit, got %dms", output2[0].LatencyMS)
	}
}

func TestCacheLogic_SimpleEviction(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000,
		"maxEntries":     1, // Only 1 entry allowed
		"evictionPolicy": "LRU",
	}

	// Add first entry
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1, _ := cache.Tick(props, req1, 1)
	if output1[0].LatencyMS != 0 {
		t.Errorf("First request should be cache miss, got %dms", output1[0].LatencyMS)
	}

	// Check it's cached
	req1Again := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1Again, _ := cache.Tick(props, req1Again, 2)
	if output1Again[0].LatencyMS != 1 {
		t.Errorf("Second request should be cache hit, got %dms", output1Again[0].LatencyMS)
	}

	// Add second entry (should evict first)
	req2 := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	output2, _ := cache.Tick(props, req2, 3)
	if output2[0].LatencyMS != 0 {
		t.Errorf("New request should be cache miss, got %dms", output2[0].LatencyMS)
	}

	// Check that first entry is now evicted
	req1Final := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1Final, _ := cache.Tick(props, req1Final, 4)
	if output1Final[0].LatencyMS != 0 {
		t.Errorf("Evicted entry should be cache miss, got %dms", output1Final[0].LatencyMS)
	}

	// Check that second entry is now also evicted (since req1 was re-added in step 4)
	req2Again := []*Request{{ID: "req2", Type: "GET", LatencyMS: 0}}
	output2Again, _ := cache.Tick(props, req2Again, 5)
	if output2Again[0].LatencyMS != 0 {
		t.Errorf("Re-evicted entry should be cache miss, got %dms", output2Again[0].LatencyMS)
	}
}

func TestCacheLogic_DifferentRequestTypes(t *testing.T) {
	cache := CacheLogic{}

	props := map[string]any{
		"cacheTTL":       10000,
		"maxEntries":     100,
		"evictionPolicy": "LRU",
	}

	// Same ID but different type should be different cache entries
	req1 := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	cache.Tick(props, req1, 1)

	req2 := []*Request{{ID: "req1", Type: "POST", LatencyMS: 0}}
	output2, _ := cache.Tick(props, req2, 2)

	// Should be cache miss since different type
	if output2[0].LatencyMS != 0 {
		t.Errorf("Expected cache miss for different request type, got %dms latency", output2[0].LatencyMS)
	}

	// Original GET should still be cached
	req1Again := []*Request{{ID: "req1", Type: "GET", LatencyMS: 0}}
	output1, _ := cache.Tick(props, req1Again, 3)

	if output1[0].LatencyMS != 1 {
		t.Errorf("Expected cache hit for original request type, got %dms latency", output1[0].LatencyMS)
	}
}
