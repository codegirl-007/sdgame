package simulation

import (
	"testing"
)

func TestMicroserviceLogic_BasicProcessing(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   2.0,
		"cpu":             4.0,
		"ramGb":           8.0,
		"rpsCapacity":     100.0,
		"scalingStrategy": "manual",
	}

	requests := []*Request{
		{ID: "1", Type: "GET", LatencyMS: 0, Path: []string{}},
		{ID: "2", Type: "POST", LatencyMS: 0, Path: []string{}},
	}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected microservice to be healthy")
	}

	if len(output) != 2 {
		t.Errorf("Expected 2 processed requests, got %d", len(output))
	}

	// Verify latency was added
	for _, req := range output {
		if req.LatencyMS == 0 {
			t.Error("Expected latency to be added to processed request")
		}
		if len(req.Path) == 0 || req.Path[len(req.Path)-1] != "microservice-processed" {
			t.Error("Expected path to be updated with microservice-processed")
		}
	}
}

func TestMicroserviceLogic_CapacityLimit(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   1.0,
		"rpsCapacity":     2.0,
		"scalingStrategy": "manual",
	}

	// Send 4 requests, capacity is 2 (1 instance * 2 RPS)
	// This should be healthy since 4 <= totalCapacity*2 (4)
	requests := make([]*Request, 4)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 0}
	}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected microservice to be healthy with moderate queuing")
	}

	// Should only process 2 requests (capacity limit)
	if len(output) != 2 {
		t.Errorf("Expected 2 processed requests due to capacity limit, got %d", len(output))
	}
}

func TestMicroserviceLogic_AutoScaling(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   1.0,
		"rpsCapacity":     10.0,
		"scalingStrategy": "auto",
	}

	// Send 25 requests to trigger scaling
	requests := make([]*Request, 25)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 0}
	}

	output, healthy := logic.Tick(props, requests, 1)

	// Check if instances were scaled up
	newInstanceCount := int(props["instanceCount"].(float64))
	if newInstanceCount <= 1 {
		t.Error("Expected auto-scaling to increase instance count")
	}

	// Should process more than 10 requests (original capacity)
	if len(output) <= 10 {
		t.Errorf("Expected auto-scaling to increase processing capacity, got %d", len(output))
	}

	if !healthy {
		t.Error("Expected microservice to be healthy after scaling")
	}
}

func TestMicroserviceLogic_ResourceBasedLatency(t *testing.T) {
	logic := MicroserviceLogic{}

	// High-resource microservice
	highResourceProps := map[string]any{
		"instanceCount":   1.0,
		"cpu":             8.0,
		"ramGb":           16.0,
		"rpsCapacity":     100.0,
		"scalingStrategy": "manual",
	}

	// Low-resource microservice
	lowResourceProps := map[string]any{
		"instanceCount":   1.0,
		"cpu":             1.0,
		"ramGb":           1.0,
		"rpsCapacity":     100.0,
		"scalingStrategy": "manual",
	}

	request := []*Request{{ID: "1", Type: "GET", LatencyMS: 0, Path: []string{}}}

	highOutput, _ := logic.Tick(highResourceProps, request, 1)
	lowOutput, _ := logic.Tick(lowResourceProps, request, 1)

	highLatency := highOutput[0].LatencyMS
	lowLatency := lowOutput[0].LatencyMS

	if lowLatency <= highLatency {
		t.Errorf("Expected low-resource microservice (%dms) to have higher latency than high-resource (%dms)",
			lowLatency, highLatency)
	}
}

