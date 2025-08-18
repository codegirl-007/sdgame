package simulation

import (
	"encoding/json"
	"os"
	"testing"

	"systemdesigngame/internal/design"
)

// TODO: Make this better
func TestSimpleChainSimulation(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "a", Type: "webserver", Props: map[string]any{"rpsCapacity": 1, "baseLatencyMs": 10}},
			{ID: "b", Type: "webserver", Props: map[string]any{"rpsCapacity": 1, "baseLatencyMs": 10}},
		},
		Connections: []design.Connection{
			{Source: "a", Target: "b"},
		},
	}

	engine := NewEngineFromDesign(d, 100)

	engine.Nodes["a"].Queue = append(engine.Nodes["a"].Queue, &Request{
		ID:        "req-1",
		Origin:    "a",
		Type:      "GET",
		Timestamp: 0,
		Path:      []string{"a"},
	})

	snaps := engine.Run(2, 100)

	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}

	if len(snaps[0].Emitted["a"]) != 1 {
		t.Errorf("expected a to emit 1 request at tick 0")
	}
	if snaps[0].QueueSizes["b"] != 0 {
		t.Errorf("expected b's queue to be 0 after tick 0 (not yet processed)")
	}

	if len(snaps[1].Emitted["b"]) != 1 {
		t.Errorf("expected b to emit 1 request at tick 1")
	}
}

func TestSingleTickRouting(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "a", Type: "webserver", Props: map[string]any{"rpsCapacity": 1.0, "baseLatencyMs": 10.0}},
			{ID: "b", Type: "webserver", Props: map[string]any{"rpsCapacity": 1.0, "baseLatencyMs": 10.0}},
		},
		Connections: []design.Connection{
			{Source: "a", Target: "b"},
		},
	}

	engine := NewEngineFromDesign(d, 100)

	engine.Nodes["a"].Queue = append(engine.Nodes["a"].Queue, &Request{
		ID:        "req-1",
		Origin:    "a",
		Type:      "GET",
		Timestamp: 0,
		Path:      []string{"a"},
	})

	snaps := engine.Run(1, 100)

	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	if len(snaps[0].Emitted["a"]) != 1 {
		t.Errorf("expected a to emit 1 request, got %d", len(snaps[0].Emitted["a"]))
	}

	if len(engine.Nodes["b"].Queue) != 1 {
		t.Errorf("expected b to have 1 request queued for next tick, got %d", len(engine.Nodes["b"].Queue))
	}
}

func TestHighRPSSimulation(t *testing.T) {
	d := design.Design{
		Nodes: []design.Node{
			{ID: "entry", Type: "webserver", Props: map[string]any{"rpsCapacity": 5000, "baseLatencyMs": 1}},
		},
		Connections: []design.Connection{},
	}

	engine := NewEngineFromDesign(d, 100)
	engine.EntryNode = "entry"
	engine.RPS = 100000

	snaps := engine.Run(10, 100)

	totalEmitted := 0
	for _, snap := range snaps {
		totalEmitted += len(snap.Emitted["entry"])
	}

	expected := 10 * 5000 // capacity-limited output
	if totalEmitted != expected {
		t.Errorf("expected %d total emitted requests, got %d", expected, totalEmitted)
	}
}

func TestDatabaseIntegration(t *testing.T) {
	design := design.Design{
		Nodes: []design.Node{
			{
				ID:   "webserver",
				Type: "webserver",
				Props: map[string]interface{}{
					"rpsCapacity": 10,
				},
			},
			{
				ID:   "database",
				Type: "database",
				Props: map[string]interface{}{
					"replication":   2,
					"maxRPS":        100,
					"baseLatencyMs": 20,
				},
			},
		},
		Connections: []design.Connection{
			{
				Source: "webserver",
				Target: "database",
			},
		},
	}

	engine := NewEngineFromDesign(design, 100)
	engine.RPS = 5
	engine.EntryNode = "webserver"

	snapshots := engine.Run(3, 100)

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify database node exists and is healthy
	if len(engine.Nodes) != 2 {
		t.Errorf("Expected 2 nodes (webserver + database), got %d", len(engine.Nodes))
	}

	dbNode, exists := engine.Nodes["database"]
	if !exists {
		t.Errorf("Database node should exist in simulation")
	}

	if !dbNode.Alive {
		t.Errorf("Database node should be alive")
	}

	if dbNode.Type != "database" {
		t.Errorf("Expected database type, got %s", dbNode.Type)
	}
}

