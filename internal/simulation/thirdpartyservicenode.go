package simulation

type ThirdPartyServiceNode struct {
	ID          string
	Label       string
	APIEndpoint string
	RateLimit   int    // Max requests per tick
	RetryPolicy string // "exponential", "fixed", etc.
	CurrentLoad int
	Queue       []*Request
	ErrorCount  int
	RetryCount  int
	Alive       bool
	Targets     []string
	Output      []*Request
}

// --- Interface Methods ---

func (n *ThirdPartyServiceNode) GetID() string { return n.ID }
func (n *ThirdPartyServiceNode) Type() string  { return "thirdpartyservice" }
func (n *ThirdPartyServiceNode) IsAlive() bool { return n.Alive }
func (n *ThirdPartyServiceNode) Receive(req *Request) {
	n.Queue = append(n.Queue, req)
}
func (n *ThirdPartyServiceNode) Emit() []*Request {
	out := append([]*Request(nil), n.Output...)
	n.Output = n.Output[:0]
	return out
}

// Add missing Queue method for interface compliance
func (n *ThirdPartyServiceNode) GetQueue() []*Request {
	return n.Queue
}

// --- Simulation Logic ---

func (n *ThirdPartyServiceNode) Tick(tick int, currentTimeMs int) {
	if !n.Alive {
		return
	}

	// Simulate third-party call behavior with success/failure
	maxProcess := min(n.RateLimit, len(n.Queue))
	newQueue := n.Queue[maxProcess:]
	n.Queue = nil

	for i := 0; i < maxProcess; i++ {
		req := newQueue[i]
		success := simulateThirdPartySuccess(req)

		if success {
			req.LatencyMS += 100 + randInt(0, 50) // simulate response time
			req.Path = append(req.Path, n.ID)
			n.Output = append(n.Output, req)
		} else {
			n.ErrorCount++
			n.RetryCount++
			if n.RetryPolicy == "exponential" && n.RetryCount < 3 {
				n.Queue = append(n.Queue, req) // retry again next tick
			}
		}
	}

	// Simulate degradation if too many errors
	if n.ErrorCount > 10 {
		n.Alive = false
	}
}

func (n *ThirdPartyServiceNode) GetTargets() []string {
	return n.Targets
}

// --- Helpers ---

func simulateThirdPartySuccess(req *Request) bool {
	// 90% success rate
	return randInt(0, 100) < 90
}
