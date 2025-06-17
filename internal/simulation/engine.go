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
	ID() string
	Type() string
	Tick(tick int)
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

func NewEngineFromDesign(design simulation.Design, duration int, tickMs int) *Engine {
	nodes := design.ToSimulationNode()
	nodeMap := make(make[string]SimulatSimulationNode)
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	return &Engine{
		Nodes: nodeMap,
		Duration: duration,
		TickMs: tickMs,
	}
}


