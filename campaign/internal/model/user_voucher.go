package model

import (
	"time"

	"github.com/google/uuid"
)

type UserVoucher struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	VoucherID uuid.UUID
	IsUsed    bool
	ClaimedAt time.Time
	UsedAt    *time.Time
}

func NewUserVoucher(userID, voucherID uuid.UUID) *UserVoucher {
	return &UserVoucher{
		ID:        uuid.New(),
		UserID:    userID,
		VoucherID: voucherID,
		IsUsed:    false,
		ClaimedAt: time.Now(),
		UsedAt:    nil,
	}
}

func (uv *UserVoucher) MarkUsed() {
	now := time.Now()
	uv.IsUsed = true
	uv.UsedAt = &now
}
