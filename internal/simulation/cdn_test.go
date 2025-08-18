package simulation

import (
	"fmt"
	"testing"
)

func TestCDNLogic(t *testing.T) {
	cdn := CDNLogic{}
	cache := map[string]int{} // shared mutable cache
	props := map[string]any{
		"ttl":    float64(1000),
		"_cache": cache,
	}

	reqA := &Request{
		ID:        "asset-123",
		Timestamp: 0,
		Path:      []string{"cdn"},
		LatencyMS: 0,
	}

	reqB := &Request{
		ID:        "asset-456",
		Timestamp: 0,
		Path:      []string{"cdn"},
		LatencyMS: 0,
	}

	// Tick 0 — both A and B should MISS and forward with added latency
	output, _ := cdn.Tick(props, []*Request{reqA, reqB}, 0)
	fmt.Printf("Tick 0 Output: %+v\n", output)
	fmt.Printf("Cache after Tick 0: %+v\n", cache)
	if len(output) != 2 {
		t.Errorf("Expected 2 forwarded requests on cache miss")
	}
	for _, o := range output {
		if o.LatencyMS != 50 {
			t.Errorf("Expected 50ms latency on miss, got %d", o.LatencyMS)
		}
	}
	if len(cache) != 2 {
		t.Errorf("Expected 2 items in cache after miss, got %d", len(cache))
	}

	// Tick 1 — both A and B should HIT and be suppressed
	reqA1 := &Request{ID: "asset-123", Timestamp: 1000, Path: []string{"cdn"}, LatencyMS: 0}
	reqB1 := &Request{ID: "asset-456", Timestamp: 1000, Path: []string{"cdn"}, LatencyMS: 0}
	output, _ = cdn.Tick(props, []*Request{reqA1, reqB1}, 1)
	fmt.Printf("Tick 1 Output: %+v\n", output)
	fmt.Printf("Cache after Tick 1: %+v\n", cache)
	if len(output) != 0 {
		t.Errorf("Expected all requests to be cached and suppressed on hit")
	}

	// Tick 11 — simulate expiry for A (TTL = 1000ms, Tick 11 = 11000ms)
	reqA2 := &Request{ID: "asset-123", Timestamp: 11000, Path: []string{"cdn"}, LatencyMS: 0}
	output, _ = cdn.Tick(props, []*Request{reqA2}, 11)
	fmt.Printf("Tick 11 Output A: %+v\n", output)
	fmt.Printf("Cache after Tick 11 A: %+v\n", cache)
	if len(output) != 1 {
		t.Errorf("Expected request A to be forwarded again after TTL expiry")
	} else if output[0].LatencyMS != 50 {
		t.Errorf("Expected 50ms latency on cache refresh, got %d", output[0].LatencyMS)
	}

	// B should still HIT (suppressed)
	reqB2 := &Request{ID: "asset-456", Timestamp: 1000, Path: []string{"cdn"}, LatencyMS: 0}
	output, _ = cdn.Tick(props, []*Request{reqB2}, 11)
	fmt.Printf("Tick 11 Output B: %+v\n", output)
	fmt.Printf("Cache after Tick 11 B: %+v\n", cache)
	if len(output) > 0 {
		t.Errorf("Expected request B to be suppressed due to valid cache")
	}
}
