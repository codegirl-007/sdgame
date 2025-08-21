package level

import (
	"path/filepath"
	"testing"
)

func TestLoadLevels(t *testing.T) {
	path := filepath.Join("..", "..", "data", "levels.json")

	levels, err := LoadLevels(path)
	if err != nil {
		t.Fatalf("failed to load levels.json: %v", err)
	}

	if len(levels) == 0 {
		t.Fatalf("expected at least one level, got 0")
	}

	InitRegistry(levels)

	lvl, err := GetLevelByID("metrics-system")
	if err != nil {
		t.Fatalf("expected to retrieve metrics-system, got %v", err)
	}

	if lvl.ID != "metrics-system" {
		t.Errorf("unexpected level ID: got %s, want %s", lvl.ID, "metrics-system")
	}
}
