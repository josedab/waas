package livemigration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Importer Tests ---

func TestNewImporter_AllPlatforms(t *testing.T) {
	platforms := []string{PlatformSvix, PlatformConvoy, PlatformHookdeck, PlatformCSV, PlatformJSON}
	for _, p := range platforms {
		imp, err := NewImporter(p)
		require.NoError(t, err, "platform: %s", p)
		assert.NotNil(t, imp, "platform: %s", p)
	}
}

func TestNewImporter_UnsupportedPlatform(t *testing.T) {
	_, err := NewImporter("unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported import platform")
}

// --- SvixImporter Tests ---

func TestSvixImporter_WithRawData(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	endpoints := []SvixAPIEndpoint{
		{UID: "ep1", URL: "https://example.com/hook1", Description: "Hook 1", FilterTypes: []string{"order.created"}},
		{UID: "ep2", URL: "https://example.com/hook2", Description: "Hook 2", FilterTypes: []string{"payment.completed"}},
	}
	data, _ := json.Marshal(endpoints)

	config := &ImportConfig{
		SourceType: PlatformSvix,
		RawData:    string(data),
		TenantID:   "tenant-1",
		JobID:      "job-1",
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Equal(t, 0, result.SkippedCount)
	assert.Equal(t, 0, result.FailedCount)
	assert.Len(t, result.Endpoints, 2)
	assert.Equal(t, "ep1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://example.com/hook1", result.Endpoints[0].URL)
	assert.Equal(t, EndpointStatusImported, result.Endpoints[0].Status)
	assert.True(t, result.Duration > 0)
}

func TestSvixImporter_SkipsDisabledEndpoints(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	endpoints := []SvixAPIEndpoint{
		{UID: "ep1", URL: "https://example.com/hook1", Disabled: false},
		{UID: "ep2", URL: "https://example.com/hook2", Disabled: true},
	}
	data, _ := json.Marshal(endpoints)

	config := &ImportConfig{
		SourceType: PlatformSvix,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestSvixImporter_FilterByEventType(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	endpoints := []SvixAPIEndpoint{
		{UID: "ep1", URL: "https://example.com/hook1", FilterTypes: []string{"order.created"}},
		{UID: "ep2", URL: "https://example.com/hook2", FilterTypes: []string{"payment.completed"}},
	}
	data, _ := json.Marshal(endpoints)

	config := &ImportConfig{
		SourceType: PlatformSvix,
		RawData:    string(data),
		Filters:    map[string]string{"event_type": "order.created"},
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestSvixImporter_DryRun(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	config := &ImportConfig{
		SourceType: PlatformSvix,
		APIKey:     "test-key",
		DryRun:     true,
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 3, result.ImportedCount)
	assert.Nil(t, result.Endpoints, "dry run should not produce endpoint objects")
}

func TestSvixImporter_MissingCredentials(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	config := &ImportConfig{
		SourceType: PlatformSvix,
	}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires api_key or raw_data")
}

func TestSvixImporter_SimulatedDiscovery(t *testing.T) {
	ctx := context.Background()
	importer := NewSvixImporter()

	config := &ImportConfig{
		SourceType: PlatformSvix,
		APIKey:     "test-key",
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 3, result.ImportedCount)
	assert.Len(t, result.Endpoints, 3)
}

// --- ConvoyImporter Tests ---

func TestConvoyImporter_WithRawData(t *testing.T) {
	ctx := context.Background()
	importer := NewConvoyImporter()

	endpoints := []ConvoyAPIEndpoint{
		{UID: "ce1", TargetURL: "https://hooks.example.com/order", Description: "Order hook", Status: "active", RateLimit: 100},
		{UID: "ce2", TargetURL: "https://hooks.example.com/payment", Description: "Payment hook", Status: "active", RateLimit: 50},
	}
	data, _ := json.Marshal(endpoints)

	config := &ImportConfig{
		SourceType: PlatformConvoy,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Len(t, result.Endpoints, 2)
	assert.Equal(t, "100", result.Endpoints[0].Metadata["rate_limit"])
}

func TestConvoyImporter_SkipsInactiveEndpoints(t *testing.T) {
	ctx := context.Background()
	importer := NewConvoyImporter()

	endpoints := []ConvoyAPIEndpoint{
		{UID: "ce1", TargetURL: "https://hooks.example.com/order", Status: "active"},
		{UID: "ce2", TargetURL: "https://hooks.example.com/payment", Status: "inactive"},
	}
	data, _ := json.Marshal(endpoints)

	config := &ImportConfig{
		SourceType: PlatformConvoy,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestConvoyImporter_MissingCredentials(t *testing.T) {
	ctx := context.Background()
	importer := NewConvoyImporter()

	config := &ImportConfig{SourceType: PlatformConvoy}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires api_key or raw_data")
}

// --- HookdeckImporter Tests ---

func TestHookdeckImporter_WithRawData(t *testing.T) {
	ctx := context.Background()
	importer := NewHookdeckImporter()

	connections := []HookdeckConnection{
		{
			ID:          "conn1",
			Source:      HookdeckSource{ID: "src1", Name: "orders", URL: "https://hd.example.com/orders"},
			Destination: HookdeckDestination{ID: "dst1", Name: "order-proc", URL: "https://api.example.com/orders", HTTPMethod: "POST"},
			FullName:    "orders -> order-proc",
		},
		{
			ID:          "conn2",
			Source:      HookdeckSource{ID: "src2", Name: "payments", URL: "https://hd.example.com/payments"},
			Destination: HookdeckDestination{ID: "dst2", Name: "pay-handler", URL: "https://api.example.com/payments"},
			FullName:    "payments -> pay-handler",
		},
	}
	data, _ := json.Marshal(connections)

	config := &ImportConfig{
		SourceType: PlatformHookdeck,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Len(t, result.Endpoints, 2)
	assert.Equal(t, "conn1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://api.example.com/orders", result.Endpoints[0].URL)
	assert.Equal(t, "orders -> order-proc", result.Endpoints[0].Description)
	assert.Equal(t, "POST", result.Endpoints[0].Metadata["http_method"])
}

func TestHookdeckImporter_SkipsPausedConnections(t *testing.T) {
	ctx := context.Background()
	importer := NewHookdeckImporter()

	now := mustParseTime(t)
	connections := []HookdeckConnection{
		{ID: "conn1", Source: HookdeckSource{ID: "src1"}, Destination: HookdeckDestination{ID: "dst1", URL: "https://a.com"}},
		{ID: "conn2", Source: HookdeckSource{ID: "src2"}, Destination: HookdeckDestination{ID: "dst2", URL: "https://b.com"}, PausedAt: &now},
	}
	data, _ := json.Marshal(connections)

	config := &ImportConfig{
		SourceType: PlatformHookdeck,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestHookdeckImporter_MissingCredentials(t *testing.T) {
	ctx := context.Background()
	importer := NewHookdeckImporter()

	config := &ImportConfig{SourceType: PlatformHookdeck}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires api_key or raw_data")
}

// --- GenericCSVImporter Tests ---

func TestCSVImporter_BasicImport(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericCSVImporter()

	csvData := "id,url,description\nep1,https://example.com/hook1,First hook\nep2,https://example.com/hook2,Second hook\n"

	config := &ImportConfig{
		SourceType: PlatformCSV,
		RawData:    csvData,
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Len(t, result.Endpoints, 2)
	assert.Equal(t, "ep1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://example.com/hook1", result.Endpoints[0].URL)
	assert.Equal(t, "First hook", result.Endpoints[0].Description)
}

func TestCSVImporter_WithFieldMapping(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericCSVImporter()

	csvData := "endpoint_id,webhook_url,label\nep1,https://example.com/hook1,My Hook\n"

	config := &ImportConfig{
		SourceType: PlatformCSV,
		RawData:    csvData,
		FieldMapping: map[string]string{
			"id":          "endpoint_id",
			"url":         "webhook_url",
			"description": "label",
		},
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, "ep1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://example.com/hook1", result.Endpoints[0].URL)
	assert.Equal(t, "My Hook", result.Endpoints[0].Description)
}

func TestCSVImporter_SkipsRowsWithoutURL(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericCSVImporter()

	csvData := "id,url\nep1,https://example.com/hook1\nep2,\n"

	config := &ImportConfig{
		SourceType: PlatformCSV,
		RawData:    csvData,
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestCSVImporter_EmptyData(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericCSVImporter()

	config := &ImportConfig{
		SourceType: PlatformCSV,
	}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires raw_data")
}

func TestCSVImporter_DryRun(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericCSVImporter()

	csvData := "id,url\nep1,https://example.com/hook1\nep2,https://example.com/hook2\n"

	config := &ImportConfig{
		SourceType: PlatformCSV,
		RawData:    csvData,
		DryRun:     true,
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Nil(t, result.Endpoints)
}

// --- GenericJSONImporter Tests ---

func TestJSONImporter_BasicImport(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericJSONImporter()

	records := []map[string]interface{}{
		{"id": "ep1", "url": "https://example.com/hook1", "description": "First"},
		{"id": "ep2", "url": "https://example.com/hook2", "description": "Second"},
	}
	data, _ := json.Marshal(records)

	config := &ImportConfig{
		SourceType: PlatformJSON,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Len(t, result.Endpoints, 2)
	assert.Equal(t, "ep1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://example.com/hook1", result.Endpoints[0].URL)
}

func TestJSONImporter_WithFieldMapping(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericJSONImporter()

	records := []map[string]interface{}{
		{"endpoint_id": "ep1", "webhook_url": "https://example.com/hook1", "label": "My Hook"},
	}
	data, _ := json.Marshal(records)

	config := &ImportConfig{
		SourceType: PlatformJSON,
		RawData:    string(data),
		FieldMapping: map[string]string{
			"id":          "endpoint_id",
			"url":         "webhook_url",
			"description": "label",
		},
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, "ep1", result.Endpoints[0].SourceID)
	assert.Equal(t, "https://example.com/hook1", result.Endpoints[0].URL)
	assert.Equal(t, "My Hook", result.Endpoints[0].Description)
}

func TestJSONImporter_SkipsRecordsWithoutURL(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericJSONImporter()

	records := []map[string]interface{}{
		{"id": "ep1", "url": "https://example.com/hook1"},
		{"id": "ep2"},
	}
	data, _ := json.Marshal(records)

	config := &ImportConfig{
		SourceType: PlatformJSON,
		RawData:    string(data),
	}

	result, err := importer.Import(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedCount)
	assert.Equal(t, 1, result.SkippedCount)
}

func TestJSONImporter_EmptyData(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericJSONImporter()

	config := &ImportConfig{
		SourceType: PlatformJSON,
	}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires raw_data")
}

func TestJSONImporter_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	importer := NewGenericJSONImporter()

	config := &ImportConfig{
		SourceType: PlatformJSON,
		RawData:    "not valid json",
	}

	_, err := importer.Import(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON data")
}

// --- DualWriteManager Tests ---

func TestDualWriteManager_RecordAndMatchRate(t *testing.T) {
	dw := NewDualWriteManager("job-1", "tenant-1")

	assert.True(t, dw.IsEnabled())

	dw.RecordResult("ep1", "ev1", 200, 200, 50, 55)
	dw.RecordResult("ep1", "ev2", 200, 500, 50, 60)
	dw.RecordResult("ep1", "ev3", 200, 200, 50, 45)

	results := dw.GetResults()
	assert.Len(t, results, 3)
	assert.True(t, results[0].Match)
	assert.False(t, results[1].Match)
	assert.True(t, results[2].Match)

	matchRate := dw.MatchRate()
	assert.InDelta(t, 0.6667, matchRate, 0.01)
}

func TestDualWriteManager_EnableDisable(t *testing.T) {
	dw := NewDualWriteManager("job-1", "tenant-1")

	assert.True(t, dw.IsEnabled())
	dw.Disable()
	assert.False(t, dw.IsEnabled())
	dw.Enable()
	assert.True(t, dw.IsEnabled())
}

func TestDualWriteManager_EmptyMatchRate(t *testing.T) {
	dw := NewDualWriteManager("job-1", "tenant-1")
	assert.Equal(t, float64(0), dw.MatchRate())
}

// --- TrafficShifter Tests ---

func TestTrafficShifter_DefaultSteps(t *testing.T) {
	ts := NewTrafficShifter("job-1", "tenant-1")

	assert.Equal(t, 0, ts.CurrentPercentage())
	assert.False(t, ts.IsComplete())

	pct, err := ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 25, pct)

	pct, err = ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 50, pct)

	pct, err = ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 75, pct)

	pct, err = ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 100, pct)
	assert.True(t, ts.IsComplete())

	// Should error at max
	_, err = ts.NextStep()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already at maximum")
}

func TestTrafficShifter_CustomSteps(t *testing.T) {
	ts := NewTrafficShifterWithSteps("job-1", "tenant-1", []int{0, 10, 50, 100})

	assert.Equal(t, 0, ts.CurrentPercentage())

	pct, err := ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 10, pct)

	pct, err = ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 50, pct)

	pct, err = ts.NextStep()
	require.NoError(t, err)
	assert.Equal(t, 100, pct)
}

func TestTrafficShifter_SetPercentage(t *testing.T) {
	ts := NewTrafficShifter("job-1", "tenant-1")

	err := ts.SetPercentage(50)
	require.NoError(t, err)
	assert.Equal(t, 50, ts.CurrentPercentage())

	err = ts.SetPercentage(-1)
	assert.Error(t, err)

	err = ts.SetPercentage(101)
	assert.Error(t, err)
}

func TestTrafficShifter_Reset(t *testing.T) {
	ts := NewTrafficShifter("job-1", "tenant-1")
	ts.NextStep()
	ts.NextStep()

	assert.Equal(t, 50, ts.CurrentPercentage())
	ts.Reset()
	assert.Equal(t, 0, ts.CurrentPercentage())
}

func TestTrafficShifter_EmptyStepsFallback(t *testing.T) {
	ts := NewTrafficShifterWithSteps("job-1", "tenant-1", []int{})
	assert.Equal(t, 0, ts.CurrentPercentage())
}

// --- CutoverService Tests ---

func TestCutoverService_StartCutover(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	// Create a migration job with endpoints
	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test Migration",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{"api_key":"test"}`,
	})
	require.NoError(t, err)

	// Discover and import endpoints
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	// Start cutover
	plan, err := cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)
	assert.Equal(t, CutoverStatusInProgress, plan.Status)
	assert.Equal(t, CutoverPhaseDualWrite, plan.CurrentPhase)
	assert.NotNil(t, plan.ValidationReport)
	assert.True(t, plan.ValidationReport.CanProceed)
	assert.NotNil(t, plan.RollbackSnapshot)
	assert.Len(t, plan.Phases, 4)
}

func TestCutoverService_StartCutover_DryRun(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test Migration",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{"api_key":"test"}`,
	})
	require.NoError(t, err)

	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	plan, err := cs.StartCutover(ctx, "tenant-1", job.ID, true)
	require.NoError(t, err)
	assert.Equal(t, CutoverStatusCompleted, plan.Status)
	assert.True(t, plan.DryRun)
	assert.NotNil(t, plan.ValidationReport)
	assert.Contains(t, plan.ValidationReport.DryRunSummary, "Dry run")
}

func TestCutoverService_StartCutover_NoEndpoints(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Empty Migration",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{"api_key":"test"}`,
	})
	require.NoError(t, err)

	plan, err := cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)
	assert.Equal(t, CutoverStatusFailed, plan.Status)
	assert.False(t, plan.ValidationReport.CanProceed)
}

func TestCutoverService_GetStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	plan, err := cs.GetStatus(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, CutoverStatusInProgress, plan.Status)
}

func TestCutoverService_GetStatus_NotFound(t *testing.T) {
	repo := newMockRepository()
	cs := NewCutoverService(repo)
	ctx := context.Background()

	_, err := cs.GetStatus(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no cutover plan found")
}

func TestCutoverService_AdjustTrafficSplit(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	// Adjust to 25%
	plan, err := cs.AdjustTrafficSplit(ctx, job.ID, 25)
	require.NoError(t, err)
	assert.Equal(t, 25, plan.TrafficSplitPct)
	assert.Equal(t, CutoverPhaseGradualShift, plan.CurrentPhase)

	// Adjust to 100% - should complete
	plan, err = cs.AdjustTrafficSplit(ctx, job.ID, 100)
	require.NoError(t, err)
	assert.Equal(t, 100, plan.TrafficSplitPct)
	assert.Equal(t, CutoverStatusCompleted, plan.Status)
}

func TestCutoverService_Rollback(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	// Adjust traffic then rollback
	_, err = cs.AdjustTrafficSplit(ctx, job.ID, 50)
	require.NoError(t, err)

	plan, err := cs.Rollback(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, CutoverStatusRolledBack, plan.Status)
	assert.Equal(t, CutoverPhaseRollback, plan.CurrentPhase)
	assert.Equal(t, 0, plan.TrafficSplitPct)
}

func TestCutoverService_Rollback_AlreadyRolledBack(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	_, err = cs.Rollback(ctx, job.ID)
	require.NoError(t, err)

	_, err = cs.Rollback(ctx, job.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already rolled back")
}

func TestCutoverService_DuplicateStart(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")
}

func TestCutoverService_GetDualWriteManager(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	dw, err := cs.GetDualWriteManager(job.ID)
	require.NoError(t, err)
	assert.NotNil(t, dw)
	assert.True(t, dw.IsEnabled())

	_, err = cs.GetDualWriteManager("nonexistent")
	assert.Error(t, err)
}

func TestCutoverService_GetTrafficShifter(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	cs := NewCutoverService(repo)
	ctx := context.Background()

	job, err := svc.CreateMigration(ctx, "tenant-1", &CreateMigrationRequest{
		Name:           "Test",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{}`,
	})
	require.NoError(t, err)
	_, err = svc.DiscoverEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)
	_, err = svc.ImportEndpoints(ctx, "tenant-1", job.ID)
	require.NoError(t, err)

	_, err = cs.StartCutover(ctx, "tenant-1", job.ID, false)
	require.NoError(t, err)

	ts, err := cs.GetTrafficShifter(job.ID)
	require.NoError(t, err)
	assert.NotNil(t, ts)
	assert.Equal(t, 0, ts.CurrentPercentage())

	_, err = cs.GetTrafficShifter("nonexistent")
	assert.Error(t, err)
}

// --- Helper ---

func mustParseTime(t *testing.T) time.Time {
	t.Helper()
	return time.Now()
}
