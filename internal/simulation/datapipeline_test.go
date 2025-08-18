package simulation

import (
	"testing"
)

func TestDataPipelineLogic_BasicProcessing(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      100.0,
		"transformation": "map",
	}

	// Create 50 requests (less than batch size)
	requests := make([]*Request, 50)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "DATA", LatencyMS: 0}
	}

	// First tick - should create batch and start processing
	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected data pipeline to be healthy")
	}

	// Should not have output yet (batch is still processing)
	if len(output) != 0 {
		t.Errorf("Expected no output during processing, got %d", len(output))
	}

	// Check that batch was created
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		t.Error("Expected pipeline state to be created")
	}

	if len(state.ProcessingQueue) != 1 {
		t.Errorf("Expected 1 batch in processing queue, got %d", len(state.ProcessingQueue))
	}

	if state.ProcessingQueue[0].RecordCount != 50 {
		t.Errorf("Expected batch with 50 records, got %d", state.ProcessingQueue[0].RecordCount)
	}
}

func TestDataPipelineLogic_BatchCompletion(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      10.0,
		"transformation": "filter", // Fast transformation
	}

	// Create 5 requests
	requests := make([]*Request, 5)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "DATA", LatencyMS: 0}
	}

	// First tick - start processing
	logic.Tick(props, requests, 1)

	// Wait enough ticks for processing to complete
	// Filter transformation should complete quickly
	var output []*Request
	var healthy bool

	for tick := 2; tick <= 10; tick++ {
		output, healthy = logic.Tick(props, []*Request{}, tick)
		if len(output) > 0 {
			break
		}
	}

	if !healthy {
		t.Error("Expected data pipeline to be healthy")
	}

	// Should have output matching input count
	if len(output) != 5 {
		t.Errorf("Expected 5 output records, got %d", len(output))
	}

	// Check output structure
	for _, req := range output {
		if req.Type != "PROCESSED" {
			t.Errorf("Expected PROCESSED type, got %s", req.Type)
		}
		if req.Origin != "data-pipeline" {
			t.Errorf("Expected data-pipeline origin, got %s", req.Origin)
		}
		if len(req.Path) == 0 || req.Path[0] != "pipeline-filter" {
			t.Error("Expected path to indicate filter transformation")
		}
	}
}

func TestDataPipelineLogic_MultipleBatches(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      10.0,
		"transformation": "map",
	}

	// Create 25 requests (should create 3 batches: 10, 10, 5)
	requests := make([]*Request, 25)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "DATA", LatencyMS: 0}
	}

	// First tick - create batches
	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected data pipeline to be healthy")
	}

	if len(output) != 0 {
		t.Error("Expected no immediate output")
	}

	// Check that 3 batches were created
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		t.Error("Expected pipeline state to be created")
	}

	if len(state.ProcessingQueue) != 3 {
		t.Errorf("Expected 3 batches in processing queue, got %d", len(state.ProcessingQueue))
	}

	// Verify batch sizes
	expectedSizes := []int{10, 10, 5}
	for i, batch := range state.ProcessingQueue {
		if batch.RecordCount != expectedSizes[i] {
			t.Errorf("Expected batch %d to have %d records, got %d",
				i, expectedSizes[i], batch.RecordCount)
		}
	}
}

func TestDataPipelineLogic_TransformationComplexity(t *testing.T) {
	logic := DataPipelineLogic{}

	transformations := []string{"filter", "map", "sort", "aggregate", "join"}

	for _, transformation := range transformations {
		t.Run(transformation, func(t *testing.T) {
			complexity := logic.getTransformationComplexity(transformation)

			// Verify relative complexity ordering
			switch transformation {
			case "filter":
				if complexity >= logic.getTransformationComplexity("map") {
					t.Error("Filter should be simpler than map")
				}
			case "join":
				if complexity <= logic.getTransformationComplexity("aggregate") {
					t.Error("Join should be more complex than aggregate")
				}
			case "sort":
				if complexity <= logic.getTransformationComplexity("map") {
					t.Error("Sort should be more complex than map")
				}
			}

			if complexity <= 0 {
				t.Errorf("Expected positive complexity for %s", transformation)
			}
		})
	}
}

func TestDataPipelineLogic_BatchOverhead(t *testing.T) {
	logic := DataPipelineLogic{}

	// Test different overhead levels
	testCases := []struct {
		transformation string
		expectedRange  [2]float64 // [min, max]
	}{
		{"map", [2]float64{0, 100}},    // Low overhead
		{"join", [2]float64{300, 600}}, // High overhead
		{"sort", [2]float64{150, 300}}, // Medium overhead
	}

	for _, tc := range testCases {
		overhead := logic.getBatchOverhead(tc.transformation)

		if overhead < tc.expectedRange[0] || overhead > tc.expectedRange[1] {
			t.Errorf("Expected %s overhead between %.0f-%.0f, got %.0f",
				tc.transformation, tc.expectedRange[0], tc.expectedRange[1], overhead)
		}
	}
}

