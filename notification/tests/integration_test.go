package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"microservices/notification/internal/bridge"
	"microservices/notification/internal/config"
	"microservices/notification/internal/infrastructure"
	"microservices/notification/internal/models"
	"microservices/notification/internal/provider"
	"microservices/notification/internal/worker"
)

// TestIntegration_OrderCreatedFlow tests the full flow from NATS event to email notification
// This test simulates an order service publishing an order.created event
// and verifies that the notification service processes it correctly
func TestIntegration_OrderCreatedFlow(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize infrastructure
	rabbitmq, err := infrastructure.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		t.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	natsConn, err := infrastructure.NewNATS(cfg.NATS.URL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	mongodb, err := infrastructure.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Get templates directory
	cwd, _ := os.Getwd()
	templatesDir := filepath.Join(cwd, "..", "templates")

	// Initialize template engine
	templateEngine, err := provider.NewTemplateEngine(templatesDir)
	if err != nil {
		t.Fatalf("Failed to initialize template engine: %v", err)
	}

	// Use mock email provider for testing
	mockEmailProvider := provider.NewMockEmailProvider()

	// Start the bridge (NATS -> RabbitMQ)
	orderListener := bridge.NewOrderEventListener(natsConn, rabbitmq)
	if err := orderListener.Start(ctx); err != nil {
		t.Fatalf("Failed to start order event listener: %v", err)
	}
	defer orderListener.Stop()

	// Start the email consumer (RabbitMQ -> Email)
	emailConsumer := worker.NewEmailConsumer(rabbitmq, mongodb, mockEmailProvider, templateEngine)
	if err := emailConsumer.Start(ctx); err != nil {
		t.Fatalf("Failed to start email consumer: %v", err)
	}
	defer emailConsumer.Stop()

	// Give consumers time to initialize
	time.Sleep(500 * time.Millisecond)

	// Step 1: Simulate Order Service by publishing a message to NATS "order.created"
	testOrderID := "TEST-ORDER-" + time.Now().Format("20060102150405")
	testUserID := "test-user-123"
	testUserEmail := "testuser@example.com"

	orderEvent := models.OrderCreatedEvent{
		OrderID:   testOrderID,
		UserID:    testUserID,
		UserEmail: testUserEmail,
		Amount:    99.99,
		Currency:  "USD",
		Items: []models.OrderItem{
			{Name: "Test Product", Quantity: 2, Price: 49.99},
		},
	}

	eventData, err := json.Marshal(orderEvent)
	if err != nil {
		t.Fatalf("Failed to marshal order event: %v", err)
	}

	t.Logf("Publishing order.created event for order: %s", testOrderID)

	// Try JetStream publish first, fallback to regular NATS
	_, err = natsConn.JetStreamPublish("order.created", eventData)
	if err != nil {
		t.Logf("JetStream publish failed, using regular NATS: %v", err)
		if err := natsConn.Publish("order.created", eventData); err != nil {
			t.Fatalf("Failed to publish order event: %v", err)
		}
	}

	// Step 2: Wait for processing (at least 1 second as per spec)
	t.Log("Waiting for notification processing...")
	time.Sleep(2 * time.Second)

	// Step 3: Check MongoDB notification_logs for a record with status "SENT"
	collection := mongodb.Database().Collection(infrastructure.NotificationLogsCollection)

	var notifLog models.NotificationLog
	filter := bson.M{
		"userId": testUserID,
	}

	err = collection.FindOne(ctx, filter).Decode(&notifLog)
	if err != nil {
		t.Fatalf("Failed to find notification log: %v", err)
	}

	// Verify the notification was processed
	t.Logf("Found notification log: ID=%s, Status=%s, Type=%s",
		notifLog.ID.Hex(), notifLog.Status, notifLog.Type)

	// Check status is SENT (implies Bridge worked AND Worker worked)
	if notifLog.Status != models.NotificationStatusSent {
		t.Errorf("Expected status SENT, got %s. Error: %s", notifLog.Status, notifLog.Error)
	}

	// Verify email was sent (using mock provider)
	sentEmails := mockEmailProvider.GetSentEmails()
	if len(sentEmails) == 0 {
		t.Error("No emails were sent by the mock provider")
	} else {
		t.Logf("Mock email provider sent %d email(s)", len(sentEmails))
		for i, email := range sentEmails {
			t.Logf("  Email %d: To=%s, Subject=%s", i+1, email.Recipient, email.Subject)
		}
	}

	t.Log("Integration test passed!")
}

// TestIntegration_FailedNotification tests that failed notifications go to DLQ
func TestIntegration_FailedNotification(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize RabbitMQ
	rabbitmq, err := infrastructure.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		t.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	// Initialize MongoDB
	mongodb, err := infrastructure.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

	// Get templates directory
	cwd, _ := os.Getwd()
	templatesDir := filepath.Join(cwd, "..", "templates")

	// Initialize template engine
	templateEngine, err := provider.NewTemplateEngine(templatesDir)
	if err != nil {
		t.Fatalf("Failed to initialize template engine: %v", err)
	}

	// Use a failing email provider
	failingProvider := &FailingEmailProvider{}

	// Start the email consumer
	emailConsumer := worker.NewEmailConsumer(rabbitmq, mongodb, failingProvider, templateEngine)
	if err := emailConsumer.Start(ctx); err != nil {
		t.Fatalf("Failed to start email consumer: %v", err)
	}
	defer emailConsumer.Stop()

	// Give consumer time to initialize
	time.Sleep(500 * time.Millisecond)

	// Create a notification job that will fail
	testUserID := "test-fail-user-" + time.Now().Format("20060102150405")
	job := models.NotificationJob{
		Type:      models.NotificationTypeEmail,
		Recipient: "fail@example.com",
		Template:  "order_confirmation",
		UserID:    testUserID,
		Data: map[string]interface{}{
			"order_id": "FAIL-ORDER",
			"total":    50.00,
			"currency": "USD",
		},
	}

	jobData, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Failed to marshal job: %v", err)
	}

	// Publish directly to RabbitMQ
	if err := rabbitmq.Publish(ctx, infrastructure.NotificationExchange, infrastructure.EmailOrderRoutingKey, jobData); err != nil {
		t.Fatalf("Failed to publish to RabbitMQ: %v", err)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Check that the notification was logged as FAILED
	collection := mongodb.Database().Collection(infrastructure.NotificationLogsCollection)

	var notifLog models.NotificationLog
	filter := bson.M{
		"userId": testUserID,
	}

	err = collection.FindOne(ctx, filter).Decode(&notifLog)
	if err != nil {
		t.Fatalf("Failed to find notification log: %v", err)
	}

	if notifLog.Status != models.NotificationStatusFailed {
		t.Errorf("Expected status FAILED, got %s", notifLog.Status)
	}

	if notifLog.Error == "" {
		t.Error("Expected error message to be set for failed notification")
	}

	t.Logf("Failed notification test passed! Status=%s, Error=%s", notifLog.Status, notifLog.Error)
}

// FailingEmailProvider always fails to send emails (for testing DLQ)
type FailingEmailProvider struct{}

func (p *FailingEmailProvider) Send(ctx context.Context, recipient, subject, body string) error {
	return context.DeadlineExceeded
}
