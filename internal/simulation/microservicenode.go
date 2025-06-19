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

func (n *MicroserviceNode) Tick(tick int, currentTimeMs int) {
	if !n.Alive {
		return
	}

	// Simulate circuit breaker state transitions
	switch n.CircuitState {
	case "open":
		// Skip processing
		return
	case "half-open":
		// Allow limited testing traffic
		if len(n.Queue) == 0 {
			return
		}
		req := n.Queue[0]
		n.Queue = n.Queue[1:]
		req.LatencyMS += 15
		req.Path = append(req.Path, n.ID+"(half-open)")
		success := simulateSuccess()
		if success {
			n.CircuitState = "closed"
			n.ErrorCount = 0
			n.Output = append(n.Output, req)
		} else {
			n.ErrorCount++
			n.CircuitState = "open"
		}
		return
	}

	// "closed" circuit - normal processing
	toProcess := min(len(n.Queue), n.RateLimit)
	for i := 0; i < toProcess; i++ {
		req := n.Queue[i]
		req.LatencyMS += 10
		req.Path = append(req.Path, n.ID)

		if simulateFailure() {
			n.ErrorCount++
			if n.CircuitBreaker && n.ErrorCount > 5 {
				n.CircuitState = "open"
				break
			}
			continue
		}

		n.Output = append(n.Output, req)
	}
	n.Queue = n.Queue[toProcess:]

	// Health check: simple example
	n.Alive = n.CircuitState != "open"
}

func simulateFailure() bool {
	// Simulate 10% failure rate
	return randInt(1, 100) <= 10
}

func simulateSuccess() bool {
	// Simulate 90% success rate
	return randInt(1, 100) <= 90
}

func randInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}
