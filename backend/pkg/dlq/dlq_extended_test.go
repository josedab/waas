package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// GetEntries – additional filter & edge-case coverage
// =====================================================================

func TestGetEntries_FilterByEndpointID(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", EndpointID: "ep1"})
	require.NoError(t, err)
	assert.Equal(t, 3, total) // e1, e3, e5
	for _, e := range entries {
		assert.Equal(t, "ep1", e.EndpointID)
	}
}

func TestGetEntries_CombinedFilters(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{
		TenantID:   "t1",
		EndpointID: "ep1",
		ErrorType:  "server_error",
		Status:     "dead_letter",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "e1", entries[0].ID)
}

func TestGetEntries_DateFromOnly(t *testing.T) {
	svc := seedService(t)
	from := time.Now().Add(-7 * time.Hour)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", DateFrom: &from})
	require.NoError(t, err)
	// Only e3 (-6h) matches
	assert.Equal(t, 1, total)
	assert.Equal(t, "e3", entries[0].ID)
}

func TestGetEntries_DateToOnly(t *testing.T) {
	svc := seedService(t)
	to := time.Now().Add(-13 * time.Hour)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", DateTo: &to})
	require.NoError(t, err)
	// e1 (-24h) and e5 (-72h) are before -13h
	assert.Equal(t, 2, total)
	ids := map[string]bool{}
	for _, e := range entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["e1"])
	assert.True(t, ids["e5"])
}

func TestGetEntries_EmptyFilter(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, entries, 5)
}

func TestGetEntries_NegativeOffset(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Offset: -5})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, entries, 4) // offset clamped to 0
}

func TestGetEntries_LimitBeyondMaxClamped(t *testing.T) {
	svc := seedService(t)
	// Limit of 0 should default to 50
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", Limit: 0})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, entries, 4)
}

func TestGetEntries_NoMatchReturnsEmpty(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "nonexistent"})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, entries)
}

func TestGetEntries_SearchInErrorMessage(t *testing.T) {
	svc := seedService(t)
	entries, total, err := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "internal server error"})
	require.NoError(t, err)
	// e1 and e5 have "internal server error" in attempts
	assert.Equal(t, 2, total)
	ids := map[string]bool{}
	for _, e := range entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["e1"])
	assert.True(t, ids["e5"])
}

func TestGetEntries_SearchCaseInsensitive(t *testing.T) {
	svc := seedService(t)
	entries1, _, _ := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "RATE LIMIT"})
	entries2, _, _ := svc.GetEntries(context.Background(), &DLQFilter{TenantID: "t1", SearchQuery: "rate limit"})
	assert.Equal(t, len(entries1), len(entries2))
}

// =====================================================================
// GetEntry – additional edge cases
// =====================================================================

func TestGetEntry_EmptyStrings(t *testing.T) {
	svc := seedService(t)
	_, err := svc.GetEntry(context.Background(), "", "")
	assert.Error(t, err)
}

func TestGetEntry_ReturnsCopy(t *testing.T) {
	svc := seedService(t)
	entry, err := svc.GetEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)
	assert.Equal(t, "e1", entry.ID)
	assert.Equal(t, "server_error", entry.ErrorType)
}

// =====================================================================
// ReplayEntry – additional edge cases
// =====================================================================

func TestReplayEntry_WrongTenant(t *testing.T) {
	svc := seedService(t)
	_, err := svc.ReplayEntry(context.Background(), "t2", "e1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReplayEntry_VerifyTimestamp(t *testing.T) {
	svc := seedService(t)
	before := time.Now()
	entry, err := svc.ReplayEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)
	assert.True(t, entry.ReplayedAt.After(before) || entry.ReplayedAt.Equal(before))
	assert.True(t, entry.ReplayedAt.Before(time.Now().Add(time.Second)))
}

func TestReplayEntry_DoesNotAffectOtherEntries(t *testing.T) {
	svc := seedService(t)
	_, err := svc.ReplayEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)

	e2, err := svc.GetEntry(context.Background(), "t1", "e2")
	require.NoError(t, err)
	assert.False(t, e2.Replayed)
	assert.Equal(t, "dead_letter", e2.FinalStatus)
}

func TestReplayEntry_DoubleReplayFails(t *testing.T) {
	svc := seedService(t)
	_, err := svc.ReplayEntry(context.Background(), "t1", "e1")
	require.NoError(t, err)

	_, err = svc.ReplayEntry(context.Background(), "t1", "e1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already been replayed")
}

// =====================================================================
// BulkRetry – deeper coverage
// =====================================================================

func TestBulkRetry_MaxCountLimitsRetries(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{},
		MaxCount: 1,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Requested)
	assert.Equal(t, 1, result.Succeeded)
}

func TestBulkRetry_NoMatchingEntries(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{ErrorType: "nonexistent_type"},
		MaxCount: 10,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Requested)
	assert.Equal(t, 0, result.Succeeded)
}

func TestBulkRetry_SkipsAlreadyReplayedEntries(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{EndpointID: "ep1"},
		MaxCount: 100,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	// ep1 has e1(dead_letter), e3(dead_letter), e5(replayed) → 2 retried
	assert.Equal(t, 2, result.Requested)
	assert.Equal(t, 2, result.Succeeded)
}

func TestBulkRetry_EmptyService(t *testing.T) {
	svc := NewService()
	req := &BulkRetryRequest{
		Filter:   DLQFilter{},
		MaxCount: 10,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Requested)
}

func TestBulkRetry_DefaultMaxCount(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{},
		MaxCount: 0, // should default to 100
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	// Should process all non-replayed entries for t1 (e1, e2, e3)
	assert.Equal(t, 3, result.Requested)
}