func TestCacheIntegration(t *testing.T) {
	design := design.Design{
		Nodes: []design.Node{
			{
				ID:   "webserver",
				Type: "webserver",
				Props: map[string]interface{}{
					"rpsCapacity": 10,
				},
			},
			{
				ID:   "cache",
				Type: "cache",
				Props: map[string]interface{}{
					"cacheTTL":       5000,
					"maxEntries":     50,
					"evictionPolicy": "LRU",
				},
			},
			{
				ID:   "database",
				Type: "database",
				Props: map[string]interface{}{
					"replication":   1,
					"maxRPS":        100,
					"baseLatencyMs": 15,
				},
			},
		},
		Connections: []design.Connection{
			{
				Source: "webserver",
				Target: "cache",
			},
			{
				Source: "cache",
				Target: "database",
			},
		},
	}

	engine := NewEngineFromDesign(design, 100)
	engine.RPS = 5
	engine.EntryNode = "webserver"

	snapshots := engine.Run(5, 100)

	if len(snapshots) != 5 {
		t.Errorf("Expected 5 snapshots, got %d", len(snapshots))
	}

	// Verify all nodes exist and are healthy
	if len(engine.Nodes) != 3 {
		t.Errorf("Expected 3 nodes (webserver + cache + database), got %d", len(engine.Nodes))
	}

	cacheNode, exists := engine.Nodes["cache"]
	if !exists {
		t.Errorf("Cache node should exist in simulation")
	}

	if !cacheNode.Alive {
		t.Errorf("Cache node should be alive")
	}

	if cacheNode.Type != "cache" {
		t.Errorf("Expected cache type, got %s", cacheNode.Type)
	}

	// Verify cache has internal state
	cacheData, ok := cacheNode.Props["_cacheData"]
	if !ok {
		t.Errorf("Cache should have internal _cacheData state")
	}

	// Cache data should be a map
	if _, ok := cacheData.(map[string]*CacheEntry); !ok {
		t.Errorf("Cache data should be map[string]*CacheEntry")
	}
}

func TestMessageQueueIntegration(t *testing.T) {
	design := design.Design{
		Nodes: []design.Node{
			{
				ID:   "producer",
				Type: "webserver",
				Props: map[string]interface{}{
					"rpsCapacity": 10,
				},
			},
			{
				ID:   "messagequeue",
				Type: "messageQueue",
				Props: map[string]interface{}{
					"queueCapacity":    50,
					"retentionSeconds": 3600,
					"processingRate":   5,
				},
			},
			{
				ID:   "consumer",
				Type: "webserver",
				Props: map[string]interface{}{
					"rpsCapacity": 20,
				},
			},
		},
		Connections: []design.Connection{
			{
				Source: "producer",
				Target: "messagequeue",
			},
			{
				Source: "messagequeue",
				Target: "consumer",
			},
		},
	}

	engine := NewEngineFromDesign(design, 100)
	engine.RPS = 3
	engine.EntryNode = "producer"

	snapshots := engine.Run(5, 100)

	if len(snapshots) != 5 {
		t.Errorf("Expected 5 snapshots, got %d", len(snapshots))
	}

	// Verify all nodes exist and are healthy
	if len(engine.Nodes) != 3 {
		t.Errorf("Expected 3 nodes (producer + queue + consumer), got %d", len(engine.Nodes))
	}

	queueNode, exists := engine.Nodes["messagequeue"]
	if !exists {
		t.Errorf("Message queue node should exist in simulation")
	}

	if !queueNode.Alive {
		t.Errorf("Message queue node should be alive")
	}

	if queueNode.Type != "messageQueue" {
		t.Errorf("Expected messageQueue type, got %s", queueNode.Type)
	}

	// Verify queue has internal state
	messageQueue, ok := queueNode.Props["_messageQueue"]
	if !ok {
		t.Errorf("Message queue should have internal _messageQueue state")
	}

	// Message queue should be a slice
	if _, ok := messageQueue.([]QueuedMessage); !ok {
		t.Errorf("Message queue should be []QueuedMessage")
	}
}

