package simulation

import (
	"fmt"
	"math/rand"
)

type LoadBalancerNode struct {
	ID        string
	Label     string
	Algorithm string
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
	lb.Queue = append(lb.Queue, req)
}

func (lb *LoadBalancerNode) Tick(tick int, currentTimeMs int) {
	lb.Processed = nil

	for _, req := range lb.Queue {
		if len(lb.Targets) == 0 {
			continue
		}

		var target string
		switch lb.Algorithm {
		case "random":
			target = lb.Targets[rand.Intn(len(lb.Targets))]
		case "round-robin":
			fallthrough
		default:
			target = lb.Targets[lb.Counter%len(lb.Targets)]
			lb.Counter++
		}

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

func (lb *LoadBalancerNode) GetTargets() []string {
	return lb.Targets
}

func (lb *LoadBalancerNode) GetQueue() []*Request {
	return lb.Queue
}
