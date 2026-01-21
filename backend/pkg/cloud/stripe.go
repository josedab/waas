package cloud

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/subscription"
)

// RealStripeClient implements StripeClient using the Stripe SDK
type RealStripeClient struct {
	apiKey string
}

// NewRealStripeClient creates a new Stripe client
func NewRealStripeClient(apiKey string) *RealStripeClient {
	if apiKey == "" {
		apiKey = os.Getenv("STRIPE_API_KEY")
	}
	stripe.Key = apiKey
	return &RealStripeClient{apiKey: apiKey}
}

// CreateCustomer creates a Stripe customer
func (c *RealStripeClient) CreateCustomer(ctx context.Context, email, name string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return cust.ID, nil
}

// CreateSubscription creates a Stripe subscription
func (c *RealStripeClient) CreateSubscription(ctx context.Context, customerID, priceID string) (string, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
	}

	sub, err := subscription.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe subscription: %w", err)
	}

	return sub.ID, nil
}

// CancelSubscription cancels a Stripe subscription
func (c *RealStripeClient) CancelSubscription(ctx context.Context, subscriptionID string, atPeriodEnd bool) error {
	if atPeriodEnd {
		params := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		_, err := subscription.Update(subscriptionID, params)
		return err
	}

	_, err := subscription.Cancel(subscriptionID, nil)
	return err
}

// UpdateSubscription updates a Stripe subscription to a new price
func (c *RealStripeClient) UpdateSubscription(ctx context.Context, subscriptionID, newPriceID string) error {
	// Get the subscription to find the item ID
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	if len(sub.Items.Data) == 0 {
		return fmt.Errorf("subscription has no items")
	}

	itemID := sub.Items.Data[0].ID

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(newPriceID),
			},
		},
	}

	_, err = subscription.Update(subscriptionID, params)
	return err
}

// CreatePaymentIntent creates a Stripe payment intent
func (c *RealStripeClient) CreatePaymentIntent(ctx context.Context, customerID string, amount int64, currency string) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		Customer: stripe.String(customerID),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create payment intent: %w", err)
	}

	return pi.ClientSecret, nil
}

// AttachPaymentMethod attaches a payment method to a customer
func (c *RealStripeClient) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	}

	_, err := paymentmethod.Attach(paymentMethodID, params)
	return err
}

// MockStripeClient implements StripeClient for testing
type MockStripeClient struct {
	Customers     map[string]string
	Subscriptions map[string]string
}

// NewMockStripeClient creates a mock Stripe client
func NewMockStripeClient() *MockStripeClient {
	return &MockStripeClient{
		Customers:     make(map[string]string),
		Subscriptions: make(map[string]string),
	}
}

// CreateCustomer creates a mock customer
func (c *MockStripeClient) CreateCustomer(ctx context.Context, email, name string) (string, error) {
	id := fmt.Sprintf("cus_mock_%d", len(c.Customers)+1)
	c.Customers[id] = email
	return id, nil
}

// CreateSubscription creates a mock subscription
func (c *MockStripeClient) CreateSubscription(ctx context.Context, customerID, priceID string) (string, error) {
	id := fmt.Sprintf("sub_mock_%d", len(c.Subscriptions)+1)
	c.Subscriptions[id] = customerID
	return id, nil
}

// CancelSubscription cancels a mock subscription
func (c *MockStripeClient) CancelSubscription(ctx context.Context, subscriptionID string, atPeriodEnd bool) error {
	delete(c.Subscriptions, subscriptionID)
	return nil
}

// UpdateSubscription updates a mock subscription
func (c *MockStripeClient) UpdateSubscription(ctx context.Context, subscriptionID, newPriceID string) error {
	return nil
}

// CreatePaymentIntent creates a mock payment intent
func (c *MockStripeClient) CreatePaymentIntent(ctx context.Context, customerID string, amount int64, currency string) (string, error) {
	return "pi_mock_secret", nil
}

// AttachPaymentMethod attaches a mock payment method
func (c *MockStripeClient) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error {
	return nil
}
