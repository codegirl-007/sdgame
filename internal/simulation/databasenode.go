package simulation

type DatabaseNode struct {
	ID               string
	Label            string
	Replication      int
	CurrentLoad      int
	Queue            []*Request
	Replicas         []*DatabaseNode
	Alive            bool
	Targets          []string
	Output           []*Request
	ReplicationQueue []*Request
}

func (n *DatabaseNode) GetID() string        { return n.ID }
func (n *DatabaseNode) Type() string         { return "database" }
func (n *DatabaseNode) IsAlive() bool        { return n.Alive }
func (n *DatabaseNode) GetQueue() []*Request { return n.Queue }

func (n *DatabaseNode) Tick(tick int, currentTimeMs int) {
	if len(n.Queue) == 0 {
		return
	}

	maxProcessPerTick := 3
	processCount := min(len(n.Queue), maxProcessPerTick)

	queue := n.Queue
	n.Queue = n.Queue[:0]

	for i := 0; i < processCount; i++ {
		req := queue[i]

		if req.Type == "READ" {
			req.LatencyMS += 20
			req.Path = append(req.Path, n.ID)
			n.Output = append(n.Output, req)
		} else {
			req.LatencyMS += 50
			req.Path = append(req.Path, n.ID)

			for _, replica := range n.Replicas {
				replicationReq := &Request{
					ID:        req.ID + "-repl",
					Timestamp: req.Timestamp,
					LatencyMS: req.LatencyMS + 5,
					Origin:    req.Origin,
					Type:      "REPLICATION",
					Path:      append(append([]string{}, req.Path...), "->"+replica.ID),
				}
				n.ReplicationQueue = append(n.ReplicationQueue, replicationReq)
			}

			n.Output = append(n.Output, req)
		}
	}

	if len(n.Queue) > 10 {
		for _, req := range n.Queue {
			req.LatencyMS += 10
		}
	}
}

func (n *DatabaseNode) Receive(req *Request) {
	if req == nil {
		return
	}
	req.LatencyMS += 2 // DB connection overhead
	n.Queue = append(n.Queue, req)
}

func (n *DatabaseNode) Emit() []*Request {
	out := append(n.Output, n.ReplicationQueue...)
	n.Output = n.Output[:0]
	n.ReplicationQueue = n.ReplicationQueue[:0]
	return out
}