func TestDataPipelineLogic_ProcessingTime(t *testing.T) {
	logic := DataPipelineLogic{}

	// Test that processing time scales with record count
	smallBatch := logic.calculateProcessingTime(10, "map")
	largeBatch := logic.calculateProcessingTime(100, "map")

	if largeBatch <= smallBatch {
		t.Error("Expected larger batch to take more time")
	}

	// Test that complex transformations take longer
	simpleTime := logic.calculateProcessingTime(50, "filter")
	complexTime := logic.calculateProcessingTime(50, "join")

	if complexTime <= simpleTime {
		t.Error("Expected complex transformation to take longer")
	}

	// Test economies of scale (large batches should be more efficient per record)
	smallPerRecord := float64(smallBatch) / 10.0
	largePerRecord := float64(largeBatch) / 100.0

	if largePerRecord >= smallPerRecord {
		t.Error("Expected economies of scale for larger batches")
	}
}

func TestDataPipelineLogic_HealthCheck(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      10.0,
		"transformation": "join", // Slow transformation
	}

	// Create a large number of requests to test backlog health
	requests := make([]*Request, 300) // 30 batches (above healthy threshold)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + (i % 26))), Type: "DATA", LatencyMS: 0}
	}

	// First tick - should create many batches
	output, healthy := logic.Tick(props, requests, 1)

	// Should be unhealthy due to large backlog
	if healthy {
		t.Error("Expected data pipeline to be unhealthy with large backlog")
	}

	if len(output) != 0 {
		t.Error("Expected no immediate output with slow transformation")
	}

	// Check backlog size
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		t.Error("Expected pipeline state to be created")
	}

	if state.BacklogSize < 200 {
		t.Errorf("Expected large backlog, got %d", state.BacklogSize)
	}
}

func TestDataPipelineLogic_DefaultValues(t *testing.T) {
	logic := DataPipelineLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	requests := []*Request{{ID: "1", Type: "DATA", LatencyMS: 0}}

	output, healthy := logic.Tick(props, requests, 1)

	if !healthy {
		t.Error("Expected pipeline to be healthy with default values")
	}

	if len(output) != 0 {
		t.Error("Expected no immediate output")
	}

	// Should use default batch size and transformation
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		t.Error("Expected pipeline state to be created with defaults")
	}

	if len(state.ProcessingQueue) != 1 {
		t.Error("Expected one batch with default settings")
	}
}

func TestDataPipelineLogic_PipelineStats(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      5.0,
		"transformation": "filter",
	}

	// Initial stats should be empty
	stats := logic.GetPipelineStats(props)
	if stats["completedBatches"] != 0 {
		t.Error("Expected initial completed batches to be 0")
	}

	// Process some data
	requests := make([]*Request, 10)
	for i := range requests {
		requests[i] = &Request{ID: string(rune('1' + i)), Type: "DATA", LatencyMS: 0}
	}

	logic.Tick(props, requests, 1)

	// Check stats after processing
	stats = logic.GetPipelineStats(props)
	if stats["queuedBatches"] != 2 {
		t.Errorf("Expected 2 queued batches, got %v", stats["queuedBatches"])
	}

	if stats["backlogSize"] != 10 {
		t.Errorf("Expected backlog size of 10, got %v", stats["backlogSize"])
	}
}

func TestDataPipelineLogic_ContinuousProcessing(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      5.0,
		"transformation": "map",
	}

	// Process multiple waves of data
	totalOutput := 0

	for wave := 0; wave < 3; wave++ {
		requests := make([]*Request, 5)
		for i := range requests {
			requests[i] = &Request{ID: string(rune('A' + wave*5 + i)), Type: "DATA", LatencyMS: 0}
		}

		// Process each wave
		for tick := wave*10 + 1; tick <= wave*10+5; tick++ {
			var output []*Request
			if tick == wave*10+1 {
				output, _ = logic.Tick(props, requests, tick)
			} else {
				output, _ = logic.Tick(props, []*Request{}, tick)
			}
			totalOutput += len(output)
		}
	}

	// Should have processed all data eventually
	if totalOutput != 15 {
		t.Errorf("Expected 15 total output records, got %d", totalOutput)
	}

	// Check final stats
	stats := logic.GetPipelineStats(props)
	if stats["totalRecords"] != 15 {
		t.Errorf("Expected 15 total records processed, got %v", stats["totalRecords"])
	}
}

func TestDataPipelineLogic_EmptyQueue(t *testing.T) {
	logic := DataPipelineLogic{}

	props := map[string]any{
		"batchSize":      10.0,
		"transformation": "map",
	}

	// Process empty queue
	output, healthy := logic.Tick(props, []*Request{}, 1)

	if !healthy {
		t.Error("Expected pipeline to be healthy with empty queue")
	}

	if len(output) != 0 {
		t.Error("Expected no output with empty queue")
	}

	// State should be initialized but empty
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		t.Error("Expected pipeline state to be initialized")
	}

	if len(state.ProcessingQueue) != 0 {
		t.Error("Expected empty processing queue")
	}
}