func TestMicroserviceIntegration(t *testing.T) {
	// Load the microservice design
	designData, err := os.ReadFile("testdata/microservice_design.json")
	if err != nil {
		t.Fatalf("Failed to read microservice design: %v", err)
	}

	var d design.Design
	if err := json.Unmarshal(designData, &d); err != nil {
		t.Fatalf("Failed to unmarshal design: %v", err)
	}

	// Create engine
	engine := NewEngineFromDesign(d, 100)
	if engine == nil {
		t.Fatalf("Failed to create engine from microservice design")
	}

	// Set up simulation parameters
	engine.RPS = 30
	engine.EntryNode = "webserver-1"

	// Run simulation for 5 ticks
	snapshots := engine.Run(5, 100)

	if len(snapshots) != 5 {
		t.Errorf("Expected 5 snapshots, got %d", len(snapshots))
	}

	// Verify microservice nodes exist and are configured correctly
	userService, exists := engine.Nodes["microservice-1"]
	if !exists {
		t.Errorf("User service microservice node should exist")
	}

	if !userService.Alive {
		t.Errorf("User service should be alive")
	}

	if userService.Type != "microservice" {
		t.Errorf("Expected microservice type, got %s", userService.Type)
	}

	orderService, exists := engine.Nodes["microservice-2"]
	if !exists {
		t.Errorf("Order service microservice node should exist")
	}

	if !orderService.Alive {
		t.Errorf("Order service should be alive")
	}

	// Verify auto-scaling properties are preserved
	userServiceInstanceCount := userService.Props["instanceCount"]
	if userServiceInstanceCount == nil {
		t.Errorf("User service should have instanceCount property")
	}

	// Verify different scaling strategies
	userScalingStrategy := userService.Props["scalingStrategy"]
	if userScalingStrategy != "auto" {
		t.Errorf("Expected auto scaling strategy for user service, got %v", userScalingStrategy)
	}

	orderScalingStrategy := orderService.Props["scalingStrategy"]
	if orderScalingStrategy != "manual" {
		t.Errorf("Expected manual scaling strategy for order service, got %v", orderScalingStrategy)
	}

	// Verify resource configurations
	userCPU := userService.Props["cpu"]
	if userCPU != 4.0 {
		t.Errorf("Expected user service to have 4 CPU cores, got %v", userCPU)
	}

	orderRAM := orderService.Props["ramGb"]
	if orderRAM != 4.0 {
		t.Errorf("Expected order service to have 4GB RAM, got %v", orderRAM)
	}

	// Check that microservices processed requests through the simulation
	lastSnapshot := snapshots[len(snapshots)-1]
	if len(lastSnapshot.QueueSizes) == 0 {
		t.Errorf("Expected queue sizes to be tracked in snapshots")
	}

	// Verify load balancer connected to microservices
	loadBalancer, exists := engine.Nodes["lb-1"]
	if !exists {
		t.Errorf("Load balancer should exist")
	}

	if !loadBalancer.Alive {
		t.Errorf("Load balancer should be alive")
	}

	// Verify database connection exists
	database, exists := engine.Nodes["db-1"]
	if !exists {
		t.Errorf("Database should exist")
	}

	if !database.Alive {
		t.Errorf("Database should be alive")
	}
}

