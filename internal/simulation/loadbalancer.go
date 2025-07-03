package simulation

import (
	"hash/fnv"
	"math/rand"
)

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

	case "random":
		for _, req := range queue {
			randomIndex := rand.Intn(len(targetIDs))
			target := targetIDs[randomIndex]

			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, target)
			output = append(output, &reqCopy)
		}

	case "ip-hash":
		for _, req := range queue {
			h := fnv.New32a()
			h.Write([]byte(req.ID))
			index := int(h.Sum32()) % len(targetIDs)
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, targetIDs[index])
			output = append(output, &reqCopy)
		}

	case "first-available":
		nodeHealth, ok := props["_nodeHealth"].(map[string]bool)
		if !ok {
			return nil, true
		}
		firstHealthy := ""
		for _, t := range targetIDs {
			if nodeHealth[t] {
				firstHealthy = t
				break
			}
		}
		if firstHealthy == "" {
			return nil, true
		}
		for _, req := range queue {
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, firstHealthy)
			output = append(output, &reqCopy)
		}

	case "weighted-round-robin":
		// Create or reuse the weighted list of targets
		weighted, ok := props["_weightedTargets"].([]string)
		if !ok {
			weights, ok := props["_weights"].(map[string]float64)
			if !ok {
				return nil, true
			}
			flattened := []string{}
			for _, id := range targetIDs {
				w := int(weights[id])
				for i := 0; i < w; i++ {
					flattened = append(flattened, id)
				}
			}
			weighted = flattened
			props["_weightedTargets"] = weighted
		}

		next := int(AsFloat64(props["_wrIndex"]))
		for _, req := range queue {
			target := weighted[next]
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, target)
			output = append(output, &reqCopy)
			next = (next + 1) % len(weighted)
		}
		props["_wrIndex"] = float64(next)

	case "last":
		last := targetIDs[len(targetIDs)-1]
		for _, req := range queue {
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, last)
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
