package livemigration

import (
	"context"
	"fmt"
	"testing"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	jobs        map[string]*MigrationJob
	endpoints   map[string]*MigrationEndpoint
	checkpoints map[string]*MigrationCheckpoint
	results     []ParallelDeliveryResult
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		jobs:        make(map[string]*MigrationJob),
		endpoints:   make(map[string]*MigrationEndpoint),
		checkpoints: make(map[string]*MigrationCheckpoint),
	}
}

func (m *mockRepository) CreateJob(_ context.Context, job *MigrationJob) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockRepository) GetJob(_ context.Context, tenantID, jobID string) (*MigrationJob, error) {
	job, ok := m.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

func (m *mockRepository) ListJobs(_ context.Context, tenantID string) ([]MigrationJob, error) {
	var jobs []MigrationJob
	for _, j := range m.jobs {
		if j.TenantID == tenantID {
			jobs = append(jobs, *j)
		}
	}
	return jobs, nil
}

func (m *mockRepository) UpdateJob(_ context.Context, job *MigrationJob) error {
	m.jobs[job.ID] = job
	return nil
}

func (m *mockRepository) DeleteJob(_ context.Context, tenantID, jobID string) error {
	delete(m.jobs, jobID)
	return nil
}

func (m *mockRepository) CreateEndpoint(_ context.Context, endpoint *MigrationEndpoint) error {
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockRepository) GetEndpoint(_ context.Context, tenantID, endpointID string) (*MigrationEndpoint, error) {
	ep, ok := m.endpoints[endpointID]
	if !ok {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}
	return ep, nil
}

func (m *mockRepository) ListEndpointsByJob(_ context.Context, tenantID, jobID string) ([]MigrationEndpoint, error) {
	var endpoints []MigrationEndpoint
	for _, ep := range m.endpoints {
		if ep.JobID == jobID {
			endpoints = append(endpoints, *ep)
		}
	}
	return endpoints, nil
}

func (m *mockRepository) UpdateEndpoint(_ context.Context, endpoint *MigrationEndpoint) error {
	m.endpoints[endpoint.ID] = endpoint
	return nil
}

func (m *mockRepository) DeleteEndpointsByJob(_ context.Context, tenantID, jobID string) error {
	for id, ep := range m.endpoints {
		if ep.JobID == jobID {
			delete(m.endpoints, id)
		}
	}
	return nil
}

func (m *mockRepository) CreateParallelResult(_ context.Context, result *ParallelDeliveryResult) error {
	m.results = append(m.results, *result)
	return nil
}

func (m *mockRepository) ListParallelResultsByJob(_ context.Context, tenantID, jobID string) ([]ParallelDeliveryResult, error) {
	var results []ParallelDeliveryResult
	for _, r := range m.results {
		if r.JobID == jobID {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *mockRepository) GetMigrationStats(_ context.Context, tenantID, jobID string) (*MigrationStats, error) {
	return &MigrationStats{JobID: jobID}, nil
}

func (m *mockRepository) GetCutoverReadiness(_ context.Context, tenantID, jobID string) (int, int, int, float64, error) {
	return 0, 0, 0, 0.0, nil
}

func (m *mockRepository) CreateCheckpoint(_ context.Context, checkpoint *MigrationCheckpoint) error {
	m.checkpoints[checkpoint.MigrationID] = checkpoint
	return nil
}

func (m *mockRepository) GetCheckpoint(_ context.Context, migrationID string) (*MigrationCheckpoint, error) {
	cp, ok := m.checkpoints[migrationID]
	if !ok {
		return nil, fmt.Errorf("checkpoint not found for migration: %s", migrationID)
	}
	return cp, nil
}

func (m *mockRepository) UpdateCheckpoint(_ context.Context, checkpoint *MigrationCheckpoint) error {
	m.checkpoints[checkpoint.MigrationID] = checkpoint
	return nil
}

func TestCreateMigration(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateMigrationRequest{
		Name:           "Test Migration",
		SourcePlatform: PlatformSvix,
		SourceConfig:   `{"api_key":"test"}`,
	}

	job, err := svc.CreateMigration(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateMigration failed: %v", err)
	}
	if job.ID == "" {
		t.Error("expected job ID to be set")
	}
	if job.Status != JobStatusPending {
		t.Errorf("expected status %s, got %s", JobStatusPending, job.Status)
	}
	if job.Name != "Test Migration" {
		t.Errorf("expected name 'Test Migration', got %s", job.Name)
	}
	if job.SourcePlatform != PlatformSvix {
		t.Errorf("expected platform %s, got %s", PlatformSvix, job.SourcePlatform)
	}
}

func TestDryRunMigration_Svix(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	config := &ImporterConfig{
		Platform: PlatformSvix,
		APIKey:   "test-key",
	}

	result, err := svc.DryRunMigration(ctx, "tenant-1", config)
	if err != nil {
		t.Fatalf("DryRunMigration failed: %v", err)
	}
	if result.Platform != PlatformSvix {
		t.Errorf("expected platform %s, got %s", PlatformSvix, result.Platform)
	}
	if result.EndpointsFound != 3 {
		t.Errorf("expected 3 endpoints, got %d", result.EndpointsFound)
	}
	if !result.Compatible {
		t.Error("expected compatible to be true")
	}
	if len(result.FieldMappings) == 0 {
		t.Error("expected field mappings to be populated")
	}
}

func TestDryRunMigration_Convoy(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	config := &ImporterConfig{
		Platform: PlatformConvoy,
		APIKey:   "test-key",
	}

	result, err := svc.DryRunMigration(ctx, "tenant-1", config)
	if err != nil {
		t.Fatalf("DryRunMigration failed: %v", err)
	}
	if result.Platform != PlatformConvoy {
		t.Errorf("expected platform %s, got %s", PlatformConvoy, result.Platform)
	}
	if result.EndpointsFound != 2 {
		t.Errorf("expected 2 endpoints, got %d", result.EndpointsFound)
	}
	if !result.Compatible {
		t.Error("expected compatible to be true")
	}
}

func TestDryRunMigration_CSV(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	config := &ImporterConfig{
		Platform: PlatformCSV,
		FilePath: "/tmp/endpoints.csv",
	}

	result, err := svc.DryRunMigration(ctx, "tenant-1", config)
	if err != nil {
		t.Fatalf("DryRunMigration failed: %v", err)
	}
	if result.Platform != PlatformCSV {
		t.Errorf("expected platform %s, got %s", PlatformCSV, result.Platform)
	}
	if !result.Compatible {
		t.Error("expected compatible to be true")
	}
	if result.EstimatedTime != "1-3 minutes" {
		t.Errorf("expected '1-3 minutes', got %s", result.EstimatedTime)
	}
}

func TestSvixFieldMapping(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	mappings := svc.svixFieldMapping()
	if len(mappings) != 5 {
		t.Fatalf("expected 5 mappings, got %d", len(mappings))
	}

	expected := map[string]string{
		"uid":         "id",
		"url":         "url",
		"description": "description",
		"filterTypes": "filter_types",
		"metadata":    "metadata",
	}
	for _, m := range mappings {
		if target, ok := expected[m.SourceField]; ok {
			if m.TargetField != target {
				t.Errorf("field %s: expected target %s, got %s", m.SourceField, target, m.TargetField)
			}
		}
	}
}

func TestConvoyFieldMapping(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	mappings := svc.convoyFieldMapping()
	if len(mappings) != 6 {
		t.Fatalf("expected 6 mappings, got %d", len(mappings))
	}

	expected := map[string]string{
		"uid":        "id",
		"target_url": "url",
		"secret":     "signing_secret",
		"rate_limit": "rate_limit",
	}
	for _, m := range mappings {
		if target, ok := expected[m.SourceField]; ok {
			if m.TargetField != target {
				t.Errorf("field %s: expected target %s, got %s", m.SourceField, target, m.TargetField)
			}
		}
	}
}

func TestSvixCompatLayer_CreateEndpoint(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	compat := NewSvixCompatLayer(svc)

	endpoint := SvixEndpointIn{
		UID:         "ep-123",
		URL:         "https://example.com/webhook",
		Version:     1,
		Description: "Test endpoint",
		FilterTypes: []string{"order.created"},
		Metadata:    map[string]string{"env": "test"},
	}

	out, err := compat.CreateEndpoint("app-1", endpoint)
	if err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}
	if out.UID != "ep-123" {
		t.Errorf("expected UID ep-123, got %s", out.UID)
	}
	if out.URL != "https://example.com/webhook" {
		t.Errorf("expected URL https://example.com/webhook, got %s", out.URL)
	}
	if out.Version != 1 {
		t.Errorf("expected version 1, got %d", out.Version)
	}
	if out.CreatedAt == "" {
		t.Error("expected createdAt to be set")
	}
}

func TestConvoyCompatLayer_CreateEndpoint(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	compat := NewConvoyCompatLayer(svc)

	endpoint := ConvoyEndpointIn{
		URL:         "https://example.com/webhook",
		Description: "Test endpoint",
		Secret:      "whsec_test",
		RateLimit:   100,
	}

	out, err := compat.CreateEndpoint("project-1", endpoint)
	if err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}
	if out.UID == "" {
		t.Error("expected UID to be set")
	}
	if out.TargetURL != "https://example.com/webhook" {
		t.Errorf("expected URL https://example.com/webhook, got %s", out.TargetURL)
	}
	if out.Status != "active" {
		t.Errorf("expected status active, got %s", out.Status)
	}
	if out.CreatedAt == "" {
		t.Error("expected createdAt to be set")
	}
}

func TestGetCheckpoint_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.GetCheckpoint(ctx, "nonexistent-migration")
	if err == nil {
		t.Error("expected error for nonexistent checkpoint")
	}
}
