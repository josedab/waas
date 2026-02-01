package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"

	"github.com/google/uuid"
)

// Aggregator handles data aggregation jobs for analytics
type Aggregator struct {
	analyticsRepo repository.AnalyticsRepositoryInterface
	logger        *utils.Logger
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewAggregator creates a new analytics aggregator
func NewAggregator(analyticsRepo repository.AnalyticsRepositoryInterface, logger *utils.Logger) *Aggregator {
	return &Aggregator{
		analyticsRepo: analyticsRepo,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the aggregation workers
func (a *Aggregator) Start(ctx context.Context) {
	a.logger.Info("Starting analytics aggregation workers", nil)

	// Start hourly aggregation worker
	a.wg.Add(1)
	go a.hourlyAggregationWorker(ctx)

	// Start daily aggregation worker
	a.wg.Add(1)
	go a.dailyAggregationWorker(ctx)

	// Start cleanup worker
	a.wg.Add(1)
	go a.cleanupWorker(ctx)

	// Start real-time metrics worker
	a.wg.Add(1)
	go a.realtimeMetricsWorker(ctx)
}

// Stop gracefully stops all aggregation workers
func (a *Aggregator) Stop() {
	a.logger.Info("Stopping analytics aggregation workers", nil)
	close(a.stopCh)
	a.wg.Wait()
}

// hourlyAggregationWorker runs hourly data aggregation
func (a *Aggregator) hourlyAggregationWorker(ctx context.Context) {
	defer a.wg.Done()

	// Run immediately on start, then every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Process the previous hour on startup
	a.processHourlyAggregation(ctx, time.Now().Add(-time.Hour))

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case t := <-ticker.C:
			// Process the previous hour
			a.processHourlyAggregation(ctx, t.Add(-time.Hour))
		}
	}
}

// dailyAggregationWorker runs daily data aggregation
func (a *Aggregator) dailyAggregationWorker(ctx context.Context) {
	defer a.wg.Done()

	// Calculate next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	
	// Wait until midnight, then run daily
	timer := time.NewTimer(time.Until(nextMidnight))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-timer.C:
			// Process the previous day
			yesterday := time.Now().AddDate(0, 0, -1)
			a.processDailyAggregation(ctx, yesterday)
			
			// Reset timer for next day
			timer.Reset(24 * time.Hour)
		}
	}
}

// cleanupWorker periodically cleans up old data
func (a *Aggregator) cleanupWorker(ctx context.Context) {
	defer a.wg.Done()

	// Run cleanup every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.performCleanup(ctx)
		}
	}
}

// realtimeMetricsWorker generates real-time metrics
func (a *Aggregator) realtimeMetricsWorker(ctx context.Context) {
	defer a.wg.Done()

	// Generate real-time metrics every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.generateRealtimeMetrics(ctx)
		}
	}
}

// processHourlyAggregation aggregates data for a specific hour
func (a *Aggregator) processHourlyAggregation(ctx context.Context, hourTime time.Time) {
	// Truncate to hour boundary
	hourStart := hourTime.Truncate(time.Hour)
	hourEnd := hourStart.Add(time.Hour)

	a.logger.Info("Processing hourly aggregation", map[string]interface{}{
		"hour_start": hourStart,
		"hour_end":   hourEnd,
	})

	// Get all tenants that had activity in this hour
	tenants, err := a.getActiveTenantsInPeriod(ctx, hourStart, hourEnd)
	if err != nil {
		a.logger.Error("Failed to get active tenants for hourly aggregation", map[string]interface{}{
			"error": err.Error(),
			"hour":  hourStart,
		})
		return
	}

	for _, tenantID := range tenants {
		// Aggregate tenant-wide metrics
		err := a.aggregateHourlyMetricsForTenant(ctx, tenantID, nil, hourStart, hourEnd)
		if err != nil {
			a.logger.Error("Failed to aggregate hourly metrics for tenant", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
				"hour":      hourStart,
			})
			continue
		}

		// Aggregate per-endpoint metrics
		endpoints, err := a.getActiveEndpointsInPeriod(ctx, tenantID, hourStart, hourEnd)
		if err != nil {
			a.logger.Error("Failed to get active endpoints for hourly aggregation", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
				"hour":      hourStart,
			})
			continue
		}

		for _, endpointID := range endpoints {
			err := a.aggregateHourlyMetricsForTenant(ctx, tenantID, &endpointID, hourStart, hourEnd)
			if err != nil {
				a.logger.Error("Failed to aggregate hourly metrics for endpoint", map[string]interface{}{
					"error":       err.Error(),
					"tenant_id":   tenantID,
					"endpoint_id": endpointID,
					"hour":        hourStart,
				})
			}
		}
	}
}

