package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"microservices/notification/internal/infrastructure"
	"microservices/notification/internal/models"
	"microservices/notification/internal/provider"
)

// EmailConsumer consumes email notification jobs from RabbitMQ
type EmailConsumer struct {
	rabbitmq       *infrastructure.RabbitMQ
	mongodb        *infrastructure.MongoDB
	emailProvider  provider.EmailProvider
	templateEngine *provider.TemplateEngine
	stopCh         chan struct{}
	doneCh         chan struct{}
}

// NewEmailConsumer creates a new email consumer
func NewEmailConsumer(
	rmq *infrastructure.RabbitMQ,
	mongo *infrastructure.MongoDB,
	emailProvider provider.EmailProvider,
	templateEngine *provider.TemplateEngine,
) *EmailConsumer {
	return &EmailConsumer{
		rabbitmq:       rmq,
		mongodb:        mongo,
		emailProvider:  emailProvider,
		templateEngine: templateEngine,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
}

// Start starts consuming messages from the email queue
func (c *EmailConsumer) Start(ctx context.Context) error {
	msgs, err := c.rabbitmq.Consume(infrastructure.EmailQueue)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("Email consumer started, listening on queue: %s", infrastructure.EmailQueue)

	go c.consume(ctx, msgs)
	return nil
}

// consume processes messages from the queue
func (c *EmailConsumer) consume(ctx context.Context, msgs <-chan amqp.Delivery) {
	defer close(c.doneCh)

	for {
		select {
		case <-c.stopCh:
			log.Println("Email consumer stopping...")
			return
		case <-ctx.Done():
			log.Println("Email consumer context cancelled")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("Message channel closed")
				return
			}
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage processes a single message
func (c *EmailConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	log.Printf("Processing email notification: %s", string(msg.Body))

	// Parse the notification job
	var job models.NotificationJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("Failed to unmarshal notification job: %v", err)
		// Reject without requeue for invalid messages
		msg.Nack(false, false)
		return
	}

	// Create pending log entry
	notifLog := &models.NotificationLog{
		UserID:  job.UserID,
		Type:    job.Type,
		Status:  models.NotificationStatusPending,
		Payload: job,
	}

	createdLog, err := c.mongodb.CreateNotificationLog(ctx, notifLog)
	if err != nil {
		log.Printf("Failed to create notification log: %v", err)
		// Requeue the message
		msg.Nack(false, true)
		return
	}

	// Render the template
	subject := c.getSubject(job.Template, job.Data)
	body, err := c.templateEngine.RenderTemplate(job.Template, job.Data)
	if err != nil {
		log.Printf("Failed to render template %s: %v", job.Template, err)
		c.updateLogStatus(ctx, createdLog.ID, models.NotificationStatusFailed, err.Error())
		// Don't requeue - template errors won't fix themselves
		msg.Nack(false, false)
		return
	}

	// Send the email
	if err := c.emailProvider.Send(ctx, job.Recipient, subject, body); err != nil {
		log.Printf("Failed to send email to %s: %v", job.Recipient, err)
		c.updateLogStatus(ctx, createdLog.ID, models.NotificationStatusFailed, err.Error())
		// Nack without requeue - message goes to DLQ
		msg.Nack(false, false)
		return
	}

	// Update log status to sent
	c.updateLogStatus(ctx, createdLog.ID, models.NotificationStatusSent, "")

	// Acknowledge the message
	if err := msg.Ack(false); err != nil {
		log.Printf("Failed to ACK message: %v", err)
	}

	log.Printf("Email sent successfully to %s", job.Recipient)
}

// getSubject returns the email subject based on template name
func (c *EmailConsumer) getSubject(template string, data map[string]interface{}) string {
	subjects := map[string]string{
		"order_confirmation": "Order Confirmation",
		"order_shipped":      "Your Order Has Shipped",
		"password_reset":     "Password Reset Request",
		"welcome":            "Welcome to Our Platform",
		"payment_received":   "Payment Received",
		"account_activated":  "Account Activated",
	}

	if subject, ok := subjects[template]; ok {
		// Add order ID to subject if available
		if orderID, ok := data["order_id"].(string); ok && template == "order_confirmation" {
			return fmt.Sprintf("%s - #%s", subject, orderID)
		}
		return subject
	}

	return "Notification"
}

// updateLogStatus updates the notification log status
func (c *EmailConsumer) updateLogStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus, errorMsg string) {
	if !id.IsZero() {
		log.Printf("Updating notification log %s to status %s", id.Hex(), status)
		if err := c.mongodb.UpdateNotificationLogStatus(ctx, id, status, errorMsg); err != nil {
			log.Printf("Failed to update notification log: %v", err)
		}
	}
}

// Stop stops the consumer
func (c *EmailConsumer) Stop() {
	close(c.stopCh)
	<-c.doneCh
	log.Println("Email consumer stopped")
}
