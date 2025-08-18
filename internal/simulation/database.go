package simulation

type DatabaseLogic struct{}

func (d DatabaseLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract database properties
	replication := int(AsFloat64(props["replication"]))
	if replication == 0 {
		replication = 1 // default
	}

	// Database capacity (could be based on instance size or explicit RPS)
	maxRPS := int(AsFloat64(props["maxRPS"]))
	if maxRPS == 0 {
		maxRPS = 1000 // default capacity
	}

	// Base latency for database operations
	baseLatencyMs := int(AsFloat64(props["baseLatencyMs"]))
	if baseLatencyMs == 0 {
		baseLatencyMs = 10 // default 10ms for local DB operations
	}

	// Process requests up to capacity
	toProcess := queue
	if len(queue) > maxRPS {
		toProcess = queue[:maxRPS]
		// TODO: Could add queue overflow logic here
	}

	output := []*Request{}

	for _, req := range toProcess {
		// Add database latency to the request
		reqCopy := *req

		// Simulate different operation types and their latencies
		operationLatency := baseLatencyMs

		// Simple heuristic: reads are faster than writes
		if req.Type == "GET" || req.Type == "READ" {
			operationLatency = baseLatencyMs
		} else if req.Type == "POST" || req.Type == "WRITE" {
			operationLatency = baseLatencyMs * 2 // writes take longer
		}

		// Replication factor affects write latency
		if req.Type == "POST" || req.Type == "WRITE" {
			operationLatency += (replication - 1) * 5 // 5ms per replica
		}

		reqCopy.LatencyMS += operationLatency
		reqCopy.Path = append(reqCopy.Path, "database-processed")

		output = append(output, &reqCopy)
	}

	// Database health (could simulate failures, connection issues, etc.)
	// For now, assume always healthy
	return output, true
}
