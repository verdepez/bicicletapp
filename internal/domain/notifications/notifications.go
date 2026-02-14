// Package notifications provides interfaces for sending notifications
package notifications

import (
	"context"
)

// EmailNotification represents an email to send
type EmailNotification struct {
	To      string
	Subject string
	Body    string
	HTML    bool
}

// SMSNotification represents an SMS to send
type SMSNotification struct {
	Phone   string
	Message string
}

// EmailProvider defines the interface for email providers
type EmailProvider interface {
	Send(ctx context.Context, notification EmailNotification) error
}

// SMSProvider defines the interface for SMS providers
type SMSProvider interface {
	Send(ctx context.Context, notification SMSNotification) error
}

// Notifier combines email and SMS capabilities
type Notifier interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendSMS(ctx context.Context, phone, message string) error
}

// CompositeNotifier implements Notifier using configurable providers
type CompositeNotifier struct {
	email EmailProvider
	sms   SMSProvider
}

// NewCompositeNotifier creates a new composite notifier
func NewCompositeNotifier(email EmailProvider, sms SMSProvider) *CompositeNotifier {
	return &CompositeNotifier{
		email: email,
		sms:   sms,
	}
}

func (n *CompositeNotifier) SendEmail(ctx context.Context, to, subject, body string) error {
	if n.email == nil {
		return nil // Silently skip if not configured
	}
	return n.email.Send(ctx, EmailNotification{
		To:      to,
		Subject: subject,
		Body:    body,
	})
}

func (n *CompositeNotifier) SendSMS(ctx context.Context, phone, message string) error {
	if n.sms == nil {
		return nil // Silently skip if not configured
	}
	return n.sms.Send(ctx, SMSNotification{
		Phone:   phone,
		Message: message,
	})
}

// MockEmailProvider is a no-op email provider for development
type MockEmailProvider struct{}

func (m *MockEmailProvider) Send(ctx context.Context, n EmailNotification) error {
	// Log email in debug mode
	return nil
}

// MockSMSProvider is a no-op SMS provider for development
type MockSMSProvider struct{}

func (m *MockSMSProvider) Send(ctx context.Context, n SMSNotification) error {
	// Log SMS in debug mode
	return nil
}
