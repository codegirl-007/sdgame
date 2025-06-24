package simulation

import "math/rand"

type MicroserviceNode struct {
	ID             string
	Label          string
	APIEndpoint    string
	RateLimit      int // max requests per tick
	CircuitBreaker bool
	CircuitState   string // "closed", "open", "half-open"
	ErrorCount     int
	CurrentLoad    int
	Queue          []*Request
	Output         []*Request
	Alive          bool
	Targets        []string
}

func (n *MicroserviceNode) GetID() string { return n.ID }

func (n *MicroserviceNode) Type() string { return "microservice" }

func (n *MicroserviceNode) IsAlive() bool { return n.Alive }

func (n *MicroserviceNode) Receive(req *Request) {
	n.Queue = append(n.Queue, req)
}

func (n *MicroserviceNode) Emit() []*Request {
	out := append([]*Request(nil), n.Output...)
	n.Output = n.Output[:0]
	return out
}

func (n *MicroserviceNode) GetTargets() []string {
	return n.Targets
}

func (n *MicroserviceNode) Tick(tick int, currentTimeMs int) {
	n.CurrentLoad = 0
	n.ErrorCount = 0
	n.Output = nil

	toProcess := min(len(n.Queue), n.RateLimit)
	for i := 0; i < toProcess; i++ {
		req := n.Queue[i]

		if rand.Float64() < 0.02 {
			n.ErrorCount++
			if n.CircuitBreaker {
				n.CircuitState = "open"
				n.Alive = false
			}

			continue
		}

		req.LatencyMS += 10
		req.Path = append(req.Path, n.ID)
		n.Output = append(n.Output, req)
		n.CurrentLoad++
	}

	if n.CircuitState == "open" && tick%10 == 0 {
		n.CircuitState = "closed"
		n.Alive = true
	}

	n.Queue = n.Queue[toProcess:]
}

func (n *MicroserviceNode) GetQueue() []*Request {
	return n.Queue
}
