package simulation

type WebServerNode struct {
	ID            string
	Queue         []*Request
	CapacityRPS   int
	BaseLatencyMs int
	PenaltyPerRPS float64
	Processed     []*Request
	Alive         bool
	Targets       []string
}

func (ws *WebServerNode) GetID() string {
	return ws.ID
}

func (ws *WebServerNode) Type() string {
	return "webserver"
}

func (ws *WebServerNode) IsAlive() bool {
	return ws.Alive
}

func (ws *WebServerNode) Tick(tick int, currentTimeMs int) {
	toProcess := min(ws.CapacityRPS, len(ws.Queue))
	for i := 0; i < toProcess; i++ {
		req := ws.Queue[i]
		req.LatencyMS += ws.BaseLatencyMs
		ws.Processed = append(ws.Processed, req)
	}

	// Remove processed requests from the queue
	ws.Queue = ws.Queue[toProcess:]

	// Apply penalty for overload
	if len(ws.Queue) > 0 {
		overload := len(ws.Queue)
		for _, req := range ws.Queue {
			req.LatencyMS += int(ws.PenaltyPerRPS * float64(overload))
		}
	}
}

func (ws *WebServerNode) Receive(req *Request) {
	ws.Queue = append(ws.Queue, req)
}

func (ws *WebServerNode) Emit() []*Request {
	out := ws.Processed
	ws.Processed = nil
	return out
}

func (ws *WebServerNode) AddTarget(targetID string) {
	ws.Targets = append(ws.Targets, targetID)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (ws *WebServerNode) GetQueue() []*Request {
	return ws.Queue
}

func (ws *WebServerNode) GetTargets() []string {
	return ws.Targets
}
