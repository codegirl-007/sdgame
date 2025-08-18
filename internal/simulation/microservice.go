package simulation

import "math"

type MicroserviceLogic struct{}

type ServiceInstance struct {
	ID           int
	CurrentLoad  int
	HealthStatus string
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

	output := []*Request{}

	// Distribute requests across instances using round-robin
	for i, req := range toProcess {

		// Create processed request copy
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
		reqCopy.Path = append(reqCopy.Path, "microservice-processed")

		output = append(output, &reqCopy)
	}

	// Health check: service is healthy if not severely overloaded
	healthy := len(queue) <= totalCapacity*2 // Allow some buffering

	return output, healthy
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
