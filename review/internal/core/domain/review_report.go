package domain

import "time"

// ReportReason represents the reason for reporting a review
type ReportReason string

const (
	ReportReasonSpam           ReportReason = "SPAM"
	ReportReasonInappropriate  ReportReason = "INAPPROPRIATE"
	ReportReasonFakeReview     ReportReason = "FAKE_REVIEW"
	ReportReasonOffensive      ReportReason = "OFFENSIVE"
	ReportReasonMisleading     ReportReason = "MISLEADING"
	ReportReasonWrongProduct   ReportReason = "WRONG_PRODUCT"
	ReportReasonPrivacyViolation ReportReason = "PRIVACY_VIOLATION"
	ReportReasonOther          ReportReason = "OTHER"
)

// ReportStatus represents the status of a review report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "PENDING"
	ReportStatusReviewing ReportStatus = "REVIEWING"
	ReportStatusResolved  ReportStatus = "RESOLVED"
	ReportStatusRejected  ReportStatus = "REJECTED"
	ReportStatusEscalated ReportStatus = "ESCALATED"
)

// ReportResolution represents the resolution of a report
type ReportResolution string

const (
	ResolutionNoAction    ReportResolution = "NO_ACTION"
	ResolutionWarning     ReportResolution = "WARNING"
	ResolutionHidden      ReportResolution = "HIDDEN"
	ResolutionRemoved     ReportResolution = "REMOVED"
	ResolutionUserBanned  ReportResolution = "USER_BANNED"
)

// ReviewReport represents a report on a review
type ReviewReport struct {
	ID             string           `json:"id" bson:"_id"`
	ReviewID       string           `json:"reviewId" bson:"reviewId"`
	ReporterID     string           `json:"reporterId" bson:"reporterId"`

	// Report details
	Reason         ReportReason     `json:"reason" bson:"reason"`
	Description    string           `json:"description,omitempty" bson:"description,omitempty"`
	Evidence       []string         `json:"evidence,omitempty" bson:"evidence,omitempty"` // URLs to screenshots, etc.

	// Review snapshot at time of report
	ReviewSnapshot *ReviewSnapshot  `json:"reviewSnapshot,omitempty" bson:"reviewSnapshot,omitempty"`

	// Status and resolution
	Status         ReportStatus     `json:"status" bson:"status"`
	Resolution     ReportResolution `json:"resolution,omitempty" bson:"resolution,omitempty"`
	ResolutionNote string           `json:"resolutionNote,omitempty" bson:"resolutionNote,omitempty"`

	// Processing
	AssignedTo     string           `json:"assignedTo,omitempty" bson:"assignedTo,omitempty"`
	ProcessedBy    string           `json:"processedBy,omitempty" bson:"processedBy,omitempty"`
	ProcessedAt    *time.Time       `json:"processedAt,omitempty" bson:"processedAt,omitempty"`

	// Priority
	Priority       int              `json:"priority" bson:"priority"` // Higher = more urgent

	// Metadata
	IPAddress      string           `json:"ipAddress,omitempty" bson:"ipAddress,omitempty"`
	UserAgent      string           `json:"userAgent,omitempty" bson:"userAgent,omitempty"`

	// Timestamps
	CreatedAt      time.Time        `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt" bson:"updatedAt"`
}

// ReviewSnapshot represents a snapshot of the review at time of report
type ReviewSnapshot struct {
	UserID    string   `json:"userId" bson:"userId"`
	UserName  string   `json:"userName" bson:"userName"`
	ProductID string   `json:"productId" bson:"productId"`
	Rating    int      `json:"rating" bson:"rating"`
	Content   string   `json:"content" bson:"content"`
	Images    []string `json:"images,omitempty" bson:"images,omitempty"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

// NewReviewReport creates a new review report
func NewReviewReport(reviewID, reporterID string, reason ReportReason, description string) *ReviewReport {
	now := time.Now()
	return &ReviewReport{
		ReviewID:    reviewID,
		ReporterID:  reporterID,
		Reason:      reason,
		Description: description,
		Evidence:    make([]string, 0),
		Status:      ReportStatusPending,
		Priority:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetReviewSnapshot sets the review snapshot
func (rr *ReviewReport) SetReviewSnapshot(review *Review) {
	rr.ReviewSnapshot = &ReviewSnapshot{
		UserID:    review.UserID,
		UserName:  review.UserName,
		ProductID: review.ProductID,
		Rating:    review.Rating,
		Content:   review.Content,
		Images:    review.Images,
		CreatedAt: review.CreatedAt,
	}
	rr.UpdatedAt = time.Now()
}

// AddEvidence adds evidence to the report
func (rr *ReviewReport) AddEvidence(url string) {
	rr.Evidence = append(rr.Evidence, url)
	rr.UpdatedAt = time.Now()
}

// SetPriority sets the priority
func (rr *ReviewReport) SetPriority(priority int) {
	rr.Priority = priority
	rr.UpdatedAt = time.Now()
}

// Assign assigns the report to a moderator
func (rr *ReviewReport) Assign(moderatorID string) {
	rr.AssignedTo = moderatorID
	rr.Status = ReportStatusReviewing
	rr.UpdatedAt = time.Now()
}

// Resolve resolves the report
func (rr *ReviewReport) Resolve(processedBy string, resolution ReportResolution, note string) {
	now := time.Now()
	rr.Status = ReportStatusResolved
	rr.Resolution = resolution
	rr.ResolutionNote = note
	rr.ProcessedBy = processedBy
	rr.ProcessedAt = &now
	rr.UpdatedAt = now
}

// Reject rejects the report
func (rr *ReviewReport) Reject(processedBy string, note string) {
	now := time.Now()
	rr.Status = ReportStatusRejected
	rr.Resolution = ResolutionNoAction
	rr.ResolutionNote = note
	rr.ProcessedBy = processedBy
	rr.ProcessedAt = &now
	rr.UpdatedAt = now
}

// Escalate escalates the report
func (rr *ReviewReport) Escalate(note string) {
	rr.Status = ReportStatusEscalated
	rr.ResolutionNote = note
	rr.Priority = 10 // High priority
	rr.UpdatedAt = time.Now()
}

// IsPending checks if the report is pending
func (rr *ReviewReport) IsPending() bool {
	return rr.Status == ReportStatusPending
}

// IsResolved checks if the report is resolved
func (rr *ReviewReport) IsResolved() bool {
	return rr.Status == ReportStatusResolved || rr.Status == ReportStatusRejected
}

// RequiresAction checks if the resolution requires action on the review
func (rr *ReviewReport) RequiresAction() bool {
	return rr.Resolution == ResolutionHidden || rr.Resolution == ResolutionRemoved || rr.Resolution == ResolutionUserBanned
}

// ReportStats represents statistics about review reports
type ReportStats struct {
	TotalReports     int64            `json:"totalReports" bson:"totalReports"`
	PendingCount     int64            `json:"pendingCount" bson:"pendingCount"`
	ReviewingCount   int64            `json:"reviewingCount" bson:"reviewingCount"`
	ResolvedCount    int64            `json:"resolvedCount" bson:"resolvedCount"`
	RejectedCount    int64            `json:"rejectedCount" bson:"rejectedCount"`
	ByReason         map[string]int64 `json:"byReason" bson:"byReason"`
	ByResolution     map[string]int64 `json:"byResolution" bson:"byResolution"`
	AvgResolutionTime float64         `json:"avgResolutionTime" bson:"avgResolutionTime"` // In hours
}
