package repository

import (
	"context"
)

type CacheRepository interface {
	Reserve(ctx context.Context, skuID string, quantity int) (bool, error)
	Release(ctx context.Context, skuID string, quantity int) error
	SetInventory(ctx context.Context, skuID string, total, reserved int) error
	GetInventory(ctx context.Context, skuID string) (total int, reserved int, err error)
}
