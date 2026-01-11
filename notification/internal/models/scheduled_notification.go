package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ScheduleStatus represents the status of a scheduled notification
type ScheduleStatus string

const (
	ScheduleStatusPending   ScheduleStatus = "PENDING"
	ScheduleStatusProcessing ScheduleStatus = "PROCESSING"
	ScheduleStatusSent      ScheduleStatus = "SENT"
	ScheduleStatusFailed    ScheduleStatus = "FAILED"
	ScheduleStatusCancelled ScheduleStatus = "CANCELLED"
)

// RecurrenceType represents the type of recurrence
type RecurrenceType string

const (
	RecurrenceTypeOnce    RecurrenceType = "ONCE"
	RecurrenceTypeDaily   RecurrenceType = "DAILY"
	RecurrenceTypeWeekly  RecurrenceType = "WEEKLY"
	RecurrenceTypeMonthly RecurrenceType = "MONTHLY"
)

// TargetType represents the type of targeting for the notification
type TargetType string

const (
	TargetTypeUser     TargetType = "USER"      // Single user
	TargetTypeSegment  TargetType = "SEGMENT"   // User segment
	TargetTypeBroadcast TargetType = "BROADCAST" // All users
	TargetTypeCustom   TargetType = "CUSTOM"    // Custom query
)

// ScheduledNotification represents a scheduled notification
type ScheduledNotification struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Description string                 `bson:"description,omitempty" json:"description,omitempty"`

	// Template reference
	TemplateID  primitive.ObjectID     `bson:"templateId" json:"templateId"`
	TemplateCode string                `bson:"templateCode" json:"templateCode"`

	// Targeting
	TargetType  TargetType             `bson:"targetType" json:"targetType"`
	TargetUserIDs []string             `bson:"targetUserIds,omitempty" json:"targetUserIds,omitempty"`
	TargetSegment string               `bson:"targetSegment,omitempty" json:"targetSegment,omitempty"`
	TargetQuery  map[string]interface{} `bson:"targetQuery,omitempty" json:"targetQuery,omitempty"`

	// Channel
	Channel     NotificationChannel    `bson:"channel" json:"channel"`

	// Content (merged with template)
	Data        map[string]interface{} `bson:"data" json:"data"` // Template variables
	CustomSubject string               `bson:"customSubject,omitempty" json:"customSubject,omitempty"`
	CustomBody    string               `bson:"customBody,omitempty" json:"customBody,omitempty"`

	// Scheduling
	ScheduledAt time.Time              `bson:"scheduledAt" json:"scheduledAt"`
	Timezone    string                 `bson:"timezone" json:"timezone"`

	// Recurrence
	RecurrenceType RecurrenceType      `bson:"recurrenceType" json:"recurrenceType"`
	RecurrenceEnd  *time.Time          `bson:"recurrenceEnd,omitempty" json:"recurrenceEnd,omitempty"`
	DaysOfWeek     []int               `bson:"daysOfWeek,omitempty" json:"daysOfWeek,omitempty"` // 0-6 for weekly
	DayOfMonth     *int                `bson:"dayOfMonth,omitempty" json:"dayOfMonth,omitempty"` // 1-28 for monthly

	// Execution tracking
	Status       ScheduleStatus        `bson:"status" json:"status"`
	LastRunAt    *time.Time            `bson:"lastRunAt,omitempty" json:"lastRunAt,omitempty"`
	NextRunAt    *time.Time            `bson:"nextRunAt,omitempty" json:"nextRunAt,omitempty"`
	RunCount     int                   `bson:"runCount" json:"runCount"`

	// Results tracking
	TotalTargets   int                 `bson:"totalTargets" json:"totalTargets"`
	SentCount      int                 `bson:"sentCount" json:"sentCount"`
	FailedCount    int                 `bson:"failedCount" json:"failedCount"`
	LastError      string              `bson:"lastError,omitempty" json:"lastError,omitempty"`

	// Priority and options
	Priority     int                   `bson:"priority" json:"priority"` // Higher = more urgent
	RespectQuietHours bool             `bson:"respectQuietHours" json:"respectQuietHours"`
	RespectPreferences bool            `bson:"respectPreferences" json:"respectPreferences"`

	// Campaign association
	CampaignID   *primitive.ObjectID   `bson:"campaignId,omitempty" json:"campaignId,omitempty"`

	// Metadata
	Metadata     map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Audit
	CreatedBy    string                `bson:"createdBy" json:"createdBy"`
	UpdatedBy    string                `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
	CancelledBy  string                `bson:"cancelledBy,omitempty" json:"cancelledBy,omitempty"`
	CancelReason string                `bson:"cancelReason,omitempty" json:"cancelReason,omitempty"`

	// Timestamps
	CreatedAt    time.Time             `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time             `bson:"updatedAt" json:"updatedAt"`
}

