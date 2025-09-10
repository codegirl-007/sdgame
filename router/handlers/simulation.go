package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"systemdesigngame/internal/design"
	"systemdesigngame/internal/level"
	"systemdesigngame/internal/simulation"
)

type SimulationHandler struct{}

type SimulationResponse struct {
	Success   bool                   `json:"success"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Timeline  []interface{}          `json:"timeline,omitempty"`
	Passed    bool                   `json:"passed"`
	Score     int                    `json:"score,omitempty"`
	Feedback  []string               `json:"feedback,omitempty"`
	LevelName string                 `json:"levelName,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

func (h *SimulationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		Design  design.Design `json:"design"`
		LevelID string        `json:"levelId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		// Try to decode as just design for backward compatibility
		r.Body.Close()
		var design design.Design
		if err2 := json.NewDecoder(r.Body).Decode(&design); err2 != nil {
			http.Error(w, "Invalid request JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		requestBody.Design = design
	}

	// Extract the design for processing
	design := requestBody.Design

	// Run the actual simulation
	engine := simulation.NewEngineFromDesign(design, 100)
	if engine == nil {
		response := SimulationResponse{
			Success: false,
			Error:   "Failed to create simulation engine - no valid components found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Set simulation parameters
	defaultRPS := 50
	targetRPS := defaultRPS

	if requestBody.LevelID != "" {
		if lvl, err := level.GetLevelByID(requestBody.LevelID); err == nil {
			targetRPS = lvl.TargetRPS
		}
	}

	engine.RPS = targetRPS

	// Find entry node by analyzing topology
	entryNode := findEntryNode(design)

	if entryNode == "" {
		response := SimulationResponse{
			Success: false,
			Error:   "No entry point found - design must include a component with no incoming connections (webserver, microservice, load balancer, etc.)",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	engine.EntryNode = entryNode

	// Run simulation for 60 ticks (6 seconds at 100ms per tick)
	snapshots := engine.Run(60, 100)

	// Calculate metrics from snapshots
	metrics := calculateMetrics(snapshots, design)

	// Convert snapshots to interface{} for JSON serialization
	timeline := make([]interface{}, len(snapshots))
	for i, snapshot := range snapshots {
		timeline[i] = snapshot
	}

	// Perform level validation if level info provided
	var passed bool
	var score int
	var feedback []string
	var levelName string

	if requestBody.LevelID != "" {
		if lvl, err := level.GetLevelByID(requestBody.LevelID); err == nil {
			levelName = lvl.Name
			passed, score, feedback = validateLevel(lvl, design, metrics)
		} else {
			feedback = []string{"Warning: Level not found, simulation ran without validation"}
		}
	}

	// Build redirect URL based on success/failure
	var redirectURL string
	if passed {
		// Success page
		redirectURL = buildSuccessURL(levelName, score, metrics, feedback, requestBody.LevelID)
	} else {
		// Failure page
		redirectURL = buildFailureURL(levelName, metrics, feedback, requestBody.LevelID)
	}

	// Redirect to appropriate result page
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// calculateMetrics computes key performance metrics from simulation snapshots
func calculateMetrics(snapshots []*simulation.TickSnapshot, design design.Design) map[string]interface{} {
	if len(snapshots) == 0 {
		return map[string]interface{}{
			"throughput":   0,
			"latency_avg":  0,
			"cost_monthly": 0,
			"availability": 0,
		}
	}

	totalRequests := 0
	totalLatency := 0
	totalHealthy := 0
	totalNodes := 0

	// Calculate aggregate metrics across all snapshots
	for _, snapshot := range snapshots {
		// Count total requests processed in this tick
		for _, requests := range snapshot.Emitted {
			totalRequests += len(requests)
			for _, req := range requests {
				totalLatency += req.LatencyMS
			}
		}

		// Count healthy vs total nodes
		for _, healthy := range snapshot.NodeHealth {
			totalNodes++
			if healthy {
				totalHealthy++
			}
		}
	}

	// Calculate throughput (requests per second)
	// snapshots represent 6 seconds of simulation (60 ticks * 100ms)
	simulationSeconds := float64(len(snapshots)) * 0.1 // 100ms per tick
	throughput := float64(totalRequests) / simulationSeconds

	// Calculate average latency
	avgLatency := 0.0
	if totalRequests > 0 {
		avgLatency = float64(totalLatency) / float64(totalRequests)
	}

	// Calculate availability percentage
	availability := 0.0
	if totalNodes > 0 {
		availability = (float64(totalHealthy) / float64(totalNodes)) * 100
	}

	// Calculate monthly cost based on component specifications
	monthlyCost := calculateRealMonthlyCost(design.Nodes)

	return map[string]interface{}{
		"throughput":   int(throughput),
		"latency_avg":  avgLatency,
		"cost_monthly": int(monthlyCost),
		"availability": availability,
	}
}

// findEntryNode analyzes the design topology to find the best entry point
func findEntryNode(design design.Design) string {
	// Build map of incoming connections
	incomingCount := make(map[string]int)

	// Initialize all nodes with 0 incoming connections
	for _, node := range design.Nodes {
		incomingCount[node.ID] = 0
	}

	// Count incoming connections for each node
	for _, conn := range design.Connections {
		incomingCount[conn.Target]++
	}

	// Find nodes with no incoming connections (potential entry points)
	var entryPoints []string
	for nodeID, count := range incomingCount {
		if count == 0 {
			entryPoints = append(entryPoints, nodeID)
		}
	}

	// If multiple entry points exist, prefer certain types
	if len(entryPoints) > 1 {
		return preferredEntryPoint(design.Nodes, entryPoints)
	} else if len(entryPoints) == 1 {
		return entryPoints[0]
	}

	return "" // No entry point found
}

// preferredEntryPoint selects the best entry point from candidates based on component type
func preferredEntryPoint(nodes []design.Node, candidateIDs []string) string {
	// Priority order for entry points (most logical first)
	priority := []string{
		"webserver",
		"microservice",
		"loadBalancer",  // Could be edge load balancer
		"cdn",           // Edge CDN
		"data pipeline", // Data ingestion entry
		"messageQueue",  // For event-driven architectures
	}

	// Create lookup for candidate nodes
	candidates := make(map[string]design.Node)
	for _, node := range nodes {
		for _, id := range candidateIDs {
			if node.ID == id {
				candidates[id] = node
				break
			}
		}
	}

	// Find highest priority candidate
	for _, nodeType := range priority {
		for id, node := range candidates {
			if node.Type == nodeType {
				return id
			}
		}
	}

	// If no preferred type, return first candidate
	if len(candidateIDs) > 0 {
		return candidateIDs[0]
	}

	return ""
}

// validateLevel checks if the design and simulation results meet level requirements
func validateLevel(lvl *level.Level, design design.Design, metrics map[string]interface{}) (bool, int, []string) {
	var feedback []string
	var failedRequirements []string
	var passedRequirements []string

	// Extract metrics
	avgLatency := int(metrics["latency_avg"].(float64))
	availability := metrics["availability"].(float64)
	monthlyCost := metrics["cost_monthly"].(int)

	// Check latency requirement (using avg latency as approximation for P95)
	if avgLatency <= lvl.MaxP95LatencyMs {
		passedRequirements = append(passedRequirements, "Latency requirement met")
	} else {
		failedRequirements = append(failedRequirements,
			fmt.Sprintf("Latency: %dms (max allowed: %dms)", avgLatency, lvl.MaxP95LatencyMs))
	}

	// Check availability requirement
	if availability >= lvl.RequiredAvailabilityPct {
		passedRequirements = append(passedRequirements, "Availability requirement met")
	} else {
		failedRequirements = append(failedRequirements,
			fmt.Sprintf("Availability: %.1f%% (required: %.1f%%)", availability, lvl.RequiredAvailabilityPct))
	}

	// Check cost requirement
	if monthlyCost <= lvl.MaxMonthlyUSD {
		passedRequirements = append(passedRequirements, "Cost requirement met")
	} else {
		failedRequirements = append(failedRequirements,
			fmt.Sprintf("Cost: $%d/month (max allowed: $%d/month)", monthlyCost, lvl.MaxMonthlyUSD))
	}

	// Check component requirements
	componentFeedback := validateComponentRequirements(lvl, design)
	if len(componentFeedback.Failed) > 0 {
		failedRequirements = append(failedRequirements, componentFeedback.Failed...)
	}
	if len(componentFeedback.Passed) > 0 {
		passedRequirements = append(passedRequirements, componentFeedback.Passed...)
	}

	// Determine if passed
	passed := len(failedRequirements) == 0

	// Calculate score (0-100)
	score := calculateScore(len(passedRequirements), len(failedRequirements), metrics)

	// Build feedback
	if passed {
		feedback = append(feedback, "Level completed successfully!")
		feedback = append(feedback, "")
		feedback = append(feedback, passedRequirements...)
	} else {
		feedback = append(feedback, "Level failed - requirements not met:")
		feedback = append(feedback, "")
		feedback = append(feedback, failedRequirements...)
		if len(passedRequirements) > 0 {
			feedback = append(feedback, "")
			feedback = append(feedback, "Requirements passed:")
			feedback = append(feedback, passedRequirements...)
		}
	}

	return passed, score, feedback
}

type ComponentValidationResult struct {
	Passed []string
	Failed []string
}

// validateComponentRequirements checks mustInclude, mustNotInclude, etc.
func validateComponentRequirements(lvl *level.Level, design design.Design) ComponentValidationResult {
	result := ComponentValidationResult{}

	// Build map of component types in design
	componentTypes := make(map[string]int)
	for _, node := range design.Nodes {
		componentTypes[node.Type]++
	}

	// Check mustInclude requirements
	for _, required := range lvl.MustInclude {
		if count, exists := componentTypes[required]; exists && count > 0 {
			result.Passed = append(result.Passed, fmt.Sprintf("Required component '%s' included", required))
		} else {
			result.Failed = append(result.Failed, fmt.Sprintf("Missing required component: '%s'", required))
		}
	}

	// Check mustNotInclude requirements
	for _, forbidden := range lvl.MustNotInclude {
		if count, exists := componentTypes[forbidden]; exists && count > 0 {
			result.Failed = append(result.Failed, fmt.Sprintf("Forbidden component used: '%s'", forbidden))
		}
	}

	// Check minReplicas requirements
	for component, minCount := range lvl.MinReplicas {
		if count, exists := componentTypes[component]; exists && count >= minCount {
			result.Passed = append(result.Passed, fmt.Sprintf("Sufficient '%s' replicas (%d)", component, count))
		} else {
			actualCount := 0
			if exists {
				actualCount = count
			}
			result.Failed = append(result.Failed,
				fmt.Sprintf("Insufficient '%s' replicas: %d (minimum: %d)", component, actualCount, minCount))
		}
	}

	return result
}

// calculateScore computes a score from 0-100 based on performance
func calculateScore(passedCount, failedCount int, metrics map[string]interface{}) int {
	if failedCount > 0 {
		// Failed level - score based on how many requirements passed
		return (passedCount * 100) / (passedCount + failedCount)
	}

	// Passed level - bonus points for performance
	baseScore := 70 // Base score for passing

	// Performance bonuses (up to 30 points)
	performanceBonus := 0

	// Throughput bonus (higher throughput = better)
	if throughput, ok := metrics["throughput"].(int); ok && throughput > 0 {
		performanceBonus += min(10, throughput/100) // 1 point per 100 RPS, max 10
	}

	// Availability bonus (higher availability = better)
	if availability, ok := metrics["availability"].(float64); ok {
		if availability >= 99.9 {
			performanceBonus += 10
		} else if availability >= 99.5 {
			performanceBonus += 5
		}
	}

	// Cost efficiency bonus (lower cost = better)
	if cost, ok := metrics["cost_monthly"].(int); ok && cost > 0 {
		if cost <= 50 {
			performanceBonus += 10
		} else if cost <= 100 {
			performanceBonus += 5
		}
	}

	return min(100, baseScore+performanceBonus)
}

// calculateRealMonthlyCost computes monthly cost based on actual component specifications
func calculateRealMonthlyCost(nodes []design.Node) float64 {
	totalCost := 0.0

	for _, node := range nodes {
		switch node.Type {
		case "user":
			// User components don't cost anything
			continue
		case "microservice":
			if monthlyUsd, ok := node.Props["monthlyUsd"].(float64); ok {
				if instanceCount, ok := node.Props["instanceCount"].(float64); ok {
					totalCost += monthlyUsd * instanceCount
				}
			}
		case "webserver":
			if monthlyCost, ok := node.Props["monthlyCostUsd"].(float64); ok {
				totalCost += monthlyCost
			}
		default:
			// Default cost for other components (cache, database, load balancer, etc.)
			totalCost += 20 // $20/month baseline
		}
	}

	return totalCost
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildSuccessURL creates a URL for the success page with simulation results
func buildSuccessURL(levelName string, score int, metrics map[string]interface{}, feedback []string, levelID string) string {
	baseURL := "/success"

	// Get level data if available
	var targetRPS, targetLatency int
	var targetAvail float64
	if levelID != "" {
		if lvl, err := level.GetLevelByID(levelID); err == nil {
			targetRPS = lvl.TargetRPS
			targetLatency = lvl.MaxP95LatencyMs
			targetAvail = lvl.RequiredAvailabilityPct
		}
	}

	// Use defaults if level not found
	if targetRPS == 0 {
		targetRPS = 10000
	}
	if targetLatency == 0 {
		targetLatency = 200
	}
	if targetAvail == 0 {
		targetAvail = 99.9
	}

	params := []string{
		fmt.Sprintf("level=%s", levelName),
		fmt.Sprintf("score=%d", score),
		fmt.Sprintf("targetRPS=%d", targetRPS),
		fmt.Sprintf("achievedRPS=%d", metrics["throughput"].(int)),
		fmt.Sprintf("targetLatency=%d", targetLatency),
		fmt.Sprintf("actualLatency=%.1f", metrics["latency_avg"].(float64)),
		fmt.Sprintf("availability=%.1f", metrics["availability"].(float64)),
		fmt.Sprintf("levelId=%s", levelID),
	}

	// Add feedback as pipe-separated values
	if len(feedback) > 0 {
		feedbackStr := strings.Join(feedback, "|")
		params = append(params, fmt.Sprintf("feedback=%s", feedbackStr))
	}

	return baseURL + "?" + strings.Join(params, "&")
}

// buildFailureURL creates a URL for the failure page with simulation results
func buildFailureURL(levelName string, metrics map[string]interface{}, feedback []string, levelID string) string {
	baseURL := "/failure"

	// Get level data if available
	var targetRPS, targetLatency int
	var targetAvail float64
	if levelID != "" {
		if lvl, err := level.GetLevelByID(levelID); err == nil {
			targetRPS = lvl.TargetRPS
			targetLatency = lvl.MaxP95LatencyMs
			targetAvail = lvl.RequiredAvailabilityPct
		}
	}

	// Use defaults if level not found
	if targetRPS == 0 {
		targetRPS = 10000
	}
	if targetLatency == 0 {
		targetLatency = 200
	}
	if targetAvail == 0 {
		targetAvail = 99.9
	}

	params := []string{
		fmt.Sprintf("level=%s", levelName),
		fmt.Sprintf("reason=performance"),
		fmt.Sprintf("targetRPS=%d", targetRPS),
		fmt.Sprintf("achievedRPS=%d", metrics["throughput"].(int)),
		fmt.Sprintf("targetLatency=%d", targetLatency),
		fmt.Sprintf("actualLatency=%.1f", metrics["latency_avg"].(float64)),
		fmt.Sprintf("targetAvail=%.1f", targetAvail),
		fmt.Sprintf("actualAvail=%.1f", metrics["availability"].(float64)),
		fmt.Sprintf("levelId=%s", levelID),
	}

	// Add failed requirements as pipe-separated values
	if len(feedback) > 0 {
		failedReqsStr := strings.Join(feedback, "|")
		params = append(params, fmt.Sprintf("failedReqs=%s", failedReqsStr))
	}

	return baseURL + "?" + strings.Join(params, "&")
}
