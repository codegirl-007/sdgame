package simulation

import (
	"testing"
)

func TestMessageQueueLogic_BasicProcessing(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    10,
		"retentionSeconds": 3600, // 1 hour
		"processingRate":   5,
	}

	// Add some messages to the queue
	reqs := []*Request{
		{ID: "msg1", Type: "SEND", LatencyMS: 0, Timestamp: 100},
		{ID: "msg2", Type: "SEND", LatencyMS: 0, Timestamp: 100},
		{ID: "msg3", Type: "SEND", LatencyMS: 0, Timestamp: 100},
	}

	output, healthy := mq.Tick(props, reqs, 1)

	if !healthy {
		t.Errorf("Message queue should be healthy")
	}

	// No immediate output since messages are queued first
	if len(output) != 0 {
		t.Errorf("Expected 0 immediate output (messages queued), got %d", len(output))
	}

	// Check that messages are in the queue
	messageQueue, ok := props["_messageQueue"].([]QueuedMessage)
	if !ok {
		t.Errorf("Expected message queue to be initialized")
	}

	if len(messageQueue) != 3 {
		t.Errorf("Expected 3 messages in queue, got %d", len(messageQueue))
	}

	// Process the queue (no new incoming messages)
	output2, _ := mq.Tick(props, []*Request{}, 2)

	// Should process up to processingRate (5) messages
	if len(output2) != 3 {
		t.Errorf("Expected 3 processed messages, got %d", len(output2))
	}

	// Queue should now be empty
	messageQueue2, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue2) != 0 {
		t.Errorf("Expected empty queue after processing, got %d messages", len(messageQueue2))
	}

	// Check output message properties
	for _, msg := range output2 {
		if msg.LatencyMS != 2 {
			t.Errorf("Expected 2ms processing latency, got %dms", msg.LatencyMS)
		}
		if msg.Type != "PROCESS" {
			t.Errorf("Expected PROCESS type, got %s", msg.Type)
		}
	}
}

func TestMessageQueueLogic_CapacityLimit(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    2, // Small capacity
		"retentionSeconds": 3600,
		"processingRate":   1,
	}

	// Add more messages than capacity
	reqs := []*Request{
		{ID: "msg1", Type: "SEND", LatencyMS: 0},
		{ID: "msg2", Type: "SEND", LatencyMS: 0},
		{ID: "msg3", Type: "SEND", LatencyMS: 0}, // This should be dropped
	}

	output, healthy := mq.Tick(props, reqs, 1)

	// Queue should be healthy (can still process messages)
	if !healthy {
		t.Errorf("Queue should be healthy (can still process)")
	}

	// Should have no immediate output (messages queued)
	if len(output) != 0 {
		t.Errorf("Expected 0 immediate output, got %d", len(output))
	}

	// Check queue size
	messageQueue, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue) != 2 {
		t.Errorf("Expected 2 messages in queue (capacity limit), got %d", len(messageQueue))
	}

	// Add another message when queue is full
	reqs2 := []*Request{{ID: "msg4", Type: "SEND", LatencyMS: 0}}
	output2, healthy2 := mq.Tick(props, reqs2, 2)

	// Queue should still be healthy (can process messages)
	if !healthy2 {
		t.Errorf("Queue should remain healthy (can still process)")
	}

	// Should have 1 processed message (processingRate = 1)
	if len(output2) != 1 {
		t.Errorf("Expected 1 processed message, got %d", len(output2))
	}

	// Queue should have 2 messages (started with 2, processed 1 leaving 1, added 1 new since space available)
	messageQueue2, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue2) != 2 {
		t.Errorf("Expected 2 messages in queue (1 remaining + 1 new), got %d", len(messageQueue2))
	}
}

func TestMessageQueueLogic_ProcessingRate(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    100,
		"retentionSeconds": 3600,
		"processingRate":   3, // Process 3 messages per tick
	}

	// Add 10 messages
	reqs := []*Request{}
	for i := 0; i < 10; i++ {
		reqs = append(reqs, &Request{ID: "msg" + string(rune(i+'0')), Type: "SEND"})
	}

	// First tick: queue all messages
	mq.Tick(props, reqs, 1)

	// Second tick: process at rate limit
	output, _ := mq.Tick(props, []*Request{}, 2)

	if len(output) != 3 {
		t.Errorf("Expected 3 processed messages (rate limit), got %d", len(output))
	}

	// Check remaining queue size
	messageQueue, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue) != 7 {
		t.Errorf("Expected 7 messages remaining in queue, got %d", len(messageQueue))
	}

	// Third tick: process 3 more
	output2, _ := mq.Tick(props, []*Request{}, 3)

	if len(output2) != 3 {
		t.Errorf("Expected 3 more processed messages, got %d", len(output2))
	}

	// Check remaining queue size
	messageQueue2, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue2) != 4 {
		t.Errorf("Expected 4 messages remaining in queue, got %d", len(messageQueue2))
	}
}

