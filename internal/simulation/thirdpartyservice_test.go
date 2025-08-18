package simulation

import (
	"testing"
)

func TestThirdPartyServiceLogic_BasicProcessing(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Stripe",
		"latency":  150.0,
	}

	requests := []*Request{
		{ID: "1", Type: "POST", LatencyMS: 50, Path: []string{}},
		{ID: "2", Type: "GET", LatencyMS: 30, Path: []string{}},
	}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected third party service to be healthy")
	}

	if len(output) != 2 {
		t.Errorf("Expected 2 processed requests, got %d", len(output))
	}

	// Verify latency was added (should be around base latency with some variance)
	for i, req := range output {
		originalLatency := requests[i].LatencyMS
		if req.LatencyMS <= originalLatency {
			t.Errorf("Expected third party service latency to be added")
		}

		// Check that path was updated
		if len(req.Path) == 0 {
			t.Error("Expected path to be updated")
		}

		lastPathElement := req.Path[len(req.Path)-1]
		if lastPathElement != "third-party-success" && lastPathElement != "third-party-failed" {
			t.Errorf("Expected path to indicate success or failure, got %s", lastPathElement)
		}
	}
}

func TestThirdPartyServiceLogic_ProviderCharacteristics(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	providers := []string{"Stripe", "AWS", "Slack", "Twilio"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			props := map[string]any{
				"provider": provider,
				"latency":  100.0,
			}

			requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 0, Path: []string{}}}

			output, healthy := logic.Tick(props, requests, 1)

			if !healthy {
				t.Errorf("Expected %s service to be healthy", provider)
			}

			if len(output) != 1 {
				t.Errorf("Expected 1 processed request for %s", provider)
			}

			// Verify latency characteristics
			addedLatency := output[0].LatencyMS
			if addedLatency <= 0 {
				t.Errorf("Expected %s to add latency", provider)
			}

			// AWS and Google should be faster than Slack
			if provider == "AWS" && addedLatency > 200 {
				t.Errorf("Expected AWS to have lower latency, got %dms", addedLatency)
			}
		})
	}
}

func TestThirdPartyServiceLogic_RateLimiting(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Slack", // Has low rate limit (30 RPS)
		"latency":  100.0,
	}

	// Send more requests than rate limit
	requests := make([]*Request, 50) // More than Slack's 30 RPS limit
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "POST", LatencyMS: 0}
	}

	output, healthy := logic.Tick(props, requests, 1)

	// Should only process up to rate limit
	if len(output) != 30 {
		t.Errorf("Expected 30 processed requests due to Slack rate limit, got %d", len(output))
	}

	// Service should still be healthy with rate limiting
	if !healthy {
		t.Error("Expected service to be healthy despite rate limiting")
	}

	// Check that rate limit hits were recorded
	status, ok := props["_serviceStatus"].(ServiceStatus)
	if !ok {
		t.Error("Expected service status to be recorded")
	}

	if status.RateLimitHits != 1 {
		t.Errorf("Expected 1 rate limit hit, got %d", status.RateLimitHits)
	}
}

func TestThirdPartyServiceLogic_ServiceFailure(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Generic",
		"latency":  100.0,
	}

	// Set up service as already having failures
	status := ServiceStatus{
		IsUp:         false,
		LastCheck:    0,
		FailureCount: 6,
	}
	props["_serviceStatus"] = status

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 50, Path: []string{}}}

	output, healthy := logic.Tick(props, requests, 1)

	if healthy {
		t.Error("Expected service to be unhealthy when external service is down")
	}

	if len(output) != 1 {
		t.Error("Expected request to be processed even when service is down")
	}

	// Should have very high latency due to timeout
	if output[0].LatencyMS < 5000 {
		t.Errorf("Expected high latency for service failure, got %dms", output[0].LatencyMS)
	}

	// Check path indicates timeout
	lastPath := output[0].Path[len(output[0].Path)-1]
	if lastPath != "third-party-timeout" {
		t.Errorf("Expected timeout path, got %s", lastPath)
	}
}

func TestThirdPartyServiceLogic_ServiceRecovery(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Stripe",
		"latency":  100.0,
	}

	// Set up service as down but with old timestamp (should recover)
	status := ServiceStatus{
		IsUp:         false,
		LastCheck:    0, // Very old timestamp
		FailureCount: 3,
	}
	props["_serviceStatus"] = status

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 50, Path: []string{}}}

	// Run with current tick that's more than 30 seconds later
	_, healthy := logic.Tick(props, requests, 400) // 40 seconds later

	if !healthy {
		t.Error("Expected service to be healthy after recovery")
	}

	// Check that service recovered
	updatedStatus, ok := props["_serviceStatus"].(ServiceStatus)
	if !ok {
		t.Error("Expected updated service status")
	}

	if !updatedStatus.IsUp {
		t.Error("Expected service to have recovered")
	}

	if updatedStatus.FailureCount != 0 {
		t.Error("Expected failure count to be reset on recovery")
	}
}

