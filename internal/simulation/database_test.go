package simulation

import (
	"testing"
)

func TestDatabaseLogic_BasicProcessing(t *testing.T) {
	db := DatabaseLogic{}

	props := map[string]any{
		"replication":   2,
		"maxRPS":        100,
		"baseLatencyMs": 15,
	}

	// Create test requests
	reqs := []*Request{
		{ID: "req1", Type: "GET", LatencyMS: 0, Path: []string{"start"}},
		{ID: "req2", Type: "POST", LatencyMS: 0, Path: []string{"start"}},
	}

	output, alive := db.Tick(props, reqs, 1)

	if !alive {
		t.Errorf("Database should be alive")
	}

	if len(output) != 2 {
		t.Errorf("Expected 2 output requests, got %d", len(output))
	}

	// Check read latency (base latency)
	readReq := output[0]
	if readReq.LatencyMS != 15 {
		t.Errorf("Expected read latency 15ms, got %dms", readReq.LatencyMS)
	}

	// Check write latency (base * 2 + replication penalty)
	writeReq := output[1]
	expectedWriteLatency := 15*2 + (2-1)*5 // 30 + 5 = 35ms
	if writeReq.LatencyMS != expectedWriteLatency {
		t.Errorf("Expected write latency %dms, got %dms", expectedWriteLatency, writeReq.LatencyMS)
	}
}

func TestDatabaseLogic_CapacityLimit(t *testing.T) {
	db := DatabaseLogic{}

	props := map[string]any{
		"maxRPS":        2,
		"baseLatencyMs": 10,
	}

	// Create more requests than capacity
	reqs := []*Request{
		{ID: "req1", Type: "GET"},
		{ID: "req2", Type: "GET"},
		{ID: "req3", Type: "GET"}, // This should be dropped
	}

	output, _ := db.Tick(props, reqs, 1)

	if len(output) != 2 {
		t.Errorf("Expected capacity limit of 2, but processed %d requests", len(output))
	}
}

func TestDatabaseLogic_DefaultValues(t *testing.T) {
	db := DatabaseLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	reqs := []*Request{
		{ID: "req1", Type: "GET", LatencyMS: 0},
	}

	output, _ := db.Tick(props, reqs, 1)

	if len(output) != 1 {
		t.Errorf("Expected 1 output request")
	}

	// Should use default 10ms base latency
	if output[0].LatencyMS != 10 {
		t.Errorf("Expected default latency 10ms, got %dms", output[0].LatencyMS)
	}
}

func TestDatabaseLogic_ReplicationEffect(t *testing.T) {
	db := DatabaseLogic{}

	// Test with high replication
	props := map[string]any{
		"replication":   5,
		"baseLatencyMs": 10,
	}

	reqs := []*Request{
		{ID: "req1", Type: "POST", LatencyMS: 0},
	}

	output, _ := db.Tick(props, reqs, 1)

	if len(output) != 1 {
		t.Errorf("Expected 1 output request")
	}

	// Write latency: base*2 + (replication-1)*5 = 10*2 + (5-1)*5 = 20 + 20 = 40ms
	expectedLatency := 10*2 + (5-1)*5
	if output[0].LatencyMS != expectedLatency {
		t.Errorf("Expected latency %dms with replication=5, got %dms", expectedLatency, output[0].LatencyMS)
	}
}

func TestDatabaseLogic_ReadVsWrite(t *testing.T) {
	db := DatabaseLogic{}

	props := map[string]any{
		"replication":   1,
		"baseLatencyMs": 20,
	}

	readReq := []*Request{{ID: "read", Type: "GET", LatencyMS: 0}}
	writeReq := []*Request{{ID: "write", Type: "POST", LatencyMS: 0}}

	readOutput, _ := db.Tick(props, readReq, 1)
	writeOutput, _ := db.Tick(props, writeReq, 1)

	// Read should be base latency
	if readOutput[0].LatencyMS != 20 {
		t.Errorf("Expected read latency 20ms, got %dms", readOutput[0].LatencyMS)
	}

	// Write should be double base latency (no replication penalty with replication=1)
	if writeOutput[0].LatencyMS != 40 {
		t.Errorf("Expected write latency 40ms, got %dms", writeOutput[0].LatencyMS)
	}
}
