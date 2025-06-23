package handlers

import (
	"encoding/json"
	"net/http"
	"systemdesigngame/internal/design"
	"systemdesigngame/internal/simulation"
)

type SimulationHandler struct{}

type SimulationResponse struct {
	Success  bool                   `json:"success"`
	Metrics  map[string]interface{} `json:"metrics,omitempty"`
	Timeline []interface{}          `json:"timeline,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

func (h *SimulationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var design design.Design
	if err := json.NewDecoder(r.Body).Decode(&design); err != nil {
		http.Error(w, "Invalid design JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	engine := simulation.NewEngineFromDesign(design, 10000, 100)
	engine.Run()

	// For now, return a mock successful response but eventually, we want to go to the results page(s)
	response := SimulationResponse{
		Success: true,
		Metrics: map[string]interface{}{
			"throughput":   250,
			"latency_p95":  85,
			"cost_monthly": 120,
			"availability": 99.5,
		},
		Timeline: []interface{}{}, // Will contain TickSnapshots later
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
