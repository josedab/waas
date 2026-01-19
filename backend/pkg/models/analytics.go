package models

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryMetric represents an individual delivery event for analytics
type DeliveryMetric struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EndpointID    uuid.UUID  `json:"endpoint_id" db:"endpoint_id"`
	DeliveryID    uuid.UUID  `json:"delivery_id" db:"delivery_id"`
	Status        string     `json:"status" db:"status"`
	HTTPStatus    *int       `json:"http_status,omitempty" db:"http_status"`
	LatencyMs     int        `json:"latency_ms" db:"latency_ms"`
	AttemptNumber int        `json:"attempt_number" db:"attempt_number"`
	ErrorMessage  *string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// HourlyMetric represents aggregated metrics for an hour
type HourlyMetric struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EndpointID           *uuid.UUID `json:"endpoint_id,omitempty" db:"endpoint_id"`
	HourTimestamp        time.Time  `json:"hour_timestamp" db:"hour_timestamp"`
	TotalDeliveries      int        `json:"total_deliveries" db:"total_deliveries"`
	SuccessfulDeliveries int        `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries     int        `json:"failed_deliveries" db:"failed_deliveries"`
	RetryingDeliveries   int        `json:"retrying_deliveries" db:"retrying_deliveries"`
	AvgLatencyMs         *float64   `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	P95LatencyMs         *float64   `json:"p95_latency_ms,omitempty" db:"p95_latency_ms"`
	P99LatencyMs         *float64   `json:"p99_latency_ms,omitempty" db:"p99_latency_ms"`
	TotalRetries         int        `json:"total_retries" db:"total_retries"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// DailyMetric represents aggregated metrics for a day
type DailyMetric struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EndpointID           *uuid.UUID `json:"endpoint_id,omitempty" db:"endpoint_id"`
	DateTimestamp        time.Time  `json:"date_timestamp" db:"date_timestamp"`
	TotalDeliveries      int        `json:"total_deliveries" db:"total_deliveries"`
	SuccessfulDeliveries int        `json:"successful_deliveries" db:"successful_deliveries"`
	FailedDeliveries     int        `json:"failed_deliveries" db:"failed_deliveries"`
	RetryingDeliveries   int        `json:"retrying_deliveries" db:"retrying_deliveries"`
	AvgLatencyMs         *float64   `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	P95LatencyMs         *float64   `json:"p95_latency_ms,omitempty" db:"p95_latency_ms"`
	P99LatencyMs         *float64   `json:"p99_latency_ms,omitempty" db:"p99_latency_ms"`
	TotalRetries         int        `json:"total_retries" db:"total_retries"`
	UniqueEndpoints      int        `json:"unique_endpoints" db:"unique_endpoints"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// RealtimeMetric represents real-time metrics for WebSocket updates
type RealtimeMetric struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	TenantID    uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	MetricType  string                 `json:"metric_type" db:"metric_type"`
	MetricValue float64                `json:"metric_value" db:"metric_value"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// AlertConfig represents alert configuration for a tenant
type AlertConfig struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	TenantID             uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	AlertType            string                 `json:"alert_type" db:"alert_type"`
	ThresholdValue       float64                `json:"threshold_value" db:"threshold_value"`
	TimeWindowMinutes    int                    `json:"time_window_minutes" db:"time_window_minutes"`
	IsEnabled            bool                   `json:"is_enabled" db:"is_enabled"`
	NotificationChannels map[string]interface{} `json:"notification_channels,omitempty" db:"notification_channels"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
}

// AlertHistory represents triggered alerts
type AlertHistory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	AlertConfigID  uuid.UUID  `json:"alert_config_id" db:"alert_config_id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	AlertType      string     `json:"alert_type" db:"alert_type"`
	TriggeredValue float64    `json:"triggered_value" db:"triggered_value"`
	ThresholdValue float64    `json:"threshold_value" db:"threshold_value"`
	Message        string     `json:"message" db:"message"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// MetricsQuery represents a query for analytics data
type MetricsQuery struct {
	TenantID    uuid.UUID    `json:"tenant_id"`
	EndpointIDs []uuid.UUID  `json:"endpoint_ids,omitempty"`
	StartDate   time.Time    `json:"start_date"`
	EndDate     time.Time    `json:"end_date"`
	GroupBy     string       `json:"group_by"` // hour, day, week
	Statuses    []string     `json:"statuses,omitempty"`
	Limit       int          `json:"limit,omitempty"`
	Offset      int          `json:"offset,omitempty"`
}

// MetricsResponse represents the response for analytics queries
type MetricsResponse struct {
	Data       []interface{} `json:"data"`
	TotalCount int           `json:"total_count"`
	Summary    MetricsSummary `json:"summary"`
}

// MetricsSummary provides aggregated summary statistics
type MetricsSummary struct {
	TotalDeliveries      int     `json:"total_deliveries"`
	SuccessfulDeliveries int     `json:"successful_deliveries"`
	FailedDeliveries     int     `json:"failed_deliveries"`
	SuccessRate          float64 `json:"success_rate"`
	AvgLatencyMs         float64 `json:"avg_latency_ms"`
	P95LatencyMs         float64 `json:"p95_latency_ms"`
	P99LatencyMs         float64 `json:"p99_latency_ms"`
}

// WebSocketMessage represents real-time updates sent via WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	TenantID  uuid.UUID   `json:"tenant_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// DashboardMetrics represents the main dashboard metrics
type DashboardMetrics struct {
	DeliveryRate    float64            `json:"delivery_rate"`    // deliveries per minute
	SuccessRate     float64            `json:"success_rate"`     // percentage
	AvgLatency      float64            `json:"avg_latency"`      // milliseconds
	ActiveEndpoints int                `json:"active_endpoints"`
	QueueSize       int                `json:"queue_size"`
	RecentAlerts    []AlertHistory     `json:"recent_alerts"`
	TopEndpoints    []EndpointMetrics  `json:"top_endpoints"`
}

// EndpointMetrics represents metrics for a specific endpoint
type EndpointMetrics struct {
	EndpointID       uuid.UUID `json:"endpoint_id"`
	EndpointURL      string    `json:"endpoint_url"`
	TotalDeliveries  int       `json:"total_deliveries"`
	SuccessRate      float64   `json:"success_rate"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	LastDeliveryAt   *time.Time `json:"last_delivery_at,omitempty"`
}