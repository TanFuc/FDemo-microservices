package domain

import "time"

// VoteType represents the type of vote
type VoteType string

const (
	VoteTypeHelpful    VoteType = "HELPFUL"
	VoteTypeNotHelpful VoteType = "NOT_HELPFUL"
)

// ReviewVote represents a vote on a review
type ReviewVote struct {
	ID        string    `json:"id" bson:"_id"`
	ReviewID  string    `json:"reviewId" bson:"reviewId"`
	UserID    string    `json:"userId" bson:"userId"`
	VoteType  VoteType  `json:"voteType" bson:"voteType"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// NewReviewVote creates a new review vote
func NewReviewVote(reviewID, userID string, voteType VoteType) *ReviewVote {
	now := time.Now()
	return &ReviewVote{
		ReviewID:  reviewID,
		UserID:    userID,
		VoteType:  voteType,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ChangeVote changes the vote type
func (rv *ReviewVote) ChangeVote(newVoteType VoteType) {
	rv.VoteType = newVoteType
	rv.UpdatedAt = time.Now()
}

// IsHelpful returns true if this is a helpful vote
func (rv *ReviewVote) IsHelpful() bool {
	return rv.VoteType == VoteTypeHelpful
}

// ReviewVoteSummary represents aggregated vote counts for a review
type ReviewVoteSummary struct {
	ReviewID       string `json:"reviewId" bson:"reviewId"`
	HelpfulCount   int    `json:"helpfulCount" bson:"helpfulCount"`
	NotHelpfulCount int   `json:"notHelpfulCount" bson:"notHelpfulCount"`
	TotalVotes     int    `json:"totalVotes" bson:"totalVotes"`
	HelpfulPercent float64 `json:"helpfulPercent" bson:"helpfulPercent"`
}

// CalculateHelpfulPercent calculates the percentage of helpful votes
func (rvs *ReviewVoteSummary) CalculateHelpfulPercent() {
	if rvs.TotalVotes == 0 {
		rvs.HelpfulPercent = 0
		return
	}
	rvs.HelpfulPercent = float64(rvs.HelpfulCount) / float64(rvs.TotalVotes) * 100
}

// UserVoteHistory represents a user's voting history for tracking
type UserVoteHistory struct {
	UserID     string         `json:"userId" bson:"userId"`
	Votes      []VoteRecord   `json:"votes" bson:"votes"`
	TotalVotes int            `json:"totalVotes" bson:"totalVotes"`
	UpdatedAt  time.Time      `json:"updatedAt" bson:"updatedAt"`
}

// VoteRecord represents a single vote record in history
type VoteRecord struct {
	ReviewID  string    `json:"reviewId" bson:"reviewId"`
	VoteType  VoteType  `json:"voteType" bson:"voteType"`
	VotedAt   time.Time `json:"votedAt" bson:"votedAt"`
}