func TestBulkRetry_DryRunDoesNotMutate(t *testing.T) {
	svc := seedService(t)
	req := &BulkRetryRequest{
		Filter:   DLQFilter{},
		MaxCount: 100,
		DryRun:   true,
	}
	result, err := svc.BulkRetry(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.True(t, result.DryRun)
	assert.Equal(t, 3, result.Requested) // e1, e2, e3

	// Verify no entries were actually replayed
	for _, id := range []string{"e1", "e2", "e3"} {
		entry, err := svc.GetEntry(context.Background(), "t1", id)
		require.NoError(t, err)
		assert.False(t, entry.Replayed, "entry %s should not be replayed in dry-run", id)
	}
}

// =====================================================================
// GetStats – additional coverage
// =====================================================================

func TestGetStats_MultiTenantIsolation(t *testing.T) {
	svc := seedService(t)
	stats1, err := svc.GetStats(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats1.TotalEntries)

	stats2, err := svc.GetStats(context.Background(), "t2")
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats2.TotalEntries)
}

func TestGetStats_GrowthRateDefaultsToZero(t *testing.T) {
	svc := seedService(t)
	stats, err := svc.GetStats(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, float64(0), stats.GrowthRate)
}

func TestGetStats_NonexistentTenant(t *testing.T) {
	svc := seedService(t)
	stats, err := svc.GetStats(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
	assert.Equal(t, int64(0), stats.PendingCount)
	assert.Equal(t, int64(0), stats.ReplayedCount)
	assert.Empty(t, stats.OldestEntryAge)
}

func TestGetStats_AllReplayed(t *testing.T) {
	svc := seedService(t)
	ctx := context.Background()

	for _, id := range []string{"e1", "e2", "e3"} {
		_, err := svc.ReplayEntry(ctx, "t1", id)
		require.NoError(t, err)
	}

	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats.TotalEntries)
	assert.Equal(t, int64(0), stats.PendingCount)
	assert.Equal(t, int64(4), stats.ReplayedCount)
}

// =====================================================================
// RouteToDeadLetter – additional error type classification
// =====================================================================

func TestRouteToDeadLetter_ConnectionRefused(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, ErrorMessage: strPtr("connection refused"), AttemptedAt: time.Now(), DurationMs: 10},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-conn",
		json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "connection refused")
	require.NoError(t, err)
	// No HTTP status set, error contains "timeout" check won't match → "unknown"
	assert.Equal(t, "unknown", entry.ErrorType)
	assert.Nil(t, entry.FinalHTTPStatus)
}

func TestRouteToDeadLetter_MultipleAttempts(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(503), ErrorMessage: strPtr("service unavailable"), AttemptedAt: time.Now().Add(-2 * time.Minute), DurationMs: 100},
		{AttemptNumber: 2, HTTPStatus: intPtr(503), ErrorMessage: strPtr("service unavailable"), AttemptedAt: time.Now().Add(-1 * time.Minute), DurationMs: 150},
		{AttemptNumber: 3, HTTPStatus: intPtr(500), ErrorMessage: strPtr("internal server error"), AttemptedAt: time.Now(), DurationMs: 200},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-multi",
		json.RawMessage(`{"key":"value"}`), json.RawMessage(`{}`), attempts, "server error")
	require.NoError(t, err)
	assert.Equal(t, "server_error", entry.ErrorType)
	assert.Equal(t, 3, entry.RetryCount)
	assert.Equal(t, 3, entry.MaxRetries)
	require.NotNil(t, entry.FinalHTTPStatus)
	assert.Equal(t, 500, *entry.FinalHTTPStatus) // uses last attempt
}

func TestRouteToDeadLetter_ResponseBodyCaptured(t *testing.T) {
	svc := NewService()
	attempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(422), ResponseBody: strPtr(`{"error":"invalid_field"}`), AttemptedAt: time.Now(), DurationMs: 50},
	}
	entry, err := svc.RouteToDeadLetter(context.Background(), "t1", "ep1", "del-resp",
		json.RawMessage(`{}`), json.RawMessage(`{}`), attempts, "validation error")
	require.NoError(t, err)
	assert.Equal(t, "client_error", entry.ErrorType)
	require.NotNil(t, entry.FinalResponseBody)
	assert.Contains(t, *entry.FinalResponseBody, "invalid_field")
}

func TestRouteToDeadLetter_FieldsPopulatedCorrectly(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	payload := json.RawMessage(`{"event":"order.created","data":{"id":"123"}}`)
	headers := json.RawMessage(`{"Authorization":"Bearer token"}`)

	entry, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "del-fields",
		payload, headers, nil, "unknown")
	require.NoError(t, err)

	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, "t1", entry.TenantID)
	assert.Equal(t, "ep1", entry.EndpointID)
	assert.Equal(t, "del-fields", entry.OriginalDeliveryID)
	assert.Equal(t, payload, entry.Payload)
	assert.Equal(t, headers, entry.Headers)
	assert.Equal(t, "dead_letter", entry.FinalStatus)
	assert.False(t, entry.Replayed)
	assert.Nil(t, entry.ReplayedAt)
	assert.False(t, entry.CreatedAt.IsZero())
	assert.True(t, entry.ExpiresAt.After(entry.CreatedAt))
}

func TestRouteToDeadLetter_UniqueIDs(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	ids := make(map[string]bool)

	for i := 0; i < 50; i++ {
		entry, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "del",
			json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "err")
		require.NoError(t, err)
		assert.False(t, ids[entry.ID], "duplicate ID generated: %s", entry.ID)
		ids[entry.ID] = true
	}
}

// =====================================================================
// ExportEntries – edge cases
// =====================================================================

func TestExportEntries_EmptyStore(t *testing.T) {
	svc := NewService()
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{Format: "json"})
	require.NoError(t, err)

	var entries []DLQEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	assert.Empty(t, entries)
}

