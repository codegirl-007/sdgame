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
	CacheHitRate    float64             // % of requests served from edge cache
	CurrentLoad     int                 // optional: can track active queue length
	Queue           []*Request          // incoming request queue
	EdgeNodes       map[string]*CDNNode // future expansion: simulate geographic edge clusters
	Alive           bool
	Targets         []string
	output          []*Request // cache HIT responses
	missQueue       []*Request // cache MISSes to forward
}

func (n *CDNNode) GetID() string          { return n.ID }
func (n *CDNNode) Type() string           { return "cdn" }
func (n *CDNNode) IsAlive() bool          { return n.Alive }
func (n *CDNNode) QueueState() []*Request { return n.Queue }

func (n *CDNNode) Tick(tick int) {
	if len(n.Queue) == 0 {
		return
	}

	maxProcessPerTick := 10
	processCount := min(len(n.Queue), maxProcessPerTick)

	// Avoid slice leak by reusing a cleared slice
	queue := n.Queue
	n.Queue = n.Queue[:0]

	for i := 0; i < processCount; i++ {
		req := queue[i]

		// Simulate cache hit or miss
		hitRate := n.CacheHitRate
		if hitRate == 0 {
			hitRate = 0.8 // default if unset
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

	// Optionally simulate cache expiration every 100 ticks
	if tick%100 == 0 {
		// e.g., simulate reduced hit rate or TTL expiration
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

	// Clear for next tick
	n.output = n.output[:0]
	n.missQueue = n.missQueue[:0]

	return out
}
