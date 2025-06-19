package simulation

import (
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
}

type Engine struct {
	Nodes    map[string]SimulationNode
	Timeline []TickSnapshot
	Duration int
	TickMs   int
}

type TickSnapshot struct {
	TickMs     int
	QueueSizes map[string]int
	NodeHealth map[string]string
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
		case "cache":
			simNode = &CacheNode{
				ID:             n.ID,
				Label:          n.Props["label"].(string),
				CacheTTL:       int(n.Props["cacheTTL"].(float64)),
				MaxEntries:     int(n.Props["maxEntries"].(float64)),
				EvictionPolicy: n.Props["evictionPolicy"].(string),
				CurrentLoad:    0,
				Queue:          []*Request{},
				Cache:          make(map[string]CacheEntry),
				Alive:          true,
			}
		case "database":
			simNode = &DatabaseNode{
				ID:          n.ID,
				Label:       n.Props["label"].(string),
				Replication: int(n.Props["replication"].(float64)),
				Queue:       []*Request{},
				Alive:       true,
			}
		case "cdn":
			simNode = &CDNNode{
				ID:              n.ID,
				Label:           n.Props["label"].(string),
				TTL:             int(n.Props["ttl"].(float64)),
				GeoReplication:  n.Props["geoReplication"].(string),
				CachingStrategy: n.Props["cachingStrategy"].(string),
				Compression:     n.Props["compression"].(string),
				HTTP2:           n.Props["http2"].(string),
				Queue:           []*Request{},
				Alive:           true,
				output:          []*Request{},
				missQueue:       []*Request{},
			}
		case "messageQueue":
			simNode = &MessageQueueNode{
				ID:         n.ID,
				Label:      n.Props["label"].(string),
				QueueSize:  int(n.Props["maxSize"].(float64)),
				MessageTTL: int(n.Props["retentionSeconds"].(float64)),
				DeadLetter: false,
				Queue:      []*Request{},
				Alive:      true,
			}
		case "microservice":
			simNode = &MicroserviceNode{
				ID:             n.ID,
				Label:          n.Props["label"].(string),
				APIEndpoint:    n.Props["apiVersion"].(string),
				RateLimit:      int(n.Props["rpsCapacity"].(float64)),
				CircuitBreaker: true,
				Queue:          []*Request{},
				CircuitState:   "closed",
				Alive:          true,
			}
		case "third party service":
			simNode = &ThirdPartyServiceNode{
				ID:          n.ID,
				Label:       n.Props["label"].(string),
				APIEndpoint: n.Props["provider"].(string),
				RateLimit:   int(n.Props["latency"].(float64)),
				RetryPolicy: "exponential",
				Queue:       []*Request{},
				Alive:       true,
			}
		case "data pipeline":
			simNode = &DataPipelineNode{
				ID:             n.ID,
				Label:          n.Props["label"].(string),
				BatchSize:      int(n.Props["batchSize"].(float64)),
				Transformation: n.Props["transformation"].(string),
				Queue:          []*Request{},
				Alive:          true,
			}
		case "monitoring/alerting":
			// Simulation not implemented yet; optionally skip
			continue
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
