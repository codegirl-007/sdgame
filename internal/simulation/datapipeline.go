package simulation

type DataPipelineLogic struct{}

type DataBatch struct {
	ID           string
	RecordCount  int
	Timestamp    int
	ProcessingMS int
}

type PipelineState struct {
	ProcessingQueue  []DataBatch
	CompletedBatches int
	TotalRecords     int
	BacklogSize      int
}

func (d DataPipelineLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract data pipeline properties
	batchSize := int(AsFloat64(props["batchSize"]))
	if batchSize == 0 {
		batchSize = 500 // default batch size
	}

	transformation := AsString(props["transformation"])
	if transformation == "" {
		transformation = "map" // default transformation
	}

	// Get pipeline state from props (persistent state)
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		state = PipelineState{
			ProcessingQueue:  []DataBatch{},
			CompletedBatches: 0,
			TotalRecords:     0,
			BacklogSize:      0,
		}
	}

	currentTime := tick * 100 // Convert tick to milliseconds

	// Convert incoming requests to data batches
	if len(queue) > 0 {
		// Group requests into batches
		batches := d.createBatches(queue, batchSize, currentTime, transformation)

		// Add batches to processing queue
		state.ProcessingQueue = append(state.ProcessingQueue, batches...)
		state.BacklogSize += len(queue)
	}

	// Process batches that are ready (completed their processing time)
	output := []*Request{}
	remainingBatches := []DataBatch{}

	for _, batch := range state.ProcessingQueue {
		if currentTime >= batch.Timestamp+batch.ProcessingMS {
			// Batch is complete - create output requests
			for i := 0; i < batch.RecordCount; i++ {
				processedReq := &Request{
					ID:        batch.ID + "-record-" + string(rune('0'+i)),
					Timestamp: batch.Timestamp,
					LatencyMS: batch.ProcessingMS,
					Origin:    "data-pipeline",
					Type:      "PROCESSED",
					Path:      []string{"pipeline-" + transformation},
				}
				output = append(output, processedReq)
			}

			state.CompletedBatches++
			state.TotalRecords += batch.RecordCount
		} else {
			// Batch still processing
			remainingBatches = append(remainingBatches, batch)
		}
	}

	state.ProcessingQueue = remainingBatches
	state.BacklogSize = len(remainingBatches) * batchSize

	// Update persistent state
	props["_pipelineState"] = state

	// Health check: pipeline is healthy if backlog is not too large
	maxBacklogSize := batchSize * 20 // Allow up to 20 batches in backlog
	healthy := state.BacklogSize < maxBacklogSize

	return output, healthy
}

// createBatches groups requests into batches and calculates processing time
func (d DataPipelineLogic) createBatches(requests []*Request, batchSize int, timestamp int, transformation string) []DataBatch {
	batches := []DataBatch{}

	for i := 0; i < len(requests); i += batchSize {
		end := i + batchSize
		if end > len(requests) {
			end = len(requests)
		}

		recordCount := end - i
		processingTime := d.calculateProcessingTime(recordCount, transformation)

		batch := DataBatch{
			ID:           "batch-" + string(rune('A'+len(batches))),
			RecordCount:  recordCount,
			Timestamp:    timestamp,
			ProcessingMS: processingTime,
		}

		batches = append(batches, batch)
	}

	return batches
}

// calculateProcessingTime determines how long a batch takes to process based on transformation type
func (d DataPipelineLogic) calculateProcessingTime(recordCount int, transformation string) int {
	// Base processing time per record
	baseTimePerRecord := d.getTransformationComplexity(transformation)

	// Total time scales with record count but with some economies of scale
	totalTime := float64(recordCount) * baseTimePerRecord

	// Add batch overhead (setup, teardown, I/O)
	batchOverhead := d.getBatchOverhead(transformation)
	totalTime += batchOverhead

	// Apply economies of scale for larger batches (slightly more efficient)
	if recordCount > 100 {
		scaleFactor := 0.9 // 10% efficiency gain for large batches
		totalTime *= scaleFactor
	}

	return int(totalTime)
}

// getTransformationComplexity returns base processing time per record in milliseconds
func (d DataPipelineLogic) getTransformationComplexity(transformation string) float64 {
	switch transformation {
	case "map":
		return 1.0 // Simple field mapping
	case "filter":
		return 0.5 // Just evaluate conditions
	case "sort":
		return 3.0 // Sorting requires more compute
	case "aggregate":
		return 2.0 // Grouping and calculating aggregates
	case "join":
		return 5.0 // Most expensive - joining with other datasets
	case "deduplicate":
		return 2.5 // Hash-based deduplication
	case "validate":
		return 1.5 // Data validation and cleaning
	case "enrich":
		return 4.0 // Enriching with external data
	case "compress":
		return 1.2 // Compression processing
	case "encrypt":
		return 2.0 // Encryption overhead
	default:
		return 1.0 // Default to simple transformation
	}
}

// getBatchOverhead returns fixed overhead time per batch in milliseconds
func (d DataPipelineLogic) getBatchOverhead(transformation string) float64 {
	switch transformation {
	case "map", "filter", "validate":
		return 50.0 // Low overhead for simple operations
	case "sort", "aggregate", "deduplicate":
		return 200.0 // Medium overhead for complex operations
	case "join", "enrich":
		return 500.0 // High overhead for operations requiring external data
	case "compress", "encrypt":
		return 100.0 // Medium overhead for I/O operations
	default:
		return 100.0 // Default overhead
	}
}

// Helper function to get pipeline statistics
func (d DataPipelineLogic) GetPipelineStats(props map[string]any) map[string]interface{} {
	state, ok := props["_pipelineState"].(PipelineState)
	if !ok {
		return map[string]interface{}{
			"completedBatches": 0,
			"totalRecords":     0,
			"backlogSize":      0,
			"queuedBatches":    0,
		}
	}

	return map[string]interface{}{
		"completedBatches": state.CompletedBatches,
		"totalRecords":     state.TotalRecords,
		"backlogSize":      state.BacklogSize,
		"queuedBatches":    len(state.ProcessingQueue),
	}
}
