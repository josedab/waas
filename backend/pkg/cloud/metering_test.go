package cloud

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockMeteringRepo struct {
	mock.Mock
}

func (m *mockMeteringRepo) IncrementUsage(ctx context.Context, tenantID, metric string, value int64) error {
	args := m.Called(ctx, tenantID, metric, value)
	return args.Error(0)
}

func (m *mockMeteringRepo) GetUsage(ctx context.Context, tenantID, period string) (*UsageRecord, error) {
	args := m.Called(ctx, tenantID, period)
	if v := args.Get(0); v != nil {
		return v.(*UsageRecord), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMeteringRepo) GetSubscriptionByTenant(ctx context.Context, tenantID string) (*Subscription, error) {
	args := m.Called(ctx, tenantID)
	if v := args.Get(0); v != nil {
		return v.(*Subscription), args.Error(1)
	}
	return nil, args.Error(1)
}

// Stub remaining Repository methods
func (m *mockMeteringRepo) CreateSubscription(ctx context.Context, s *Subscription) error { return nil }
func (m *mockMeteringRepo) GetSubscription(ctx context.Context, id string) (*Subscription, error) {
	return nil, nil
}
func (m *mockMeteringRepo) UpdateSubscription(ctx context.Context, s *Subscription) error { return nil }
func (m *mockMeteringRepo) ListSubscriptions(ctx context.Context, status SubscriptionStatus, limit int) ([]*Subscription, error) {
	return nil, nil
}
func (m *mockMeteringRepo) ListUsage(ctx context.Context, tenantID string, limit int) ([]*UsageRecord, error) {
	return nil, nil
}
func (m *mockMeteringRepo) CreateInvoice(ctx context.Context, i *Invoice) error { return nil }
func (m *mockMeteringRepo) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	return nil, nil
}
func (m *mockMeteringRepo) ListInvoices(ctx context.Context, tenantID string, limit int) ([]*Invoice, error) {
	return nil, nil
}
func (m *mockMeteringRepo) UpdateInvoice(ctx context.Context, i *Invoice) error   { return nil }
func (m *mockMeteringRepo) CreateCustomer(ctx context.Context, c *Customer) error { return nil }
func (m *mockMeteringRepo) GetCustomer(ctx context.Context, tenantID string) (*Customer, error) {
	return nil, nil
}
func (m *mockMeteringRepo) UpdateCustomer(ctx context.Context, c *Customer) error { return nil }
func (m *mockMeteringRepo) CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error {
	return nil
}
func (m *mockMeteringRepo) ListPaymentMethods(ctx context.Context, tenantID string) ([]*PaymentMethod, error) {
	return nil, nil
}
func (m *mockMeteringRepo) DeletePaymentMethod(ctx context.Context, id string) error { return nil }
func (m *mockMeteringRepo) SetDefaultPaymentMethod(ctx context.Context, tenantID, pmID string) error {
	return nil
}
func (m *mockMeteringRepo) CreateTeamMember(ctx context.Context, tm *TeamMember) error { return nil }
func (m *mockMeteringRepo) GetTeamMember(ctx context.Context, id string) (*TeamMember, error) {
	return nil, nil
}
func (m *mockMeteringRepo) ListTeamMembers(ctx context.Context, tenantID string) ([]*TeamMember, error) {
	return nil, nil
}
func (m *mockMeteringRepo) UpdateTeamMember(ctx context.Context, tm *TeamMember) error { return nil }
func (m *mockMeteringRepo) DeleteTeamMember(ctx context.Context, id string) error      { return nil }
func (m *mockMeteringRepo) CreateAuditLog(ctx context.Context, log *AuditLog) error    { return nil }
func (m *mockMeteringRepo) ListAuditLogs(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error) {
	return nil, nil
}

func TestMeteringService_RecordUsageEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   *UsageMeterEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &UsageMeterEvent{
				TenantID:   "tenant-1",
				MetricName: "webhooks_sent",
				Quantity:   1,
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			event: &UsageMeterEvent{
				MetricName: "webhooks_sent",
				Quantity:   1,
			},
			wantErr: true,
		},
		{
			name: "missing metric_name",
			event: &UsageMeterEvent{
				TenantID: "tenant-1",
				Quantity: 1,
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			event: &UsageMeterEvent{
				TenantID:   "tenant-1",
				MetricName: "webhooks_sent",
				Quantity:   0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockMeteringRepo)
			if !tt.wantErr {
				repo.On("IncrementUsage", mock.Anything, tt.event.TenantID, tt.event.MetricName, tt.event.Quantity).Return(nil)
				repo.On("GetSubscriptionByTenant", mock.Anything, tt.event.TenantID).Return(nil, ErrSubscriptionNotFound)
			}

			svc := NewMeteringService(repo, nil)
			err := svc.RecordUsageEvent(context.Background(), tt.event)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Allow async alert check to complete
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestMeteringService_CalculatePeriodBilling(t *testing.T) {
	tests := []struct {
		name          string
		planID        string
		webhooksSent  int64
		expectedTotal int64
		expectOverage bool
	}{
		{
			name:          "within limits",
			planID:        "starter",
			webhooksSent:  50000,
			expectedTotal: 2900, // base only
			expectOverage: false,
		},
		{
			name:          "over limits",
			planID:        "starter",
			webhooksSent:  150000,
			expectedTotal: 2900 + 500, // base + (50000/100)*1
			expectOverage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockMeteringRepo)
			repo.On("GetSubscriptionByTenant", mock.Anything, "tenant-1").Return(&Subscription{
				PlanID: tt.planID,
				Status: SubscriptionStatusActive,
			}, nil)
			repo.On("GetUsage", mock.Anything, "tenant-1", mock.Anything).Return(&UsageRecord{
				WebhooksSent: tt.webhooksSent,
			}, nil)

			svc := NewMeteringService(repo, nil)
			summary, err := svc.CalculatePeriodBilling(context.Background(), "tenant-1", "2026-02")

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTotal, summary.TotalCharge)
			if tt.expectOverage {
				assert.NotEmpty(t, summary.UsageCharges)
			}
		})
	}
}

func TestMeteringService_SetAlert(t *testing.T) {
	svc := NewMeteringService(nil, nil)

	err := svc.SetAlert(context.Background(), &UsageAlert{
		TenantID:     "tenant-1",
		MetricName:   "webhooks_sent",
		ThresholdPct: 80,
	})
	assert.NoError(t, err)

	alerts := svc.GetAlerts(context.Background(), "tenant-1")
	assert.Len(t, alerts, 1)
	assert.Equal(t, float64(80), alerts[0].ThresholdPct)

	// Invalid threshold
	err = svc.SetAlert(context.Background(), &UsageAlert{
		TenantID:     "tenant-1",
		ThresholdPct: 0,
	})
	assert.Error(t, err)
}