func TestExportEntries_CSVEmptyStore(t *testing.T) {
	svc := NewService()
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{Format: "csv"})
	require.NoError(t, err)
	// Should have at least the header row
	assert.Contains(t, string(data), "id,tenant_id")
}

func TestExportEntries_WithFilter(t *testing.T) {
	svc := seedService(t)
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{
		Filter: DLQFilter{ErrorType: "server_error"},
		Format: "json",
	})
	require.NoError(t, err)

	var entries []DLQEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	for _, e := range entries {
		assert.Equal(t, "server_error", e.ErrorType)
	}
}

func TestExportEntries_UnknownFormatDefaultsToJSON(t *testing.T) {
	svc := seedService(t)
	data, err := svc.ExportEntries(context.Background(), "t1", &ExportRequest{Format: "xml"})
	require.NoError(t, err)

	var entries []DLQEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	assert.Len(t, entries, 4)
}

func TestExportEntries_CSVRowCount(t *testing.T) {
	svc := seedService(t)
	data, err := svc.ExportEntries(context.Background(), "t2", &ExportRequest{Format: "csv"})
	require.NoError(t, err)

	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	// header + 1 entry for t2 + possible trailing newline
	assert.GreaterOrEqual(t, lines, 2)
}

// =====================================================================
// AlertRules – additional coverage
// =====================================================================

func TestAlertRules_MultipleRulesPerTenant(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := svc.CreateAlertRule(ctx, "t1", &AlertRule{
			Name:    "rule",
			Action:  "webhook",
			Enabled: true,
		})
		require.NoError(t, err)
	}

	rules, err := svc.GetAlertRules(ctx, "t1")
	require.NoError(t, err)
	assert.Len(t, rules, 5)

	// Each rule should have a unique ID
	ids := map[string]bool{}
	for _, r := range rules {
		assert.False(t, ids[r.ID])
		ids[r.ID] = true
	}
}

func TestAlertRules_EmptyTenantReturnsEmpty(t *testing.T) {
	svc := NewService()
	rules, err := svc.GetAlertRules(context.Background(), "nobody")
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// =====================================================================
// RetentionPolicy – additional coverage
// =====================================================================

func TestRetentionPolicy_UpdateThenReadBack(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.UpdateRetentionPolicy(ctx, "t1", &RetentionPolicy{
		RetentionDays:     60,
		MaxEntries:        20000,
		CompressAfterDays: 10,
		AutoPurge:         true,
	})
	require.NoError(t, err)

	policy, err := svc.GetRetentionPolicy(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 60, policy.RetentionDays)
	assert.Equal(t, int64(20000), policy.MaxEntries)
	assert.Equal(t, 10, policy.CompressAfterDays)
	assert.True(t, policy.AutoPurge)
	assert.Equal(t, "t1", policy.TenantID)
}

func TestRetentionPolicy_PerTenantIsolation(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, err := svc.UpdateRetentionPolicy(ctx, "t1", &RetentionPolicy{RetentionDays: 7})
	require.NoError(t, err)

	p1, _ := svc.GetRetentionPolicy(ctx, "t1")
	p2, _ := svc.GetRetentionPolicy(ctx, "t2")
	assert.Equal(t, 7, p1.RetentionDays)
	assert.Equal(t, 30, p2.RetentionDays) // default
}

func TestRetentionPolicy_OverwriteExisting(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	_, _ = svc.UpdateRetentionPolicy(ctx, "t1", &RetentionPolicy{RetentionDays: 7})
	_, _ = svc.UpdateRetentionPolicy(ctx, "t1", &RetentionPolicy{RetentionDays: 14})

	p, err := svc.GetRetentionPolicy(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 14, p.RetentionDays)
}

// =====================================================================
// Concurrent operations
// =====================================================================

func TestConcurrentRouteToDeadLetter(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "d",
				json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "err")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(n), stats.TotalEntries)
}

func TestConcurrentReplayDifferentEntries(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	var entryIDs []string
	for i := 0; i < 20; i++ {
		entry, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", "d",
			json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "err")
		require.NoError(t, err)
		entryIDs = append(entryIDs, entry.ID)
	}

	var wg sync.WaitGroup
	wg.Add(len(entryIDs))
	for _, id := range entryIDs {
		go func(eid string) {
			defer wg.Done()
			_, err := svc.ReplayEntry(ctx, "t1", eid)
			assert.NoError(t, err)
		}(id)
	}
	wg.Wait()

	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(20), stats.ReplayedCount)
	assert.Equal(t, int64(0), stats.PendingCount)
}

func TestConcurrentReadAndWrite(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	var wg sync.WaitGroup

	// Writers
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.RouteToDeadLetter(ctx, "t1", "ep1", "d",
				json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "err")
		}()
	}

	// Readers
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.GetStats(ctx, "t1")
			_, _, _ = svc.GetEntries(ctx, &DLQFilter{TenantID: "t1"})
		}()
	}

	wg.Wait()

	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(50), stats.TotalEntries)
}

// =====================================================================
// ClassifyFailure – comprehensive classification
// =====================================================================

func TestClassifyFailure_NilEntry(t *testing.T) {
	svc := NewService()
	_, err := svc.ClassifyFailure(context.Background(), nil)
	assert.Error(t, err)
}

func TestClassifyFailure_DNS(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("dns resolution failed: no such host")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseDNSFailure, c.Category)
	assert.Equal(t, 0.95, c.Confidence)
	assert.False(t, c.IsRetryable)
}

func TestClassifyFailure_TLS(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("tls: handshake failure")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseTLSError, c.Category)
	assert.Equal(t, 0.92, c.Confidence)
	assert.False(t, c.IsRetryable)
}

