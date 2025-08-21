package level

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

type Level struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

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

	Hints                     []string `json:"hints,omitempty"`
	InterviewerRequirements   []string `json:"interviewerRequirements,omitempty"`
	FunctionalRequirements    []string `json:"functionalRequirements,omitempty"`
	NonFunctionalRequirements []string `json:"nonFunctionalRequirements,omitempty"`
}

var Registry map[string]Level

type FailureEvent struct {
	Type     string `json:"type"`
	TimeSec  int    `json:"timeSec"`
	TargetID string `json:"targetId,omitempty"`
}

func LoadLevels(path string) ([]Level, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error opening levels.json: %w", err)
	}

	var levels []Level
	if err := json.Unmarshal(data, &levels); err != nil {
		return nil, fmt.Errorf("Error parsing levels.json: %w", err)
	}

	return levels, nil
}

func InitRegistry(levels []Level) {
	Registry = make(map[string]Level)
	for _, lvl := range levels {
		Registry[lvl.ID] = lvl
	}
}

func GetLevelByID(id string) (*Level, error) {
	lvl, ok := Registry[id]
	if !ok {
		return nil, fmt.Errorf("level with ID %s not found", id)
	}
	return &lvl, nil
}

func AllLevels() []Level {
	var levels []Level
	for _, lvl := range Registry {
		levels = append(levels, lvl)
	}
	slices.SortFunc(levels, func(i Level, j Level) int {
		if i.Name < j.Name {
			return -1
		}
		if i.Name > j.Name {
			return 1
		}
		return 0
	})
	return levels
}
