package simulation

import (
	"testing"

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

