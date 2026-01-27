package dlq

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func seedService(t *testing.T) *Service {
	t.Helper()
	svc := NewService()

	now := time.Now()
	entries := []*DLQEntry{
		{
			ID: "e1", TenantID: "t1", EndpointID: "ep1", OriginalDeliveryID: "d1",
			Payload: json.RawMessage(`{"event":"order.created"}`),
			Headers: json.RawMessage(`{"Content-Type":"application/json"}`),
			AllAttempts: []AttemptDetail{
				{AttemptNumber: 1, HTTPStatus: intPtr(500), ErrorMessage: strPtr("internal server error"), AttemptedAt: now.Add(-2 * time.Hour), DurationMs: 120},
			},
			ErrorType: "server_error", FinalStatus: "dead_letter", FinalHTTPStatus: intPtr(500),
			RetryCount: 1, MaxRetries: 3, CreatedAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(29 * 24 * time.Hour),
		},
		{
			ID: "e2", TenantID: "t1", EndpointID: "ep2", OriginalDeliveryID: "d2",
			Payload: json.RawMessage(`{"event":"payment.failed"}`),
			Headers: json.RawMessage(`{"Content-Type":"application/json"}`),
			AllAttempts: []AttemptDetail{
				{AttemptNumber: 1, HTTPStatus: intPtr(429), ErrorMessage: strPtr("rate limit exceeded"), AttemptedAt: now.Add(-1 * time.Hour), DurationMs: 50},
			},
			ErrorType: "rate_limit", FinalStatus: "dead_letter", FinalHTTPStatus: intPtr(429),
			RetryCount: 1, MaxRetries: 3, CreatedAt: now.Add(-12 * time.Hour), ExpiresAt: now.Add(29 * 24 * time.Hour),
		},
		{
			ID: "e3", TenantID: "t1", EndpointID: "ep1", OriginalDeliveryID: "d3",
			Payload: json.RawMessage(`{"event":"user.signup"}`),
			Headers: json.RawMessage(`{"Content-Type":"application/json"}`),
			AllAttempts: []AttemptDetail{
				{AttemptNumber: 1, ErrorMessage: strPtr("connection timeout"), AttemptedAt: now.Add(-30 * time.Minute), DurationMs: 30000},
			},
			ErrorType: "timeout", FinalStatus: "dead_letter",
			RetryCount: 1, MaxRetries: 3, CreatedAt: now.Add(-6 * time.Hour), ExpiresAt: now.Add(29 * 24 * time.Hour),
		},
		{
			ID: "e4", TenantID: "t2", EndpointID: "ep3", OriginalDeliveryID: "d4",
			Payload: json.RawMessage(`{"event":"invoice.sent"}`),
			Headers: json.RawMessage(`{"Content-Type":"application/json"}`),
			AllAttempts: []AttemptDetail{
				{AttemptNumber: 1, HTTPStatus: intPtr(502), ErrorMessage: strPtr("bad gateway"), AttemptedAt: now.Add(-3 * time.Hour), DurationMs: 200},
			},
			ErrorType: "server_error", FinalStatus: "dead_letter", FinalHTTPStatus: intPtr(502),
			RetryCount: 1, MaxRetries: 3, CreatedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(28 * 24 * time.Hour),
		},
		{
			ID: "e5", TenantID: "t1", EndpointID: "ep1", OriginalDeliveryID: "d5",
			Payload: json.RawMessage(`{"event":"order.updated"}`),
			Headers: json.RawMessage(`{"Content-Type":"application/json"}`),
			AllAttempts: []AttemptDetail{
				{AttemptNumber: 1, HTTPStatus: intPtr(500), ErrorMessage: strPtr("internal server error"), AttemptedAt: now.Add(-4 * time.Hour), DurationMs: 100},
			},
			ErrorType: "server_error", FinalStatus: "replayed", FinalHTTPStatus: intPtr(500),
			RetryCount: 1, MaxRetries: 3, CreatedAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(27 * 24 * time.Hour),
			Replayed: true, ReplayedAt: timePtr(now.Add(-1 * time.Hour)),
		},
	}

	for _, e := range entries {
		svc.entries[e.ID] = e
	}
	return svc
}

func timePtr(t time.Time) *time.Time { return &t }

// ---------- NewService ----------

func TestNewService(t *testing.T) {
	svc := NewService()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.entries)
	assert.NotNil(t, svc.alertRules)
	assert.NotNil(t, svc.retentionPolicies)
}

// ---------- GetEntries ----------

func TestGetEntries_EmptyStore(t *testing.T) {
	svc := NewService()
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestGetEntries_FilterByTenant(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1"})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	for _, e := range entries {
		assert.Equal(t, "t1", e.TenantID)
	}
}

func TestGetEntries_FilterByErrorType(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", ErrorType: "timeout"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "e3", entries[0].ID)
}

func TestGetEntries_FilterByStatus(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Status: "replayed"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "e5", entries[0].ID)
}