func TestMonitoringIntegration(t *testing.T) {
	// Load the monitoring design
	designData, err := os.ReadFile("testdata/monitoring_design.json")
	if err != nil {
		t.Fatalf("Failed to read monitoring design: %v", err)
	}

	var d design.Design
	if err := json.Unmarshal(designData, &d); err != nil {
		t.Fatalf("Failed to unmarshal design: %v", err)
	}

	// Create engine
	engine := NewEngineFromDesign(d, 100)
	if engine == nil {
		t.Fatalf("Failed to create engine from monitoring design")
	}

	// Set up simulation parameters
	engine.RPS = 20
	engine.EntryNode = "webserver-1"

	// Run simulation for 10 ticks to allow metrics collection
	snapshots := engine.Run(10, 100)

	if len(snapshots) != 10 {
		t.Errorf("Expected 10 snapshots, got %d", len(snapshots))
	}

	// Verify monitoring nodes exist and are configured correctly
	monitor1, exists := engine.Nodes["monitor-1"]
	if !exists {
		t.Errorf("Latency monitor node should exist")
	}

	if !monitor1.Alive {
		t.Errorf("Latency monitor should be alive")
	}

	if monitor1.Type != "monitoring/alerting" {
		t.Errorf("Expected monitoring/alerting type, got %s", monitor1.Type)
	}

	monitor2, exists := engine.Nodes["monitor-2"]
	if !exists {
		t.Errorf("Error rate monitor node should exist")
	}

	if !monitor2.Alive {
		t.Errorf("Error rate monitor should be alive")
	}

	// Verify monitoring properties are preserved
	tool1 := monitor1.Props["tool"]
	if tool1 != "Prometheus" {
		t.Errorf("Expected Prometheus tool for monitor-1, got %v", tool1)
	}

	tool2 := monitor2.Props["tool"]
	if tool2 != "Datadog" {
		t.Errorf("Expected Datadog tool for monitor-2, got %v", tool2)
	}

	alertMetric1 := monitor1.Props["alertMetric"]
	if alertMetric1 != "latency" {
		t.Errorf("Expected latency alert metric for monitor-1, got %v", alertMetric1)
	}

	alertMetric2 := monitor2.Props["alertMetric"]
	if alertMetric2 != "error_rate" {
		t.Errorf("Expected error_rate alert metric for monitor-2, got %v", alertMetric2)
	}

	// Check that metrics were collected during simulation
	metrics1, ok := monitor1.Props["_metrics"]
	if !ok {
		t.Errorf("Expected monitor-1 to have collected metrics")
	}

	if metrics1 == nil {
		t.Errorf("Expected monitor-1 metrics to be non-nil")
	}

	// Check alert count tracking
	alertCount1, ok := monitor1.Props["_alertCount"]
	if !ok {
		t.Errorf("Expected monitor-1 to track alert count")
	}

	if alertCount1 == nil {
		t.Errorf("Expected monitor-1 alert count to be tracked")
	}

	// Verify other components in the chain
	webserver, exists := engine.Nodes["webserver-1"]
	if !exists {
		t.Errorf("Web server should exist")
	}

	if !webserver.Alive {
		t.Errorf("Web server should be alive")
	}

	loadBalancer, exists := engine.Nodes["lb-1"]
	if !exists {
		t.Errorf("Load balancer should exist")
	}

	if !loadBalancer.Alive {
		t.Errorf("Load balancer should be alive")
	}

	// Verify microservices
	userService, exists := engine.Nodes["microservice-1"]
	if !exists {
		t.Errorf("User service should exist")
	}

	if !userService.Alive {
		t.Errorf("User service should be alive")
	}

	orderService, exists := engine.Nodes["microservice-2"]
	if !exists {
		t.Errorf("Order service should exist")
	}

	if !orderService.Alive {
		t.Errorf("Order service should be alive")
	}

	// Verify database
	database, exists := engine.Nodes["db-1"]
	if !exists {
		t.Errorf("Database should exist")
	}

	if !database.Alive {
		t.Errorf("Database should be alive")
	}

	// Check that requests flowed through the monitoring chain
	lastSnapshot := snapshots[len(snapshots)-1]
	if len(lastSnapshot.QueueSizes) == 0 {
		t.Errorf("Expected queue sizes to be tracked in snapshots")
	}

	// Verify monitoring nodes processed requests
	if lastSnapshot.NodeHealth["monitor-1"] != true {
		t.Errorf("Expected monitor-1 to be healthy in final snapshot")
	}

	if lastSnapshot.NodeHealth["monitor-2"] != true {
		t.Errorf("Expected monitor-2 to be healthy in final snapshot")
	}
}

