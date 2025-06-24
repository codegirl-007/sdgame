package simulation

import (
	"fmt"
	"math/rand"
	"systemdesigngame/internal/design"
)

// a unit that flows through the system
type Request struct {
	ID string
	// when a request was created
	Timestamp int
	// total time spent on system
	LatencyMS int
	// where the request originated from (node ID)
	Origin string
	// could be GET or POST
	Type string
	// records where it's been (used to prevent loops)
	Path []string
}

// Every node implements this interface and is used by the engine to operate all nodes in a uniform way.
type SimulationNode interface {
	GetID() string
	Type() string
	Tick(tick int, currentTimeMs int) // Advance the node's state
	Receive(req *Request)             // Accept new requests
	Emit() []*Request
	IsAlive() bool
	GetTargets() []string // Connection to other nodes
	GetQueue() []*Request // Requests currently pending
}

type Engine struct {
	// Map of Node ID -> actual node, Represents all nodes in the graph
	Nodes map[string]SimulationNode
	// all tick snapshots
	Timeline []*TickSnapshot
	// how many ticks to run
	Duration int
	// no used here but we could use it if we want it configurable
	TickMs int
}

// what hte system looks like given a tick
type TickSnapshot struct {
	TickMs int
	// Queue size at each node
	QueueSizes map[string]int
	NodeHealth map[string]NodeState
	// what each node output that tick before routing
	Emitted map[string][]*Request
}

// used for tracking health/debugging each node at tick
type NodeState struct {
	QueueSize int
	Alive     bool
}

// Takes a level design and produces a runnable engine from it.
func NewEngineFromDesign(design design.Design, duration int, tickMs int) *Engine {

	// Iterate over each nodes and then construct the simulation nodes
	// Each constructed simulation node is then stored in the nodeMap
	nodeMap := make(map[string]SimulationNode)

	for _, n := range design.Nodes {
		var simNode SimulationNode

		switch n.Type {
		case "webserver":
			simNode = &WebServerNode{
				ID:    n.ID,
				Alive: true,
				Queue: []*Request{},
			}
		case "loadBalancer":
			props := n.Props
			simNode = &LoadBalancerNode{
				ID:        n.ID,
				Label:     asString(props["label"]),
				Algorithm: asString(props["algorithm"]),
				Queue:     []*Request{},
				Alive:     true,
				Targets:   []string{},
			}
		case "cache":
			simNode = &CacheNode{
				ID:             n.ID,
				Label:          asString(n.Props["label"]),
				CacheTTL:       int(asFloat64(n.Props["cacheTTL"])),
				MaxEntries:     int(asFloat64(n.Props["maxEntries"])),
				EvictionPolicy: asString(n.Props["evictionPolicy"]),
				CurrentLoad:    0,
				Queue:          []*Request{},
				Cache:          make(map[string]CacheEntry),
				Alive:          true,
			}
		case "database":
			simNode = &DatabaseNode{
				ID:          n.ID,
				Label:       asString(n.Props["label"]),
				Replication: int(asFloat64(n.Props["replication"])),
				Queue:       []*Request{},
				Alive:       true,
			}
		case "cdn":
			simNode = &CDNNode{
				ID:              n.ID,
				Label:           asString(n.Props["label"]),
				TTL:             int(asFloat64(n.Props["ttl"])),
				GeoReplication:  asString(n.Props["geoReplication"]),
				CachingStrategy: asString(n.Props["cachingStrategy"]),
				Compression:     asString(n.Props["compression"]),
				HTTP2:           asString(n.Props["http2"]),
				Queue:           []*Request{},
				Alive:           true,
				output:          []*Request{},
				missQueue:       []*Request{},
			}
		case "messageQueue":
			simNode = &MessageQueueNode{
				ID:         n.ID,
				Label:      asString(n.Props["label"]),
				QueueSize:  int(asFloat64(n.Props["maxSize"])),
				MessageTTL: int(asFloat64(n.Props["retentionSeconds"])),
				DeadLetter: false,
				Queue:      []*Request{},
				Alive:      true,
			}
		case "microservice":
			simNode = &MicroserviceNode{
				ID:             n.ID,
				Label:          asString(n.Props["label"]),
				APIEndpoint:    asString(n.Props["apiVersion"]),
				RateLimit:      int(asFloat64(n.Props["rpsCapacity"])),
				CircuitBreaker: true,
				Queue:          []*Request{},
				CircuitState:   "closed",
				Alive:          true,
			}
		case "third party service":
			simNode = &ThirdPartyServiceNode{
				ID:          n.ID,
				Label:       asString(n.Props["label"]),
				APIEndpoint: asString(n.Props["provider"]),
				RateLimit:   int(asFloat64(n.Props["latency"])),
				RetryPolicy: "exponential",
				Queue:       []*Request{},
				Alive:       true,
			}
		case "data pipeline":
			simNode = &DataPipelineNode{
				ID:             n.ID,
				Label:          asString(n.Props["label"]),
				BatchSize:      int(asFloat64(n.Props["batchSize"])),
				Transformation: asString(n.Props["transformation"]),
				Queue:          []*Request{},
				Alive:          true,
			}
		case "monitoring/alerting":
			simNode = &MonitoringNode{
				ID:             n.ID,
				Label:          asString(n.Props["label"]),
				Tool:           asString(n.Props["tool"]),
				AlertMetric:    asString(n.Props["metric"]),
				ThresholdValue: int(asFloat64(n.Props["threshold"])),
				ThresholdUnit:  asString(n.Props["unit"]),
				Queue:          []*Request{},
				Alive:          true,
			}
		default:
			continue
		}

		if simNode != nil {
			nodeMap[simNode.GetID()] = simNode
		}
	}

	// Wire up connections
	for _, conn := range design.Connections {
		if sourceNode, ok := nodeMap[conn.Source]; ok {
			if targetSetter, ok := sourceNode.(interface{ AddTarget(string) }); ok {
				targetSetter.AddTarget(conn.Target)
			}
		}
	}

	return &Engine{
		Nodes:    nodeMap,
		Duration: duration,
		TickMs:   tickMs,
	}
}

