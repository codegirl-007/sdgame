package simulation

type LoadBalancerNode struct {
	ID        string
	Queue     []*Request
	Targets   []string
	Counter   int
	Alive     bool
	Processed []*Request
}

func (lb *LoadBalancerNode) GetID() string {
	return lb.ID
}

func (lb *LoadBalancerNode) Type() string {
	return "loadBalancer"
}

func (lb *LoadBalancerNode) IsAlive() bool {
	return lb.Alive
}

func (lb *LoadBalancerNode) Receive(req *Request) {
	lb.Queue = append(lb.Queue)
}

func (lb *LoadBalancerNode) Tick(tick int) {
	lb.Processed = nil

	for _, req := range lb.Queue {
		if len(lb.Targets) == 0 {
			continue
		}

		target := lb.Targets[lb.Counter%len(lb.Targets)]
		lb.Counter++

		req.Path = append([]string{target}, req.Path...)

		req.LatencyMS += 10

		lb.Processed = append(lb.Processed, req)
	}

	lb.Queue = lb.Queue[:0]
}

func (lb *LoadBalancerNode) Emit() []*Request {
	out := lb.Processed
	lb.Processed = nil
	return out
}