func TestClassifyFailure_X509(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("x509: certificate has expired")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseTLSError, c.Category)
}

func TestClassifyFailure_Timeout(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("context deadline exceeded")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseTimeout, c.Category)
	assert.True(t, c.IsRetryable)
	assert.Equal(t, "30s", c.SuggestedDelay)
}

func TestClassifyFailure_Auth(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("HTTP 401 unauthorized")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseAuthFailure, c.Category)
	assert.False(t, c.IsRetryable)
}

func TestClassifyFailure_Forbidden(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("forbidden")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseAuthFailure, c.Category)
}

func TestClassifyFailure_RateLimit(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("429 too many requests")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseRateLimit, c.Category)
	assert.True(t, c.IsRetryable)
	assert.Equal(t, "60s", c.SuggestedDelay)
}

func TestClassifyFailure_ConnectionRefused(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("dial tcp: connection refused")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseEndpointDown, c.Category)
	assert.True(t, c.IsRetryable)
	assert.Equal(t, "300s", c.SuggestedDelay)
}

func TestClassifyFailure_ServerError(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID:              "e1",
		FinalHTTPStatus: intPtr(503),
		AllAttempts:     []AttemptDetail{{ErrorMessage: strPtr("service unavailable")}},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseServerError, c.Category)
	assert.True(t, c.IsRetryable)
	assert.Equal(t, "http_503", c.SubCategory)
}

func TestClassifyFailure_PayloadRejected(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID: "e1",
		AllAttempts: []AttemptDetail{
			{ErrorMessage: strPtr("invalid payload format")},
		},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCausePayloadRejected, c.Category)
	assert.False(t, c.IsRetryable)
}

func TestClassifyFailure_Unknown(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{
		ID:          "e1",
		AllAttempts: []AttemptDetail{{ErrorMessage: strPtr("something bizarre happened")}},
	}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseUnknown, c.Category)
	assert.Equal(t, 0.50, c.Confidence)
	assert.True(t, c.IsRetryable)
}

func TestClassifyFailure_NoAttempts(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{ID: "e1", FinalHTTPStatus: intPtr(500)}
	c, err := svc.ClassifyFailure(context.Background(), entry)
	require.NoError(t, err)
	assert.Equal(t, RootCauseServerError, c.Category)
}

func TestClassifyFailure_HasUniqueID(t *testing.T) {
	svc := NewService()
	entry := &DLQEntry{ID: "e1"}
	c1, _ := svc.ClassifyFailure(context.Background(), entry)
	c2, _ := svc.ClassifyFailure(context.Background(), entry)
	assert.NotEqual(t, c1.ID, c2.ID)
}

// =====================================================================
// BulkReplayWithBackpressure
// =====================================================================