func (e *Engine) Run() {
	// clear and set defaults
	const tickMS = 100
	currentTimeMs := 0
	e.Timeline = e.Timeline[:0]

	// start ticking. This is really where the simulation begins
	for tick := 0; tick < e.Duration; tick++ {

		// find the entry points (where traffic enters) or else print a warning
		entries := e.findEntryPoints()
		if len(entries) == 0 {
			fmt.Println("[ERROR] No entry points found! Simulation will not inject requests.")
		}

		// inject new requests of each entry node every tick
		for _, node := range entries {
			if shouldInject(tick) {
				req := &Request{
					ID:        generateRequestID(tick),
					Timestamp: currentTimeMs,
					LatencyMS: 0,
					Origin:    node.GetID(),
					Type:      "GET",
					Path:      []string{node.GetID()},
				}
				node.Receive(req)
			}
		}

		// snapshot to record what happened this tick
		snapshot := &TickSnapshot{
			TickMs:     tick,
			NodeHealth: make(map[string]NodeState),
		}

		for id, node := range e.Nodes {
			// capture health data before processing
			snapshot.NodeHealth[id] = NodeState{
				QueueSize: len(node.GetQueue()),
				Alive:     node.IsAlive(),
			}

			// tick all nodes
			node.Tick(tick, currentTimeMs)

			// get all processed requets and fan it out to all connected targets
			for _, req := range node.Emit() {
				snapshot.Emitted[node.GetID()] = append(snapshot.Emitted[node.GetID()], req)

				for _, targetID := range node.GetTargets() {
					if target, ok := e.Nodes[targetID]; ok && target.IsAlive() && !hasVisited(req, targetID) {
						// Deep copy request and update path
						newReq := *req
						newReq.Path = append([]string{}, req.Path...)
						newReq.Path = append(newReq.Path, targetID)
						target.Receive(&newReq)
					}
				}
			}

		}

		// store the snapshot and advance time
		e.Timeline = append(e.Timeline, snapshot)
		currentTimeMs += tickMS
	}
}

func (e *Engine) findEntryPoints() []SimulationNode {
	var entries []SimulationNode
	for _, node := range e.Nodes {
		if node.Type() == "loadBalancer" {
			entries = append(entries, node)
		}
	}
	return entries
}

func (e *Engine) injectRequests(entries []SimulationNode, requests []*Request) {
	for i, req := range requests {
		node := entries[i%len(entries)]
		node.Receive(req)
	}
}

func shouldInject(tick int) bool {
	return tick%100 == 0
}

func generateRequestID(tick int) string {
	return fmt.Sprintf("req-%d-%d", tick, rand.Intn(1000))
}

func asFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func asString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}

func hasVisited(req *Request, nodeID string) bool {
	for _, id := range req.Path {
		if id == nodeID {
			return true
		}
	}
	return false
}
