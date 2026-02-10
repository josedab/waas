package streaming

import (
	"context"
	"time"
)

// MonitoringDashboard provides monitoring data for streaming bridges
type MonitoringDashboard struct {
	TenantID        string                 `json:"tenant_id"`
	Overview        *StreamingOverview     `json:"overview"`
	BridgeHealth    []BridgeMonitorHealth  `json:"bridge_health"`
	ThroughputChart []ThroughputDataPoint  `json:"throughput_chart"`
	ErrorBreakdown  []ErrorBreakdownEntry  `json:"error_breakdown"`
	ConsumerLag     []ConsumerLagDataPoint `json:"consumer_lag"`
	TopBridges      []TopBridgeEntry       `json:"top_bridges"`
	Alerts          []StreamingAlert       `json:"alerts"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// StreamingOverview provides high-level overview metrics
type StreamingOverview struct {
	TotalBridges       int     `json:"total_bridges"`
	ActiveBridges      int     `json:"active_bridges"`
	ErrorBridges       int     `json:"error_bridges"`
	TotalEventsIn      int64   `json:"total_events_in"`
	TotalEventsOut     int64   `json:"total_events_out"`
	TotalEventsFailed  int64   `json:"total_events_failed"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	P99LatencyMs       float64 `json:"p99_latency_ms"`
	TotalBytesIn       int64   `json:"total_bytes_in"`
	TotalBytesOut      int64   `json:"total_bytes_out"`
	DeadLetterCount    int64   `json:"dead_letter_count"`
	ConsumerGroupCount int     `json:"consumer_group_count"`
}

// BridgeMonitorHealth represents the health of a single bridge for monitoring
type BridgeMonitorHealth struct {
	BridgeID     string       `json:"bridge_id"`
	BridgeName   string       `json:"bridge_name"`
	StreamType   StreamType   `json:"stream_type"`
	Status       BridgeStatus `json:"status"`
	HealthScore  float64      `json:"health_score"` // 0-100
	EventsPerSec float64      `json:"events_per_sec"`
	ErrorRate    float64      `json:"error_rate"`
	AvgLatencyMs float64      `json:"avg_latency_ms"`
	ConsumerLag  int64        `json:"consumer_lag"`
	LastEventAt  *time.Time   `json:"last_event_at,omitempty"`
}

// ThroughputDataPoint represents a data point in the throughput chart
type ThroughputDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	EventsIn   int64     `json:"events_in"`
	EventsOut  int64     `json:"events_out"`
	ErrorCount int64     `json:"error_count"`
	BytesIn    int64     `json:"bytes_in"`
	BytesOut   int64     `json:"bytes_out"`
}

