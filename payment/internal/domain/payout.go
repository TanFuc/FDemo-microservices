package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayoutStatus represents the status of a payout request
type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "PENDING"
	PayoutStatusApproved   PayoutStatus = "APPROVED"
	PayoutStatusProcessing PayoutStatus = "PROCESSING"
	PayoutStatusCompleted  PayoutStatus = "COMPLETED"
	PayoutStatusFailed     PayoutStatus = "FAILED"
	PayoutStatusCancelled  PayoutStatus = "CANCELLED"
	PayoutStatusOnHold     PayoutStatus = "ON_HOLD"
)

// PayoutMethod represents the method of payout
type PayoutMethod string

const (
	PayoutMethodBankTransfer PayoutMethod = "BANK_TRANSFER"
	PayoutMethodEWallet      PayoutMethod = "E_WALLET"
)

// PayoutRequest represents a seller payout request
type PayoutRequest struct {
	ID              uuid.UUID       `json:"id"`
	SellerID        uuid.UUID       `json:"seller_id"`
	ShopID          uuid.UUID       `json:"shop_id"`

	// Amount details
	RequestedAmount decimal.Decimal `json:"requested_amount"`
	ProcessingFee   decimal.Decimal `json:"processing_fee"`
	TaxAmount       decimal.Decimal `json:"tax_amount"`
	NetAmount       decimal.Decimal `json:"net_amount"`
	Currency        Currency        `json:"currency"`

	// Payout method
	Method          PayoutMethod    `json:"method"`
	PaymentMethodID *uuid.UUID      `json:"payment_method_id,omitempty"`

	// Bank details (snapshot at request time)
	BankName        string          `json:"bank_name,omitempty"`
	BankAccountNo   string          `json:"bank_account_no,omitempty"`
	AccountHolder   string          `json:"account_holder,omitempty"`
	BankBranch      string          `json:"bank_branch,omitempty"`
	SwiftCode       string          `json:"swift_code,omitempty"`

	// E-wallet details
	WalletType      string          `json:"wallet_type,omitempty"`
	WalletID        string          `json:"wallet_id,omitempty"`

	// Status and tracking
	Status          PayoutStatus    `json:"status"`
	ProviderRef     string          `json:"provider_ref,omitempty"`
	TransactionID   string          `json:"transaction_id,omitempty"`

	// Period covered
	PeriodStart     time.Time       `json:"period_start"`
	PeriodEnd       time.Time       `json:"period_end"`

	// Approval workflow
	RequestedAt     time.Time       `json:"requested_at"`
	ApprovedBy      *uuid.UUID      `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`

	// Failure handling
	FailureReason   string          `json:"failure_reason,omitempty"`
	RetryCount      int             `json:"retry_count"`
	LastRetryAt     *time.Time      `json:"last_retry_at,omitempty"`

	// Notes
	SellerNote      string          `json:"seller_note,omitempty"`
	AdminNote       string          `json:"admin_note,omitempty"`

	// Metadata and audit
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	IPAddress       string          `json:"ip_address,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewPayoutRequest creates a new payout request
func NewPayoutRequest(sellerID, shopID uuid.UUID, amount decimal.Decimal, currency Currency, method PayoutMethod) *PayoutRequest {
	now := time.Now()
	return &PayoutRequest{
		ID:              uuid.New(),
		SellerID:        sellerID,
		ShopID:          shopID,
		RequestedAmount: amount,
		ProcessingFee:   decimal.Zero,
		TaxAmount:       decimal.Zero,
		NetAmount:       amount,
		Currency:        currency,
		Method:          method,
		Status:          PayoutStatusPending,
		RetryCount:      0,
		RequestedAt:     now,
		Metadata:        json.RawMessage("{}"),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// SetBankDetails sets the bank transfer details
func (pr *PayoutRequest) SetBankDetails(bankName, accountNo, holder, branch, swift string) {
	pr.BankName = bankName
	pr.BankAccountNo = accountNo
	pr.AccountHolder = holder
	pr.BankBranch = branch
	pr.SwiftCode = swift
	pr.UpdatedAt = time.Now()
}

// SetEWalletDetails sets the e-wallet details
func (pr *PayoutRequest) SetEWalletDetails(walletType, walletID string) {
	pr.WalletType = walletType
	pr.WalletID = walletID
	pr.UpdatedAt = time.Now()
}

// SetPeriod sets the period covered by this payout
func (pr *PayoutRequest) SetPeriod(start, end time.Time) {
	pr.PeriodStart = start
	pr.PeriodEnd = end
	pr.UpdatedAt = time.Now()
}

// CalculateFees calculates and sets the processing fee and net amount
func (pr *PayoutRequest) CalculateFees(feeRate decimal.Decimal, taxRate decimal.Decimal) {
	pr.ProcessingFee = pr.RequestedAmount.Mul(feeRate).Round(2)
	pr.TaxAmount = pr.RequestedAmount.Mul(taxRate).Round(2)
	pr.NetAmount = pr.RequestedAmount.Sub(pr.ProcessingFee).Sub(pr.TaxAmount)
	pr.UpdatedAt = time.Now()
}

// Approve approves the payout request
func (pr *PayoutRequest) Approve(approvedBy uuid.UUID) {
	now := time.Now()
	pr.Status = PayoutStatusApproved
	pr.ApprovedBy = &approvedBy
	pr.ApprovedAt = &now
	pr.UpdatedAt = now
}

// StartProcessing marks the payout as being processed
func (pr *PayoutRequest) StartProcessing() {
	now := time.Now()
	pr.Status = PayoutStatusProcessing
	pr.ProcessedAt = &now
	pr.UpdatedAt = now
}

// Complete marks the payout as completed
func (pr *PayoutRequest) Complete(providerRef, transactionID string) {
	now := time.Now()
	pr.Status = PayoutStatusCompleted
	pr.ProviderRef = providerRef
	pr.TransactionID = transactionID
	pr.CompletedAt = &now
	pr.UpdatedAt = now
}

// Fail marks the payout as failed
func (pr *PayoutRequest) Fail(reason string) {
	now := time.Now()
	pr.Status = PayoutStatusFailed
	pr.FailureReason = reason
	pr.RetryCount++
	pr.LastRetryAt = &now
	pr.UpdatedAt = now
}

// Cancel cancels the payout request
func (pr *PayoutRequest) Cancel(adminNote string) {
	pr.Status = PayoutStatusCancelled
	pr.AdminNote = adminNote
	pr.UpdatedAt = time.Now()
}

// PutOnHold puts the payout request on hold
func (pr *PayoutRequest) PutOnHold(reason string) {
	pr.Status = PayoutStatusOnHold
	pr.AdminNote = reason
	pr.UpdatedAt = time.Now()
}

// CanRetry checks if the payout can be retried
func (pr *PayoutRequest) CanRetry(maxRetries int) bool {
	return pr.Status == PayoutStatusFailed && pr.RetryCount < maxRetries
}

// IsTerminal checks if the payout is in a terminal state
func (pr *PayoutRequest) IsTerminal() bool {
	return pr.Status == PayoutStatusCompleted || pr.Status == PayoutStatusCancelled
}

// PayoutSummary represents aggregated payout information for a seller
type PayoutSummary struct {
	SellerID        uuid.UUID       `json:"seller_id"`
	ShopID          uuid.UUID       `json:"shop_id"`
	TotalPending    decimal.Decimal `json:"total_pending"`
	TotalProcessing decimal.Decimal `json:"total_processing"`
	TotalCompleted  decimal.Decimal `json:"total_completed"`
	TotalFailed     decimal.Decimal `json:"total_failed"`
	LastPayoutAt    *time.Time      `json:"last_payout_at,omitempty"`
	NextPayoutDate  *time.Time      `json:"next_payout_date,omitempty"`
}

// PayoutSchedule represents the payout schedule configuration for a shop
type PayoutSchedule struct {
	ID           uuid.UUID     `json:"id"`
	ShopID       uuid.UUID     `json:"shop_id"`
	Frequency    string        `json:"frequency"` // DAILY, WEEKLY, BIWEEKLY, MONTHLY
	DayOfWeek    *int          `json:"day_of_week,omitempty"` // 0-6 for weekly
	DayOfMonth   *int          `json:"day_of_month,omitempty"` // 1-28 for monthly
	MinAmount    decimal.Decimal `json:"min_amount"`
	IsAutomatic  bool          `json:"is_automatic"`
	IsActive     bool          `json:"is_active"`
	NextRunAt    time.Time     `json:"next_run_at"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}
