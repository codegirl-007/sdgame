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

	var output []*Request
	for _, req := range toProcess {
		output = append(output, &Request{
			ID:        req.ID,
			Timestamp: req.Timestamp,
			Origin:    req.Origin,
			Type:      req.Type,
		})
	}

	return output, true
}
