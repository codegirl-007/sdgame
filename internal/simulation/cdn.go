package simulation

import "math/rand"

type CDNLogic struct{}

func (c CDNLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	hitRate := AsFloat64("hitRate")
	var output []*Request

	for _, req := range queue {
		if rand.Float64() < hitRate {
			continue
		}

		reqCopy := *req
		reqCopy.Path = append(reqCopy.Path, "target-0")
		output = append(output, &reqCopy)
	}

	return output, true
}
