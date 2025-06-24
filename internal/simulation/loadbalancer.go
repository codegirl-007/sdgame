package simulation

import (
	"math/rand"
)

type LoadBalancerNode struct {
	// unique identifier for the node
	ID string
	// human readable name
	Label string
	// load balancing strategy
	Algorithm string
	// list of incoming requests to be processed
	Queue []*Request
	// IDs of downstream nodes (e.g. webservers)
	Targets []string
	// use to track round-robin state (i.e. which target is next)
	Counter int
	// bool for health check
	Alive bool
	// requests that this node has handled (ready to be emitted)
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

// Acceps an incoming request by adding it to the Queue which will be processed on the next tick
func (lb *LoadBalancerNode) Receive(req *Request) {
	lb.Queue = append(lb.Queue, req)
}

func (lb *LoadBalancerNode) Tick(tick int, currentTimeMs int) {
	// clear out the process so it starts fresh
	lb.Processed = nil

	// for each pending request...
	for _, req := range lb.Queue {
		// if there are no targets to forward to, skip processing
		if len(lb.Targets) == 0 {
			continue
		}

		// placeholder for algorithm-specific logic. TODO.
		switch lb.Algorithm {
		case "random":
			fallthrough
		case "round-robin":
			fallthrough
		default:
			lb.Counter++
		}

		// Append the load balancer's ID to the request's path to record it's journey through the system
		req.Path = append(req.Path, lb.ID)

		// Simulate networking delay
		req.LatencyMS += 10

		// Mark the request as processed so it can be emitted to targets
		lb.Processed = append(lb.Processed, req)
	}

	// clear the queue after processing. Ready for next tick.
	lb.Queue = lb.Queue[:0]
}

// return the list of process requests and then clear the processed requests
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
