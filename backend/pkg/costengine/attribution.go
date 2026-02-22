package costengine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Attribution granularity constants
const (
	GranularityTenant   = "tenant"
	GranularityEndpoint = "endpoint"
	GranularityEvent    = "event_type"
	GranularityHourly   = "hourly"
	GranularityDaily    = "daily"
	GranularityMonthly  = "monthly"
)

// CostAttribution represents attributed cost for a specific dimension.
type CostAttributionEntry struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	EndpointID      string    `json:"endpoint_id,omitempty"`
	EventType       string    `json:"event_type,omitempty"`
	Period          string    `json:"period"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	DeliveryCount   int64     `json:"delivery_count"`
	SuccessCount    int64     `json:"success_count"`
	FailedCount     int64     `json:"failed_count"`
	RetryCount      int64     `json:"retry_count"`
	TotalCost       float64   `json:"total_cost"`
	DeliveryCost    float64   `json:"delivery_cost"`
	BandwidthCost   float64   `json:"bandwidth_cost"`
	ComputeCost     float64   `json:"compute_cost"`
	StorageCost     float64   `json:"storage_cost"`
	TotalBytesOut   int64     `json:"total_bytes_out"`
}

// TenantCostSummary provides a tenant-level cost overview.
type TenantCostSummary struct {
	TenantID           string                    `json:"tenant_id"`
	Period             string                    `json:"period"`
	TotalCost          float64                   `json:"total_cost"`
	DeliveryCount      int64                     `json:"delivery_count"`
	CostPerDelivery    float64                   `json:"cost_per_delivery"`
	TopEndpoints       []EndpointCostSummary     `json:"top_endpoints"`
	TopEventTypes      []EventTypeCostSummary    `json:"top_event_types"`
	CostTrend          []CostDataPoint           `json:"cost_trend"`
	ProjectedMonthlyCost float64                 `json:"projected_monthly_cost"`
}

// EndpointCostSummary summarizes costs per endpoint.
type EndpointCostSummary struct {
	EndpointID    string  `json:"endpoint_id"`
	DeliveryCount int64   `json:"delivery_count"`
	TotalCost     float64 `json:"total_cost"`
	CostPercent   float64 `json:"cost_percent"`
}

// EventTypeCostSummary summarizes costs per event type.
type EventTypeCostSummary struct {
	EventType     string  `json:"event_type"`
	DeliveryCount int64   `json:"delivery_count"`
	TotalCost     float64 `json:"total_cost"`
	CostPercent   float64 `json:"cost_percent"`
}

// CostDataPoint is a time-series cost data point.
type CostDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Cost      float64   `json:"cost"`
	Deliveries int64    `json:"deliveries"`
}

// ChargebackReport generates chargeback data for billing integration.
type ChargebackReport struct {
	ID           string                `json:"id"`
	TenantID     string                `json:"tenant_id"`
	PeriodStart  time.Time             `json:"period_start"`
	PeriodEnd    time.Time             `json:"period_end"`
	TotalCost    float64               `json:"total_cost"`
	LineItems    []ChargebackLineItem  `json:"line_items"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// ChargebackLineItem is a single line in a chargeback report.
type ChargebackLineItem struct {
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Quantity      int64   `json:"quantity"`
	UnitCost      float64 `json:"unit_cost"`
	TotalCost     float64 `json:"total_cost"`
}

// CostForecast predicts future costs.
type CostForecastResult struct {
	TenantID     string          `json:"tenant_id"`
	ForecastDays int             `json:"forecast_days"`
	DataPoints   []CostDataPoint `json:"data_points"`
	TotalCost    float64         `json:"total_cost"`
	Confidence   float64         `json:"confidence"`
}

// Request DTOs

type RecordCostEventRequest struct {
	EndpointID  string  `json:"endpoint_id" binding:"required"`
	EventType   string  `json:"event_type" binding:"required"`
	BytesOut    int64   `json:"bytes_out"`
	Success     bool    `json:"success"`
	IsRetry     bool    `json:"is_retry"`
}

type GetCostSummaryRequest struct {
	Period string `json:"period" form:"period"`
}

