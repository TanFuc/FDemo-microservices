package provider

import (
	"context"
	"fmt"
	"log"

	"gopkg.in/gomail.v2"

	"microservices/notification/internal/config"
)

// EmailProvider defines the interface for sending emails
type EmailProvider interface {
	Send(ctx context.Context, recipient, subject, body string) error
}

// SMTPEmailProvider implements EmailProvider using SMTP
type SMTPEmailProvider struct {
	dialer *gomail.Dialer
	from   string
}

// NewSMTPEmailProvider creates a new SMTP email provider
func NewSMTPEmailProvider(cfg config.SMTPConfig) *SMTPEmailProvider {
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Pass)

	return &SMTPEmailProvider{
		dialer: dialer,
		from:   cfg.From,
	}
}

// Send sends an email to the recipient
func (p *SMTPEmailProvider) Send(ctx context.Context, recipient, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", p.from)
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := p.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent to %s: %s", recipient, subject)
	return nil
}

// MockEmailProvider is a mock implementation for testing
type MockEmailProvider struct {
	SentEmails []SentEmail
}

// SentEmail represents a sent email for testing
type SentEmail struct {
	Recipient string
	Subject   string
	Body      string
}

// NewMockEmailProvider creates a new mock email provider
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		SentEmails: make([]SentEmail, 0),
	}
}

// Send records the email for testing
func (p *MockEmailProvider) Send(ctx context.Context, recipient, subject, body string) error {
	p.SentEmails = append(p.SentEmails, SentEmail{
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
	})
	log.Printf("[MOCK] Email sent to %s: %s", recipient, subject)
	return nil
}

// GetSentEmails returns all sent emails
func (p *MockEmailProvider) GetSentEmails() []SentEmail {
	return p.SentEmails
}

// ClearSentEmails clears the sent emails list
func (p *MockEmailProvider) ClearSentEmails() {
	p.SentEmails = make([]SentEmail, 0)
}