func TestThirdPartyServiceIntegration(t *testing.T) {
	// Load the third party service design
	designData, err := os.ReadFile("testdata/thirdpartyservice_design.json")
	if err != nil {
		t.Fatalf("Failed to read third party service design: %v", err)
	}

	var d design.Design
	if err := json.Unmarshal(designData, &d); err != nil {
		t.Fatalf("Failed to unmarshal design: %v", err)
	}

	// Create engine
	engine := NewEngineFromDesign(d, 100)
	if engine == nil {
		t.Fatalf("Failed to create engine from third party service design")
	}

	// Set up simulation parameters
	engine.RPS = 10 // Lower RPS to reduce chance of random failures affecting health
	engine.EntryNode = "webserver-1"

	// Run simulation for 5 ticks (shorter run to reduce random failure impact)
	snapshots := engine.Run(5, 100)

	if len(snapshots) != 5 {
		t.Errorf("Expected 5 snapshots, got %d", len(snapshots))
	}

	// Verify third party service nodes exist and are configured correctly
	stripeService, exists := engine.Nodes["stripe-service"]
	if !exists {
		t.Errorf("Stripe service node should exist")
	}

	if stripeService.Type != "third party service" {
		t.Errorf("Expected third party service type, got %s", stripeService.Type)
	}

	twilioService, exists := engine.Nodes["twilio-service"]
	if !exists {
		t.Errorf("Twilio service node should exist")
	}

	sendgridService, exists := engine.Nodes["sendgrid-service"]
	if !exists {
		t.Errorf("SendGrid service node should exist")
	}

	slackService, exists := engine.Nodes["slack-service"]
	if !exists {
		t.Errorf("Slack service node should exist")
	}

	// Note: We don't check if services are alive here because the random failure
	// simulation can cause services to go down, which is realistic behavior

	// Verify provider configurations are preserved
	stripeProvider := stripeService.Props["provider"]
	if stripeProvider != "Stripe" {
		t.Errorf("Expected Stripe provider, got %v", stripeProvider)
	}

	twilioProvider := twilioService.Props["provider"]
	if twilioProvider != "Twilio" {
		t.Errorf("Expected Twilio provider, got %v", twilioProvider)
	}

	sendgridProvider := sendgridService.Props["provider"]
	if sendgridProvider != "SendGrid" {
		t.Errorf("Expected SendGrid provider, got %v", sendgridProvider)
	}

	slackProvider := slackService.Props["provider"]
	if slackProvider != "Slack" {
		t.Errorf("Expected Slack provider, got %v", slackProvider)
	}

	// Verify latency configurations
	stripeLatency := stripeService.Props["latency"]
	if stripeLatency != 180.0 {
		t.Errorf("Expected Stripe latency 180, got %v", stripeLatency)
	}

	twilioLatency := twilioService.Props["latency"]
	if twilioLatency != 250.0 {
		t.Errorf("Expected Twilio latency 250, got %v", twilioLatency)
	}

	// Check that service status was initialized and tracked
	stripeStatus, ok := stripeService.Props["_serviceStatus"]
	if !ok {
		t.Errorf("Expected Stripe service status to be tracked")
	}

	if stripeStatus == nil {
		t.Errorf("Expected Stripe service status to be non-nil")
	}

	// Verify other components in the chain
	webserver, exists := engine.Nodes["webserver-1"]
	if !exists {
		t.Errorf("Web server should exist")
	}

	if !webserver.Alive {
		t.Errorf("Web server should be alive")
	}

	// Verify microservices
	paymentService, exists := engine.Nodes["microservice-1"]
	if !exists {
		t.Errorf("Payment service should exist")
	}

	if !paymentService.Alive {
		t.Errorf("Payment service should be alive")
	}

	notificationService, exists := engine.Nodes["microservice-2"]
	if !exists {
		t.Errorf("Notification service should exist")
	}

	if !notificationService.Alive {
		t.Errorf("Notification service should be alive")
	}

	// Verify monitoring and database
	monitor, exists := engine.Nodes["monitor-1"]
	if !exists {
		t.Errorf("Monitor should exist")
	}

	if !monitor.Alive {
		t.Errorf("Monitor should be alive")
	}

	database, exists := engine.Nodes["db-1"]
	if !exists {
		t.Errorf("Database should exist")
	}

	if !database.Alive {
		t.Errorf("Database should be alive")
	}

	// Check that requests flowed through the third party services
	lastSnapshot := snapshots[len(snapshots)-1]
	if len(lastSnapshot.QueueSizes) == 0 {
		t.Errorf("Expected queue sizes to be tracked in snapshots")
	}

	// Verify third party services are being tracked in snapshots
	// Note: We don't assert health status because random failures are realistic
	_, stripeHealthTracked := lastSnapshot.NodeHealth["stripe-service"]
	if !stripeHealthTracked {
		t.Errorf("Expected Stripe service health to be tracked in snapshots")
	}

	_, twilioHealthTracked := lastSnapshot.NodeHealth["twilio-service"]
	if !twilioHealthTracked {
		t.Errorf("Expected Twilio service health to be tracked in snapshots")
	}

	_, sendgridHealthTracked := lastSnapshot.NodeHealth["sendgrid-service"]
	if !sendgridHealthTracked {
		t.Errorf("Expected SendGrid service health to be tracked in snapshots")
	}

	_, slackHealthTracked := lastSnapshot.NodeHealth["slack-service"]
	if !slackHealthTracked {
		t.Errorf("Expected Slack service health to be tracked in snapshots")
	}
}