type GenerateChargebackRequest struct {
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
}

type GetCostForecastRequest struct {
	Days int `json:"days" form:"days"`
}

// In-memory cost tracking
type costTracker struct {
	mu           sync.RWMutex
	attributions map[string]map[string]*CostAttributionEntry // tenantID -> key -> attribution
}

var globalCostTracker = &costTracker{
	attributions: make(map[string]map[string]*CostAttributionEntry),
}

// resetCostTracker resets global state; used by tests to prevent state leakage.
func resetCostTracker() {
	globalCostTracker.mu.Lock()
	defer globalCostTracker.mu.Unlock()
	globalCostTracker.attributions = make(map[string]map[string]*CostAttributionEntry)
}

// Default cost rates
const (
	defaultCostPerDelivery = 0.0001
	defaultCostPerMB       = 0.01
	defaultCostPerRetry    = 0.00005
	defaultComputeRate     = 0.00002
)

// RecordCostEvent records a webhook delivery cost event.
func (s *Service) RecordCostEvent(ctx context.Context, tenantID string, req *RecordCostEventRequest) (*CostAttributionEntry, error) {
	if req.EndpointID == "" || req.EventType == "" {
		return nil, fmt.Errorf("endpoint_id and event_type are required")
	}

	globalCostTracker.mu.Lock()
	defer globalCostTracker.mu.Unlock()

	tenantMap, exists := globalCostTracker.attributions[tenantID]
	if !exists {
		tenantMap = make(map[string]*CostAttributionEntry)
		globalCostTracker.attributions[tenantID] = tenantMap
	}

	now := time.Now()
	key := fmt.Sprintf("%s:%s:%s", req.EndpointID, req.EventType, now.Format("2006-01-02"))
	attr, exists := tenantMap[key]
	if !exists {
		attr = &CostAttributionEntry{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			EndpointID:  req.EndpointID,
			EventType:   req.EventType,
			Period:       GranularityDaily,
			PeriodStart:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
			PeriodEnd:    time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
		}
		tenantMap[key] = attr
	}

	attr.DeliveryCount++
	attr.TotalBytesOut += req.BytesOut

	if req.Success {
		attr.SuccessCount++
	} else {
		attr.FailedCount++
	}
	if req.IsRetry {
		attr.RetryCount++
	}

	// Calculate costs
	attr.DeliveryCost = float64(attr.DeliveryCount) * defaultCostPerDelivery
	attr.BandwidthCost = float64(attr.TotalBytesOut) / (1024 * 1024) * defaultCostPerMB
	attr.ComputeCost = float64(attr.DeliveryCount) * defaultComputeRate
	attr.StorageCost = float64(attr.TotalBytesOut) / (1024 * 1024 * 1024) * 0.023 // $0.023/GB
	attr.TotalCost = attr.DeliveryCost + attr.BandwidthCost + attr.ComputeCost + attr.StorageCost

	return attr, nil
}

