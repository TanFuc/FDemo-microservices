package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/tafu/campaign-service/internal/domain"
)

const (
	stockKeyPrefix = "voucher:%s:stock"
	usersKeyPrefix = "voucher:%s:users"
)

// claimScript is the Lua script for atomic voucher claiming
// It ensures:
// 1. User hasn't already claimed
// 2. Stock is available
// 3. Atomically decrements stock and adds user to claimed set
var claimScript = redis.NewScript(`
local stock_key = KEYS[1]
local users_key = KEYS[2]
local user_id = ARGV[1]

-- Check if user already claimed
local already_claimed = redis.call('SISMEMBER', users_key, user_id)
if already_claimed == 1 then
    return {err = "ALREADY_CLAIMED"}
end

-- Check stock
local stock = tonumber(redis.call('GET', stock_key) or 0)
if stock <= 0 then
    return {err = "OUT_OF_STOCK"}
end

-- Atomic claim: decrement stock and add user
redis.call('DECR', stock_key)
redis.call('SADD', users_key, user_id)

return {ok = "SUCCESS"}
`)

type VoucherCacheRepository struct {
	client *redis.Client
}

func NewVoucherCacheRepository(client *redis.Client) *VoucherCacheRepository {
	return &VoucherCacheRepository{
		client: client,
	}
}

func (r *VoucherCacheRepository) stockKey(code string) string {
	return fmt.Sprintf(stockKeyPrefix, code)
}

func (r *VoucherCacheRepository) usersKey(code string) string {
	return fmt.Sprintf(usersKeyPrefix, code)
}

func (r *VoucherCacheRepository) InitializeStock(ctx context.Context, code string, stock int) error {
	return r.client.Set(ctx, r.stockKey(code), stock, 0).Err()
}

func (r *VoucherCacheRepository) AtomicClaim(ctx context.Context, code string, userID uuid.UUID) error {
	keys := []string{r.stockKey(code), r.usersKey(code)}
	args := []interface{}{userID.String()}

	result, err := claimScript.Run(ctx, r.client, keys, args...).Result()
	if err != nil {
		// Check if it's a Lua table with error
		if err.Error() == "ALREADY_CLAIMED" {
			return domain.ErrVoucherAlreadyClaimed
		}
		if err.Error() == "OUT_OF_STOCK" {
			return domain.ErrVoucherOutOfStock
		}
		return fmt.Errorf("redis claim script error: %w", err)
	}

	// Handle result as map
	if resultMap, ok := result.(map[interface{}]interface{}); ok {
		if errVal, exists := resultMap["err"]; exists {
			errStr := fmt.Sprintf("%v", errVal)
			switch errStr {
			case "ALREADY_CLAIMED":
				return domain.ErrVoucherAlreadyClaimed
			case "OUT_OF_STOCK":
				return domain.ErrVoucherOutOfStock
			default:
				return fmt.Errorf("claim error: %s", errStr)
			}
		}
	}

	return nil
}

func (r *VoucherCacheRepository) GetStock(ctx context.Context, code string) (int, error) {
	val, err := r.client.Get(ctx, r.stockKey(code)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get stock: %w", err)
	}
	return val, nil
}

func (r *VoucherCacheRepository) HasUserClaimed(ctx context.Context, code string, userID uuid.UUID) (bool, error) {
	result, err := r.client.SIsMember(ctx, r.usersKey(code), userID.String()).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check user claim: %w", err)
	}
	return result, nil
}

func (r *VoucherCacheRepository) InvalidateVoucher(ctx context.Context, code string) error {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.stockKey(code))
	pipe.Del(ctx, r.usersKey(code))
	_, err := pipe.Exec(ctx)
	return err
}

// Ensure interface compliance
var _ domain.VoucherCacheRepository = (*VoucherCacheRepository)(nil)