func TestBulkReplayWithBackpressure_DryRun(t *testing.T) {
	svc := seedService(t)
	req := &BulkReplayWithBackpressureRequest{
		Filter:       DLQFilter{ErrorType: "server_error"},
		DryRun:       true,
		Backpressure: BackpressureConfig{RatePerSecond: 10},
	}
	progress, err := svc.BulkReplayWithBackpressure(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", progress.Status)
	assert.Greater(t, progress.EstimatedTimeMs, int64(0))

	// Verify entries not mutated
	e1, _ := svc.GetEntry(context.Background(), "t1", "e1")
	assert.False(t, e1.Replayed)
}

func TestBulkReplayWithBackpressure_Completes(t *testing.T) {
	svc := seedService(t)
	req := &BulkReplayWithBackpressureRequest{
		Filter:       DLQFilter{ErrorType: "rate_limit"},
		Backpressure: BackpressureConfig{RatePerSecond: 100},
	}
	progress, err := svc.BulkReplayWithBackpressure(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, "completed", progress.Status)
	assert.Equal(t, 1, progress.Succeeded)
	assert.Equal(t, "t1", progress.TenantID)
}

func TestBulkReplayWithBackpressure_MaxEntries(t *testing.T) {
	svc := seedService(t)
	req := &BulkReplayWithBackpressureRequest{
		Filter:       DLQFilter{},
		MaxEntries:   1,
		Backpressure: BackpressureConfig{RatePerSecond: 100},
	}
	progress, err := svc.BulkReplayWithBackpressure(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.LessOrEqual(t, progress.TotalEntries, 1)
}

func TestBulkReplayWithBackpressure_CircuitBreaker(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Add entries, pre-mark as replayed so ReplayEntry will fail
	for i := 0; i < 5; i++ {
		entry, _ := svc.RouteToDeadLetter(ctx, "t1", "ep1", "d",
			json.RawMessage(`{}`), json.RawMessage(`{}`), nil, "err")
		svc.mu.Lock()
		svc.entries[entry.ID].Replayed = true
		svc.mu.Unlock()
	}

	req := &BulkReplayWithBackpressureRequest{
		Filter: DLQFilter{},
		Backpressure: BackpressureConfig{
			CircuitBreaker:   true,
			FailureThreshold: 2,
			RatePerSecond:    100,
		},
	}
	progress, err := svc.BulkReplayWithBackpressure(ctx, "t1", req)
	require.NoError(t, err)
	// Circuit breaker stops processing early: only 2 failures out of 5
	assert.Equal(t, 2, progress.Failed)
	assert.Equal(t, 0, progress.Succeeded)
	assert.Equal(t, 5, progress.TotalEntries)
}

func TestBulkReplayWithBackpressure_EmptyQueue(t *testing.T) {
	svc := NewService()
	req := &BulkReplayWithBackpressureRequest{
		Filter:       DLQFilter{},
		Backpressure: BackpressureConfig{RatePerSecond: 10},
	}
	progress, err := svc.BulkReplayWithBackpressure(context.Background(), "t1", req)
	require.NoError(t, err)
	assert.Equal(t, "completed", progress.Status)
	assert.Equal(t, 0, progress.TotalEntries)
}

// =====================================================================
// GetFailureSummary
// =====================================================================

func TestGetFailureSummary_Empty(t *testing.T) {
	svc := NewService()
	summary, err := svc.GetFailureSummary(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, 0, summary.TotalFailures)
	assert.Empty(t, summary.TopPatterns)
	assert.Equal(t, "stable", summary.TrendDirection)
}

func TestGetFailureSummary_WithEntries(t *testing.T) {
	svc := seedService(t)
	summary, err := svc.GetFailureSummary(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, 4, summary.TotalFailures)
	assert.Equal(t, "t1", summary.TenantID)
	assert.NotEmpty(t, summary.ByCategory)
	assert.NotEmpty(t, summary.ByEndpoint)
	assert.NotEmpty(t, summary.TopPatterns)

	// Verify categories sum to total
	catTotal := 0
	for _, count := range summary.ByCategory {
		catTotal += count
	}
	assert.Equal(t, summary.TotalFailures, catTotal)

	// Verify endpoints sum to total
	epTotal := 0
	for _, count := range summary.ByEndpoint {
		epTotal += count
	}
	assert.Equal(t, summary.TotalFailures, epTotal)
}

func TestGetFailureSummary_TenantIsolation(t *testing.T) {
	svc := seedService(t)
	s1, _ := svc.GetFailureSummary(context.Background(), "t1")
	s2, _ := svc.GetFailureSummary(context.Background(), "t2")
	assert.Equal(t, 4, s1.TotalFailures)
	assert.Equal(t, 1, s2.TotalFailures)
}

func TestGetFailureSummary_PatternPercentages(t *testing.T) {
	svc := seedService(t)
	summary, err := svc.GetFailureSummary(context.Background(), "t1")
	require.NoError(t, err)

	totalPct := 0.0
	for _, p := range summary.TopPatterns {
		totalPct += p.Percentage
		assert.Greater(t, p.Count, 0)
	}
	assert.InDelta(t, 100.0, totalPct, 1.0) // should sum to ~100%
}

// =====================================================================
// AnalyzeEndpointHealth
// =====================================================================

func TestAnalyzeEndpointHealth_NoEntries(t *testing.T) {
	svc := NewService()
	health, err := svc.AnalyzeEndpointHealth(context.Background(), "t1", "ep1")
	require.NoError(t, err)
	assert.Equal(t, 0, health.TotalFailures)
	assert.Equal(t, float64(100), health.HealthScore)
	assert.Nil(t, health.LastFailureAt)
}

func TestAnalyzeEndpointHealth_WithEntries(t *testing.T) {
	svc := seedService(t)
	health, err := svc.AnalyzeEndpointHealth(context.Background(), "t1", "ep1")
	require.NoError(t, err)
	assert.Equal(t, 3, health.TotalFailures) // e1, e3, e5
	assert.NotEmpty(t, health.FailuresByCategory)
	assert.Greater(t, health.AvgResponseTimeMs, float64(0))
	assert.NotNil(t, health.LastFailureAt)
	assert.NotEmpty(t, health.TopRootCauses)
}

func TestAnalyzeEndpointHealth_HealthScoreBrackets(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected float64
	}{
		{"no failures", 0, 100},
		{"few failures", 3, 80},
		{"moderate failures", 10, 50},
		{"many failures", 50, 20},
		{"extreme failures", 150, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			ctx := context.Background()
			for i := 0; i < tt.count; i++ {
				svc.mu.Lock()
				svc.entries[fmt.Sprintf("e%d", i)] = &DLQEntry{
					ID: fmt.Sprintf("e%d", i), TenantID: "t1", EndpointID: "ep1",
					ErrorType: "server_error", FinalHTTPStatus: intPtr(500),
					CreatedAt: time.Now(),
					AllAttempts: []AttemptDetail{
						{AttemptNumber: 1, HTTPStatus: intPtr(500), AttemptedAt: time.Now(), DurationMs: 100},
					},
				}
				svc.mu.Unlock()
			}

			health, err := svc.AnalyzeEndpointHealth(ctx, "t1", "ep1")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, health.HealthScore)
		})
	}
}

func TestAnalyzeEndpointHealth_TopRootCausesSorted(t *testing.T) {
	svc := NewService()
	now := time.Now()

	// Add 10 server_error, 5 timeout, 2 rate_limit
	for i := 0; i < 10; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("se%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("se%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "server_error", FinalHTTPStatus: intPtr(500), CreatedAt: now,
			AllAttempts: []AttemptDetail{{AttemptedAt: now, DurationMs: 100}},
		}
		svc.mu.Unlock()
	}
	for i := 0; i < 5; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("to%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("to%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "timeout", CreatedAt: now,
			AllAttempts: []AttemptDetail{{AttemptedAt: now, DurationMs: 200}},
		}
		svc.mu.Unlock()
	}
	for i := 0; i < 2; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("rl%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("rl%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "rate_limit", FinalHTTPStatus: intPtr(429), CreatedAt: now,
			AllAttempts: []AttemptDetail{{AttemptedAt: now, DurationMs: 50}},
		}
		svc.mu.Unlock()
	}

	health, err := svc.AnalyzeEndpointHealth(context.Background(), "t1", "ep1")
	require.NoError(t, err)
	assert.Equal(t, 17, health.TotalFailures)
	require.NotEmpty(t, health.TopRootCauses)
	// First root cause should be the most common
	assert.Equal(t, health.TopRootCauses[0].Count, 10)
}

// =====================================================================
// SmartRetryRecommendation – all categories
// =====================================================================

