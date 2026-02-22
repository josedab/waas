package costengine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordCostEvent(t *testing.T) {
	svc := NewService(nil)

	attr, err := svc.RecordCostEvent(context.Background(), "tenant-cost-1", &RecordCostEventRequest{
		EndpointID: "ep-1",
		EventType:  "order.created",
		BytesOut:   1024,
		Success:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), attr.DeliveryCount)
	assert.Equal(t, int64(1), attr.SuccessCount)
	assert.Greater(t, attr.TotalCost, float64(0))

	// Record another event
	attr2, err := svc.RecordCostEvent(context.Background(), "tenant-cost-1", &RecordCostEventRequest{
		EndpointID: "ep-1",
		EventType:  "order.created",
		BytesOut:   2048,
		Success:    false,
		IsRetry:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), attr2.DeliveryCount)
	assert.Equal(t, int64(1), attr2.FailedCount)
	assert.Equal(t, int64(1), attr2.RetryCount)
}

func TestRecordCostEventValidation(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.RecordCostEvent(context.Background(), "t1", &RecordCostEventRequest{})
	assert.Error(t, err)
}

func TestGetTenantCostSummary(t *testing.T) {
	svc := NewService(nil)

	// Record some events
	for i := 0; i < 5; i++ {
		_, _ = svc.RecordCostEvent(context.Background(), "tenant-cost-summary", &RecordCostEventRequest{
			EndpointID: "ep-1", EventType: "order.created", BytesOut: 1024, Success: true,
		})
	}
	for i := 0; i < 3; i++ {
		_, _ = svc.RecordCostEvent(context.Background(), "tenant-cost-summary", &RecordCostEventRequest{
			EndpointID: "ep-2", EventType: "payment.completed", BytesOut: 512, Success: true,
		})
	}

	summary, err := svc.GetTenantCostSummary(context.Background(), "tenant-cost-summary", "daily")
	require.NoError(t, err)
	assert.Equal(t, int64(8), summary.DeliveryCount)
	assert.Greater(t, summary.TotalCost, float64(0))
	assert.Len(t, summary.TopEndpoints, 2)
	assert.Len(t, summary.TopEventTypes, 2)
}

func TestGenerateChargebackReport(t *testing.T) {
	svc := NewService(nil)

	_, _ = svc.RecordCostEvent(context.Background(), "tenant-chargeback", &RecordCostEventRequest{
		EndpointID: "ep-1", EventType: "order.created", BytesOut: 1024, Success: true,
	})

	report, err := svc.GenerateChargebackReport(context.Background(), "tenant-chargeback", &GenerateChargebackRequest{
		PeriodStart: "2026-02-01",
		PeriodEnd:   "2026-02-28",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, report.ID)
	assert.NotEmpty(t, report.LineItems)
}

func TestGetCostForecast(t *testing.T) {
	svc := NewService(nil)

	forecast, err := svc.GetCostForecast(context.Background(), "tenant-forecast", 7)
	require.NoError(t, err)
	assert.Equal(t, 7, forecast.ForecastDays)
	assert.Len(t, forecast.DataPoints, 7)
	assert.Greater(t, forecast.TotalCost, float64(0))
}
