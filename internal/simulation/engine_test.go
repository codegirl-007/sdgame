package simulation

import (
	"testing"

	"systemdesigngame/internal/design"
)

func TestSimpleChainSimulation(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "a", Type: "webserver", Props: map[string]any{"capacityRPS": 1, "baseLatencyMs": 10}},
			{ID: "b", Type: "webserver", Props: map[string]any{"capacityRPS": 1, "baseLatencyMs": 10}},
		},
		Connections: []design.Connection{
			{Source: "a", Target: "b"},
		},
	}

	engine := NewEngineFromDesign(d, 100)

	engine.Nodes["a"].Queue = append(engine.Nodes["a"].Queue, &Request{
		ID:        "req-1",
		Origin:    "a",
		Type:      "GET",
		Timestamp: 0,
		Path:      []string{"a"},
	})

	snaps := engine.Run(2, 100)

	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}

	if len(snaps[0].Emitted["a"]) != 1 {
		t.Errorf("expected a to emit 1 request at tick 0")
	}
	if snaps[0].QueueSizes["b"] != 0 {
		t.Errorf("expected b's queue to be 0 after tick 0 (not yet processed)")
	}

	if len(snaps[1].Emitted["b"]) != 1 {
		t.Errorf("expected b to emit 1 request at tick 1")
	}
}

func TestSingleTickRouting(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "a", Type: "webserver", Props: map[string]any{"capacityRPS": 1.0, "baseLatencyMs": 10.0}},
			{ID: "b", Type: "webserver", Props: map[string]any{"capacityRPS": 1.0, "baseLatencyMs": 10.0}},
		},
		Connections: []design.Connection{
			{Source: "a", Target: "b"},
		},
	}

	engine := NewEngineFromDesign(d, 100)

	engine.Nodes["a"].Queue = append(engine.Nodes["a"].Queue, &Request{
		ID:        "req-1",
		Origin:    "a",
		Type:      "GET",
		Timestamp: 0,
		Path:      []string{"a"},
	})

	snaps := engine.Run(1, 100)

	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	if len(snaps[0].Emitted["a"]) != 1 {
		t.Errorf("expected a to emit 1 request, got %d", len(snaps[0].Emitted["a"]))
	}

	if len(engine.Nodes["b"].Queue) != 1 {
		t.Errorf("expected b to have 1 request queued for next tick, got %d", len(engine.Nodes["b"].Queue))
	}
}

func TestHighRPSSimulation(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "entry", Type: "webserver", Props: map[string]any{"capacityRPS": 5000, "baseLatencyMs": 1}},
		},
		Connections: []design.Connection{},
	}

	engine := NewEngineFromDesign(d, 100)
	engine.EntryNode = "entry"
	engine.RPS = 100000

	snaps := engine.Run(10, 100)

	totalEmitted := 0
	for _, snap := range snaps {
		totalEmitted += len(snap.Emitted["entry"])
	}

	expected := 10 * 5000 // capacity-limited output
	if totalEmitted != expected {
		t.Errorf("expected %d total emitted requests, got %d", expected, totalEmitted)
	}
}
