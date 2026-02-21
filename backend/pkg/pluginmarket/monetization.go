package pluginmarket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Default revenue share: 70% developer, 30% platform
const (
	DefaultDeveloperSharePct = 70
	DefaultPlatformSharePct  = 30
)

// TransactionStatus represents the state of a monetization transaction
type TransactionStatus string

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionCompleted TransactionStatus = "completed"
	TransactionFailed    TransactionStatus = "failed"
	TransactionRefunded  TransactionStatus = "refunded"
)

// PayoutStatus represents the state of a developer payout
type PayoutStatus string

const (
	PayoutPending    PayoutStatus = "pending"
	PayoutProcessing PayoutStatus = "processing"
	PayoutPaid       PayoutStatus = "paid"
	PayoutFailed     PayoutStatus = "failed"
)

// RevenueShare defines the split between developer and platform
type RevenueShare struct {
	DeveloperPct int `json:"developer_pct"`
	PlatformPct  int `json:"platform_pct"`
}

// DefaultRevenueShare returns the default 70/30 split
func DefaultRevenueShare() RevenueShare {
	return RevenueShare{
		DeveloperPct: DefaultDeveloperSharePct,
		PlatformPct:  DefaultPlatformSharePct,
	}
}

// DeveloperAccount tracks a developer's Stripe Connect account and earnings
type DeveloperAccount struct {
	ID                string    `json:"id"`
	DeveloperID       string    `json:"developer_id"`
	StripeConnectID   string    `json:"stripe_connect_id"`
	Email             string    `json:"email"`
	DisplayName       string    `json:"display_name"`
	TotalEarnings     float64   `json:"total_earnings"`
	PendingEarnings   float64   `json:"pending_earnings"`
	PaidEarnings      float64   `json:"paid_earnings"`
	RevenueShare      RevenueShare `json:"revenue_share"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Transaction records a plugin purchase or payout event
type Transaction struct {
	ID              string            `json:"id"`
	ListingID       string            `json:"listing_id"`
	BuyerTenantID   string            `json:"buyer_tenant_id"`
	DeveloperID     string            `json:"developer_id"`
	Amount          float64           `json:"amount"`
	Currency        string            `json:"currency"`
	DeveloperAmount float64           `json:"developer_amount"`
	PlatformAmount  float64           `json:"platform_amount"`
	Status          TransactionStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
}

// Payout represents a developer payout
type Payout struct {
	ID          string       `json:"id"`
	DeveloperID string       `json:"developer_id"`
	Amount      float64      `json:"amount"`
	Currency    string       `json:"currency"`
	Status      PayoutStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	PaidAt      *time.Time   `json:"paid_at,omitempty"`
}

// EarningsDashboard aggregates a developer's financial data
type EarningsDashboard struct {
	DeveloperID     string        `json:"developer_id"`
	TotalEarnings   float64       `json:"total_earnings"`
	PendingEarnings float64       `json:"pending_earnings"`
	PaidEarnings    float64       `json:"paid_earnings"`
	RevenueShare    RevenueShare  `json:"revenue_share"`
	Transactions    []Transaction `json:"recent_transactions,omitempty"`
	Payouts         []Payout      `json:"recent_payouts,omitempty"`
}

// MonetizationService manages developer accounts, purchases, and payouts
type MonetizationService struct {
	mu           sync.RWMutex
	accounts     map[string]*DeveloperAccount // keyed by developerID
	transactions []Transaction
	payouts      []Payout
	revenueShare RevenueShare
}

// NewMonetizationService creates a new MonetizationService with default revenue share
func NewMonetizationService() *MonetizationService {
	return &MonetizationService{
		accounts:     make(map[string]*DeveloperAccount),
		revenueShare: DefaultRevenueShare(),
	}
}

// RegisterDeveloper creates a developer account with Stripe Connect info
func (s *MonetizationService) RegisterDeveloper(ctx context.Context, developerID, email, displayName, stripeConnectID string) (*DeveloperAccount, error) {
	if developerID == "" {
		return nil, fmt.Errorf("developer ID is required")
	}
	if stripeConnectID == "" {
		return nil, fmt.Errorf("stripe connect ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[developerID]; exists {
		return nil, fmt.Errorf("developer account already exists for %q", developerID)
	}

	now := time.Now()
	account := &DeveloperAccount{
		ID:              uuid.New().String(),
		DeveloperID:     developerID,
		StripeConnectID: stripeConnectID,
		Email:           email,
		DisplayName:     displayName,
		RevenueShare:    s.revenueShare,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.accounts[developerID] = account
	return account, nil
}

// GetDeveloper retrieves a developer account
func (s *MonetizationService) GetDeveloper(ctx context.Context, developerID string) (*DeveloperAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[developerID]
	if !ok {
		return nil, fmt.Errorf("developer account %q not found", developerID)
	}
	return acc, nil
}

// RecordPurchase records a plugin purchase and splits revenue
func (s *MonetizationService) RecordPurchase(ctx context.Context, listingID, buyerTenantID, developerID string, amount float64, currency string) (*Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[developerID]
	if !ok {
		return nil, fmt.Errorf("developer account %q not found", developerID)
	}

	devAmount := amount * float64(acc.RevenueShare.DeveloperPct) / 100
	platAmount := amount - devAmount

	tx := Transaction{
		ID:              uuid.New().String(),
		ListingID:       listingID,
		BuyerTenantID:   buyerTenantID,
		DeveloperID:     developerID,
		Amount:          amount,
		Currency:        currency,
		DeveloperAmount: devAmount,
		PlatformAmount:  platAmount,
		Status:          TransactionCompleted,
		CreatedAt:       time.Now(),
	}
	s.transactions = append(s.transactions, tx)

	acc.TotalEarnings += devAmount
	acc.PendingEarnings += devAmount
	acc.UpdatedAt = time.Now()

	return &tx, nil
}

// GetEarnings returns the earnings dashboard for a developer
func (s *MonetizationService) GetEarnings(ctx context.Context, developerID string) (*EarningsDashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[developerID]
	if !ok {
		return nil, fmt.Errorf("developer account %q not found", developerID)
	}

	var txs []Transaction
	for _, tx := range s.transactions {
		if tx.DeveloperID == developerID {
			txs = append(txs, tx)
		}
	}

	var payouts []Payout
	for _, p := range s.payouts {
		if p.DeveloperID == developerID {
			payouts = append(payouts, p)
		}
	}

	return &EarningsDashboard{
		DeveloperID:     developerID,
		TotalEarnings:   acc.TotalEarnings,
		PendingEarnings: acc.PendingEarnings,
		PaidEarnings:    acc.PaidEarnings,
		RevenueShare:    acc.RevenueShare,
		Transactions:    txs,
		Payouts:         payouts,
	}, nil
}

// ProcessPayout moves pending earnings to a paid payout
func (s *MonetizationService) ProcessPayout(ctx context.Context, developerID string) (*Payout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[developerID]
	if !ok {
		return nil, fmt.Errorf("developer account %q not found", developerID)
	}
	if acc.PendingEarnings <= 0 {
		return nil, fmt.Errorf("no pending earnings to pay out")
	}

	now := time.Now()
	payout := Payout{
		ID:          uuid.New().String(),
		DeveloperID: developerID,
		Amount:      acc.PendingEarnings,
		Currency:    "USD",
		Status:      PayoutPaid,
		CreatedAt:   now,
		PaidAt:      &now,
	}
	s.payouts = append(s.payouts, payout)

	acc.PaidEarnings += acc.PendingEarnings
	acc.PendingEarnings = 0
	acc.UpdatedAt = now

	return &payout, nil
}
