package simulation

import (
	"testing"
)

func TestMonitoringLogic_BasicPassthrough(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "latency",
		"thresholdValue": 100.0,
		"thresholdUnit":  "ms",
	}

	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 50, Path: []string{}},
		{ID: "2", Type: "POST", LatencyMS: 75, Path: []string{}},
	}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy")
	}

	if len(output) != 2 {
		t.Errorf("Expected 2 requests to pass through monitoring, got %d", len(output))
	}

	// Verify minimal latency overhead was added
	for i, req := range output {
		originalLatency := requests[i].LatencyMS
		if req.LatencyMS <= originalLatency {
			t.Errorf("Expected monitoring overhead to be added to latency")
		}
		if req.LatencyMS > originalLatency+5 {
			t.Errorf("Expected minimal monitoring overhead, got %d ms added", req.LatencyMS-originalLatency)
		}
		if len(req.Path) == 0 || req.Path[len(req.Path)-1] != "monitored" {
			t.Error("Expected path to be updated with 'monitored'")
		}
	}
}

func TestMonitoringLogic_MetricsCollection(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Datadog",
		"alertMetric":    "latency",
		"thresholdValue": 100.0,
		"thresholdUnit":  "ms",
	}

	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 50},
		{ID: "2", Type: "POST", LatencyMS: 150},
		{ID: "3", Type: "GET", LatencyMS: 75},
	}

	_, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy")
	}

	// Check that metrics were collected
	metrics, ok := props["_metrics"].([]MetricData)
	if !ok {
		t.Error("Expected metrics to be collected in props")
	}

	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric data point, got %d", len(metrics))
	}

	metric := metrics[0]
	if metric.RequestCount != 3 {
		t.Errorf("Expected 3 requests counted, got %d", metric.RequestCount)
	}

	if metric.LatencySum != 275 { // 50 + 150 + 75
		t.Errorf("Expected latency sum of 275, got %d", metric.LatencySum)
	}

	// Check current latency calculation
	currentLatency, ok := props["_currentLatency"].(float64)
	if !ok {
		t.Error("Expected current latency to be calculated")
	}

	if currentLatency < 90 || currentLatency > 95 {
		t.Errorf("Expected average latency around 91.67, got %f", currentLatency)
	}
}

func TestMonitoringLogic_LatencyAlert(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "latency",
		"thresholdValue": 80.0,
		"thresholdUnit":  "ms",
	}

	// Send requests that exceed latency threshold
	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 100},
		{ID: "2", Type: "POST", LatencyMS: 120},
	}

	_, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy despite alerts")
	}

	// Check that alert was generated
	alerts, ok := props["_alerts"].([]AlertEvent)
	if !ok {
		t.Error("Expected alerts to be stored in props")
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert to be generated, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.MetricType != "latency" {
		t.Errorf("Expected latency alert, got %s", alert.MetricType)
	}

	if alert.Threshold != 80.0 {
		t.Errorf("Expected threshold of 80, got %f", alert.Threshold)
	}

	if alert.Value < 80.0 {
		t.Errorf("Expected alert value to exceed threshold, got %f", alert.Value)
	}

	if alert.Severity != "warning" {
		t.Errorf("Expected warning severity, got %s", alert.Severity)
	}
}

func TestMonitoringLogic_ErrorRateAlert(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "error_rate",
		"thresholdValue": 20.0, // 20% error rate threshold
		"thresholdUnit":  "percent",
	}

	// Send mix of normal and high-latency (error) requests
	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 100},   // normal
		{ID: "2", Type: "POST", LatencyMS: 1200}, // error (>1000ms)
		{ID: "3", Type: "GET", LatencyMS: 200},   // normal
		{ID: "4", Type: "POST", LatencyMS: 1500}, // error
	}

	_, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy")
	}

	// Check that error rate alert was generated (50% error rate > 20% threshold)
	alerts, ok := props["_alerts"].([]AlertEvent)
	if !ok {
		t.Error("Expected alerts to be stored in props")
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert to be generated, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.MetricType != "error_rate" {
		t.Errorf("Expected error_rate alert, got %s", alert.MetricType)
	}

	if alert.Value != 50.0 { // 2 errors out of 4 requests = 50%
		t.Errorf("Expected 50%% error rate, got %f", alert.Value)
	}
}

func TestMonitoringLogic_QueueSizeAlert(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "queue_size",
		"thresholdValue": 5.0,
		"thresholdUnit":  "requests",
	}

	// Send more requests than threshold
	requests := make([]*Request, 8)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 50}
	}

	_, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy with queue size alert")
	}

	// Check that queue size alert was generated
	alerts, ok := props["_alerts"].([]AlertEvent)
	if !ok {
		t.Error("Expected alerts to be stored in props")
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert to be generated, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.MetricType != "queue_size" {
		t.Errorf("Expected queue_size alert, got %s", alert.MetricType)
	}

	if alert.Value != 8.0 {
		t.Errorf("Expected queue size of 8, got %f", alert.Value)
	}
}

