package capacityplanner

import (
	"context"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.config.DefaultLookbackDays != 30 {
		t.Errorf("expected default lookback 30, got %d", svc.config.DefaultLookbackDays)
	}
	if svc.config.HighUtilThreshold != 80.0 {
		t.Errorf("expected default threshold 80.0, got %f", svc.config.HighUtilThreshold)
	}
}

func TestNewServiceCustomConfig(t *testing.T) {
	repo := NewMemoryRepository()
	config := &ServiceConfig{DefaultLookbackDays: 7, HighUtilThreshold: 90.0}
	svc := NewService(repo, config)
	if svc.config.DefaultLookbackDays != 7 {
		t.Errorf("expected lookback 7, got %d", svc.config.DefaultLookbackDays)
	}
}

func seedTrafficHistory(repo *MemoryRepository, tenantID string, count int) {
	now := time.Now()
	for i := 0; i < count; i++ {
		_ = repo.SaveTrafficSnapshot(tenantID, &TrafficSnapshot{
			Timestamp:       now.Add(-time.Duration(count-i) * time.Hour),
			RequestsPerSec:  100 + float64(i*10),
			AvgLatencyMs:    50 + float64(i),
			P99LatencyMs:    200 + float64(i*2),
			ErrorRate:       0.01,
			ActiveEndpoints: 5 + i%3,
			QueueDepth:      10 + i,
		})
	}
}

func TestGenerateReport(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	seedTrafficHistory(repo, "tenant-1", 10)

	req := &GenerateReportRequest{
		PeriodStart: time.Now().AddDate(0, 0, -30),
		PeriodEnd:   time.Now(),
	}

	report, err := svc.GenerateReport(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ID == "" {
		t.Error("expected non-empty report ID")
	}
	if report.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", report.TenantID)
	}
	if len(report.Projections) != 4 {
		t.Errorf("expected 4 projections, got %d", len(report.Projections))
	}
	if report.GeneratedAt.IsZero() {
		t.Error("expected non-zero generated_at")
	}
	if report.CurrentUsage.DailyAvgRequests <= 0 {
		t.Error("expected positive daily avg requests")
	}

	// Verify report is retrievable
	fetched, err := svc.GetReport(ctx, report.ID)
	if err != nil {
		t.Fatalf("failed to get report: %v", err)
	}
	if fetched.ID != report.ID {
		t.Error("fetched report ID mismatch")
	}
}

func TestGenerateReportNoData(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	req := &GenerateReportRequest{
		PeriodStart: time.Now().AddDate(0, 0, -30),
		PeriodEnd:   time.Now(),
	}

	report, err := svc.GenerateReport(ctx, "empty-tenant", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.CurrentUsage.DailyAvgRequests != 0 {
		t.Errorf("expected 0 daily avg requests for empty tenant, got %f", report.CurrentUsage.DailyAvgRequests)
	}
}

func TestGetCurrentUsage(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	seedTrafficHistory(repo, "tenant-1", 20)

	usage, err := svc.GetCurrentUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.DailyAvgRequests <= 0 {
		t.Error("expected positive daily avg requests")
	}
	if usage.PeakRequestsPerSec <= 0 {
		t.Error("expected positive peak RPS")
	}
	if usage.ActiveEndpoints <= 0 {
		t.Error("expected positive active endpoints")
	}
}

func TestGetCurrentUsageEmpty(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	usage, err := svc.GetCurrentUsage(ctx, "no-data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.DailyAvgRequests != 0 {
		t.Errorf("expected 0 for empty tenant, got %f", usage.DailyAvgRequests)
	}
}

func TestGetProjections(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	seedTrafficHistory(repo, "tenant-1", 10)

	projections, err := svc.GetProjections(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projections) != 4 {
		t.Fatalf("expected 4 projections, got %d", len(projections))
	}

	expectedPeriods := []string{"30d", "90d", "180d", "365d"}
	for i, p := range projections {
		if p.Period != expectedPeriods[i] {
			t.Errorf("expected period %s, got %s", expectedPeriods[i], p.Period)
		}
		if p.ConfidenceLevel <= 0 || p.ConfidenceLevel > 1.0 {
			t.Errorf("expected confidence between 0 and 1, got %f", p.ConfidenceLevel)
		}
		if p.GrowthRatePct <= 0 {
			t.Errorf("expected positive growth rate for %s, got %f", p.Period, p.GrowthRatePct)
		}
	}

	// Confidence should decrease with longer periods
	if projections[0].ConfidenceLevel <= projections[3].ConfidenceLevel {
		t.Error("expected confidence to decrease over longer periods")
	}
}

func TestGetRecommendations(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	// Seed high-traffic data to trigger recommendations
	now := time.Now()
	for i := 0; i < 10; i++ {
		_ = repo.SaveTrafficSnapshot("tenant-high", &TrafficSnapshot{
			Timestamp:       now.Add(-time.Duration(10-i) * time.Hour),
			RequestsPerSec:  2000 + float64(i*100),
			AvgLatencyMs:    100,
			P99LatencyMs:    500,
			ErrorRate:       0.05,
			ActiveEndpoints: 20,
			QueueDepth:      50,
		})
	}

	recs, err := svc.GetRecommendations(ctx, "tenant-high")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recs) == 0 {
		t.Error("expected at least one recommendation for high-traffic tenant")
	}

	foundCompute := false
	for _, rec := range recs {
		if rec.Resource == "compute" {
			foundCompute = true
			if rec.Urgency == "" {
				t.Error("expected non-empty urgency")
			}
		}
	}
	if !foundCompute {
		t.Error("expected compute recommendation for high-traffic tenant")
	}
}

