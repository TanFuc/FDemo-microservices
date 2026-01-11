package domain

import "time"

// ShopReviewStatus represents the status of a shop review
type ShopReviewStatus string

const (
	ShopReviewStatusVisible ShopReviewStatus = "VISIBLE"
	ShopReviewStatusHidden  ShopReviewStatus = "HIDDEN"
	ShopReviewStatusPending ShopReviewStatus = "PENDING" // For moderation
)

// ShopReviewType represents the type of shop review
type ShopReviewType string

const (
	ShopReviewTypeOrder    ShopReviewType = "ORDER"    // Based on order experience
	ShopReviewTypeService  ShopReviewType = "SERVICE"  // General service review
	ShopReviewTypeDelivery ShopReviewType = "DELIVERY" // Delivery experience
)

// ShopReply represents the shop's reply to a review
type ShopReply struct {
	Content   string    `json:"content" bson:"content"`
	RepliedBy string    `json:"repliedBy" bson:"repliedBy"` // Staff member ID
	RepliedAt time.Time `json:"repliedAt" bson:"repliedAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}

// ShopReview represents a review of a shop
type ShopReview struct {
	ID          string           `json:"id" bson:"_id"`
	ShopID      string           `json:"shopId" bson:"shopId"`
	UserID      string           `json:"userId" bson:"userId"`
	OrderID     string           `json:"orderId,omitempty" bson:"orderId,omitempty"`

	// User info (denormalized)
	UserName    string           `json:"userName" bson:"userName"`
	UserAvatar  string           `json:"userAvatar,omitempty" bson:"userAvatar,omitempty"`

	// Review type
	Type        ShopReviewType   `json:"type" bson:"type"`

	// Ratings (1-5)
	OverallRating    int          `json:"overallRating" bson:"overallRating"`
	ServiceRating    int          `json:"serviceRating,omitempty" bson:"serviceRating,omitempty"`
	QualityRating    int          `json:"qualityRating,omitempty" bson:"qualityRating,omitempty"`
	DeliveryRating   int          `json:"deliveryRating,omitempty" bson:"deliveryRating,omitempty"`
	CommunicationRating int       `json:"communicationRating,omitempty" bson:"communicationRating,omitempty"`

	// Review content
	Title       string           `json:"title,omitempty" bson:"title,omitempty"`
	Content     string           `json:"content" bson:"content"`
	Pros        []string         `json:"pros,omitempty" bson:"pros,omitempty"`     // What was good
	Cons        []string         `json:"cons,omitempty" bson:"cons,omitempty"`     // What could improve
	Tags        []string         `json:"tags,omitempty" bson:"tags,omitempty"`     // e.g., "fast_shipping", "helpful"

	// Media
	Images      []string         `json:"images,omitempty" bson:"images,omitempty"`
	Videos      []string         `json:"videos,omitempty" bson:"videos,omitempty"`

	// Shop's reply
	Reply       *ShopReply       `json:"reply,omitempty" bson:"reply,omitempty"`

	// Interaction stats
	HelpfulCount    int          `json:"helpfulCount" bson:"helpfulCount"`
	NotHelpfulCount int          `json:"notHelpfulCount" bson:"notHelpfulCount"`

	// Verification
	IsPurchased bool             `json:"isPurchased" bson:"isPurchased"` // Verified buyer
	IsEdited    bool             `json:"isEdited" bson:"isEdited"`

	// Status
	Status      ShopReviewStatus `json:"status" bson:"status"`

	// Timestamps
	CreatedAt   time.Time        `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt" bson:"updatedAt"`
	EditedAt    *time.Time       `json:"editedAt,omitempty" bson:"editedAt,omitempty"`
}

