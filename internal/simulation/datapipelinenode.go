package simulation

import (
	"math/rand"
)

type DataPipelineNode struct {
	ID             string
	Label          string
	BatchSize      int
	Transformation string
	CurrentLoad    int
	Queue          []*Request
	Alive          bool
	Targets        []string
	output         []*Request
}

func (n *DataPipelineNode) GetID() string { return n.ID }

func (n *DataPipelineNode) Tick(tick int) {
	if len(n.Queue) == 0 {
		return
	}

	if len(n.Queue) < n.BatchSize {
		if tick%50 == 0 {
			n.processBatch(len(n.Queue))
		}
		return
	}

	batchesToProcess := len(n.Queue) / n.BatchSize
	maxBatchesPerTick := 2
	actualBatches := min(batchesToProcess, maxBatchesPerTick)

	for i := 0; i < actualBatches; i++ {
		n.processBatch(n.BatchSize)
	}
}

func (n *DataPipelineNode) processBatch(size int) {
	if size == 0 {
		return
	}

	batch := n.Queue[:size]
	n.Queue = n.Queue[size:]
	batchLatency := 100 + (size * 5)

	for _, req := range batch {
		req.LatencyMS += batchLatency
		req.Path = append(req.Path, n.ID+"(batch)")

		switch n.Transformation {
		case "aggregation":
			req.Type = "AGGREGATED"
		case "filtering":
			if rand.Float64() < 0.9 {
				n.output = append(n.output, req)
			}
			continue
		case "enrichment":
			req.Type = "ENRICHED"
		}

		n.output = append(n.output, req)
	}
}

func (n *DataPipelineNode) Receive(req *Request) {
	if req == nil {
		return
	}
	req.LatencyMS += 1
	n.Queue = append(n.Queue, req)
}

func (n *DataPipelineNode) Emit() []*Request {
	out := append([]*Request(nil), n.output...)
	n.output = n.output[:0]
	return out
}

func (n *DataPipelineNode) Type() string  { return "datapipeline" }
func (n *DataPipelineNode) IsAlive() bool { return n.Alive }