// GetTenantCostSummary generates a cost summary for a tenant.
func (s *Service) GetTenantCostSummary(ctx context.Context, tenantID, period string) (*TenantCostSummary, error) {
	globalCostTracker.mu.RLock()
	defer globalCostTracker.mu.RUnlock()

	summary := &TenantCostSummary{
		TenantID: tenantID,
		Period:   period,
	}

	tenantMap, exists := globalCostTracker.attributions[tenantID]
	if !exists {
		return summary, nil
	}

	endpointCosts := make(map[string]*EndpointCostSummary)
	eventCosts := make(map[string]*EventTypeCostSummary)

	for _, attr := range tenantMap {
		summary.TotalCost += attr.TotalCost
		summary.DeliveryCount += attr.DeliveryCount

		// Aggregate by endpoint
		ep, ok := endpointCosts[attr.EndpointID]
		if !ok {
			ep = &EndpointCostSummary{EndpointID: attr.EndpointID}
			endpointCosts[attr.EndpointID] = ep
		}
		ep.TotalCost += attr.TotalCost
		ep.DeliveryCount += attr.DeliveryCount

		// Aggregate by event type
		evt, ok := eventCosts[attr.EventType]
		if !ok {
			evt = &EventTypeCostSummary{EventType: attr.EventType}
			eventCosts[attr.EventType] = evt
		}
		evt.TotalCost += attr.TotalCost
		evt.DeliveryCount += attr.DeliveryCount
	}

	if summary.DeliveryCount > 0 {
		summary.CostPerDelivery = summary.TotalCost / float64(summary.DeliveryCount)
	}

	// Convert to sorted slices
	for _, ep := range endpointCosts {
		if summary.TotalCost > 0 {
			ep.CostPercent = ep.TotalCost / summary.TotalCost * 100
		}
		summary.TopEndpoints = append(summary.TopEndpoints, *ep)
	}
	sort.Slice(summary.TopEndpoints, func(i, j int) bool {
		return summary.TopEndpoints[i].TotalCost > summary.TopEndpoints[j].TotalCost
	})

	for _, evt := range eventCosts {
		if summary.TotalCost > 0 {
			evt.CostPercent = evt.TotalCost / summary.TotalCost * 100
		}
		summary.TopEventTypes = append(summary.TopEventTypes, *evt)
	}
	sort.Slice(summary.TopEventTypes, func(i, j int) bool {
		return summary.TopEventTypes[i].TotalCost > summary.TopEventTypes[j].TotalCost
	})

	// Project monthly cost
	summary.ProjectedMonthlyCost = summary.TotalCost * 30

	return summary, nil
}

// GenerateChargebackReport generates a chargeback report for billing.
func (s *Service) GenerateChargebackReport(ctx context.Context, tenantID string, req *GenerateChargebackRequest) (*ChargebackReport, error) {
	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start format: use YYYY-MM-DD")
	}
	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid period_end format: use YYYY-MM-DD")
	}

	globalCostTracker.mu.RLock()
	defer globalCostTracker.mu.RUnlock()

	report := &ChargebackReport{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		GeneratedAt: time.Now(),
	}

	tenantMap := globalCostTracker.attributions[tenantID]

	var totalDeliveries, totalRetries int64
	var totalBandwidth float64

	for _, attr := range tenantMap {
		totalDeliveries += attr.DeliveryCount
		totalRetries += attr.RetryCount
		totalBandwidth += float64(attr.TotalBytesOut)
		report.TotalCost += attr.TotalCost
	}

	report.LineItems = []ChargebackLineItem{
		{
			Description: "Webhook Deliveries",
			Category:    "compute",
			Quantity:    totalDeliveries,
			UnitCost:    defaultCostPerDelivery,
			TotalCost:   float64(totalDeliveries) * defaultCostPerDelivery,
		},
		{
			Description: "Bandwidth (MB)",
			Category:    "bandwidth",
			Quantity:    int64(totalBandwidth / (1024 * 1024)),
			UnitCost:    defaultCostPerMB,
			TotalCost:   totalBandwidth / (1024 * 1024) * defaultCostPerMB,
		},
		{
			Description: "Retries",
			Category:    "compute",
			Quantity:    totalRetries,
			UnitCost:    defaultCostPerRetry,
			TotalCost:   float64(totalRetries) * defaultCostPerRetry,
		},
	}

	return report, nil
}

// GetCostForecast generates a cost forecast.
func (s *Service) GetCostForecast(ctx context.Context, tenantID string, days int) (*CostForecastResult, error) {
	if days <= 0 {
		days = 30
	}

	summary, _ := s.GetTenantCostSummary(ctx, tenantID, GranularityDaily)

	dailyCost := summary.TotalCost
	if dailyCost == 0 {
		dailyCost = 0.01
	}

	forecast := &CostForecastResult{
		TenantID:     tenantID,
		ForecastDays: days,
		Confidence:   0.7,
	}

	for i := 0; i < days; i++ {
		forecast.DataPoints = append(forecast.DataPoints, CostDataPoint{
			Timestamp:  time.Now().AddDate(0, 0, i),
			Cost:       dailyCost * (1 + float64(i)*0.02), // 2% daily growth
			Deliveries: summary.DeliveryCount,
		})
		forecast.TotalCost += dailyCost * (1 + float64(i)*0.02)
	}

	return forecast, nil
}
