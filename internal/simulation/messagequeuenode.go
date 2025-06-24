package simulation

type MessageQueueNode struct {
	ID               string
	Label            string
	QueueSize        int
	MessageTTL       int // TTL in milliseconds
	DeadLetter       bool
	CurrentLoad      int
	Queue            []*Request
	Alive            bool
	Targets          []string
	output           []*Request
	deadLetterOutput []*Request
}

func (n *MessageQueueNode) GetID() string { return n.ID }
func (n *MessageQueueNode) Type() string  { return "messagequeue" }
func (n *MessageQueueNode) IsAlive() bool { return n.Alive }

func (n *MessageQueueNode) Tick(tick int, currentTimeMs int) {
	if len(n.Queue) == 0 {
		return
	}

	// Message queues have very high throughput
	maxProcessPerTick := 20 // Higher than database (3) or CDN (10)
	processCount := min(len(n.Queue), maxProcessPerTick)

	// Check for queue overflow (simulate back pressure)
	if len(n.Queue) > n.QueueSize {
		// Move oldest messages to dead letter queue if enabled
		if n.DeadLetter {
			overflow := len(n.Queue) - n.QueueSize
			for i := 0; i < overflow; i++ {
				deadReq := n.Queue[i]
				deadReq.Type = "DEAD_LETTER"
				deadReq.Path = append(deadReq.Path, n.ID+"(dead)")
				n.deadLetterOutput = append(n.deadLetterOutput, deadReq)
			}
			n.Queue = n.Queue[overflow:]
		} else {
			// Drop messages if no dead letter queue
			n.Queue = n.Queue[:n.QueueSize]
		}
	}

	// Process messages with TTL check
	for i := 0; i < processCount; i++ {
		req := n.Queue[0]
		n.Queue = n.Queue[1:]

		// Check TTL (time to live) - use current time in milliseconds
		messageAgeMs := currentTimeMs - req.Timestamp
		if messageAgeMs > n.MessageTTL {
			// Message expired
			if n.DeadLetter {
				req.Type = "EXPIRED"
				req.Path = append(req.Path, n.ID+"(expired)")
				n.deadLetterOutput = append(n.deadLetterOutput, req)
			}
			// Otherwise drop expired message
			continue
		}

		// Message queue adds minimal latency (very fast)
		req.LatencyMS += 2
		req.Path = append(req.Path, n.ID)
		n.output = append(n.output, req)
	}
}

func (n *MessageQueueNode) Receive(req *Request) {
	if req == nil {
		return
	}

	// Message queues have very low receive overhead
	req.LatencyMS += 1

	n.Queue = append(n.Queue, req)
}

func (n *MessageQueueNode) Emit() []*Request {
	// Return both normal messages and dead letter messages
	allRequests := append(n.output, n.deadLetterOutput...)

	// Clear queues
	n.output = n.output[:0]
	n.deadLetterOutput = n.deadLetterOutput[:0]

	return allRequests
}

func (n *MessageQueueNode) GetTargets() []string {
	return n.Targets
}

func (n *MessageQueueNode) GetQueue() []*Request {
	return n.Queue
}
