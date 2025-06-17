package simulation

type WebServerNode struct {
	ID            string
	Queue         []*Request
	CapacityRPS   int
	BaseLatencyMs int
	PenaltyPerRPS float64
	Processed     []*Request
	Alive         bool
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

func (ws *WebServerNode) Tick(tick int) {
	toProcess := min(ws.CapacityRPS, len(ws.Queue))
	for i := 0; i < toProcess; i++ {
		req := ws.Queue[0]
		req.LatencyMS += ws.BaseLatencyMs
		ws.Processed = append(ws.Processed, req)

		ws.Queue[i] = nil
	}

	ws.Queue = ws.Queue[toProcess:]

	if len(ws.Queue) > ws.CapacityRPS {
		overload := len(ws.Queue) - ws.CapacityRPS
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