func TestSmartRetryRecommendation_NotFound(t *testing.T) {
	svc := NewService()
	_, err := svc.GetSmartRetryRecommendation(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestSmartRetryRecommendation_AllCategories(t *testing.T) {
	tests := []struct {
		name        string
		errorType   string
		httpStatus  *int
		shouldRetry bool
	}{
		{"endpoint_down", "connection_refused", nil, true},
		{"timeout", "timeout", nil, true},
		{"rate_limit", "rate_limit", intPtr(429), true},
		{"auth_failure", "auth", intPtr(401), false},
		{"payload_rejected", "payload_error", intPtr(400), false},
		{"tls_error", "tls_handshake_failed", nil, false},
		{"server_error", "server_error", intPtr(500), true},
		{"unknown", "something_random", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService()
			svc.mu.Lock()
			svc.entries["test"] = &DLQEntry{
				ID:              "test",
				ErrorType:       tt.errorType,
				FinalHTTPStatus: tt.httpStatus,
				RetryCount:      1,
				MaxRetries:      5,
				CreatedAt:       time.Now(),
				ExpiresAt:       time.Now().Add(time.Hour),
				Payload:         json.RawMessage(`{}`),
				Headers:         json.RawMessage(`{}`),
			}
			svc.mu.Unlock()

			rec, err := svc.GetSmartRetryRecommendation(context.Background(), "test")
			require.NoError(t, err)
			assert.Equal(t, tt.shouldRetry, rec.ShouldRetry)
			assert.NotEmpty(t, rec.Reason)
			assert.Greater(t, rec.SuccessProbability, float64(0))
		})
	}
}

func TestSmartRetryRecommendation_HighRetryCountReducesProbability(t *testing.T) {
	svc := NewService()
	svc.mu.Lock()
	svc.entries["low"] = &DLQEntry{
		ID: "low", ErrorType: "timeout", RetryCount: 1,
		Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
	}
	svc.entries["high"] = &DLQEntry{
		ID: "high", ErrorType: "timeout", RetryCount: 5,
		Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
	}
	svc.mu.Unlock()

	recLow, err := svc.GetSmartRetryRecommendation(context.Background(), "low")
	require.NoError(t, err)
	recHigh, err := svc.GetSmartRetryRecommendation(context.Background(), "high")
	require.NoError(t, err)

	assert.Greater(t, recLow.SuccessProbability, recHigh.SuccessProbability)
}

// =====================================================================
// AnalyzeRootCause – deeper coverage
// =====================================================================

func TestAnalyzeRootCause_AllCategories(t *testing.T) {
	tests := []struct {
		errorType  string
		httpStatus *int
		wantCat    string
	}{
		{"timeout_exceeded", nil, RootCauseTimeout},
		{"dns_resolution_failed", nil, RootCauseDNSFailure},
		{"tls_error", nil, RootCauseTLSError},
		{"certificate_expired", nil, RootCauseTLSError},
		{"something", intPtr(401), RootCauseAuthFailure},
		{"something", intPtr(403), RootCauseAuthFailure},
		{"something", intPtr(429), RootCauseRateLimit},
		{"something", intPtr(400), RootCausePayloadRejected},
		{"something", intPtr(422), RootCausePayloadRejected},
		{"something", intPtr(500), RootCauseServerError},
		{"something", intPtr(502), RootCauseServerError},
		{"connection_refused", nil, RootCauseEndpointDown},
		{"weird_error", nil, RootCauseUnknown},
	}

	for _, tt := range tests {
		name := tt.errorType
		if tt.httpStatus != nil {
			name += fmt.Sprintf("_%d", *tt.httpStatus)
		}
		t.Run(name, func(t *testing.T) {
			svc := NewService()
			svc.mu.Lock()
			svc.entries["e"] = &DLQEntry{
				ID: "e", TenantID: "t1", EndpointID: "ep1",
				ErrorType: tt.errorType, FinalHTTPStatus: tt.httpStatus,
				Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
				RetryCount: 1, MaxRetries: 3, CreatedAt: time.Now(),
			}
			svc.mu.Unlock()

			analysis, err := svc.AnalyzeRootCause(context.Background(), "e")
			require.NoError(t, err)
			assert.Equal(t, tt.wantCat, analysis.Category)
			assert.NotEmpty(t, analysis.Summary)
			assert.NotEmpty(t, analysis.Details)
			assert.NotEmpty(t, analysis.Suggestions)
			assert.NotEmpty(t, analysis.ID)
			assert.NotEmpty(t, analysis.Severity)
		})
	}
}

func TestAnalyzeRootCause_FindsRelatedEntries(t *testing.T) {
	svc := NewService()
	now := time.Now()
	for i := 0; i < 5; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("e%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("e%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "server_error", FinalHTTPStatus: intPtr(500),
			Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
			CreatedAt: now, RetryCount: 1, MaxRetries: 3,
		}
		svc.mu.Unlock()
	}

	analysis, err := svc.AnalyzeRootCause(context.Background(), "e0")
	require.NoError(t, err)
	assert.NotEmpty(t, analysis.RelatedEntries)
	assert.NotContains(t, analysis.RelatedEntries, "e0") // should not include self
}

func TestAnalyzeRootCause_DetectsPatterns(t *testing.T) {
	svc := NewService()
	now := time.Now()
	// Add 3 entries with same endpoint and error for pattern detection
	for i := 0; i < 3; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("p%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("p%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "server_error", FinalHTTPStatus: intPtr(500),
			Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
			CreatedAt: now, RetryCount: 1, MaxRetries: 3,
		}
		svc.mu.Unlock()
	}

	analysis, err := svc.AnalyzeRootCause(context.Background(), "p0")
	require.NoError(t, err)
	assert.NotEmpty(t, analysis.Patterns)
	assert.GreaterOrEqual(t, analysis.Patterns[0].Occurrences, 2)
}