// processDailyAggregation aggregates data for a specific day
func (a *Aggregator) processDailyAggregation(ctx context.Context, dayTime time.Time) {
	// Truncate to day boundary
	dayStart := time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(), 0, 0, 0, 0, dayTime.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	a.logger.Info("Processing daily aggregation", map[string]interface{}{
		"day_start": dayStart,
		"day_end":   dayEnd,
	})

	// Similar logic to hourly but for daily aggregation
	// This would aggregate from hourly_metrics table instead of raw delivery_metrics
	// Implementation would be similar but working with different time boundaries
}

// aggregateHourlyMetricsForTenant calculates and stores hourly metrics
func (a *Aggregator) aggregateHourlyMetricsForTenant(ctx context.Context, tenantID uuid.UUID, endpointID *uuid.UUID, hourStart, hourEnd time.Time) error {
	// Build the aggregation query
	query := &models.MetricsQuery{
		TenantID:  tenantID,
		StartDate: hourStart,
		EndDate:   hourEnd,
	}
	
	if endpointID != nil {
		query.EndpointIDs = []uuid.UUID{*endpointID}
	}

	// Get summary statistics
	summary, err := a.analyticsRepo.GetMetricsSummary(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get metrics summary: %w", err)
	}

	// Skip if no data
	if summary.TotalDeliveries == 0 {
		return nil
	}

	// Create hourly metric record
	hourlyMetric := &models.HourlyMetric{
		TenantID:             tenantID,
		EndpointID:           endpointID,
		HourTimestamp:        hourStart,
		TotalDeliveries:      summary.TotalDeliveries,
		SuccessfulDeliveries: summary.SuccessfulDeliveries,
		FailedDeliveries:     summary.FailedDeliveries,
		RetryingDeliveries:   summary.TotalDeliveries - summary.SuccessfulDeliveries - summary.FailedDeliveries,
		AvgLatencyMs:         &summary.AvgLatencyMs,
		P95LatencyMs:         &summary.P95LatencyMs,
		P99LatencyMs:         &summary.P99LatencyMs,
		TotalRetries:         0, // Would need to calculate from attempt_number > 1
	}

	// Store the aggregated metric
	return a.analyticsRepo.UpsertHourlyMetric(ctx, hourlyMetric)
}

// generateRealtimeMetrics creates real-time metrics for WebSocket updates
func (a *Aggregator) generateRealtimeMetrics(ctx context.Context) {
	// Get active tenants from the last few minutes
	since := time.Now().Add(-5 * time.Minute)
	tenants, err := a.getActiveTenantsInPeriod(ctx, since, time.Now())
	if err != nil {
		a.logger.Error("Failed to get active tenants for real-time metrics", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for _, tenantID := range tenants {
		// Calculate delivery rate (deliveries per minute)
		deliveryRate, err := a.calculateDeliveryRate(ctx, tenantID, 5*time.Minute)
		if err != nil {
			a.logger.Error("Failed to calculate delivery rate", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
			continue
		}

		// Record real-time metric
		realtimeMetric := &models.RealtimeMetric{
			TenantID:    tenantID,
			MetricType:  "delivery_rate",
			MetricValue: deliveryRate,
			Metadata: map[string]interface{}{
				"window_minutes": 5,
			},
		}

		err = a.analyticsRepo.RecordRealtimeMetric(ctx, realtimeMetric)
		if err != nil {
			a.logger.Error("Failed to record real-time metric", map[string]interface{}{
				"error":     err.Error(),
				"tenant_id": tenantID,
			})
		}
	}
}

// performCleanup removes old data based on retention policies
func (a *Aggregator) performCleanup(ctx context.Context) {
	a.logger.Info("Starting data cleanup", nil)

	// Clean up raw delivery metrics older than 30 days
	err := a.analyticsRepo.CleanupOldMetrics(ctx, 30)
	if err != nil {
		a.logger.Error("Failed to cleanup old metrics", map[string]interface{}{
			"error": err.Error(),
		})
	}

	a.logger.Info("Data cleanup completed", nil)
}

// Helper functions

func (a *Aggregator) getActiveTenantsInPeriod(ctx context.Context, start, end time.Time) ([]uuid.UUID, error) {
	// This would query the database to get tenants that had activity in the period
	// For now, returning empty slice - would need actual implementation
	return []uuid.UUID{}, nil
}

func (a *Aggregator) getActiveEndpointsInPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]uuid.UUID, error) {
	// This would query the database to get endpoints that had activity in the period
	// For now, returning empty slice - would need actual implementation
	return []uuid.UUID{}, nil
}

func (a *Aggregator) calculateDeliveryRate(ctx context.Context, tenantID uuid.UUID, window time.Duration) (float64, error) {
	// This would calculate deliveries per minute for the tenant
	// For now, returning 0 - would need actual implementation
	return 0.0, nil
}