func TestThirdPartyServiceLogic_ReliabilityDifferences(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	// Test different reliability levels
	testCases := []struct {
		provider            string
		expectedReliability float64
	}{
		{"AWS", 0.9995},
		{"Google", 0.9999},
		{"Stripe", 0.999},
		{"Slack", 0.995},
		{"Generic", 0.99},
	}

	for _, tc := range testCases {
		reliability := logic.getProviderReliability(tc.provider)
		if reliability != tc.expectedReliability {
			t.Errorf("Expected %s reliability %.4f, got %.4f",
				tc.provider, tc.expectedReliability, reliability)
		}
	}
}

func TestThirdPartyServiceLogic_RateLimitDifferences(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	// Test different rate limits
	testCases := []struct {
		provider      string
		expectedLimit int
	}{
		{"AWS", 1000},
		{"Stripe", 100},
		{"Slack", 30},
		{"SendGrid", 200},
		{"Twilio", 50},
	}

	for _, tc := range testCases {
		rateLimit := logic.getProviderRateLimit(tc.provider)
		if rateLimit != tc.expectedLimit {
			t.Errorf("Expected %s rate limit %d, got %d",
				tc.provider, tc.expectedLimit, rateLimit)
		}
	}
}

func TestThirdPartyServiceLogic_LatencyVariance(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Stripe",
		"latency":  100.0,
	}

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 0, Path: []string{}}}

	latencies := []int{}

	// Run multiple times to observe variance
	for i := 0; i < 10; i++ {
		output, _ := logic.Tick(props, requests, i)
		latencies = append(latencies, output[0].LatencyMS)
	}

	// Check that we have variance (not all latencies are the same)
	allSame := true
	firstLatency := latencies[0]
	for _, latency := range latencies[1:] {
		if latency != firstLatency {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Expected latency variance, but all latencies were the same")
	}

	// All latencies should be reasonable (between 50ms and 300ms for Stripe)
	for _, latency := range latencies {
		if latency < 50 || latency > 300 {
			t.Errorf("Expected reasonable latency for Stripe, got %dms", latency)
		}
	}
}

func TestThirdPartyServiceLogic_DefaultValues(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 0, Path: []string{}}}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected service to be healthy with default values")
	}

	if len(output) != 1 {
		t.Error("Expected 1 processed request with defaults")
	}

	// Should have reasonable default latency (around 200ms base)
	if output[0].LatencyMS < 100 || output[0].LatencyMS > 400 {
		t.Errorf("Expected reasonable default latency, got %dms", output[0].LatencyMS)
	}
}

func TestThirdPartyServiceLogic_SuccessCountTracking(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "AWS", // High reliability
		"latency":  50.0,
	}

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 0, Path: []string{}}}

	// Run multiple successful requests
	for i := 0; i < 5; i++ {
		logic.Tick(props, requests, i)
	}

	status, ok := props["_serviceStatus"].(ServiceStatus)
	if !ok {
		t.Error("Expected service status to be tracked")
	}

	// Should have accumulated success count
	if status.SuccessCount == 0 {
		t.Error("Expected success count to be tracked")
	}

	// Should be healthy
	if !status.IsUp {
		t.Error("Expected service to remain up with successful calls")
	}
}

func TestThirdPartyServiceLogic_FailureRecovery(t *testing.T) {
	logic := ThirdPartyServiceLogic{}

	props := map[string]any{
		"provider": "Generic",
		"latency":  100.0,
	}

	// Set up service with some failures but still up
	status := ServiceStatus{
		IsUp:         true,
		FailureCount: 3,
		SuccessCount: 0,
	}
	props["_serviceStatus"] = status

	requests := []*Request{{ID: "1", Type: "POST", LatencyMS: 0, Path: []string{}}}

	// Simulate a successful call (with high probability for Generic service)
	// We'll run this multiple times to ensure we get at least one success
	successFound := false
	for i := 0; i < 10 && !successFound; i++ {
		output, _ := logic.Tick(props, requests, i)
		if len(output[0].Path) > 0 && output[0].Path[len(output[0].Path)-1] == "third-party-success" {
			successFound = true
		}
	}

	if successFound {
		updatedStatus, _ := props["_serviceStatus"].(ServiceStatus)
		// Failure count should have decreased
		if updatedStatus.FailureCount >= 3 {
			t.Error("Expected failure count to decrease after successful call")
		}
	}
}
