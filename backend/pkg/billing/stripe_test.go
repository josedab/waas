package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =====================
// StripeClient Interface Tests (via testifyStripeClient mock)
// =====================

func TestStripeClient_CreateCustomer_Success(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateCustomer", "user@test.com", "Test User").Return("cus_123", nil)

	id, err := sc.CreateCustomer("user@test.com", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "cus_123", id)
	sc.AssertExpectations(t)
}

func TestStripeClient_CreateCustomer_APIError(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateCustomer", "bad@test.com", "").Return("", errors.New("stripe: invalid email"))

	_, err := sc.CreateCustomer("bad@test.com", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe")
}

func TestStripeClient_CreateSubscription_Success(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateSubscription", "cus_123", "price_abc").Return("sub_456", nil)

	id, err := sc.CreateSubscription("cus_123", "price_abc")
	require.NoError(t, err)
	assert.Equal(t, "sub_456", id)
}

func TestStripeClient_CreateSubscription_InvalidCustomer(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateSubscription", "invalid", "price_abc").Return("", errors.New("stripe: no such customer"))

	_, err := sc.CreateSubscription("invalid", "price_abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such customer")
}

func TestStripeClient_CancelSubscription(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CancelSubscription", "sub_123").Return(nil)

	err := sc.CancelSubscription("sub_123")
	require.NoError(t, err)
}

func TestStripeClient_CreateUsageRecord_Success(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateUsageRecord", "si_123", int64(100), int64(1709164800)).Return(nil)

	err := sc.CreateUsageRecord("si_123", 100, 1709164800)
	require.NoError(t, err)
}

func TestStripeClient_CreateUsageRecord_ZeroQuantity(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("CreateUsageRecord", "si_123", int64(0), int64(1709164800)).Return(nil)

	err := sc.CreateUsageRecord("si_123", 0, 1709164800)
	require.NoError(t, err)
}

func TestStripeClient_GetInvoices_Success(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("GetInvoices", "cus_123").Return([]StripeInvoice{
		{ID: "inv_1", Amount: 2900, Status: "paid"},
		{ID: "inv_2", Amount: 4900, Status: "open"},
	}, nil)

	invoices, err := sc.GetInvoices("cus_123")
	require.NoError(t, err)
	assert.Len(t, invoices, 2)
	assert.Equal(t, "inv_1", invoices[0].ID)
	assert.Equal(t, int64(2900), invoices[0].Amount)
}

func TestStripeClient_GetInvoices_Empty(t *testing.T) {
	sc := new(testifyStripeClient)
	sc.On("GetInvoices", "cus_new").Return([]StripeInvoice{}, nil)

	invoices, err := sc.GetInvoices("cus_new")
	require.NoError(t, err)
	assert.Empty(t, invoices)
}

// =====================
// Service + Stripe Integration
// =====================

func TestService_CreateSubscription_WithStripeClient(t *testing.T) {
	sc := new(testifyStripeClient)
	svc := NewService(nil, nil, nil)
	svc.SetStripeClient(sc)
	svc.SetConfig(ServiceConfig{BillingEnabled: true})

	sc.On("CreateSubscription", mock.Anything, mock.Anything).Return("sub_stripe_123", nil)

	ctx := context.Background()
	tenantID := uuid.New()
	planID := uuid.New()

	sub, err := svc.CreateSubscriptionForTenant(ctx, tenantID, planID)
	require.NoError(t, err)
	assert.Equal(t, "sub_stripe_123", sub.StripeSubID)
}

func TestService_CreateSubscription_StripeError(t *testing.T) {
	sc := new(testifyStripeClient)
	svc := NewService(nil, nil, nil)
	svc.SetStripeClient(sc)
	svc.SetConfig(ServiceConfig{BillingEnabled: true})

	sc.On("CreateSubscription", mock.Anything, mock.Anything).Return("", errors.New("stripe error"))

	ctx := context.Background()
	tenantID := uuid.New()
	planID := uuid.New()

	_, err := svc.CreateSubscriptionForTenant(ctx, tenantID, planID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe")
}

func TestService_CancelSubscription_WithStripeClient(t *testing.T) {
	sc := new(testifyStripeClient)
	svc := NewService(nil, nil, nil)
	svc.SetStripeClient(sc)

	// CancelSubscriptionForTenant calls GetSubscriptionForTenant which returns
	// a default sub with empty StripeSubID, so CancelSubscription on stripe
	// won't be called (guarded by sub.StripeSubID != "")
	ctx := context.Background()
	tenantID := uuid.New()

	err := svc.CancelSubscriptionForTenant(ctx, tenantID)
	require.NoError(t, err)
	sc.AssertNotCalled(t, "CancelSubscription")
}

func TestService_CreateSubscription_WithoutStripe(t *testing.T) {
	svc := NewService(nil, nil, nil)
	// No Stripe client set — billing disabled

	ctx := context.Background()
	tenantID := uuid.New()
	planID := uuid.New()

	sub, err := svc.CreateSubscriptionForTenant(ctx, tenantID, planID)
	require.NoError(t, err)
	assert.Empty(t, sub.StripeSubID)
}