// NewScheduledNotification creates a new scheduled notification
func NewScheduledNotification(name string, templateID primitive.ObjectID, templateCode string, channel NotificationChannel, scheduledAt time.Time, createdBy string) *ScheduledNotification {
	now := time.Now()
	return &ScheduledNotification{
		ID:                primitive.NewObjectID(),
		Name:              name,
		TemplateID:        templateID,
		TemplateCode:      templateCode,
		TargetType:        TargetTypeBroadcast,
		Channel:           channel,
		Data:              make(map[string]interface{}),
		ScheduledAt:       scheduledAt,
		Timezone:          "Asia/Ho_Chi_Minh",
		RecurrenceType:    RecurrenceTypeOnce,
		Status:            ScheduleStatusPending,
		RunCount:          0,
		Priority:          0,
		RespectQuietHours: true,
		RespectPreferences: true,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// SetTargetUsers sets target users
func (sn *ScheduledNotification) SetTargetUsers(userIDs []string) {
	sn.TargetType = TargetTypeUser
	sn.TargetUserIDs = userIDs
	sn.UpdatedAt = time.Now()
}

// SetTargetSegment sets target segment
func (sn *ScheduledNotification) SetTargetSegment(segment string) {
	sn.TargetType = TargetTypeSegment
	sn.TargetSegment = segment
	sn.UpdatedAt = time.Now()
}

// SetTargetQuery sets custom target query
func (sn *ScheduledNotification) SetTargetQuery(query map[string]interface{}) {
	sn.TargetType = TargetTypeCustom
	sn.TargetQuery = query
	sn.UpdatedAt = time.Now()
}

// SetBroadcast sets broadcast targeting (all users)
func (sn *ScheduledNotification) SetBroadcast() {
	sn.TargetType = TargetTypeBroadcast
	sn.TargetUserIDs = nil
	sn.TargetSegment = ""
	sn.TargetQuery = nil
	sn.UpdatedAt = time.Now()
}

// SetRecurrence sets recurrence configuration
func (sn *ScheduledNotification) SetRecurrence(recurrenceType RecurrenceType, endDate *time.Time) {
	sn.RecurrenceType = recurrenceType
	sn.RecurrenceEnd = endDate
	sn.UpdatedAt = time.Now()
}

// SetWeeklyRecurrence sets weekly recurrence on specific days
func (sn *ScheduledNotification) SetWeeklyRecurrence(daysOfWeek []int, endDate *time.Time) {
	sn.RecurrenceType = RecurrenceTypeWeekly
	sn.DaysOfWeek = daysOfWeek
	sn.RecurrenceEnd = endDate
	sn.UpdatedAt = time.Now()
}

// SetMonthlyRecurrence sets monthly recurrence on specific day
func (sn *ScheduledNotification) SetMonthlyRecurrence(dayOfMonth int, endDate *time.Time) {
	sn.RecurrenceType = RecurrenceTypeMonthly
	sn.DayOfMonth = &dayOfMonth
	sn.RecurrenceEnd = endDate
	sn.UpdatedAt = time.Now()
}

// StartProcessing marks the notification as being processed
func (sn *ScheduledNotification) StartProcessing() {
	sn.Status = ScheduleStatusProcessing
	sn.UpdatedAt = time.Now()
}

// MarkSent marks the notification as sent
func (sn *ScheduledNotification) MarkSent(totalTargets, sentCount, failedCount int) {
	now := time.Now()
	sn.Status = ScheduleStatusSent
	sn.LastRunAt = &now
	sn.RunCount++
	sn.TotalTargets = totalTargets
	sn.SentCount = sentCount
	sn.FailedCount = failedCount
	sn.LastError = ""

	// Calculate next run for recurring notifications
	if sn.RecurrenceType != RecurrenceTypeOnce {
		sn.calculateNextRun()
		if sn.NextRunAt != nil {
			sn.Status = ScheduleStatusPending
		}
	}
	sn.UpdatedAt = now
}

// MarkFailed marks the notification as failed
func (sn *ScheduledNotification) MarkFailed(err string) {
	now := time.Now()
	sn.Status = ScheduleStatusFailed
	sn.LastRunAt = &now
	sn.LastError = err
	sn.UpdatedAt = now
}

// Cancel cancels the scheduled notification
func (sn *ScheduledNotification) Cancel(cancelledBy, reason string) {
	sn.Status = ScheduleStatusCancelled
	sn.CancelledBy = cancelledBy
	sn.CancelReason = reason
	sn.UpdatedAt = time.Now()
}

// calculateNextRun calculates the next run time for recurring notifications
func (sn *ScheduledNotification) calculateNextRun() {
	now := time.Now()
	var nextRun time.Time

	switch sn.RecurrenceType {
	case RecurrenceTypeDaily:
		nextRun = now.AddDate(0, 0, 1)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(),
			sn.ScheduledAt.Hour(), sn.ScheduledAt.Minute(), 0, 0, now.Location())

	case RecurrenceTypeWeekly:
		// Find the next scheduled day of week
		nextRun = now
		for i := 1; i <= 7; i++ {
			nextRun = now.AddDate(0, 0, i)
			weekday := int(nextRun.Weekday())
			for _, day := range sn.DaysOfWeek {
				if day == weekday {
					nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(),
						sn.ScheduledAt.Hour(), sn.ScheduledAt.Minute(), 0, 0, now.Location())
					goto found
				}
			}
		}
	found:

	case RecurrenceTypeMonthly:
		if sn.DayOfMonth != nil {
			nextRun = time.Date(now.Year(), now.Month()+1, *sn.DayOfMonth,
				sn.ScheduledAt.Hour(), sn.ScheduledAt.Minute(), 0, 0, now.Location())
		}

	default:
		sn.NextRunAt = nil
		return
	}

	// Check if past recurrence end date
	if sn.RecurrenceEnd != nil && nextRun.After(*sn.RecurrenceEnd) {
		sn.NextRunAt = nil
		return
	}

	sn.NextRunAt = &nextRun
}

