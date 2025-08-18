package simulation

type MessageQueueLogic struct{}

type QueuedMessage struct {
	RequestID   string
	Timestamp   int
	MessageData string
	RetryCount  int
}

func (mq MessageQueueLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// Extract message queue properties
	queueCapacity := int(AsFloat64(props["queueCapacity"]))
	if queueCapacity == 0 {
		queueCapacity = 1000 // default capacity
	}

	retentionSeconds := int(AsFloat64(props["retentionSeconds"]))
	if retentionSeconds == 0 {
		retentionSeconds = 86400 // default 24 hours in seconds
	}

	// Processing rate (messages per tick)
	processingRate := int(AsFloat64(props["processingRate"]))
	if processingRate == 0 {
		processingRate = 100 // default 100 messages per tick
	}

	// Current timestamp for this tick
	currentTime := tick * 100 // assuming 100ms per tick

	// Initialize queue storage in props
	messageQueue, ok := props["_messageQueue"].([]QueuedMessage)
	if !ok {
		messageQueue = []QueuedMessage{}
	}

	// Clean up expired messages based on retention policy
	messageQueue = mq.cleanExpiredMessages(messageQueue, currentTime, retentionSeconds*1000)

	// First, process existing messages from the queue (FIFO order)
	output := []*Request{}
	messagesToProcess := len(messageQueue)
	if messagesToProcess > processingRate {
		messagesToProcess = processingRate
	}

	for i := 0; i < messagesToProcess; i++ {
		if len(messageQueue) == 0 {
			break
		}

		// Dequeue message (FIFO - take from front)
		message := messageQueue[0]
		messageQueue = messageQueue[1:]

		// Create request for downstream processing
		processedReq := &Request{
			ID:        message.RequestID,
			Timestamp: message.Timestamp,
			LatencyMS: 2, // Small latency for queue processing
			Origin:    "message-queue",
			Type:      "PROCESS",
			Path:      []string{"queued-message"},
		}

		output = append(output, processedReq)
	}

	// Then, add incoming requests to the queue for next tick
	for _, req := range queue {
		// Check if queue is at capacity
		if len(messageQueue) >= queueCapacity {
			// Queue full - message is dropped (or could implement backpressure)
			// For now, we'll drop the message and add latency penalty
			reqCopy := *req
			reqCopy.LatencyMS += 1000 // High latency penalty for dropped messages
			reqCopy.Path = append(reqCopy.Path, "queue-full-dropped")
			// Don't add to output as message was dropped
			continue
		}

		// Add message to queue
		message := QueuedMessage{
			RequestID:   req.ID,
			Timestamp:   currentTime,
			MessageData: "message-payload", // In real system, this would be the actual message
			RetryCount:  0,
		}
		messageQueue = append(messageQueue, message)
	}

	// Update queue storage in props
	props["_messageQueue"] = messageQueue

	// Queue is healthy if not at capacity or if we can still process messages
	// Queue becomes unhealthy only when completely full AND we can't process anything
	healthy := len(messageQueue) < queueCapacity || processingRate > 0

	return output, healthy
}

func (mq MessageQueueLogic) cleanExpiredMessages(messageQueue []QueuedMessage, currentTime, retentionMs int) []QueuedMessage {
	cleaned := []QueuedMessage{}

	for _, message := range messageQueue {
		if (currentTime - message.Timestamp) <= retentionMs {
			cleaned = append(cleaned, message)
		}
		// Expired messages are dropped
	}

	return cleaned
}