func TestMessageQueueLogic_MessageRetention(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    100,
		"retentionSeconds": 1, // 1 second retention
		"processingRate":   0, // Don't process messages, just test retention
	}

	// Add messages at tick 1
	reqs := []*Request{
		{ID: "msg1", Type: "SEND", Timestamp: 100},
		{ID: "msg2", Type: "SEND", Timestamp: 100},
	}

	mq.Tick(props, reqs, 1)

	// Check messages are queued
	messageQueue, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue) != 2 {
		t.Errorf("Expected 2 messages in queue, got %d", len(messageQueue))
	}

	// Tick at time that should expire messages (tick 20 = 2000ms, retention = 1000ms)
	output, _ := mq.Tick(props, []*Request{}, 20)

	// Messages should be expired and removed
	messageQueue2, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue2) != 0 {
		t.Errorf("Expected messages to be expired and removed, got %d", len(messageQueue2))
	}

	// No output since processingRate = 0
	if len(output) != 0 {
		t.Errorf("Expected no output with processingRate=0, got %d", len(output))
	}
}

func TestMessageQueueLogic_FIFOOrdering(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    10,
		"retentionSeconds": 3600,
		"processingRate":   2,
	}

	// Add messages in order
	reqs := []*Request{
		{ID: "first", Type: "SEND"},
		{ID: "second", Type: "SEND"},
		{ID: "third", Type: "SEND"},
	}

	mq.Tick(props, reqs, 1)

	// Process 2 messages
	output, _ := mq.Tick(props, []*Request{}, 2)

	if len(output) != 2 {
		t.Errorf("Expected 2 processed messages, got %d", len(output))
	}

	// Check FIFO order
	if output[0].ID != "first" {
		t.Errorf("Expected first message to be 'first', got '%s'", output[0].ID)
	}

	if output[1].ID != "second" {
		t.Errorf("Expected second message to be 'second', got '%s'", output[1].ID)
	}

	// Process remaining message
	output2, _ := mq.Tick(props, []*Request{}, 3)

	if len(output2) != 1 {
		t.Errorf("Expected 1 remaining message, got %d", len(output2))
	}

	if output2[0].ID != "third" {
		t.Errorf("Expected remaining message to be 'third', got '%s'", output2[0].ID)
	}
}

func TestMessageQueueLogic_DefaultValues(t *testing.T) {
	mq := MessageQueueLogic{}

	// Empty props should use defaults
	props := map[string]any{}

	reqs := []*Request{{ID: "msg1", Type: "SEND"}}
	output, healthy := mq.Tick(props, reqs, 1)

	if !healthy {
		t.Errorf("Queue should be healthy with default values")
	}

	// Should queue the message (no immediate output)
	if len(output) != 0 {
		t.Errorf("Expected message to be queued (0 output), got %d", len(output))
	}

	// Check that message was queued with defaults
	messageQueue, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue) != 1 {
		t.Errorf("Expected 1 message queued with defaults, got %d", len(messageQueue))
	}

	// Process with defaults (should process up to default rate)
	output2, _ := mq.Tick(props, []*Request{}, 2)

	if len(output2) != 1 {
		t.Errorf("Expected 1 processed message with defaults, got %d", len(output2))
	}
}

func TestMessageQueueLogic_ContinuousFlow(t *testing.T) {
	mq := MessageQueueLogic{}

	props := map[string]any{
		"queueCapacity":    5,
		"retentionSeconds": 3600,
		"processingRate":   2,
	}

	// Tick 1: Add 3 messages
	reqs1 := []*Request{
		{ID: "msg1", Type: "SEND"},
		{ID: "msg2", Type: "SEND"},
		{ID: "msg3", Type: "SEND"},
	}
	output1, _ := mq.Tick(props, reqs1, 1)

	// Should queue all 3 messages
	if len(output1) != 0 {
		t.Errorf("Expected 0 output on first tick, got %d", len(output1))
	}

	// Tick 2: Add 2 more messages, process 2
	reqs2 := []*Request{
		{ID: "msg4", Type: "SEND"},
		{ID: "msg5", Type: "SEND"},
	}
	output2, _ := mq.Tick(props, reqs2, 2)

	// Should process 2 messages
	if len(output2) != 2 {
		t.Errorf("Expected 2 processed messages, got %d", len(output2))
	}

	// Should have 3 messages in queue (3 remaining + 2 new - 2 processed)
	messageQueue, _ := props["_messageQueue"].([]QueuedMessage)
	if len(messageQueue) != 3 {
		t.Errorf("Expected 3 messages in queue, got %d", len(messageQueue))
	}

	// Check processing order
	if output2[0].ID != "msg1" || output2[1].ID != "msg2" {
		t.Errorf("Expected FIFO processing order, got %s, %s", output2[0].ID, output2[1].ID)
	}
}
