package level

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLevels(t *testing.T) {
	path := filepath.Join("..", "..", "data", "levels.json")

	cwd, _ := os.Getwd()
	fmt.Println("Current working directory: ", cwd)
	fmt.Println("loading path: ", path)

	levels, err := LoadLevels(path)
	if err != nil {
		t.Fatalf("failed to load levels.json: %v", err)
	}

	if len(levels) == 0 {
		t.Fatalf("expected at least one level, got 0")
	}

	InitRegistry(levels)

	lvl, err := GetLevel("Metrics System", DifficultyHard)
	if err != nil {
		t.Fatalf("expected to retrieve Metrics System (hard), got %v", err)
	}

	if lvl.Difficulty != DifficultyHard {
		t.Errorf("unexpected difficulty: got %s, want %s", lvl.Difficulty, DifficultyHard)
	}
}
