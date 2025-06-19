package simulation

import (
	"math/rand"
)

type DataPipelineNode struct {
	ID              string
	Label           string
	BatchSize       int
	Transformation  string
	CurrentLoad     int
	Queue           []*Request
	Alive           bool
	Targets         []string
	Output          []*Request
	LastFlushTimeMS int
}

func (n *DataPipelineNode) GetID() string { return n.ID }

func (n *DataPipelineNode) Tick(tick int, currentTimeMs int) {
	if len(n.Queue) == 0 {
		return
	}

	if len(n.Queue) < n.BatchSize {
		if n.LastFlushTimeMS == 0 {
			n.LastFlushTimeMS = currentTimeMs
		}
		if currentTimeMs-n.LastFlushTimeMS >= 5000 {
			n.processBatch(len(n.Queue), currentTimeMs)
			n.LastFlushTimeMS = currentTimeMs
		}
		return
	}

	batchesToProcess := len(n.Queue) / n.BatchSize
	maxBatchesPerTick := 2
	actualBatches := min(batchesToProcess, maxBatchesPerTick)

	for i := 0; i < actualBatches; i++ {
		n.processBatch(n.BatchSize, currentTimeMs)
		n.LastFlushTimeMS = currentTimeMs
	}
}

func (n *DataPipelineNode) processBatch(size int, currentTimeMs int) {
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
		case "aggregate":
			req.Type = "AGGREGATED"
			n.Output = append(n.Output, req)
		case "filter":
			if rand.Float64() < 0.9 {
				n.Output = append(n.Output, req)
			}
			continue
		case "enrich":
			req.Type = "ENRICHED"
			n.Output = append(n.Output, req)
		case "normalize":
			req.Type = "NORMALIZED"
			n.Output = append(n.Output, req)
		case "dedupe":
			if rand.Float64() < 0.95 {
				req.Type = "DEDUPED"
				n.Output = append(n.Output, req)
			}
			continue
		default:
			n.Output = append(n.Output, req)
		}
	}
}

func (n *DataPipelineNode) Receive(req *Request) {
	if req == nil {
		return
	}
	req.LatencyMS += 1
	n.Queue = append(n.Queue, req)

	if n.LastFlushTimeMS == 0 {
		n.LastFlushTimeMS = req.Timestamp
	}
}

func (n *DataPipelineNode) Emit() []*Request {
	out := append([]*Request(nil), n.Output...)
	n.Output = n.Output[:0]
	return out
}

func (n *DataPipelineNode) Type() string  { return "datapipeline" }
func (n *DataPipelineNode) IsAlive() bool { return n.Alive }
func (n *DataPipelineNode) GetTargets() []string {
	return n.Targets
}