func TestDataPipelineIntegration(t *testing.T) {
	// Load the data pipeline design
	designData, err := os.ReadFile("testdata/datapipeline_design.json")
	if err != nil {
		t.Fatalf("Failed to read data pipeline design: %v", err)
	}

	var d design.Design
	if err := json.Unmarshal(designData, &d); err != nil {
		t.Fatalf("Failed to unmarshal design: %v", err)
	}

	// Create engine
	engine := NewEngineFromDesign(d, 100)
	if engine == nil {
		t.Fatalf("Failed to create engine from data pipeline design")
	}

	// Set up simulation parameters
	engine.RPS = 20
	engine.EntryNode = "data-source"

	// Run simulation for 10 ticks to test data pipeline processing
	snapshots := engine.Run(10, 100)

	if len(snapshots) != 10 {
		t.Errorf("Expected 10 snapshots, got %d", len(snapshots))
	}

	// Verify data pipeline nodes exist and are configured correctly
	etlPipeline1, exists := engine.Nodes["etl-pipeline-1"]
	if !exists {
		t.Errorf("ETL Pipeline 1 node should exist")
	}

	if etlPipeline1.Type != "data pipeline" {
		t.Errorf("Expected data pipeline type, got %s", etlPipeline1.Type)
	}

	etlPipeline2, exists := engine.Nodes["etl-pipeline-2"]
	if !exists {
		t.Errorf("ETL Pipeline 2 node should exist")
	}

	mlPipeline, exists := engine.Nodes["ml-pipeline"]
	if !exists {
		t.Errorf("ML Pipeline node should exist")
	}

	analyticsPipeline, exists := engine.Nodes["analytics-pipeline"]
	if !exists {
		t.Errorf("Analytics Pipeline node should exist")
	}

	compressionPipeline, exists := engine.Nodes["compression-pipeline"]
	if !exists {
		t.Errorf("Compression Pipeline node should exist")
	}

	// Verify pipeline configurations are preserved
	etl1BatchSize := etlPipeline1.Props["batchSize"]
	if etl1BatchSize != 100.0 {
		t.Errorf("Expected ETL Pipeline 1 batch size 100, got %v", etl1BatchSize)
	}

	etl1Transformation := etlPipeline1.Props["transformation"]
	if etl1Transformation != "validate" {
		t.Errorf("Expected validate transformation, got %v", etl1Transformation)
	}

	etl2BatchSize := etlPipeline2.Props["batchSize"]
	if etl2BatchSize != 50.0 {
		t.Errorf("Expected ETL Pipeline 2 batch size 50, got %v", etl2BatchSize)
	}

	etl2Transformation := etlPipeline2.Props["transformation"]
	if etl2Transformation != "aggregate" {
		t.Errorf("Expected aggregate transformation, got %v", etl2Transformation)
	}

	mlTransformation := mlPipeline.Props["transformation"]
	if mlTransformation != "enrich" {
		t.Errorf("Expected enrich transformation for ML pipeline, got %v", mlTransformation)
	}

	analyticsTransformation := analyticsPipeline.Props["transformation"]
	if analyticsTransformation != "join" {
		t.Errorf("Expected join transformation for analytics pipeline, got %v", analyticsTransformation)
	}

	compressionTransformation := compressionPipeline.Props["transformation"]
	if compressionTransformation != "compress" {
		t.Errorf("Expected compress transformation, got %v", compressionTransformation)
	}

	// Check that pipeline state was initialized and tracked
	etl1State, ok := etlPipeline1.Props["_pipelineState"]
	if !ok {
		t.Errorf("Expected ETL Pipeline 1 to have pipeline state")
	}

	if etl1State == nil {
		t.Errorf("Expected ETL Pipeline 1 state to be non-nil")
	}

	// Verify other components in the data flow
	dataSource, exists := engine.Nodes["data-source"]
	if !exists {
		t.Errorf("Data source should exist")
	}

	if !dataSource.Alive {
		t.Errorf("Data source should be alive")
	}

	rawDataQueue, exists := engine.Nodes["raw-data-queue"]
	if !exists {
		t.Errorf("Raw data queue should exist")
	}

	if !rawDataQueue.Alive {
		t.Errorf("Raw data queue should be alive")
	}

	// Verify storage components
	cache, exists := engine.Nodes["cache-1"]
	if !exists {
		t.Errorf("Feature cache should exist")
	}

	if !cache.Alive {
		t.Errorf("Feature cache should be alive")
	}

	dataWarehouse, exists := engine.Nodes["data-warehouse"]
	if !exists {
		t.Errorf("Data warehouse should exist")
	}

	if !dataWarehouse.Alive {
		t.Errorf("Data warehouse should be alive")
	}

	// Verify monitoring
	monitor, exists := engine.Nodes["monitoring-1"]
	if !exists {
		t.Errorf("Pipeline monitor should exist")
	}

	if !monitor.Alive {
		t.Errorf("Pipeline monitor should be alive")
	}

	// Check that data pipelines are being tracked in snapshots
	lastSnapshot := snapshots[len(snapshots)-1]
	if len(lastSnapshot.QueueSizes) == 0 {
		t.Errorf("Expected queue sizes to be tracked in snapshots")
	}

	// Verify data pipeline health is tracked
	_, etl1HealthTracked := lastSnapshot.NodeHealth["etl-pipeline-1"]
	if !etl1HealthTracked {
		t.Errorf("Expected ETL Pipeline 1 health to be tracked in snapshots")
	}

	_, etl2HealthTracked := lastSnapshot.NodeHealth["etl-pipeline-2"]
	if !etl2HealthTracked {
		t.Errorf("Expected ETL Pipeline 2 health to be tracked in snapshots")
	}

	_, mlHealthTracked := lastSnapshot.NodeHealth["ml-pipeline"]
	if !mlHealthTracked {
		t.Errorf("Expected ML Pipeline health to be tracked in snapshots")
	}

	_, analyticsHealthTracked := lastSnapshot.NodeHealth["analytics-pipeline"]
	if !analyticsHealthTracked {
		t.Errorf("Expected Analytics Pipeline health to be tracked in snapshots")
	}

	_, compressionHealthTracked := lastSnapshot.NodeHealth["compression-pipeline"]
	if !compressionHealthTracked {
		t.Errorf("Expected Compression Pipeline health to be tracked in snapshots")
	}

	// Verify the data flow chain exists (all components are connected)
	// This ensures the integration test validates the complete data processing architecture
	totalNodes := len(engine.Nodes)
	expectedNodes := 10 // From the design JSON
	if totalNodes != expectedNodes {
		t.Errorf("Expected %d total nodes in data pipeline architecture, got %d", expectedNodes, totalNodes)
	}
}