func TestGetEntries_FilterByDateRange(t *testing.T) {
	svc := seedService(t)
	now := time.Now()
	from := now.Add(-13 * time.Hour)
	to := now.Add(-5 * time.Hour)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", DateFrom: &from, DateTo: &to})
	require.NoError(t, err)
	// e2 created -12h, e3 created -6h match
	assert.Equal(t, 2, total)
	ids := map[string]bool{}
	for _, e := range entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["e2"])
	assert.True(t, ids["e3"])
}

func TestGetEntries_Pagination(t *testing.T) {
	svc := seedService(t)
	// Get all for t1 (4 entries) with limit 2
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, entries, 2)

	// Second page
	entries2, total2, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, 4, total2)
	assert.Len(t, entries2, 2)

	// Beyond range
	entries3, total3, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Limit: 2, Offset: 10})
	require.NoError(t, err)
	assert.Equal(t, 4, total3)
	assert.Empty(t, entries3)
}

func TestGetEntries_SearchQuery(t *testing.T) {
	svc := seedService(t)

	// Search in payload
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "payment"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "e2", entries[0].ID)

	// Search in error message (case-insensitive)
	entries, total, err = svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "TIMEOUT"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "e3", entries[0].ID)

	// No match
	entries, total, err = svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "nonexistent"})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

// ---------- GetEntry ----------

func TestGetEntry_Found(t *testing.T) {
	svc := seedService(t)
	entry, err := svc.GetEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)
	assert.Equal(t, "e1", entry.ID)
	assert.Equal(t, "t1", entry.TenantID)
}

func TestGetEntry_NotFound(t *testing.T) {
	svc := seedService(t)
	_, err := svc.GetEntry(context.Background(), "t1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEntry_WrongTenant(t *testing.T) {
	svc := seedService(t)
	_, err := svc.GetEntry(context.Background(), "t2", "e1") // e1 belongs to t1
	assert.Error(t, err)
}

// ---------- ReplayEntry ----------

func TestReplayEntry_Success(t *testing.T) {
	svc := seedService(t)
	entry, err := svc.ReplayEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)
	assert.True(t, entry.Replayed)
	assert.NotNil(t, entry.ReplayedAt)
	assert.Equal(t, "replayed", entry.FinalStatus)
}

func TestReplayEntry_AlreadyReplayed(t *testing.T) {
	svc := seedService(t)
	_, err := svc.ReplayEntry(context.Background(), "t1", "e5") // e5 is already replayed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already been replayed")
}

func TestReplayEntry_NotFound(t *testing.T) {
	svc := seedService(t)
	_, err := svc.ReplayEntry(context.Background(), "t1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------- BulkRetry ----------

func TestBulkRetry_WithFilter(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{ErrorType: "server_error"},
		MaxCount: 10,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.False(t, result.DryRun)
	// e1 is server_error+dead_letter for t1; e5 is already replayed so skipped
	assert.Equal(t, 1, result.Requested)
	assert.Equal(t, 1, result.Succeeded)

	// Verify e1 is now replayed
	entry, _ := svc.GetEntry(context.Background(), "t1", "e1")
	assert.True(t, entry.Replayed)
}

func TestBulkRetry_DryRun(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{ErrorType: "rate_limit"},
		MaxCount: 10,
		DryRun:   true,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, 1, result.Requested)
	assert.Equal(t, 1, result.Succeeded)

	// Verify e2 is NOT actually replayed
	entry, _ := svc.GetEntry(context.Background(), "t1", "e2")
	assert.False(t, entry.Replayed)
}

// ---------- GetStats ----------

func TestGetStats_Accurate(t *testing.T) {
	svc := seedService(t)
	stats, err := svc.GetStats(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats.TotalEntries)
	assert.Equal(t, int64(3), stats.PendingCount)  // e1,e2,e3
	assert.Equal(t, int64(1), stats.ReplayedCount) // e5
	assert.NotEmpty(t, stats.OldestEntryAge)
}

func TestGetStats_Empty(t *testing.T) {
	svc := NewService()
	stats, err := svc.GetStats(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
	assert.Equal(t, int64(0), stats.PendingCount)
	assert.Equal(t, int64(0), stats.ReplayedCount)
	assert.Empty(t, stats.OldestEntryAge)
}

func TestGetStats_DynamicAfterReplay(t *testing.T) {
	svc := seedService(t)

	// Replay an entry then verify stats change
	_, err := svc.ReplayEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)

	stats, err := svc.GetStats(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats.TotalEntries)
	assert.Equal(t, int64(2), stats.PendingCount)  // e2,e3
	assert.Equal(t, int64(2), stats.ReplayedCount) // e1,e5
}

// ---------- AlertRules ----------

func TestCreateAndGetAlertRules(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	rule := &AlertRule{
		Name: "high error rate",
		Condition: AlertCondition{
			MetricType:    "error_count",
			Threshold:     100,
			WindowMinutes: 5,
			Operator:      "gt",
		},
		Action:  "webhook",
		Enabled: true,
	}

	created, err := svc.CreateAlertRule(ctx, "t1", rule)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "t1", created.TenantID)
	assert.False(t, created.CreatedAt.IsZero())

	rules, err := svc.GetAlertRules(ctx, "t1")
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, "high error rate", rules[0].Name)

	// Different tenant sees nothing
	rules, err = svc.GetAlertRules(ctx, "t2")
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// ---------- RetentionPolicy ----------

