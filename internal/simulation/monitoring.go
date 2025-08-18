package simulation

type MonitoringLogic struct{}

type MetricData struct {
	Timestamp    int
	LatencySum   int
	RequestCount int
	ErrorCount   int
	QueueSize    int
}

type AlertEvent struct {
	Timestamp  int
	MetricType string
	Value      float64
	Threshold  float64
	Unit       string
	Severity   string
}

func (m MonitoringLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract monitoring properties
	tool := AsString(props["tool"])
	if tool == "" {
		tool = "Prometheus" // default monitoring tool
	}

	alertMetric := AsString(props["alertMetric"])
	if alertMetric == "" {
		alertMetric = "latency" // default to latency monitoring
	}

	thresholdValue := int(AsFloat64(props["thresholdValue"]))
	if thresholdValue == 0 {
		thresholdValue = 100 // default threshold
	}

	thresholdUnit := AsString(props["thresholdUnit"])
	if thresholdUnit == "" {
		thresholdUnit = "ms" // default unit
	}

	// Get historical metrics from props
	metrics, ok := props["_metrics"].([]MetricData)
	if !ok {
		metrics = []MetricData{}
	}

	// Get alert history
	alerts, ok := props["_alerts"].([]AlertEvent)
	if !ok {
		alerts = []AlertEvent{}
	}

	currentTime := tick * 100 // Convert tick to milliseconds

	// Process all incoming requests (monitoring is pass-through)
	output := []*Request{}
	totalLatency := 0
	errorCount := 0

	for _, req := range queue {
		// Create a copy of the request to forward
		reqCopy := *req

		// Add minimal monitoring overhead (1-2ms for metric collection)
		monitoringOverhead := 1
		if tool == "Datadog" || tool == "New Relic" {
			monitoringOverhead = 2 // More feature-rich tools have slightly higher overhead
		}

		reqCopy.LatencyMS += monitoringOverhead
		reqCopy.Path = append(reqCopy.Path, "monitored")

		// Collect metrics from the request
		totalLatency += req.LatencyMS

		// Simple heuristic: requests with high latency are considered errors
		if req.LatencyMS > 1000 { // 1 second threshold for errors
			errorCount++
		}

		output = append(output, &reqCopy)
	}

	// Calculate current metrics
	avgLatency := 0.0
	if len(queue) > 0 {
		avgLatency = float64(totalLatency) / float64(len(queue))
	}

	// Store current metrics
	currentMetric := MetricData{
		Timestamp:    currentTime,
		LatencySum:   totalLatency,
		RequestCount: len(queue),
		ErrorCount:   errorCount,
		QueueSize:    len(queue),
	}

	// Add to metrics history (keep last 10 data points)
	metrics = append(metrics, currentMetric)
	if len(metrics) > 10 {
		metrics = metrics[1:]
	}

	// Check alert conditions
	shouldAlert := false
	alertValue := 0.0

	switch alertMetric {
	case "latency":
		alertValue = avgLatency
		if avgLatency > float64(thresholdValue) && len(queue) > 0 {
			shouldAlert = true
		}
	case "throughput":
		alertValue = float64(len(queue))
		if len(queue) < thresholdValue { // Low throughput alert
			shouldAlert = true
		}
	case "error_rate":
		errorRate := 0.0
		if len(queue) > 0 {
			errorRate = float64(errorCount) / float64(len(queue)) * 100
		}
		alertValue = errorRate
		if errorRate > float64(thresholdValue) {
			shouldAlert = true
		}
	case "queue_size":
		alertValue = float64(len(queue))
		if len(queue) > thresholdValue {
			shouldAlert = true
		}
	}

	// Generate alert if threshold exceeded
	if shouldAlert {
		severity := "warning"
		if alertValue > float64(thresholdValue)*1.5 { // 150% of threshold
			severity = "critical"
		}

		alert := AlertEvent{
			Timestamp:  currentTime,
			MetricType: alertMetric,
			Value:      alertValue,
			Threshold:  float64(thresholdValue),
			Unit:       thresholdUnit,
			Severity:   severity,
		}

		// Only add alert if it's not a duplicate of the last alert
		if len(alerts) == 0 || !m.isDuplicateAlert(alerts[len(alerts)-1], alert) {
			alerts = append(alerts, alert)
		}

		// Keep only last 20 alerts
		if len(alerts) > 20 {
			alerts = alerts[1:]
		}
	}

	// Update props with collected data
	props["_metrics"] = metrics
	props["_alerts"] = alerts
	props["_currentLatency"] = avgLatency
	props["_alertCount"] = len(alerts)

	// Monitoring system health - it's healthy unless it's completely overloaded
	healthy := len(queue) < 10000 // Can handle very high loads

	// If too many critical alerts recently, mark as unhealthy
	recentCriticalAlerts := 0
	for _, alert := range alerts {
		if currentTime-alert.Timestamp < 10000 && alert.Severity == "critical" { // Last 10 seconds
			recentCriticalAlerts++
		}
	}

	if recentCriticalAlerts > 5 {
		healthy = false
	}

	return output, healthy
}

// isDuplicateAlert checks if an alert is similar to the previous one to avoid spam
func (m MonitoringLogic) isDuplicateAlert(prev, current AlertEvent) bool {
	return prev.MetricType == current.MetricType &&
		prev.Severity == current.Severity &&
		(current.Timestamp-prev.Timestamp) < 5000 // Within 5 seconds
}

// Helper function to calculate moving average
func (m MonitoringLogic) calculateMovingAverage(metrics []MetricData, window int) float64 {
	if len(metrics) == 0 {
		return 0
	}

	start := 0
	if len(metrics) > window {
		start = len(metrics) - window
	}

	sum := 0.0
	count := 0
	for i := start; i < len(metrics); i++ {
		if metrics[i].RequestCount > 0 {
			sum += float64(metrics[i].LatencySum) / float64(metrics[i].RequestCount)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
