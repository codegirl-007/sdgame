package simulation

import (
	"systemdesigngame/internal/design"
	"testing"
)

func TestLoadBalancerAlgorithms(t *testing.T) {
	t.Run("round-rouble", func(t *testing.T) {
		d := design.Design{
			Nodes: []design.Node{
				{ID: "lb", Type: "loadbalancer", Props: map[string]any{"algorithm": "round-robin"}},
				{ID: "a", Type: "webserver", Props: map[string]any{"capacityRPS": 10}},
				{ID: "b", Type: "webserver", Props: map[string]any{"capacityRPS": 10}},
			},
			Connections: []design.Connection{
				{Source: "lb", Target: "a"},
				{Source: "lb", Target: "b"},
			},
		}

		e := NewEngineFromDesign(d, 100)
		e.EntryNode = "lb"
		e.RPS = 4

		snaps := e.Run(1, 100)
		if len(snaps[0].Emitted["lb"]) != 4 {
			t.Errorf("expected lb to emit 4 requests")
		}

		path0 := snaps[0].Emitted["lb"][0].Path[1]
		path1 := snaps[0].Emitted["lb"][1].Path[1]

		if path0 == path1 {
			t.Errorf("expecting alternating targets, got %s and %s", path0, path1)
		}
	})

	t.Run("least-connection", func(t *testing.T) {
		d := design.Design{
			Nodes: []design.Node{
				{ID: "lb", Type: "loadbalancer", Props: map[string]any{"algorithm": "least-connection"}},
				{ID: "a", Type: "webserver", Props: map[string]any{"capacityRPS": 1}},
				{ID: "b", Type: "webserver", Props: map[string]any{"capacityRPS": 1}},
			},
			Connections: []design.Connection{
				{Source: "lb", Target: "a"},
				{Source: "lb", Target: "b"},
			},
		}

		e := NewEngineFromDesign(d, 100)
		e.EntryNode = "lb"
		e.RPS = 2

		snaps := e.Run(1, 100)
		if len(snaps[0].Emitted["lb"]) != 2 {
			t.Errorf("expected lb to emit 2 requests")
		}

		paths := []string{
			snaps[0].Emitted["lb"][0].Path[1],
			snaps[0].Emitted["lb"][1].Path[1],
		}

		if paths[0] == paths[1] {
			t.Errorf("expected requests to be balanced, go %v", paths)
		}
	})
}
