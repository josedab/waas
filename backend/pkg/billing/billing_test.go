package billing

import (
	"testing"
)

func TestDefaultPricing(t *testing.T) {
	pricing := DefaultPricing

	if pricing.WebhookRequestCost <= 0 {
		t.Error("WebhookRequestCost should be positive")
	}
	if pricing.Currency == "" {
		t.Error("Currency should not be empty")
	}
}

func TestAlertThreshold_Validate(t *testing.T) {
	tests := []struct {
		name      string
		threshold *AlertThreshold
		wantErr   bool
	}{
		{
			name:      "valid threshold",
			threshold: &AlertThreshold{Percentage: 80, Channels: []AlertChannel{ChannelEmail}},
			wantErr:   false,
		},
		{
			name:      "zero percentage",
			threshold: &AlertThreshold{Percentage: 0, Channels: []AlertChannel{ChannelEmail}},
			wantErr:   true,
		},
		{
			name:      "over 100 percentage",
			threshold: &AlertThreshold{Percentage: 150, Channels: []AlertChannel{ChannelEmail}},
			wantErr:   true,
		},
		{
			name:      "no channels",
			threshold: &AlertThreshold{Percentage: 80},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.threshold.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSpendTracker_Fields(t *testing.T) {
	tracker := &SpendTracker{
		ID:           "test-id",
		TenantID:     "tenant-1",
		BudgetLimit:  1000,
		CurrentSpend: 500,
		Currency:     "USD",
		Period:       PeriodMonthly,
	}

	if tracker.ID != "test-id" {
		t.Errorf("ID = %s, want test-id", tracker.ID)
	}
	if tracker.BudgetLimit != 1000 {
		t.Errorf("BudgetLimit = %f, want 1000", tracker.BudgetLimit)
	}
}

func TestBudgetConfig_Fields(t *testing.T) {
	budget := &BudgetConfig{
		TenantID: "tenant-1",
		Amount:   1000,
		Period:   PeriodMonthly,
		Currency: "USD",
	}

	if budget.Amount <= 0 {
		t.Error("Budget amount should be positive")
	}
}

func TestBillingPeriod_Constants(t *testing.T) {
	periods := []BillingPeriod{PeriodDaily, PeriodWeekly, PeriodMonthly}

	for _, p := range periods {
		if p == "" {
			t.Error("BillingPeriod constant should not be empty")
		}
	}
}

func TestAlertChannel_Constants(t *testing.T) {
	channels := []AlertChannel{ChannelEmail, ChannelSlack, ChannelWebhook, ChannelSMS}

	for _, c := range channels {
		if c == "" {
			t.Error("AlertChannel constant should not be empty")
		}
	}
}
