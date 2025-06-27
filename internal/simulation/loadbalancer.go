package simulation

import (
	"fmt"
)

type LoadBalancerLogic struct{}

func (l LoadBalancerLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract the load balancing algorithm from the props.
	algorithm := AsString(props["algorithm"])
	// Number of downstream targets
	targets := int(AsFloat64(props["_numTargets"]))

	if len(queue) == 0 {
		return nil, true
	}

	// Hold the processed requests to be emitted
	output := []*Request{}

	switch algorithm {
	case "least-connection":
		// extrat current queue sizes from downstream targets
		queueSizesRaw, ok := props["_queueSizes"].(map[string]interface{})
		if !ok {
			return nil, true
		}

		// find target with smallest queue
		for _, req := range queue {
			minTarget := "target-0"
			minSize := int(AsFloat64(queueSizesRaw[minTarget]))
			for i := 1; i < targets; i++ {
				targetKey := fmt.Sprintf("target-%d", i)
				size := int(AsFloat64(queueSizesRaw[targetKey]))
				if size < minSize {
					minTarget = targetKey
					minSize = size
				}
			}

			// Clone the request and append the selected target to its path
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, minTarget)
			output = append(output, &reqCopy)
		}
	default:
		// Retrieve the last used index
		next := int(AsFloat64(props["_rrIndex"]))
		for _, req := range queue {
			// Clone ther equest and append the selected target to its path
			reqCopy := *req
			reqCopy.Path = append(reqCopy.Path, fmt.Sprintf("target-%d", next))
			output = append(output, &reqCopy)
			// Advance to next target
			next = (next + 1) % targets
		}

		props["_rrIndex"] = float64(next)
	}

	return output, true
}