func TestMonitoringLogic_CriticalAlert(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "latency",
		"thresholdValue": 100.0,
		"thresholdUnit":  "ms",
	}

	// Send requests with very high latency (150% of threshold)
	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 180}, // 180 > 150 (1.5 * 100)
		{ID: "2", Type: "POST", LatencyMS: 200},
	}

	_, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy")
	}

	alerts, ok := props["_alerts"].([]AlertEvent)
	if !ok {
		t.Error("Expected alerts to be stored in props")
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert to be generated, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.Severity != "critical" {
		t.Errorf("Expected critical severity for high threshold breach, got %s", alert.Severity)
	}
}

func TestMonitoringLogic_DuplicateAlertSuppression(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "latency",
		"thresholdValue": 80.0,
		"thresholdUnit":  "ms",
	}

	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 100},
	}

	// First tick - should generate alert
	logic.Tick(props, requests, 1)

	alerts, _ := props["_alerts"].([]AlertEvent)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert after first tick, got %d", len(alerts))
	}

	// Second tick immediately after - should suppress duplicate
	logic.Tick(props, requests, 2)

	alerts, _ = props["_alerts"].([]AlertEvent)
	if len(alerts) != 1 {
		t.Errorf("Expected duplicate alert to be suppressed, got %d alerts", len(alerts))
	}
}

func TestMonitoringLogic_DefaultValues(t *testing.T) {
	logic := MonitoringLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	requests := []*Request{{ID: "1", Type: "GET", LatencyMS: 50, Path: []string{}}}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected monitoring to be healthy with default values")
	}

	if len(output) != 1 {
		t.Errorf("Expected 1 request to pass through, got %d", len(output))
	}

	// Should have reasonable default monitoring overhead
	if output[0].LatencyMS <= 50 || output[0].LatencyMS > 55 {
		t.Errorf("Expected default monitoring overhead, got %dms total", output[0].LatencyMS)
	}
}

func TestMonitoringLogic_ToolSpecificOverhead(t *testing.T) {
	logic := MonitoringLogic{}

	// Test Prometheus (lower overhead)
	propsPrometheus := map[string]any{
		"tool": "Prometheus",
	}

	// Test Datadog (higher overhead)
	propsDatadog := map[string]any{
		"tool": "Datadog",
	}

	request := []*Request{{ID: "1", Type: "GET", LatencyMS: 50, Path: []string{}}}

	prometheusOutput, _ := logic.Tick(propsPrometheus, request, 1)
	datadogOutput, _ := logic.Tick(propsDatadog, request, 1)

	prometheusOverhead := prometheusOutput[0].LatencyMS - 50
	datadogOverhead := datadogOutput[0].LatencyMS - 50

	if datadogOverhead <= prometheusOverhead {
		t.Errorf("Expected Datadog (%dms) to have higher overhead than Prometheus (%dms)",
			datadogOverhead, prometheusOverhead)
	}
}

func TestMonitoringLogic_UnhealthyWithManyAlerts(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool":           "Prometheus",
		"alertMetric":    "latency",
		"thresholdValue": 50.0,
		"thresholdUnit":  "ms",
	}

	// Manually create many recent critical alerts to simulate an unhealthy state
	currentTime := 10000 // 10 seconds
	recentAlerts := []AlertEvent{
		{Timestamp: currentTime - 1000, MetricType: "latency", Severity: "critical", Value: 200},
		{Timestamp: currentTime - 2000, MetricType: "latency", Severity: "critical", Value: 180},
		{Timestamp: currentTime - 3000, MetricType: "latency", Severity: "critical", Value: 190},
		{Timestamp: currentTime - 4000, MetricType: "latency", Severity: "critical", Value: 170},
		{Timestamp: currentTime - 5000, MetricType: "latency", Severity: "critical", Value: 160},
		{Timestamp: currentTime - 6000, MetricType: "latency", Severity: "critical", Value: 150},
	}

	// Set up the props with existing critical alerts
	props["_alerts"] = recentAlerts

	// Make a request that would trigger another alert (low latency to avoid triggering new alert)
	requests := []*Request{{ID: "1", Type: "GET", LatencyMS: 40}}

	// This tick should recognize the existing critical alerts and mark system as unhealthy
	_, healthy := logic.Tick(props, requests, 100) // tick 100 = 10000ms

	if healthy {
		t.Error("Expected monitoring to be unhealthy due to many recent critical alerts")
	}
}

func TestMonitoringLogic_MetricsHistoryLimit(t *testing.T) {
	logic := MonitoringLogic{}

	props := map[string]any{
		"tool": "Prometheus",
	}

	request := []*Request{{ID: "1", Type: "GET", LatencyMS: 50}}

	// Generate more than 10 metric data points
	for i := 0; i < 15; i++ {
		logic.Tick(props, request, i)
	}

	metrics, ok := props["_metrics"].([]MetricData)
	if !ok {
		t.Error("Expected metrics to be stored")
	}

	if len(metrics) != 10 {
		t.Errorf("Expected metrics history to be limited to 10, got %d", len(metrics))
	}
}
