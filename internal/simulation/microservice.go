package simulation

import (
	"fmt"
	"hash/fnv"
	"math"
)

type MicroserviceLogic struct{}

type ServiceInstance struct {
	ID           int
	CurrentLoad  int
	HealthStatus string
}

// CacheEntry represents a cached item in the microservice's cache
type MicroserviceCacheEntry struct {
	Data        string
	Timestamp   int
	AccessTime  int
	AccessCount int
}

// hash function for cache keys
func hashKey(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (m MicroserviceLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract microservice properties
	instanceCount := int(AsFloat64(props["instanceCount"]))
	if instanceCount == 0 {
		instanceCount = 1 // default to 1 instance
	}

	cpu := int(AsFloat64(props["cpu"]))
	if cpu == 0 {
		cpu = 2 // default 2 CPU cores
	}

	ramGb := int(AsFloat64(props["ramGb"]))
	if ramGb == 0 {
		ramGb = 4 // default 4GB RAM
	}

	rpsCapacity := int(AsFloat64(props["rpsCapacity"]))
	if rpsCapacity == 0 {
		rpsCapacity = 100 // default capacity per instance
	}

	scalingStrategy := AsString(props["scalingStrategy"])
	if scalingStrategy == "" {
		scalingStrategy = "auto"
	}

	// Calculate base latency based on resource specs
	baseLatencyMs := m.calculateBaseLatency(cpu, ramGb)

	// Auto-scaling logic: adjust instance count based on load
	currentLoad := len(queue)
	if scalingStrategy == "auto" {
		instanceCount = m.autoScale(instanceCount, currentLoad, rpsCapacity)
		props["instanceCount"] = float64(instanceCount) // update for next tick
	}

	// Total capacity across all instances
	totalCapacity := instanceCount * rpsCapacity

	// Process requests up to total capacity
	toProcess := queue
	if len(queue) > totalCapacity {
		toProcess = queue[:totalCapacity]
	}

	// Initialize cache in microservice props
	cache, ok := props["_microserviceCache"].(map[string]*MicroserviceCacheEntry)
	if !ok {
		cache = make(map[string]*MicroserviceCacheEntry)
		props["_microserviceCache"] = cache
	}

	cacheTTL := 300000        // 5 minutes default TTL
	currentTime := tick * 100 // assuming 100ms per tick

	output := []*Request{}     // Only cache misses go here (forwarded to database)
	cacheHits := []*Request{}  // Cache hits - completed locally
	dbRequests := []*Request{} // Requests that need to go to database

	// Process each request with cache-aside logic
	for i, req := range toProcess {
		// Generate cache key for this request (simulate URL patterns)
		hashValue := hashKey(req.ID) % 100 // Create 100 possible "URLs"
		cacheKey := fmt.Sprintf("url-%d-%s", hashValue, req.Type)

		// Check cache first (Cache-Aside pattern)
		entry, hit := cache[cacheKey]
		if hit && !m.isCacheExpired(entry, currentTime, cacheTTL) {
			// CACHE HIT - serve from cache (NO DATABASE QUERY)

			reqCopy := *req
			reqCopy.LatencyMS += 1 // 1ms for cache access
			reqCopy.Path = append(reqCopy.Path, "microservice-cache-hit-completed")

			// Update cache access tracking
			entry.AccessTime = currentTime
			entry.AccessCount++

			// Cache hits do NOT go to database - they complete here
			// In a real system, this response would go back to the client
			// Store separately - these do NOT get forwarded to database
			cacheHits = append(cacheHits, &reqCopy)

		} else {
			// CACHE MISS - need to query database

			reqCopy := *req

			// Add microservice processing latency
			processingLatency := baseLatencyMs

			// Simulate CPU-bound vs I/O-bound operations
			if req.Type == "GET" {
				processingLatency = baseLatencyMs // Fast reads
			} else if req.Type == "POST" || req.Type == "PUT" {
				processingLatency = baseLatencyMs + 10 // Writes take longer
			} else if req.Type == "COMPUTE" {
				processingLatency = baseLatencyMs + 50 // CPU-intensive operations
			}

			// Instance load affects latency (queuing delay)
			instanceLoad := m.calculateInstanceLoad(i, len(toProcess), instanceCount)
			if float64(instanceLoad) > float64(rpsCapacity)*0.8 { // Above 80% capacity
				processingLatency += int(float64(processingLatency) * 0.5) // 50% penalty
			}

			reqCopy.LatencyMS += processingLatency
			reqCopy.Path = append(reqCopy.Path, "microservice-cache-miss")

			// Store cache key in request for when database response comes back
			reqCopy.CacheKey = cacheKey

			// Forward to database for actual data
			dbRequests = append(dbRequests, &reqCopy)
		}
	}

	// For cache misses, we would normally wait for database response and then cache it
	// In this simulation, we'll immediately cache the "result" for future requests
	for _, req := range dbRequests {
		// Simulate caching the database response
		cache[req.CacheKey] = &MicroserviceCacheEntry{
			Data:        "cached-response-data",
			Timestamp:   currentTime,
			AccessTime:  currentTime,
			AccessCount: 1,
		}

		// Forward request to database
		output = append(output, req)
	}

	// Health check: service is healthy if not severely overloaded
	healthy := len(queue) <= totalCapacity*2 // Allow some buffering

	return output, healthy
}

// isCacheExpired checks if a cache entry has expired
func (m MicroserviceLogic) isCacheExpired(entry *MicroserviceCacheEntry, currentTime, ttl int) bool {
	return (currentTime - entry.Timestamp) > ttl
}

// calculateBaseLatency determines base processing time based on resources
func (m MicroserviceLogic) calculateBaseLatency(cpu, ramGb int) int {
	// Better CPU and RAM = lower base latency
	// Formula: base latency inversely proportional to resources
	cpuFactor := float64(cpu)
	ramFactor := float64(ramGb) / 4.0 // Normalize to 4GB baseline

	resourceScore := cpuFactor * ramFactor
	if resourceScore < 1 {
		resourceScore = 1
	}

	baseLatency := int(50.0 / resourceScore) // 50ms baseline for 2CPU/4GB
	if baseLatency < 5 {
		baseLatency = 5 // Minimum 5ms processing time
	}

	return baseLatency
}

// autoScale implements simple auto-scaling logic
func (m MicroserviceLogic) autoScale(currentInstances, currentLoad, rpsPerInstance int) int {
	// Calculate desired instances based on current load
	desiredInstances := int(math.Ceil(float64(currentLoad) / float64(rpsPerInstance)))

	// Scale up/down gradually (max 25% change per tick)
	maxChange := int(math.Max(1, float64(currentInstances)*0.25))

	if desiredInstances > currentInstances {
		// Scale up
		newInstances := currentInstances + maxChange
		if newInstances > desiredInstances {
			newInstances = desiredInstances
		}
		// Cap at reasonable maximum
		if newInstances > 20 {
			newInstances = 20
		}
		return newInstances
	} else if desiredInstances < currentInstances {
		// Scale down (more conservative)
		newInstances := currentInstances - int(math.Max(1, float64(maxChange)*0.5))
		if newInstances < desiredInstances {
			newInstances = desiredInstances
		}
		// Always maintain at least 1 instance
		if newInstances < 1 {
			newInstances = 1
		}
		return newInstances
	}

	return currentInstances
}

// calculateInstanceLoad estimates load on a specific instance
func (m MicroserviceLogic) calculateInstanceLoad(instanceID, totalRequests, instanceCount int) int {
	// Simple round-robin distribution
	baseLoad := totalRequests / instanceCount
	remainder := totalRequests % instanceCount

	if instanceID < remainder {
		return baseLoad + 1
	}
	return baseLoad
}