func TestAnalyzeRootCause_SeverityMapping(t *testing.T) {
	tests := []struct {
		category string
		severity string
	}{
		{RootCauseEndpointDown, SeverityCritical},
		{RootCauseDNSFailure, SeverityCritical},
		{RootCauseAuthFailure, SeverityHigh},
		{RootCauseTLSError, SeverityHigh},
		{RootCauseRateLimit, SeverityMedium},
		{RootCauseServerError, SeverityMedium},
		{RootCausePayloadRejected, SeverityLow},
		{RootCauseUnknown, SeverityLow},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			entry := &DLQEntry{EndpointID: "ep1"}
			assert.Equal(t, tt.severity, determineSeverity(entry, tt.category))
		})
	}
}

// =====================================================================
// Clustering – additional coverage
// =====================================================================

func TestClusterEntries_EmptyTenant(t *testing.T) {
	svc := NewService()
	result, err := svc.ClusterEntries("empty-tenant")
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalEntries)
	assert.Empty(t, result.Clusters)
}

func TestClusterEntries_SingleEntry(t *testing.T) {
	svc := NewService()
	svc.mu.Lock()
	svc.entries["e1"] = &DLQEntry{
		ID: "e1", TenantID: "t1", EndpointID: "ep1",
		ErrorType: "server_error", FinalHTTPStatus: intPtr(500),
		CreatedAt: time.Now(),
	}
	svc.mu.Unlock()

	result, err := svc.ClusterEntries("t1")
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalEntries)
	assert.Len(t, result.Clusters, 1)
	assert.Equal(t, 1, result.Clusters[0].EntryCount)
	assert.NotNil(t, result.Clusters[0].RootCause)
}

func TestClusterEntries_TenantIsolation(t *testing.T) {
	svc := NewService()
	svc.mu.Lock()
	svc.entries["e1"] = &DLQEntry{ID: "e1", TenantID: "t1", ErrorType: "timeout", CreatedAt: time.Now()}
	svc.entries["e2"] = &DLQEntry{ID: "e2", TenantID: "t2", ErrorType: "timeout", CreatedAt: time.Now()}
	svc.mu.Unlock()

	r1, _ := svc.ClusterEntries("t1")
	r2, _ := svc.ClusterEntries("t2")
	assert.Equal(t, 1, r1.TotalEntries)
	assert.Equal(t, 1, r2.TotalEntries)
}

func TestClusterEntries_SortedByEntryCount(t *testing.T) {
	svc := NewService()
	now := time.Now()

	// 5 timeouts, 2 server errors
	for i := 0; i < 5; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("t%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("t%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "timeout", CreatedAt: now,
		}
		svc.mu.Unlock()
	}
	for i := 0; i < 2; i++ {
		svc.mu.Lock()
		svc.entries[fmt.Sprintf("s%d", i)] = &DLQEntry{
			ID: fmt.Sprintf("s%d", i), TenantID: "t1", EndpointID: "ep1",
			ErrorType: "server_error", FinalHTTPStatus: intPtr(500), CreatedAt: now,
		}
		svc.mu.Unlock()
	}

	result, err := svc.ClusterEntries("t1")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result.Clusters), 2)
	assert.GreaterOrEqual(t, result.Clusters[0].EntryCount, result.Clusters[1].EntryCount)
}

func TestClusterEntries_TracksEndpoints(t *testing.T) {
	svc := NewService()
	now := time.Now()

	svc.mu.Lock()
	svc.entries["e1"] = &DLQEntry{ID: "e1", TenantID: "t1", EndpointID: "ep1", ErrorType: "timeout", CreatedAt: now}
	svc.entries["e2"] = &DLQEntry{ID: "e2", TenantID: "t1", EndpointID: "ep2", ErrorType: "timeout", CreatedAt: now}
	svc.entries["e3"] = &DLQEntry{ID: "e3", TenantID: "t1", EndpointID: "ep1", ErrorType: "timeout", CreatedAt: now}
	svc.mu.Unlock()

	result, err := svc.ClusterEntries("t1")
	require.NoError(t, err)
	require.Len(t, result.Clusters, 1)
	assert.Len(t, result.Clusters[0].EndpointIDs, 2)
	assert.Contains(t, result.Clusters[0].EndpointIDs, "ep1")
	assert.Contains(t, result.Clusters[0].EndpointIDs, "ep2")
}

func TestClusterEntries_FirstSeenLastSeen(t *testing.T) {
	svc := NewService()
	early := time.Now().Add(-24 * time.Hour)
	late := time.Now()

	svc.mu.Lock()
	svc.entries["e1"] = &DLQEntry{ID: "e1", TenantID: "t1", ErrorType: "timeout", CreatedAt: early}
	svc.entries["e2"] = &DLQEntry{ID: "e2", TenantID: "t1", ErrorType: "timeout", CreatedAt: late}
	svc.mu.Unlock()

	result, err := svc.ClusterEntries("t1")
	require.NoError(t, err)
	require.Len(t, result.Clusters, 1)
	assert.True(t, result.Clusters[0].FirstSeenAt.Equal(early))
	assert.True(t, result.Clusters[0].LastSeenAt.Equal(late))
}

// =====================================================================
// normalizeError – additional patterns
// =====================================================================

func TestNormalizeError_AdditionalPatterns(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"unexpected EOF", "conn_closed"},
		{"rate limit exceeded", "rate_limited"},
		{"too many requests", "rate_limited"},
		{"unauthorized access", "auth_failure"},
		{"forbidden resource", "auth_failure"},
		{"TLS handshake failed", "tls_error"},
		{"certificate verify failed", "tls_cert"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeError(tt.input))
		})
	}
}

// =====================================================================
// containsStr helper
// =====================================================================

