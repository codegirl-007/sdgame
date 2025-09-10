package simulation

// keep this stateless. Trying to store state here is a big mistake.
// This is meant to be a pure logic handler.
type WebServerLogic struct {
}

func (l WebServerLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	maxRPS := int(AsFloat64(props["rpsCapacity"]))

	toProcess := queue
	if len(queue) > maxRPS {
		toProcess = queue[:maxRPS]
	}

	// Get base latency for web server operations
	baseLatencyMs := int(AsFloat64(props["baseLatencyMs"]))
	if baseLatencyMs == 0 {
		baseLatencyMs = 20 // default 20ms for web server processing
	}

	var output []*Request
	for _, req := range toProcess {
		// Create a copy of the request to preserve existing latency
		reqCopy := *req

		// Add web server processing latency
		reqCopy.LatencyMS += baseLatencyMs
		reqCopy.Path = append(reqCopy.Path, "webserver-processed")

		output = append(output, &reqCopy)
	}

	return output, true
}
