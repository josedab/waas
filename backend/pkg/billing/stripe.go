package billing

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	stripeinvoice "github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/usagerecord"
)

// StripeClient defines the interface for Stripe operations.
type StripeClient interface {
	CreateCustomer(email, name string) (string, error)
	CreateSubscription(customerID, priceID string) (string, error)
	CancelSubscription(subID string) error
	CreateUsageRecord(subItemID string, quantity int64, timestamp int64) error
	GetInvoices(customerID string) ([]StripeInvoice, error)
}

type stripeClientImpl struct {
	apiKey string
}

// NewStripeClient creates a new Stripe client.
func NewStripeClient(apiKey string) StripeClient {
	stripe.Key = apiKey
	return &stripeClientImpl{apiKey: apiKey}
}

func (s *stripeClientImpl) CreateCustomer(email, name string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create customer: %w", err)
	}
	return c.ID, nil
}

func (s *stripeClientImpl) CreateSubscription(customerID, priceID string) (string, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(priceID)},
		},
	}
	sub, err := subscription.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe create subscription: %w", err)
	}
	return sub.ID, nil
}

func (s *stripeClientImpl) CancelSubscription(subID string) error {
	params := &stripe.SubscriptionCancelParams{}
	_, err := subscription.Cancel(subID, params)
	if err != nil {
		return fmt.Errorf("stripe cancel subscription: %w", err)
	}
	return nil
}

func (s *stripeClientImpl) CreateUsageRecord(subItemID string, quantity int64, timestamp int64) error {
	params := &stripe.UsageRecordParams{
		SubscriptionItem: stripe.String(subItemID),
		Quantity:         stripe.Int64(quantity),
		Timestamp:        stripe.Int64(timestamp),
		Action:           stripe.String(string(stripe.UsageRecordActionIncrement)),
	}
	_, err := usagerecord.New(params)
	if err != nil {
		return fmt.Errorf("stripe create usage record: %w", err)
	}
	return nil
}

func (s *stripeClientImpl) GetInvoices(customerID string) ([]StripeInvoice, error) {
	params := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID),
	}
	params.Filters.AddFilter("limit", "", "10")

	var invoices []StripeInvoice
	i := stripeinvoice.List(params)
	for i.Next() {
		inv := i.Invoice()
		invoices = append(invoices, StripeInvoice{
			ID:     inv.ID,
			Amount: inv.AmountDue,
			Status: string(inv.Status),
		})
	}
	if err := i.Err(); err != nil {
		return nil, fmt.Errorf("stripe list invoices: %w", err)
	}
	return invoices, nil
}
