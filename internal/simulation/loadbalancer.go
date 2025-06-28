package simulation

type LoadBalancerLogic struct{}

func (l LoadBalancerLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	algorithm := AsString(props["algorithm"])

	targetIDs, ok := props["_targetIDs"].([]string)
	if !ok || len(targetIDs) == 0 || len(queue) == 0 {
		return nil, true
	}

	output := []*Request{}

	switch algorithm {
	case "least-connection":
		queueSizesRaw, ok := props["_queueSizes"].(map[string]interface{})

		if !ok {
			return nil, true
		}

		for _, req := range queue {
			minTarget := targetIDs[0]
			minSize := int(AsFloat64(queueSizesRaw[minTarget]))
			for _, targetID := range targetIDs[1:] {
				size := int(AsFloat64(queueSizesRaw[targetID]))
				if size < minSize {
					minTarget = targetID
					minSize = size
				}
			}

			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, minTarget)
			output = append(output, &reqCopy)
		}

	default: // round-robin
		next := int(AsFloat64(props["_rrIndex"]))
		for _, req := range queue {
			target := targetIDs[next]
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, target)
			output = append(output, &reqCopy)
			next = (next + 1) % len(targetIDs)
		}
		props["_rrIndex"] = float64(next)
	}

	return output, true
}