func TestGetRecommendationsLowTraffic(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	// Seed very low traffic that won't trigger any recommendations
	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = repo.SaveTrafficSnapshot("tenant-low", &TrafficSnapshot{
			Timestamp:       now.Add(-time.Duration(5-i) * time.Hour),
			RequestsPerSec:  0.5,
			AvgLatencyMs:    10,
			P99LatencyMs:    20,
			ErrorRate:       0.0,
			ActiveEndpoints: 1,
			QueueDepth:      0,
		})
	}

	recs, err := svc.GetRecommendations(ctx, "tenant-low")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Very low traffic should produce no recommendations
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations for low traffic, got %d", len(recs))
	}
}

func TestSetAlertThreshold(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	req := &SetAlertThresholdRequest{
		Resource:       "compute",
		ThresholdValue: 85.0,
		Severity:       AlertSeverityWarning,
	}

	err := svc.SetAlertThreshold(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify threshold was persisted
	val, err := repo.GetAlertThreshold("tenant-1", "compute")
	if err != nil {
		t.Fatalf("failed to get threshold: %v", err)
	}
	if val != 85.0 {
		t.Errorf("expected threshold 85.0, got %f", val)
	}
}

func TestSetAlertThresholdMultipleResources(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	resources := []struct {
		resource  string
		threshold float64
	}{
		{"compute", 80.0},
		{"storage", 90.0},
		{"bandwidth", 75.0},
	}

	for _, r := range resources {
		err := svc.SetAlertThreshold(ctx, "tenant-1", &SetAlertThresholdRequest{
			Resource:       r.resource,
			ThresholdValue: r.threshold,
			Severity:       AlertSeverityWarning,
		})
		if err != nil {
			t.Fatalf("failed to set threshold for %s: %v", r.resource, err)
		}
	}

	for _, r := range resources {
		val, err := repo.GetAlertThreshold("tenant-1", r.resource)
		if err != nil {
			t.Fatalf("failed to get threshold for %s: %v", r.resource, err)
		}
		if val != r.threshold {
			t.Errorf("expected %s threshold %f, got %f", r.resource, r.threshold, val)
		}
	}
}

func TestAcknowledgeAlert(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	alert := &CapacityAlert{
		ID:       "alert-1",
		TenantID: "tenant-1",
		Resource: "compute",
		Severity: AlertSeverityCritical,
		Message:  "CPU utilization exceeded threshold",
	}
	_ = repo.SaveAlert(alert)

	err := svc.AcknowledgeAlert(ctx, "alert-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Alert should be removed
	_, err = repo.GetAlert("alert-1")
	if err == nil {
		t.Error("expected alert to be removed after acknowledgment")
	}
}

func TestAcknowledgeAlertNotFound(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	err := svc.AcknowledgeAlert(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent alert")
	}
}

func TestForecastCosts(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	seedTrafficHistory(repo, "tenant-1", 10)

	for _, provider := range []string{"aws", "gcp", "azure"} {
		forecast, err := svc.ForecastCosts(ctx, "tenant-1", provider)
		if err != nil {
			t.Fatalf("ForecastCosts(%s) failed: %v", provider, err)
		}
		if forecast.Provider != provider {
			t.Errorf("expected provider %s, got %s", provider, forecast.Provider)
		}
		if forecast.CurrentMonthlyCost <= 0 {
			t.Errorf("expected positive cost for %s", provider)
		}
		if len(forecast.CostBreakdown) != 3 {
			t.Errorf("expected 3 cost line items, got %d", len(forecast.CostBreakdown))
		}
		if len(forecast.ProjectedCosts) != 4 {
			t.Errorf("expected 4 projected costs, got %d", len(forecast.ProjectedCosts))
		}
	}

	_, err := svc.ForecastCosts(ctx, "tenant-1", "invalid")
	if err == nil {
		t.Error("expected error for invalid provider")
	}
}

func TestGenerateWeeklyDigest(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)
	ctx := context.Background()

	seedTrafficHistory(repo, "tenant-1", 10)

	digest, err := svc.GenerateWeeklyDigest(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GenerateWeeklyDigest failed: %v", err)
	}
	if digest.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", digest.TenantID)
	}
	if digest.TrendDirection == "" {
		t.Error("expected non-empty trend direction")
	}
	if digest.GeneratedAt.IsZero() {
		t.Error("expected non-zero generated_at")
	}
}
