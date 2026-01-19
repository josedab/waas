package repository

import (
	"context"
	"time"
	"webhook-platform/pkg/models"

	"github.com/google/uuid"
)

// AnalyticsRepositoryInterface defines the interface for analytics repository operations
type AnalyticsRepositoryInterface interface {
	RecordDeliveryMetric(ctx context.Context, metric *models.DeliveryMetric) error
	GetDeliveryMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.DeliveryMetric, error)
	UpsertHourlyMetric(ctx context.Context, metric *models.HourlyMetric) error
	GetHourlyMetrics(ctx context.Context, query *models.MetricsQuery) ([]models.HourlyMetric, error)
	RecordRealtimeMetric(ctx context.Context, metric *models.RealtimeMetric) error
	GetRealtimeMetrics(ctx context.Context, tenantID uuid.UUID, metricType string, since time.Time) ([]models.RealtimeMetric, error)
	GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID, timeWindow time.Duration) (*models.DashboardMetrics, error)
	GetMetricsSummary(ctx context.Context, query *models.MetricsQuery) (*models.MetricsSummary, error)
	CleanupOldMetrics(ctx context.Context, retentionDays int) error
}