func TestGetRetentionPolicy_Default(t *testing.T) {
	svc := NewService()
	policy, err := svc.GetRetentionPolicy(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, 30, policy.RetentionDays)
	assert.Equal(t, int64(10000), policy.MaxEntries)
	assert.True(t, policy.AutoPurge)
}

func TestUpdateRetentionPolicy(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	updated, err := svc.UpdateRetentionPolicy(ctx, "t1", &RetentionPolicy{
		RetentionDays:     90,
		MaxEntries:        50000,
		CompressAfterDays: 14,
		AutoPurge:         false,
	})
	require.NoError(t, err)
	assert.Equal(t, 90, updated.RetentionDays)
	assert.Equal(t, "t1", updated.TenantID)

	// Re-read
	policy, err := svc.GetRetentionPolicy(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 90, policy.RetentionDays)
	assert.False(t, policy.AutoPurge)
}

// ---------- ExportEntries ----------

func TestExportEntries_JSON(t *testing.T) {
	svc := seedService(t)
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{Format: "json"})
	require.NoError(t, err)

	var entries []DLQEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	assert.Len(t, entries, 4)
}

func TestExportEntries_CSV(t *testing.T) {
	svc := seedService(t)
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{Format: "csv"})
	require.NoError(t, err)

	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	require.NoError(t, err)
	// header + 4 entries
	assert.Len(t, records, 5)
	assert.Equal(t, "id", records[0][0])
}

// ---------- RouteToDeadLetter ----------

func TestRouteToDeadLetter_ServerError(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(500), ErrorMessage: strPtr("internal server error"), AttemptedAt: time.Now(), DurationMs: 100},
		{AttemptNumber: 2, HTTPStatus: intPtr(500), ErrorMessage: strPtr("internal server error"), AttemptedAt: time.Now(), DurationMs: 120},
	}

	entry, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "del-1",
		json.RawMessage(`{"event":"test"}`),
		json.RawMessage(`{"X-Custom":"val"}`),
		attempts, "internal server error",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, "t1", entry.TenantID)
	assert.Equal(t, "ep1", entry.EndpointID)
	assert.Equal(t, "del-1", entry.OriginalDeliveryID)
	assert.Equal(t, "server_error", entry.ErrorType)
	assert.Equal(t, "dead_letter", entry.FinalStatus)
	assert.Equal(t, 2, entry.RetryCount)
	assert.False(t, entry.Replayed)

	// Verify it's in the store
	fetched, err := svc.GetEntry(ctx, "t1", entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, fetched.ID)
}

func TestRouteToDeadLetter_RateLimit(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(429), ErrorMessage: strPtr("too many requests"), AttemptedAt: time.Now(), DurationMs: 10},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-2",
		json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "rate limited")
	require.NoError(t, err)
	assert.Equal(t, "rate_limit", entry.ErrorType)
}

func TestRouteToDeadLetter_Timeout(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, ErrorMessage: strPtr("connection timeout"), AttemptedAt: time.Now(), DurationMs: 30000},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-3",
		json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "timeout")
	require.NoError(t, err)
	assert.Equal(t, "timeout", entry.ErrorType)
}

func TestRouteToDeadLetter_ClientError(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(404), ErrorMessage: strPtr("not found"), AttemptedAt: time.Now(), DurationMs: 20},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-4",
		json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "not found")
	require.NoError(t, err)
	assert.Equal(t, "client_error", entry.ErrorType)
}

func TestRouteToDeadLetter_NoAttempts(t *testing.T) {
	svc := NewService()
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-5",
		json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "unknown failure")
	require.NoError(t, err)
	assert.Equal(t, "unknown", entry.ErrorType)
	assert.Equal(t, 0, entry.RetryCount)
}

// ---------- Stats after RouteToDeadLetter ----------

func TestGetStats_AfterRouteToDeadLetter(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(500), ErrorMessage: strPtr("error"), AttemptedAt: time.Now(), DurationMs: 50},
	}

	_, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "d1", json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "err")
	require.NoError(t, err)
	_, err = svc.RouteToDeadLetter(ctx, "t1", "ep2", "d2", json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "err")
	require.NoError(t, err)

	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalEntries)
	assert.Equal(t, int64(2), stats.PendingCount)
	assert.Equal(t, int64(0), stats.ReplayedCount)
}
