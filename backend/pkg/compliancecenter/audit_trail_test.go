package compliancecenter

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAuditTrailRecordAndVerify(t *testing.T) {
	repo := newMemoryAuditRepo()
	service := NewAuditTrailService(repo, nil)
	ctx := context.Background()

	// Record several events
	events := []struct {
		eventType AuditEventType
		action    string
	}{
		{AuditEndpointCreated, "create"},
		{AuditWebhookCreated, "create"},
		{AuditDeliveryAttempted, "deliver"},
		{AuditDeliverySucceeded, "deliver"},
		{AuditConfigChanged, "update"},
	}

	for _, e := range events {
		err := service.RecordEvent(ctx, "tenant-1", e.eventType,
			AuditActor{ID: "user-1", Type: "user", Name: "Alice"},
			AuditResource{Type: "webhook", ID: "wh-1"},
			e.action, "success",
			[]byte(`{"key":"value"}`), json.RawMessage(`{"detail":"test"}`),
			"127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to record event %s: %v", e.eventType, err)
		}
	}

	// Verify integrity
	report, err := service.VerifyIntegrity(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to verify integrity: %v", err)
	}
	if !report.IsIntact {
		t.Fatalf("audit trail should be intact, violations: %+v", report.Violations)
	}
	if report.TotalEntries != 5 {
		t.Fatalf("expected 5 entries, got %d", report.TotalEntries)
	}
}

func TestAuditTrailTamperDetection(t *testing.T) {
	repo := newMemoryAuditRepo()
	service := NewAuditTrailService(repo, nil)
	ctx := context.Background()

	// Record events
	for i := 0; i < 5; i++ {
		service.RecordEvent(ctx, "tenant-1", AuditDeliverySucceeded,
			AuditActor{ID: "sys", Type: "system"},
			AuditResource{Type: "delivery", ID: "d-1"},
			"deliver", "success", nil, nil, "", "")
	}

	// Tamper with an entry
	entries, _, _ := repo.ListEntries(ctx, "tenant-1", &AuditTrailFilters{Limit: 100})
	if len(entries) >= 3 {
		entries[2].EntryHash = "tampered_hash"
		repo.entries["tenant-1"][2] = entries[2]
	}

	// Verify should detect tampering
	report, _ := service.VerifyIntegrity(ctx, "tenant-1")
	if report.IsIntact {
		t.Fatal("should detect tampering")
	}
	if len(report.Violations) == 0 {
		t.Fatal("should have violations")
	}
}

func TestPayloadFingerprint(t *testing.T) {
	payload1 := []byte(`{"order_id":"123","amount":99.99}`)
	payload2 := []byte(`{"order_id":"123","amount":99.99}`)
	payload3 := []byte(`{"order_id":"456","amount":50.00}`)

	fp1 := ComputePayloadFingerprint(payload1)
	fp2 := ComputePayloadFingerprint(payload2)
	fp3 := ComputePayloadFingerprint(payload3)

	if fp1 != fp2 {
		t.Fatal("identical payloads should have same fingerprint")
	}
	if fp1 == fp3 {
		t.Fatal("different payloads should have different fingerprints")
	}
	if fp1[:7] != "sha256:" {
		t.Fatalf("fingerprint should start with 'sha256:', got %s", fp1[:7])
	}
}

func TestComplianceExportSOC2(t *testing.T) {
	repo := newMemoryAuditRepo()
	service := NewAuditTrailService(repo, nil)
	ctx := context.Background()

	// Record diverse events
	service.RecordEvent(ctx, "tenant-1", AuditLoginSuccess, AuditActor{ID: "u1", Type: "user"}, AuditResource{Type: "session", ID: "s1"}, "login", "success", nil, nil, "", "")
	service.RecordEvent(ctx, "tenant-1", AuditWebhookCreated, AuditActor{ID: "u1", Type: "user"}, AuditResource{Type: "webhook", ID: "w1"}, "create", "success", nil, nil, "", "")
	service.RecordEvent(ctx, "tenant-1", AuditDeliverySucceeded, AuditActor{ID: "sys", Type: "system"}, AuditResource{Type: "delivery", ID: "d1"}, "deliver", "success", nil, nil, "", "")

	export, err := service.GenerateComplianceExport(ctx, "tenant-1", FrameworkSOC2,
		time.Now().Add(-24*time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to generate export: %v", err)
	}

	if len(export.Sections) == 0 {
		t.Fatal("should have SOC2 sections")
	}
	if export.TotalEntries != 3 {
		t.Fatalf("expected 3 entries, got %d", export.TotalEntries)
	}
}

func TestComplianceExportGDPR(t *testing.T) {
	repo := newMemoryAuditRepo()
	service := NewAuditTrailService(repo, nil)
	ctx := context.Background()

	service.RecordEvent(ctx, "tenant-1", AuditDataDeleted, AuditActor{ID: "u1", Type: "user"}, AuditResource{Type: "data", ID: "d1"}, "delete", "success", nil, nil, "", "")

	export, err := service.GenerateComplianceExport(ctx, "tenant-1", FrameworkGDPR,
		time.Now().Add(-24*time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to generate GDPR export: %v", err)
	}

	if len(export.Sections) != 3 {
		t.Fatalf("expected 3 GDPR sections, got %d", len(export.Sections))
	}
}

// --- In-memory repository for testing ---

type memoryAuditRepo struct {
	entries map[string][]ImmutableAuditEntry
}

func newMemoryAuditRepo() *memoryAuditRepo {
	return &memoryAuditRepo{entries: make(map[string][]ImmutableAuditEntry)}
}

func (r *memoryAuditRepo) AppendEntry(ctx context.Context, entry *ImmutableAuditEntry) error {
	r.entries[entry.TenantID] = append(r.entries[entry.TenantID], *entry)
	return nil
}

func (r *memoryAuditRepo) GetEntry(ctx context.Context, tenantID, entryID string) (*ImmutableAuditEntry, error) {
	for _, e := range r.entries[tenantID] {
		if e.ID == entryID {
			return &e, nil
		}
	}
	return nil, nil
}

func (r *memoryAuditRepo) ListEntries(ctx context.Context, tenantID string, filters *AuditTrailFilters) ([]ImmutableAuditEntry, int, error) {
	all := r.entries[tenantID]
	var filtered []ImmutableAuditEntry

	for _, e := range all {
		if filters.StartTime != nil && e.Timestamp.Before(*filters.StartTime) {
			continue
		}
		if filters.EndTime != nil && e.Timestamp.After(*filters.EndTime) {
			continue
		}
		if len(filters.EventTypes) > 0 {
			match := false
			for _, t := range filters.EventTypes {
				if e.EventType == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)
	start := filters.Offset
	if start >= total {
		return nil, total, nil
	}
	end := start + filters.Limit
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (r *memoryAuditRepo) GetLatestEntry(ctx context.Context, tenantID string) (*ImmutableAuditEntry, error) {
	entries := r.entries[tenantID]
	if len(entries) == 0 {
		return nil, nil
	}
	latest := entries[len(entries)-1]
	return &latest, nil
}

func (r *memoryAuditRepo) GetSequenceRange(ctx context.Context, tenantID string, startSeq, endSeq int64) ([]ImmutableAuditEntry, error) {
	var result []ImmutableAuditEntry
	for _, e := range r.entries[tenantID] {
		if e.SequenceNumber >= startSeq && e.SequenceNumber <= endSeq {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *memoryAuditRepo) CountEntries(ctx context.Context, tenantID string, filters *AuditTrailFilters) (int64, error) {
	return int64(len(r.entries[tenantID])), nil
}