// ErrorBreakdownEntry represents an error category breakdown
type ErrorBreakdownEntry struct {
	Category string  `json:"category"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

// ConsumerLagDataPoint represents consumer lag over time
type ConsumerLagDataPoint struct {
	Timestamp     time.Time     `json:"timestamp"`
	BridgeID      string        `json:"bridge_id"`
	BridgeName    string        `json:"bridge_name"`
	Lag           int64         `json:"lag"`
	PartitionLags map[int]int64 `json:"partition_lags,omitempty"`
}

// TopBridgeEntry represents a top-performing or underperforming bridge
type TopBridgeEntry struct {
	BridgeID     string     `json:"bridge_id"`
	BridgeName   string     `json:"bridge_name"`
	StreamType   StreamType `json:"stream_type"`
	EventCount   int64      `json:"event_count"`
	ErrorCount   int64      `json:"error_count"`
	AvgLatencyMs float64    `json:"avg_latency_ms"`
}

// StreamingAlertSeverity represents alert severity levels
type StreamingAlertSeverity string

const (
	AlertSeverityInfo     StreamingAlertSeverity = "info"
	AlertSeverityWarning  StreamingAlertSeverity = "warning"
	AlertSeverityCritical StreamingAlertSeverity = "critical"
)

// StreamingAlert represents a monitoring alert
type StreamingAlert struct {
	ID          string                 `json:"id"`
	BridgeID    string                 `json:"bridge_id"`
	BridgeName  string                 `json:"bridge_name"`
	Severity    StreamingAlertSeverity `json:"severity"`
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Threshold   float64                `json:"threshold,omitempty"`
	CurrentVal  float64                `json:"current_value,omitempty"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// MonitoringRepository defines storage for monitoring data
type MonitoringRepository interface {
	GetOverview(ctx context.Context, tenantID string) (*StreamingOverview, error)
	GetBridgeHealth(ctx context.Context, tenantID string) ([]BridgeMonitorHealth, error)
	GetThroughputHistory(ctx context.Context, tenantID string, start, end time.Time, interval string) ([]ThroughputDataPoint, error)
	GetErrorBreakdown(ctx context.Context, tenantID string, start, end time.Time) ([]ErrorBreakdownEntry, error)
	GetConsumerLag(ctx context.Context, tenantID string) ([]ConsumerLagDataPoint, error)
	GetTopBridges(ctx context.Context, tenantID string, limit int, sortBy string) ([]TopBridgeEntry, error)
	GetAlerts(ctx context.Context, tenantID string, activeOnly bool) ([]StreamingAlert, error)
	CreateAlert(ctx context.Context, alert *StreamingAlert) error
	ResolveAlert(ctx context.Context, alertID string) error
}

// MonitoringService provides monitoring dashboard operations
type MonitoringService struct {
	repo MonitoringRepository
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(repo MonitoringRepository) *MonitoringService {
	return &MonitoringService{repo: repo}
}

// GetDashboard generates the full monitoring dashboard
func (m *MonitoringService) GetDashboard(ctx context.Context, tenantID string) (*MonitoringDashboard, error) {
	overview, err := m.repo.GetOverview(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	health, _ := m.repo.GetBridgeHealth(ctx, tenantID)
	if health == nil {
		health = []BridgeMonitorHealth{}
	}

	end := time.Now()
	start := end.Add(-24 * time.Hour)
	throughput, _ := m.repo.GetThroughputHistory(ctx, tenantID, start, end, "1h")
	if throughput == nil {
		throughput = []ThroughputDataPoint{}
	}

	errorBreakdown, _ := m.repo.GetErrorBreakdown(ctx, tenantID, start, end)
	if errorBreakdown == nil {
		errorBreakdown = []ErrorBreakdownEntry{}
	}

	lag, _ := m.repo.GetConsumerLag(ctx, tenantID)
	if lag == nil {
		lag = []ConsumerLagDataPoint{}
	}

	topBridges, _ := m.repo.GetTopBridges(ctx, tenantID, 10, "event_count")
	if topBridges == nil {
		topBridges = []TopBridgeEntry{}
	}

	alerts, _ := m.repo.GetAlerts(ctx, tenantID, true)
	if alerts == nil {
		alerts = []StreamingAlert{}
	}

	return &MonitoringDashboard{
		TenantID:        tenantID,
		Overview:        overview,
		BridgeHealth:    health,
		ThroughputChart: throughput,
		ErrorBreakdown:  errorBreakdown,
		ConsumerLag:     lag,
		TopBridges:      topBridges,
		Alerts:          alerts,
		GeneratedAt:     time.Now(),
	}, nil
}

// CheckBridgeHealth evaluates bridge health and generates alerts
func (m *MonitoringService) CheckBridgeHealth(ctx context.Context, tenantID string) ([]StreamingAlert, error) {
	health, err := m.repo.GetBridgeHealth(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var newAlerts []StreamingAlert
	for _, h := range health {
		if h.ErrorRate > 0.1 {
			alert := StreamingAlert{
				BridgeID:    h.BridgeID,
				BridgeName:  h.BridgeName,
				Severity:    AlertSeverityCritical,
				Type:        "high_error_rate",
				Message:     "Error rate exceeds 10%",
				Threshold:   0.1,
				CurrentVal:  h.ErrorRate,
				TriggeredAt: time.Now(),
			}
			_ = m.repo.CreateAlert(ctx, &alert)
			newAlerts = append(newAlerts, alert)
		}

		if h.ConsumerLag > 10000 {
			alert := StreamingAlert{
				BridgeID:    h.BridgeID,
				BridgeName:  h.BridgeName,
				Severity:    AlertSeverityWarning,
				Type:        "high_consumer_lag",
				Message:     "Consumer lag exceeds 10,000 messages",
				Threshold:   10000,
				CurrentVal:  float64(h.ConsumerLag),
				TriggeredAt: time.Now(),
			}
			_ = m.repo.CreateAlert(ctx, &alert)
			newAlerts = append(newAlerts, alert)
		}

		if h.HealthScore < 50 {
			alert := StreamingAlert{
				BridgeID:    h.BridgeID,
				BridgeName:  h.BridgeName,
				Severity:    AlertSeverityWarning,
				Type:        "low_health_score",
				Message:     "Bridge health score below 50",
				Threshold:   50,
				CurrentVal:  h.HealthScore,
				TriggeredAt: time.Now(),
			}
			_ = m.repo.CreateAlert(ctx, &alert)
			newAlerts = append(newAlerts, alert)
		}
	}

	return newAlerts, nil
}

// GetAlerts retrieves active alerts
func (m *MonitoringService) GetAlerts(ctx context.Context, tenantID string, activeOnly bool) ([]StreamingAlert, error) {
	return m.repo.GetAlerts(ctx, tenantID, activeOnly)
}

// ResolveAlert resolves an alert
func (m *MonitoringService) ResolveAlert(ctx context.Context, alertID string) error {
	return m.repo.ResolveAlert(ctx, alertID)
}
