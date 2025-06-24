package simulation

import (
	"fmt"
	"math/rand"
	"systemdesigngame/internal/design"
)

type Request struct {
	ID        string
	Timestamp int
	LatencyMS int
	Origin    string
	Type      string
	Path      []string
}

type SimulationNode interface {
	GetID() string
	Type() string
	Tick(tick int, currentTimeMs int)
	Receive(req *Request)
	Emit() []*Request
	IsAlive() bool
	GetTargets() []string
	GetQueue() []*Request
}

type Engine struct {
	Nodes    map[string]SimulationNode
	Timeline []*TickSnapshot
	Duration int
	TickMs   int
}

type TickSnapshot struct {
	TickMs     int
	QueueSizes map[string]int
	NodeHealth map[string]NodeState
	Emitted    map[string][]*Request
}

type NodeState struct {
	QueueSize int
	Alive     bool
}

func NewEngineFromDesign(design design.Design, duration int, tickMs int) *Engine {
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
	const tickMS = 100
	currentTimeMs := 0
	e.Timeline = e.Timeline[:0]

	for tick := 0; tick < e.Duration; tick++ {
		// inject new requests
		for _, node := range e.findEntryPoints() {
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

		// snapshot for this tick
		snapshot := &TickSnapshot{
			TickMs:     tick,
			NodeHealth: make(map[string]NodeState),
		}

		for id, node := range e.Nodes {
			// tick all nodes
			node.Tick(tick, currentTimeMs)
			emitted := node.Emit()

			// emit and forward requests to connected nodes
			for _, req := range emitted {
				for _, targetID := range node.GetTargets() {
					if target, ok := e.Nodes[targetID]; ok && target.IsAlive() {
						target.Receive(req)
					}
				}
			}

			snapshot.NodeHealth[id] = NodeState{
				QueueSize: len(emitted),
				Alive:     node.IsAlive(),
			}
		}

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
	if v == nil {
		return 0
	}

	return v.(float64)
}

func asString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	return s
}
