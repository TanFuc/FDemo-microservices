package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TemplateType represents the type of notification template
type TemplateType string

const (
	TemplateTypeEmail TemplateType = "EMAIL"
	TemplateTypePush  TemplateType = "PUSH"
	TemplateTypeSMS   TemplateType = "SMS"
	TemplateTypeInApp TemplateType = "IN_APP"
)

// TemplateStatus represents the status of a notification template
type TemplateStatus string

const (
	TemplateStatusActive   TemplateStatus = "ACTIVE"
	TemplateStatusInactive TemplateStatus = "INACTIVE"
	TemplateStatusDraft    TemplateStatus = "DRAFT"
)

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Code        string                 `bson:"code" json:"code"`       // Unique identifier like ORDER_CREATED, PAYMENT_SUCCESS
	Name        string                 `bson:"name" json:"name"`
	Description string                 `bson:"description,omitempty" json:"description,omitempty"`
	Type        TemplateType           `bson:"type" json:"type"`
	Category    string                 `bson:"category" json:"category"` // ORDER, PAYMENT, PROMOTION, SYSTEM

	// Content templates (supports i18n)
	Subject     map[string]string      `bson:"subject" json:"subject"`     // {en: "...", vi: "..."}
	Body        map[string]string      `bson:"body" json:"body"`           // HTML or text body
	PlainText   map[string]string      `bson:"plainText,omitempty" json:"plainText,omitempty"` // Plain text fallback

	// Push notification specific
	PushTitle   map[string]string      `bson:"pushTitle,omitempty" json:"pushTitle,omitempty"`
	PushBody    map[string]string      `bson:"pushBody,omitempty" json:"pushBody,omitempty"`
	PushImage   string                 `bson:"pushImage,omitempty" json:"pushImage,omitempty"`
	PushAction  string                 `bson:"pushAction,omitempty" json:"pushAction,omitempty"` // Deep link

	// Email specific
	FromName    string                 `bson:"fromName,omitempty" json:"fromName,omitempty"`
	FromEmail   string                 `bson:"fromEmail,omitempty" json:"fromEmail,omitempty"`
	ReplyTo     string                 `bson:"replyTo,omitempty" json:"replyTo,omitempty"`

	// Template variables
	Variables   []TemplateVariable     `bson:"variables" json:"variables"`

	// Metadata
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Status
	Status      TemplateStatus         `bson:"status" json:"status"`
	Version     int                    `bson:"version" json:"version"`

	// Audit
	CreatedBy   string                 `bson:"createdBy,omitempty" json:"createdBy,omitempty"`
	UpdatedBy   string                 `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}

// TemplateVariable represents a variable in a template
type TemplateVariable struct {
	Key         string `bson:"key" json:"key"`
	Description string `bson:"description" json:"description"`
	Type        string `bson:"type" json:"type"` // string, number, date, currency
	Required    bool   `bson:"required" json:"required"`
	Default     string `bson:"default,omitempty" json:"default,omitempty"`
}

// NewNotificationTemplate creates a new notification template
func NewNotificationTemplate(code, name string, templateType TemplateType, category string) *NotificationTemplate {
	now := time.Now()
	return &NotificationTemplate{
		ID:        primitive.NewObjectID(),
		Code:      code,
		Name:      name,
		Type:      templateType,
		Category:  category,
		Subject:   make(map[string]string),
		Body:      make(map[string]string),
		Variables: make([]TemplateVariable, 0),
		Status:    TemplateStatusDraft,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetSubject sets the subject for a language
func (nt *NotificationTemplate) SetSubject(lang, subject string) {
	nt.Subject[lang] = subject
	nt.UpdatedAt = time.Now()
}

// SetBody sets the body for a language
func (nt *NotificationTemplate) SetBody(lang, body string) {
	nt.Body[lang] = body
	nt.UpdatedAt = time.Now()
}

// AddVariable adds a template variable
func (nt *NotificationTemplate) AddVariable(key, description, varType string, required bool) {
	nt.Variables = append(nt.Variables, TemplateVariable{
		Key:         key,
		Description: description,
		Type:        varType,
		Required:    required,
	})
	nt.UpdatedAt = time.Now()
}

// Activate activates the template
func (nt *NotificationTemplate) Activate() {
	nt.Status = TemplateStatusActive
	nt.UpdatedAt = time.Now()
}

// Deactivate deactivates the template
func (nt *NotificationTemplate) Deactivate() {
	nt.Status = TemplateStatusInactive
	nt.UpdatedAt = time.Now()
}

// IncrementVersion increments the template version
func (nt *NotificationTemplate) IncrementVersion() {
	nt.Version++
	nt.UpdatedAt = time.Now()
}

// GetSubject returns the subject for the given language, falling back to default
func (nt *NotificationTemplate) GetSubject(lang string) string {
	if subject, ok := nt.Subject[lang]; ok {
		return subject
	}
	if subject, ok := nt.Subject["en"]; ok {
		return subject
	}
	return ""
}

// GetBody returns the body for the given language, falling back to default
func (nt *NotificationTemplate) GetBody(lang string) string {
	if body, ok := nt.Body[lang]; ok {
		return body
	}
	if body, ok := nt.Body["en"]; ok {
		return body
	}
	return ""
}
