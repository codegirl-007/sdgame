package level

import (
	"encoding/json"
	"fmt"
	"os"
)

type Level struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Difficulty  Difficulty `json:"difficulty"`

	TargetRPS               int     `json:"targetRps"`
	DurationSec             int     `json:"durationSec"`
	MaxMonthlyUSD           int     `json:"maxMonthlyUsd"`
	MaxP95LatencyMs         int     `json:"maxP95LatencyMs"`
	RequiredAvailabilityPct float64 `json:"requiredAvailabilityPct"`

	MustInclude           []string       `json:"mustInclude,omitempty"`
	MustNotInclude        []string       `json:"mustNotInclude,omitempty"`
	EncouragedComponents  []string       `json:"encouragedComponents,omitempty"`
	DiscouragedComponents []string       `json:"discouragedComponents,omitempty"`
	MinReplicas           map[string]int `json:"minReplicas,omitempty"`
	MaxLatencyPerNodeType map[string]int `json:"maxLatencyPerNodeType,omitempty"`
	CustomValidators      []string       `json:"customValidators,omitempty"`

	FailureEvents  []FailureEvent     `json:"failureEvents,omitempty"`
	ScoringWeights map[string]float64 `json:"scoringWeights,omitempty"`

	Hints []string `json:"hints,omitempty"`
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

var Registry map[string]map[string]Level

type FailureEvent struct {
	Type     string `json:"type"`
	TimeSec  int    `json:"timeSec"`
	TargetID string `json:"targetId,omitempty"`
}

func LoadLevels(path string) ([]Level, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error opening levels.json: %w", err)
	}
	defer file.Close()

	var levels []Level
	err = json.NewDecoder(file).Decode(&levels)
	if err != nil {
		return nil, fmt.Errorf("Error decoding levels.json: %w", err)
	}

	return levels, nil
}

func InitRegistry(levels []Level) {
	Registry = make(map[string]map[string]Level)
	for _, lvl := range levels {
		// check if level already exists here
		if _, ok := Registry[lvl.Name]; !ok {
			Registry[lvl.Name] = make(map[string]Level)
		}
		// populate it
		Registry[lvl.Name][string(lvl.Difficulty)] = lvl
	}
}

func GetLevel(name string, difficulty Difficulty) (*Level, error) {
	diffMap, ok := Registry[name]
	if !ok {
		return nil, fmt.Errorf("level name %s not found", name)
	}

	lvl, ok := diffMap[string(difficulty)]
	if !ok {
		return nil, fmt.Errorf("difficulty %s not available for level '%s'", difficulty, name)
	}
	return &lvl, nil
}

func (d *Difficulty) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch s {
	case string(DifficultyEasy), string(DifficultyMedium), string(DifficultyHard):
		*d = Difficulty(s)
		return nil
	default:
		return fmt.Errorf("invalid difficulty: %q", s)
	}
}
