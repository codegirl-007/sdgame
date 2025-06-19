package simulation

type MonitoringNode struct {
	ID             string
	Label          string
	Tool           string
	AlertMetric    string
	ThresholdValue int
	ThresholdUnit  string
	Queue          []*Request
	Alive          bool
	Targets        []string
	Metrics        map[string]int
	Alerts         []*Request
}

func (n *MonitoringNode) GetID() string { return n.ID }

func (n *MonitoringNode) Type() string { return "monitoring" }

func (n *MonitoringNode) IsAlive() bool { return n.Alive }

func (n *MonitoringNode) Receive(req *Request) {
	n.Queue = append(n.Queue, req)
}

func (n *MonitoringNode) Emit() []*Request {
	out := append([]*Request(nil), n.Alerts...)
	n.Alerts = n.Alerts[:0]
	return out
}

func (n *MonitoringNode) Tick(tick int, currentTimeMs int) {
	if !n.Alive {
		return
	}

	if n.Metrics == nil {
		n.Metrics = make(map[string]int)
	}

	// Simulate processing requests as metrics
	for _, req := range n.Queue {
		// For now, pretend all requests are relevant to the AlertMetric
		n.Metrics[n.AlertMetric] += 1
		req.LatencyMS += 1
		req.Path = append(req.Path, n.ID)

		if n.Metrics[n.AlertMetric] > n.ThresholdValue {
			alert := &Request{
				ID:        "alert-" + req.ID,
				Timestamp: currentTimeMs,
				Origin:    n.ID,
				Type:      "alert",
				LatencyMS: 0,
				Path:      []string{n.ID, "alert"},
			}
			n.Alerts = append(n.Alerts, alert)

			// Reset after alert (or you could continue accumulating)
			n.Metrics[n.AlertMetric] = 0
		}
	}

	n.Queue = nil
}
