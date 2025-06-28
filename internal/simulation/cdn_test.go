package simulation

import (
	"testing"
)

func TestCDNLogic(t *testing.T) {
	cdn := CDNLogic{}
	props := map[string]any{
		"ttlMs":  float64(1000),
		"_cache": map[string]int{}, // initial empty cache
	}

	req := &Request{
		ID:        "asset-123",
		Timestamp: 0,
		Path:      []string{"cdn"},
		LatencyMS: 0,
	}

	// Tick 0 — should MISS and forward with added latency
	output, _ := cdn.Tick(props, []*Request{req}, 0)
	if len(output) != 1 {
		t.Errorf("Expected request to pass through on first miss")
	} else if output[0].LatencyMS != 50 {
		t.Errorf("Expected latency to be 50ms on cache miss, got %d", output[0].LatencyMS)
	}

	// Tick 1 — should HIT and suppress
	output, _ = cdn.Tick(props, []*Request{req}, 1)
	if len(output) != 0 {
		t.Errorf("Expected request to be cached and suppressed on hit")
	}

	// Tick 11 — simulate expiry (assuming TickMs = 100, so Tick 11 = 1100ms)
	output, _ = cdn.Tick(props, []*Request{req}, 11)
	if len(output) != 1 {
		t.Errorf("Expected request to be forwarded again after TTL expiry")
	} else if output[0].LatencyMS != 50 {
		t.Errorf("Expected latency to be 50ms on cache refresh, got %d", output[0].LatencyMS)
	}
}
