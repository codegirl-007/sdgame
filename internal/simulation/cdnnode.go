package simulation

import (
	"math/rand"
)

type CDNNode struct {
	ID              string
	Label           string
	TTL             int
	GeoReplication  string
	CachingStrategy string
	Compression     string
	HTTP2           string
	CacheHitRate    float64
	CurrentLoad     int
	Queue           []*Request
	EdgeNodes       map[string]*CDNNode
	Alive           bool
	Targets         []string
	output          []*Request
	missQueue       []*Request
}

func (n *CDNNode) GetID() string          { return n.ID }
func (n *CDNNode) Type() string           { return "cdn" }
func (n *CDNNode) IsAlive() bool          { return n.Alive }
func (n *CDNNode) QueueState() []*Request { return n.Queue }

func (n *CDNNode) Tick(tick int, currentTimeMs int) {
	if len(n.Queue) == 0 {
		return
	}

	maxProcessPerTick := 10
	processCount := min(len(n.Queue), maxProcessPerTick)

	queue := n.Queue
	n.Queue = n.Queue[:0]

	for i := 0; i < processCount; i++ {
		req := queue[i]

		hitRate := n.CacheHitRate
		if hitRate == 0 {
			hitRate = 0.8
		}

		if rand.Float64() < hitRate {
			// Cache HIT
			req.LatencyMS += 5
			req.Path = append(req.Path, n.ID)
			n.output = append(n.output, req)
		} else {
			// Cache MISS
			req.LatencyMS += 10
			req.Path = append(req.Path, n.ID+"(miss)")
			n.missQueue = append(n.missQueue, req)
		}
	}

	if len(queue) > processCount {
		n.Queue = append(n.Queue, queue[processCount:]...)
	}
}

func (n *CDNNode) Receive(req *Request) {
	if req == nil {
		return
	}

	geoLatency := rand.Intn(20) + 5 // 5–25ms routing delay
	req.LatencyMS += geoLatency

	n.Queue = append(n.Queue, req)
}

func (n *CDNNode) Emit() []*Request {
	out := append(n.output, n.missQueue...)

	n.output = n.output[:0]
	n.missQueue = n.missQueue[:0]

	return out
}

func (n *CDNNode) GetTargets() []string {
	return n.Targets
}
