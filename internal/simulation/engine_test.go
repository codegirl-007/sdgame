package simulation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"encoding/json"
	"systemdesigngame/internal/design"
)

func TestNewEngineFromDesign(t *testing.T) {
	designInput := &design.Design{
		Nodes: []design.Node{
			{
				ID:   "web1",
				Type: "webserver",
				Props: map[string]interface{}{
					"cpu":            float64(2),
					"ramGb":          float64(4),
					"rpsCapacity":    float64(100),
					"monthlyCostUsd": float64(20),
				},
			},
			{
				ID:   "cache1",
				Type: "cache",
				Props: map[string]interface{}{
					"label":          "L1 Cache",
					"cacheTTL":       float64(60),
					"maxEntries":     float64(1000),
					"evictionPolicy": "LRU",
				},
			},
		},
		Connections: []design.Connection{
			{
				Source: "web1",
				Target: "cache1",
			},
		},
	}

	engine := NewEngineFromDesign(*designInput, 10, 100)

	if len(engine.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(engine.Nodes))
	}

	if len(engine.Nodes["web1"].GetTargets()) != 1 {
		t.Fatalf("expected web1 to have 1 target, got %d", len(engine.Nodes["web1"].GetTargets()))
	}

	if engine.Nodes["web1"].GetTargets()[0] != "cache1" {
		t.Fatalf("expected web1 target to be cache1")
	}
}

func TestComplexSimulationRun(t *testing.T) {
	filePath := filepath.Join("testdata", "complex_design.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	var d design.Design
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	engine := NewEngineFromDesign(d, 10, 100)
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	engine.Run()

	if len(engine.Timeline) == 0 {
		t.Fatal("Expected timeline snapshots after Run, got none")
	}

	// Optional: check that some nodes received or emitted requests
	for id, node := range engine.Nodes {
		if len(node.Emit()) > 0 {
			t.Logf("Node %s has activity", id)
		}
	}
}

func TestSimulationRunEndToEnd(t *testing.T) {
	data, err := os.ReadFile("testdata/simple_design.json")
	fmt.Print(data)
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	var design design.Design
	if err := json.Unmarshal(data, &design); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	engine := NewEngineFromDesign(design, 20, 100) // 20 ticks, 100ms per tick
	engine.Run()

	if len(engine.Timeline) != 20 {
		t.Errorf("Expected 20 timeline entries, got %d", len(engine.Timeline))
	}

	anyTraffic := false
	for _, snapshot := range engine.Timeline {
		for _, nodeState := range snapshot.NodeHealth {
			if nodeState.QueueSize > 0 {
				anyTraffic = true
				break
			}
		}
	}

	if !anyTraffic {
		t.Errorf("Expected at least one node to have non-zero queue size over time")
	}

	// Optional: check a few expected node IDs
	for _, id := range []string{"node-1", "node-2"} {
		if _, ok := engine.Nodes[id]; !ok {
			t.Errorf("Expected node %s to be present in simulation", id)
		}
	}
}