func TestMicroserviceLogic_RequestTypeLatency(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   1.0,
		"cpu":             2.0,
		"ramGb":           4.0,
		"rpsCapacity":     100.0,
		"scalingStrategy": "manual",
	}

	getRequest := []*Request{{ID: "1", Type: "GET", LatencyMS: 0, Path: []string{}}}
	postRequest := []*Request{{ID: "2", Type: "POST", LatencyMS: 0, Path: []string{}}}
	computeRequest := []*Request{{ID: "3", Type: "COMPUTE", LatencyMS: 0, Path: []string{}}}

	getOutput, _ := logic.Tick(props, getRequest, 1)
	postOutput, _ := logic.Tick(props, postRequest, 1)
	computeOutput, _ := logic.Tick(props, computeRequest, 1)

	getLatency := getOutput[0].LatencyMS
	postLatency := postOutput[0].LatencyMS
	computeLatency := computeOutput[0].LatencyMS

	if getLatency >= postLatency {
		t.Errorf("Expected GET (%dms) to be faster than POST (%dms)", getLatency, postLatency)
	}

	if postLatency >= computeLatency {
		t.Errorf("Expected POST (%dms) to be faster than COMPUTE (%dms)", postLatency, computeLatency)
	}
}

func TestMicroserviceLogic_HighLoadLatencyPenalty(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   1.0,
		"cpu":             2.0,
		"ramGb":           4.0,
		"rpsCapacity":     10.0,
		"scalingStrategy": "manual",
	}

	// Low load scenario
	lowLoadRequest := []*Request{{ID: "1", Type: "GET", LatencyMS: 0, Path: []string{}}}
	lowOutput, _ := logic.Tick(props, lowLoadRequest, 1)
	lowLatency := lowOutput[0].LatencyMS

	// High load scenario (above 80% capacity threshold)
	highLoadRequests := make([]*Request, 9) // 90% of 10 RPS capacity
	for i := range highLoadRequests {
		highLoadRequests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 0, Path: []string{}}
	}
	highOutput, _ := logic.Tick(props, highLoadRequests, 1)

	// Check if first request has higher latency due to load
	highLatency := highOutput[0].LatencyMS

	if highLatency <= lowLatency {
		t.Errorf("Expected high load scenario (%dms) to have higher latency than low load (%dms)",
			highLatency, lowLatency)
	}
}

func TestMicroserviceLogic_DefaultValues(t *testing.T) {
	logic := MicroserviceLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	requests := []*Request{{ID: "1", Type: "GET", LatencyMS: 0, Path: []string{}}}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected microservice to be healthy with default values")
	}

	if len(output) != 1 {
		t.Errorf("Expected 1 processed request with defaults, got %d", len(output))
	}

	// Should have reasonable default latency
	if output[0].LatencyMS <= 0 || output[0].LatencyMS > 100 {
		t.Errorf("Expected reasonable default latency, got %dms", output[0].LatencyMS)
	}
}

func TestMicroserviceLogic_UnhealthyWhenOverloaded(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   1.0,
		"rpsCapacity":     5.0,
		"scalingStrategy": "manual", // No auto-scaling
	}

	// Send way more requests than capacity (5 * 2 = 10 max before unhealthy)
	requests := make([]*Request, 15) // 3x capacity
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 0}
	}

	output, healthy := logic.Tick(props, requests, 1)

	if healthy {
		t.Error("Expected microservice to be unhealthy when severely overloaded")
	}

	// Should still process up to capacity
	if len(output) != 5 {
		t.Errorf("Expected 5 processed requests despite being overloaded, got %d", len(output))
	}
}

func TestMicroserviceLogic_RoundRobinDistribution(t *testing.T) {
	logic := MicroserviceLogic{}

	props := map[string]any{
		"instanceCount":   3.0,
		"rpsCapacity":     10.0,
		"scalingStrategy": "manual",
	}

	// Send 6 requests to be distributed across 3 instances
	requests := make([]*Request, 6)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "GET", LatencyMS: 0, Path: []string{}}
	}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected microservice to be healthy")
	}

	if len(output) != 6 {
		t.Errorf("Expected 6 processed requests, got %d", len(output))
	}

	// All requests should be processed (within total capacity of 30)
	for _, req := range output {
		if req.LatencyMS <= 0 {
			t.Error("Expected all requests to have added latency")
		}
	}
}