// NewShopReview creates a new shop review
func NewShopReview(shopID, userID, userName string, reviewType ShopReviewType, overallRating int, content string) *ShopReview {
	now := time.Now()
	return &ShopReview{
		ShopID:        shopID,
		UserID:        userID,
		UserName:      userName,
		Type:          reviewType,
		OverallRating: overallRating,
		Content:       content,
		Pros:          make([]string, 0),
		Cons:          make([]string, 0),
		Tags:          make([]string, 0),
		Images:        make([]string, 0),
		Videos:        make([]string, 0),
		HelpfulCount:  0,
		NotHelpfulCount: 0,
		IsPurchased:  false,
		IsEdited:     false,
		Status:       ShopReviewStatusVisible,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetOrderID sets the order ID for verified buyer
func (sr *ShopReview) SetOrderID(orderID string) {
	sr.OrderID = orderID
	sr.IsPurchased = true
	sr.UpdatedAt = time.Now()
}

// SetDetailedRatings sets the detailed ratings
func (sr *ShopReview) SetDetailedRatings(service, quality, delivery, communication int) {
	sr.ServiceRating = service
	sr.QualityRating = quality
	sr.DeliveryRating = delivery
	sr.CommunicationRating = communication
	sr.UpdatedAt = time.Now()
}

// SetProsAndCons sets the pros and cons
func (sr *ShopReview) SetProsAndCons(pros, cons []string) {
	sr.Pros = pros
	sr.Cons = cons
	sr.UpdatedAt = time.Now()
}

// AddTag adds a tag to the review
func (sr *ShopReview) AddTag(tag string) {
	for _, t := range sr.Tags {
		if t == tag {
			return
		}
	}
	sr.Tags = append(sr.Tags, tag)
	sr.UpdatedAt = time.Now()
}

// AddImage adds an image to the review
func (sr *ShopReview) AddImage(imageURL string) {
	sr.Images = append(sr.Images, imageURL)
	sr.UpdatedAt = time.Now()
}

// AddVideo adds a video to the review
func (sr *ShopReview) AddVideo(videoURL string) {
	sr.Videos = append(sr.Videos, videoURL)
	sr.UpdatedAt = time.Now()
}

// SetReply sets the shop's reply
func (sr *ShopReview) SetReply(content, repliedBy string) {
	now := time.Now()
	sr.Reply = &ShopReply{
		Content:   content,
		RepliedBy: repliedBy,
		RepliedAt: now,
	}
	sr.UpdatedAt = now
}

// UpdateReply updates the shop's reply
func (sr *ShopReview) UpdateReply(content string) {
	if sr.Reply != nil {
		now := time.Now()
		sr.Reply.Content = content
		sr.Reply.UpdatedAt = &now
		sr.UpdatedAt = now
	}
}

// Edit edits the review content
func (sr *ShopReview) Edit(rating int, content string) {
	now := time.Now()
	sr.OverallRating = rating
	sr.Content = content
	sr.IsEdited = true
	sr.EditedAt = &now
	sr.UpdatedAt = now
}

// IncrementHelpful increments the helpful count
func (sr *ShopReview) IncrementHelpful() {
	sr.HelpfulCount++
	sr.UpdatedAt = time.Now()
}

// DecrementHelpful decrements the helpful count
func (sr *ShopReview) DecrementHelpful() {
	if sr.HelpfulCount > 0 {
		sr.HelpfulCount--
	}
	sr.UpdatedAt = time.Now()
}

// IncrementNotHelpful increments the not helpful count
func (sr *ShopReview) IncrementNotHelpful() {
	sr.NotHelpfulCount++
	sr.UpdatedAt = time.Now()
}

// DecrementNotHelpful decrements the not helpful count
func (sr *ShopReview) DecrementNotHelpful() {
	if sr.NotHelpfulCount > 0 {
		sr.NotHelpfulCount--
	}
	sr.UpdatedAt = time.Now()
}

// Hide hides the review
func (sr *ShopReview) Hide() {
	sr.Status = ShopReviewStatusHidden
	sr.UpdatedAt = time.Now()
}

// Show shows the review
func (sr *ShopReview) Show() {
	sr.Status = ShopReviewStatusVisible
	sr.UpdatedAt = time.Now()
}

// IsVisible returns true if the review is visible
func (sr *ShopReview) IsVisible() bool {
	return sr.Status == ShopReviewStatusVisible
}

// HasMedia returns true if the review has images or videos
func (sr *ShopReview) HasMedia() bool {
	return len(sr.Images) > 0 || len(sr.Videos) > 0
}

// HasReply returns true if the shop has replied
func (sr *ShopReview) HasReply() bool {
	return sr.Reply != nil
}

// GetAverageDetailedRating calculates the average of detailed ratings
func (sr *ShopReview) GetAverageDetailedRating() float64 {
	count := 0
	total := 0

	if sr.ServiceRating > 0 {
		total += sr.ServiceRating
		count++
	}
	if sr.QualityRating > 0 {
		total += sr.QualityRating
		count++
	}
	if sr.DeliveryRating > 0 {
		total += sr.DeliveryRating
		count++
	}
	if sr.CommunicationRating > 0 {
		total += sr.CommunicationRating
		count++
	}

	if count == 0 {
		return float64(sr.OverallRating)
	}
	return float64(total) / float64(count)
}

// ShopRating represents aggregated shop ratings
type ShopRating struct {
	ShopID               string             `json:"shopId" bson:"shopId"`
	AverageRating        float64            `json:"averageRating" bson:"averageRating"`
	TotalReviews         int64              `json:"totalReviews" bson:"totalReviews"`

	// Detailed averages
	AverageServiceRating       float64 `json:"averageServiceRating" bson:"averageServiceRating"`
	AverageQualityRating       float64 `json:"averageQualityRating" bson:"averageQualityRating"`
	AverageDeliveryRating      float64 `json:"averageDeliveryRating" bson:"averageDeliveryRating"`
	AverageCommunicationRating float64 `json:"averageCommunicationRating" bson:"averageCommunicationRating"`

	// Distribution
	RatingDistribution map[int]int64      `json:"ratingDistribution" bson:"ratingDistribution"` // 1-5 stars
	ReviewsWithMedia   int64              `json:"reviewsWithMedia" bson:"reviewsWithMedia"`
	VerifiedPurchases  int64              `json:"verifiedPurchases" bson:"verifiedPurchases"`

	// Recent metrics
	RecentRating       float64            `json:"recentRating" bson:"recentRating"` // Last 30 days
	RecentReviewCount  int64              `json:"recentReviewCount" bson:"recentReviewCount"`

	// Response stats
	ResponseRate       float64            `json:"responseRate" bson:"responseRate"` // % of reviews with reply
	AvgResponseTime    float64            `json:"avgResponseTime" bson:"avgResponseTime"` // Hours

	// Top tags
	TopTags            []TagCount         `json:"topTags" bson:"topTags"`

	// Timestamps
	CalculatedAt       time.Time          `json:"calculatedAt" bson:"calculatedAt"`
}

// TagCount represents a tag with its count
type TagCount struct {
	Tag   string `json:"tag" bson:"tag"`
	Count int64  `json:"count" bson:"count"`
}