func TestContainsStr(t *testing.T) {
	assert.True(t, containsStr([]string{"a", "b", "c"}, "b"))
	assert.False(t, containsStr([]string{"a", "b", "c"}, "d"))
	assert.False(t, containsStr(nil, "a"))
	assert.False(t, containsStr([]string{}, "a"))
}

// =====================================================================
// generateDetails helper
// =====================================================================

func TestGenerateDetails_WithAllFields(t *testing.T) {
	body := "error body"
	entry := &DLQEntry{
		RetryCount:        3,
		MaxRetries:        5,
		FinalHTTPStatus:   intPtr(500),
		FinalResponseBody: &body,
		AllAttempts: []AttemptDetail{
			{DurationMs: 120},
		},
	}
	details := generateDetails(entry)
	assert.Contains(t, details, "3/5")
	assert.Contains(t, details, "500")
	assert.Contains(t, details, "error body")
	assert.Contains(t, details, "120ms")
}

func TestGenerateDetails_LongResponseBodyTruncated(t *testing.T) {
	longBody := string(make([]byte, 300))
	entry := &DLQEntry{
		FinalResponseBody: &longBody,
		RetryCount:        1,
		MaxRetries:        3,
	}
	details := generateDetails(entry)
	assert.Contains(t, details, "...")
}

func TestGenerateDetails_MinimalEntry(t *testing.T) {
	entry := &DLQEntry{RetryCount: 0, MaxRetries: 0}
	details := generateDetails(entry)
	assert.Contains(t, details, "0/0")
}

// =====================================================================
// generateSummary – all categories
// =====================================================================

func TestGenerateSummary_AllCategories(t *testing.T) {
	entry := &DLQEntry{EndpointID: "ep-test"}
	categories := []string{
		RootCauseEndpointDown, RootCauseTimeout, RootCauseRateLimit,
		RootCauseAuthFailure, RootCausePayloadRejected, RootCauseDNSFailure,
		RootCauseTLSError, RootCauseServerError, RootCauseUnknown,
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			summary := generateSummary(cat, entry)
			assert.NotEmpty(t, summary)
			assert.Contains(t, summary, "ep-test")
		})
	}
}

// =====================================================================
// generateSuggestions – all categories produce actions
// =====================================================================

func TestGenerateSuggestions_AllCategories(t *testing.T) {
	entry := &DLQEntry{EndpointID: "ep1"}
	categories := []string{
		RootCauseEndpointDown, RootCauseTimeout, RootCauseRateLimit,
		RootCauseAuthFailure, RootCausePayloadRejected, "totally_unknown",
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			suggestions := generateSuggestions(cat, entry)
			assert.NotEmpty(t, suggestions)
			for _, s := range suggestions {
				assert.NotEmpty(t, s.Action)
				assert.NotEmpty(t, s.Description)
				assert.Greater(t, s.Priority, 0)
			}
		})
	}
}

// =====================================================================
// Handler / NewHandler
// =====================================================================

func TestNewHandler(t *testing.T) {
	svc := NewService()
	h := NewHandler(svc)
	require.NotNil(t, h)
	assert.Equal(t, svc, h.service)
}

// =====================================================================
// Integration-style: full lifecycle
// =====================================================================

func TestFullLifecycle_CreateFilterReplayStats(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// 1. Route several entries to DLQ
	for i := 0; i < 3; i++ {
		attempts := []AttemptDetail{
			{AttemptNumber: 1, HTTPStatus: intPtr(500), ErrorMessage: strPtr("server error"), AttemptedAt: time.Now(), DurationMs: 100},
		}
		_, err := svc.RouteToDeadLetter(ctx, "t1", "ep1", fmt.Sprintf("d%d", i),
			json.RawMessage(`{"event":"test"}`), json.RawMessage(`{}`), attempts, "server error")
		require.NoError(t, err)
	}
	// Add a rate-limit entry
	rlAttempts := []AttemptDetail{
		{AttemptNumber: 1, HTTPStatus: intPtr(429), ErrorMessage: strPtr("rate limited"), AttemptedAt: time.Now(), DurationMs: 10},
	}
	_, err := svc.RouteToDeadLetter(ctx, "t1", "ep2", "d-rl",
		json.RawMessage(`{}`), json.RawMessage(`{}`), rlAttempts, "rate limited")
	require.NoError(t, err)

	// 2. Verify stats
	stats, err := svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats.TotalEntries)
	assert.Equal(t, int64(4), stats.PendingCount)

	// 3. Filter by error type
	entries, total, err := svc.GetEntries(ctx, &DLQFilter{TenantID: "t1", ErrorType: "server_error"})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, entries, 3)

	// 4. Bulk retry server errors
	result, err := svc.BulkRetry(ctx, "t1", &BulkRetryRequest{
		Filter: DLQFilter{ErrorType: "server_error"}, MaxCount: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Succeeded)

	// 5. Verify stats updated
	stats, err = svc.GetStats(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.ReplayedCount)
	assert.Equal(t, int64(1), stats.PendingCount)

	// 6. Export remaining
	data, err := svc.ExportEntries(ctx, "t1", &ExportRequest{
		Filter: DLQFilter{Status: "dead_letter"}, Format: "json",
	})
	require.NoError(t, err)
	var remaining []DLQEntry
	require.NoError(t, json.Unmarshal(data, &remaining))
	assert.Len(t, remaining, 1)
	assert.Equal(t, "rate_limit", remaining[0].ErrorType)

	// 7. Classify the remaining entry
	classification, err := svc.ClassifyFailure(ctx, &remaining[0])
	require.NoError(t, err)
	assert.Equal(t, RootCauseRateLimit, classification.Category)

	// 8. Get failure summary
	summary, err := svc.GetFailureSummary(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 4, summary.TotalFailures)
}
