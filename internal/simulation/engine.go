package simulation

import (
	"fmt"
	"systemdesigngame/internal/design"
)

// TODO list
type TODO interface{}
type NodeLogic interface {
	Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool)
}

type NodeInstance struct {
	ID    string
	Type  string
	Props map[string]any
	Queue []*Request
	Alive bool
	Logic NodeLogic
}

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

// what hte system looks like given a tick
type TickSnapshot struct {
	TickMs int
	// Queue size at each node
	QueueSizes map[string]int
	NodeHealth map[string]bool
	// what each node output that tick before routing
	Emitted map[string][]*Request
}

type SimulationEngine struct {
	Nodes     map[string]*NodeInstance
	Edges     map[string][]string
	TickMS    int
	EntryNode string // this should be the node ID where traffic should enter
	RPS       int    // how many requests per second should be injected while running
}

func NewEngineFromDesign(d design.Design, tickMS int) *SimulationEngine {
	nodes := make(map[string]*NodeInstance)
	edges := make(map[string][]string)

	for _, n := range d.Nodes {
		logic := GetLogicForType(n.Type)

		if logic == nil {
			continue
		}

		// create a NodeInstance using data from the json
		nodes[n.ID] = &NodeInstance{
			ID:    n.ID,
			Type:  n.Type,
			Props: n.Props,
			Queue: []*Request{},
			Alive: true,
			Logic: logic,
		}
	}

	// build a map of the connections (edges)
	for _, c := range d.Connections {
		edges[c.Source] = append(edges[c.Source], c.Target)
	}

	return &SimulationEngine{
		Nodes:     nodes,
		Edges:     edges,
		TickMS:    tickMS,
		RPS:       0,  // ideally, this will come from the design (serialized json)
		EntryNode: "", // default to empty, we check this later in the run method
	}
}

func (e *SimulationEngine) Run(duration int, tickMs int) []*TickSnapshot {
	snapshots := []*TickSnapshot{}
	currentTime := 0

	for tick := 0; tick < duration; tick++ {
		if e.RPS > 0 && e.EntryNode != "" {
			count := int(float64(e.RPS) * float64(e.TickMS) / 1000.0)

			reqs := make([]*Request, count)

			for i := 0; i < count; i++ {
				reqs[i] = &Request{
					ID:        fmt.Sprintf("req-%d-%d", tick, i),
					Origin:    e.EntryNode,
					Type:      "GET",
					Timestamp: tick * e.TickMS,
					Path:      []string{e.EntryNode},
				}
			}

			node := e.Nodes[e.EntryNode]
			node.Queue = append(node.Queue, reqs...)
		}

		emitted := map[string][]*Request{}
		snapshot := &TickSnapshot{
			TickMs:     tick,
			QueueSizes: map[string]int{},
			NodeHealth: map[string]bool{},
			Emitted:    map[string][]*Request{},
		}

		for id, node := range e.Nodes {
			// if the node is not alive, don't even bother.
			if !node.Alive {
				continue
			}

			// this will preopulate some props so that we can use different load balancing algorithms
			if node.Type == "loadbalancer" {
				targets := e.Edges[id]
				node.Props["_numTargets"] = float64(len(targets))
				node.Props["_targetIDs"] = targets

				algo, ok := node.Props["algorithm"].(string)
				if ok && algo == "least-connection" {
					queueSizes := make(map[string]interface{})
					for _, targetID := range e.Edges[id] {
						queueSizes[targetID] = len(e.Nodes[targetID].Queue)
					}
					node.Props["_queueSizes"] = queueSizes
				}
			}

			// simulate the node. outputs is the emitted requests (request post-processing) and alive tells you if the node is healthy
			outputs, alive := node.Logic.Tick(node.Props, node.Queue, tick)

			// at this point, all nodes have ticked. record the emitted requests
			emitted[id] = outputs

			// clear the queue after processing. Queues should only contain requests for the next tick that need processing.
			node.Queue = nil

			// update if the node is still alive
			node.Alive = alive

			// populate snapshot
			snapshot.QueueSizes[id] = len(node.Queue)
			snapshot.NodeHealth[id] = node.Alive
			snapshot.Emitted[id] = outputs
		}

		// iterate over each emitted request
		for from, reqs := range emitted {
			// figure out where those requests should go to
			for _, to := range e.Edges[from] {
				// add those requests to the target node's input (queue)
				e.Nodes[to].Queue = append(e.Nodes[to].Queue, reqs...)
			}
		}

		snapshots = append(snapshots, snapshot)
		currentTime += e.TickMS
	}

	return snapshots
}

func GetLogicForType(t string) NodeLogic {
	switch t {
	case "webserver":
		return WebServerLogic{}
	case "loadbalancer":
		return LoadBalancerLogic{}
	case "cdn":
		return CDNLogic{}
	default:
		return nil
	}
}