// IsDue checks if the notification is due to be sent
func (sn *ScheduledNotification) IsDue() bool {
	if sn.Status != ScheduleStatusPending {
		return false
	}
	return time.Now().After(sn.ScheduledAt) || time.Now().Equal(sn.ScheduledAt)
}

// Reschedule reschedules the notification
func (sn *ScheduledNotification) Reschedule(newScheduledAt time.Time, updatedBy string) {
	sn.ScheduledAt = newScheduledAt
	sn.Status = ScheduleStatusPending
	sn.UpdatedBy = updatedBy
	sn.UpdatedAt = time.Now()
}

// Clone creates a copy of the notification for retry or duplication
func (sn *ScheduledNotification) Clone(newScheduledAt time.Time, createdBy string) *ScheduledNotification {
	clone := *sn
	clone.ID = primitive.NewObjectID()
	clone.ScheduledAt = newScheduledAt
	clone.Status = ScheduleStatusPending
	clone.RunCount = 0
	clone.SentCount = 0
	clone.FailedCount = 0
	clone.TotalTargets = 0
	clone.LastRunAt = nil
	clone.NextRunAt = nil
	clone.LastError = ""
	clone.CreatedBy = createdBy
	clone.UpdatedBy = ""
	clone.CancelledBy = ""
	clone.CancelReason = ""
	now := time.Now()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	return &clone
}
