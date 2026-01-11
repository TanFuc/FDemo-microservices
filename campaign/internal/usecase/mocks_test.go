package usecase

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/tafu/campaign-service/internal/domain"
)

// MockVoucherRepository is a mock implementation of VoucherRepository for testing
type MockVoucherRepository struct {
	vouchers map[string]*domain.Voucher
	mu       sync.RWMutex
}

func NewMockVoucherRepository() *MockVoucherRepository {
	return &MockVoucherRepository{
		vouchers: make(map[string]*domain.Voucher),
	}
}

func (m *MockVoucherRepository) Create(ctx context.Context, voucher *domain.Voucher) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vouchers[voucher.Code] = voucher
	return nil
}

func (m *MockVoucherRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Voucher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.vouchers {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, domain.ErrVoucherNotFound
}

func (m *MockVoucherRepository) GetByCode(ctx context.Context, code string) (*domain.Voucher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vouchers[code]
	if !ok {
		return nil, domain.ErrVoucherNotFound
	}
	return v, nil
}

func (m *MockVoucherRepository) Update(ctx context.Context, voucher *domain.Voucher) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vouchers[voucher.Code] = voucher
	return nil
}

func (m *MockVoucherRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range m.vouchers {
		if v.ID == id {
			if v.UsedCount >= v.TotalCount {
				return domain.ErrVoucherOutOfStock
			}
			v.UsedCount++
			return nil
		}
	}
	return domain.ErrVoucherNotFound
}

func (m *MockVoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for code, v := range m.vouchers {
		if v.ID == id {
			delete(m.vouchers, code)
			return nil
		}
	}
	return nil
}

func (m *MockVoucherRepository) ListByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Voucher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Voucher
	for _, v := range m.vouchers {
		if v.CampaignID == campaignID {
			result = append(result, v)
		}
	}
	return result, nil
}

// MockUserVoucherRepository is a mock implementation of UserVoucherRepository for testing
type MockUserVoucherRepository struct {
	userVouchers map[string]*domain.UserVoucher // key: userID:voucherID
	mu           sync.RWMutex
}

func NewMockUserVoucherRepository() *MockUserVoucherRepository {
	return &MockUserVoucherRepository{
		userVouchers: make(map[string]*domain.UserVoucher),
	}
}

func (m *MockUserVoucherRepository) key(userID, voucherID uuid.UUID) string {
	return userID.String() + ":" + voucherID.String()
}

func (m *MockUserVoucherRepository) Create(ctx context.Context, uv *domain.UserVoucher) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userVouchers[m.key(uv.UserID, uv.VoucherID)] = uv
	return nil
}

func (m *MockUserVoucherRepository) GetByUserAndVoucher(ctx context.Context, userID, voucherID uuid.UUID) (*domain.UserVoucher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	uv, ok := m.userVouchers[m.key(userID, voucherID)]
	if !ok {
		return nil, nil
	}
	return uv, nil
}

func (m *MockUserVoucherRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.UserVoucher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.UserVoucher
	for _, uv := range m.userVouchers {
		if uv.UserID == userID {
			result = append(result, uv)
		}
	}
	return result, nil
}

func (m *MockUserVoucherRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, uv := range m.userVouchers {
		if uv.ID == id {
			uv.MarkUsed()
			return nil
		}
	}
	return nil
}

func (m *MockUserVoucherRepository) Exists(ctx context.Context, userID, voucherID uuid.UUID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.userVouchers[m.key(userID, voucherID)]
	return ok, nil
}

// MockVoucherCacheRepository is a mock implementation of VoucherCacheRepository for testing
type MockVoucherCacheRepository struct {
	stocks map[string]int            // voucher code -> stock count
	users  map[string]map[string]bool // voucher code -> set of user IDs
	mu     sync.Mutex
}

func NewMockVoucherCacheRepository() *MockVoucherCacheRepository {
	return &MockVoucherCacheRepository{
		stocks: make(map[string]int),
		users:  make(map[string]map[string]bool),
	}
}

func (m *MockVoucherCacheRepository) InitializeStock(ctx context.Context, code string, stock int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stocks[code] = stock
	if m.users[code] == nil {
		m.users[code] = make(map[string]bool)
	}
	return nil
}

func (m *MockVoucherCacheRepository) AtomicClaim(ctx context.Context, code string, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user already claimed
	if m.users[code] != nil && m.users[code][userID.String()] {
		return domain.ErrVoucherAlreadyClaimed
	}

	// Check stock
	stock, ok := m.stocks[code]
	if !ok || stock <= 0 {
		return domain.ErrVoucherOutOfStock
	}

	// Atomic claim
	m.stocks[code]--
	if m.users[code] == nil {
		m.users[code] = make(map[string]bool)
	}
	m.users[code][userID.String()] = true

	return nil
}

func (m *MockVoucherCacheRepository) GetStock(ctx context.Context, code string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stocks[code], nil
}

func (m *MockVoucherCacheRepository) HasUserClaimed(ctx context.Context, code string, userID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.users[code] == nil {
		return false, nil
	}
	return m.users[code][userID.String()], nil
}

func (m *MockVoucherCacheRepository) InvalidateVoucher(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stocks, code)
	delete(m.users, code)
	return nil
}

// GetClaimedCount returns the number of successful claims for testing
func (m *MockVoucherCacheRepository) GetClaimedCount(code string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.users[code] == nil {
		return 0
	}
	return len(m.users[code])
}
