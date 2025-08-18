package simulation

import (
	"math/rand"
)

type ThirdPartyServiceLogic struct{}

type ServiceStatus struct {
	IsUp          bool
	LastCheck     int
	FailureCount  int
	SuccessCount  int
	RateLimitHits int
}

func (t ThirdPartyServiceLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract third-party service properties
	provider := AsString(props["provider"])
	if provider == "" {
		provider = "Generic" // default provider
	}

	baseLatency := int(AsFloat64(props["latency"]))
	if baseLatency == 0 {
		baseLatency = 200 // default 200ms latency
	}

	// Get service status from props (persistent state)
	status, ok := props["_serviceStatus"].(ServiceStatus)
	if !ok {
		status = ServiceStatus{
			IsUp:          true,
			LastCheck:     tick,
			FailureCount:  0,
			SuccessCount:  0,
			RateLimitHits: 0,
		}
	}

	currentTime := tick * 100 // Convert tick to milliseconds

	// Simulate service availability and characteristics based on provider
	reliability := t.getProviderReliability(provider)
	rateLimitRPS := t.getProviderRateLimit(provider)
	latencyVariance := t.getProviderLatencyVariance(provider)

	// Check if service is down and should recover
	if !status.IsUp {
		// Services typically recover after some time
		if currentTime-status.LastCheck > 30000 { // 30 seconds downtime
			status.IsUp = true
			status.FailureCount = 0
		}
	}

	// Apply rate limiting - third-party services often have strict limits
	requestsThisTick := len(queue)
	if requestsThisTick > rateLimitRPS {
		status.RateLimitHits++
		// Only process up to rate limit
		queue = queue[:rateLimitRPS]
	}

	output := []*Request{}

	for _, req := range queue {
		reqCopy := *req

		// Simulate service availability
		if !status.IsUp {
			// Service is down - simulate timeout/error
			reqCopy.LatencyMS += 10000 // 10 second timeout
			reqCopy.Path = append(reqCopy.Path, "third-party-timeout")
			status.FailureCount++
		} else {
			// Service is up - calculate response time
			serviceLatency := t.calculateServiceLatency(provider, baseLatency, latencyVariance)

			// Random failure based on reliability
			if rand.Float64() > reliability {
				// Service call failed
				serviceLatency += 5000 // 5 second timeout on failure
				reqCopy.Path = append(reqCopy.Path, "third-party-failed")
				status.FailureCount++

				// If too many failures, mark service as down
				if status.FailureCount > 5 {
					status.IsUp = false
					status.LastCheck = currentTime
				}
			} else {
				// Successful service call
				reqCopy.Path = append(reqCopy.Path, "third-party-success")
				status.SuccessCount++

				// Reset failure count on successful calls
				if status.FailureCount > 0 {
					status.FailureCount--
				}
			}

			reqCopy.LatencyMS += serviceLatency
		}

		output = append(output, &reqCopy)
	}

	// Update persistent state
	props["_serviceStatus"] = status

	// Health check: service is healthy if external service is up and not excessively rate limited
	// Allow some rate limiting but not too much
	maxRateLimitHits := 10 // Allow up to 10 rate limit hits before considering unhealthy
	healthy := status.IsUp && status.RateLimitHits < maxRateLimitHits

	return output, healthy
}

// getProviderReliability returns the reliability percentage for different providers
func (t ThirdPartyServiceLogic) getProviderReliability(provider string) float64 {
	switch provider {
	case "Stripe":
		return 0.999 // 99.9% uptime
	case "Twilio":
		return 0.998 // 99.8% uptime
	case "SendGrid":
		return 0.997 // 99.7% uptime
	case "AWS":
		return 0.9995 // 99.95% uptime
	case "Google":
		return 0.9999 // 99.99% uptime
	case "Slack":
		return 0.995 // 99.5% uptime
	case "GitHub":
		return 0.996 // 99.6% uptime
	case "Shopify":
		return 0.998 // 99.8% uptime
	default:
		return 0.99 // 99% uptime for generic services
	}
}

// getProviderRateLimit returns the rate limit (requests per tick) for different providers
func (t ThirdPartyServiceLogic) getProviderRateLimit(provider string) int {
	switch provider {
	case "Stripe":
		return 100 // 100 requests per second (per tick in our sim)
	case "Twilio":
		return 50 // More restrictive
	case "SendGrid":
		return 200 // Email is typically higher volume
	case "AWS":
		return 1000 // Very high limits
	case "Google":
		return 500 // High but controlled
	case "Slack":
		return 30 // Very restrictive for chat APIs
	case "GitHub":
		return 60 // GitHub API limits
	case "Shopify":
		return 80 // E-commerce API limits
	default:
		return 100 // Default rate limit
	}
}

// getProviderLatencyVariance returns the latency variance factor for different providers
func (t ThirdPartyServiceLogic) getProviderLatencyVariance(provider string) float64 {
	switch provider {
	case "Stripe":
		return 0.3 // Low variance, consistent performance
	case "Twilio":
		return 0.5 // Moderate variance
	case "SendGrid":
		return 0.4 // Email services are fairly consistent
	case "AWS":
		return 0.2 // Very consistent
	case "Google":
		return 0.25 // Very consistent
	case "Slack":
		return 0.6 // Chat services can be variable
	case "GitHub":
		return 0.4 // Moderate variance
	case "Shopify":
		return 0.5 // E-commerce can be variable under load
	default:
		return 0.5 // Default variance
	}
}

// calculateServiceLatency computes the actual latency including variance
func (t ThirdPartyServiceLogic) calculateServiceLatency(provider string, baseLatency int, variance float64) int {
	// Add random variance to base latency
	varianceMs := float64(baseLatency) * variance
	randomVariance := (rand.Float64() - 0.5) * 2 * varianceMs // -variance to +variance

	finalLatency := float64(baseLatency) + randomVariance

	// Ensure minimum latency (can't be negative or too low)
	if finalLatency < 10 {
		finalLatency = 10
	}

	// Add provider-specific baseline adjustments
	switch provider {
	case "AWS", "Google":
		// Cloud providers are typically fast
		finalLatency *= 0.8
	case "Slack":
		// Chat APIs can be slower
		finalLatency *= 1.2
	case "Twilio":
		// Telecom APIs have processing overhead
		finalLatency *= 1.1
	}

	return int(finalLatency)
}
