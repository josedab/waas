package compliancecenter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestImmutableAuditVault_AppendAndVerify(t *testing.T) {
	vault := NewImmutableAuditVault()

	// Append entries
	e1, err := vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success",
		json.RawMessage(`{"endpoint": "https://api.example.com"}`))
	assert.NoError(t, err)
	assert.Equal(t, int64(1), e1.SequenceNumber)
	assert.Equal(t, "genesis", e1.PreviousHash)
	assert.NotEmpty(t, e1.EntryHash)

	e2, err := vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "failure",
		json.RawMessage(`{"error": "timeout"}`))
	assert.NoError(t, err)
	assert.Equal(t, int64(2), e2.SequenceNumber)
	assert.Equal(t, e1.EntryHash, e2.PreviousHash)

	// Verify chain
	valid, issues := vault.VerifyChain()
	assert.True(t, valid, "chain should be valid")
	assert.Empty(t, issues)
}

func TestImmutableAuditVault_TamperDetection(t *testing.T) {
	vault := NewImmutableAuditVault()

	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success", nil)
	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success", nil)
	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success", nil)

	// Tamper with an entry
	vault.entries[1].Action = "tampered"

	valid, issues := vault.VerifyChain()
	assert.False(t, valid, "chain should detect tampering")
	assert.NotEmpty(t, issues)
}

func TestImmutableAuditVault_GetEntries(t *testing.T) {
	vault := NewImmutableAuditVault()

	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success", nil)
	vault.AppendEntry("tenant-2", AuditDeliverySucceeded, "deliver", "success", nil)
	vault.AppendEntry("tenant-1", AuditEndpointCreated, "create", "success", nil)

	entries := vault.GetEntries("tenant-1", 10)
	assert.Len(t, entries, 2)
}

func TestProcessDSAR(t *testing.T) {
	vault := NewImmutableAuditVault()

	// Add entries with subject data
	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success",
		json.RawMessage(`{"email": "user@example.com", "data": "test"}`))
	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success",
		json.RawMessage(`{"email": "other@example.com"}`))
	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success",
		json.RawMessage(`{"email": "user@example.com", "action": "update"}`))

	req := &DSARRequest{
		ID:           "dsar-1",
		TenantID:     "tenant-1",
		SubjectEmail: "user@example.com",
		RequestType:  "access",
		RequestedAt:  time.Now(),
		DueDate:      time.Now().AddDate(0, 0, 30),
	}

	response, err := ProcessDSAR(req, vault)
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", response.SubjectEmail)
	assert.Greater(t, response.TotalRecords, 0)
}

func TestProcessDSAR_MissingSubject(t *testing.T) {
	vault := NewImmutableAuditVault()
	_, err := ProcessDSAR(&DSARRequest{ID: "1", TenantID: "t1"}, vault)
	assert.Error(t, err)
}

func TestGenerateSOC2Report(t *testing.T) {
	vault := NewImmutableAuditVault()

	vault.AppendEntry("tenant-1", AuditDeliverySucceeded, "deliver", "success", nil)
	vault.AppendEntry("tenant-1", AuditEndpointCreated, "create", "success", nil)

	period := ReportPeriod{
		StartDate: time.Now().AddDate(0, -3, 0),
		EndDate:   time.Now(),
	}

	report := GenerateSOC2Report("tenant-1", period, vault)
	assert.Equal(t, FrameworkSOC2, report.Framework)
	assert.Equal(t, "detailed", report.ReportType)
	assert.GreaterOrEqual(t, report.Summary.OverallScore, 80)
	assert.Len(t, report.Sections, 5)

	// Verify sections cover all TSC
	sectionTitles := make([]string, len(report.Sections))
	for i, s := range report.Sections {
		sectionTitles[i] = s.Title
	}
	assert.Contains(t, sectionTitles, "Security")
	assert.Contains(t, sectionTitles, "Availability")
	assert.Contains(t, sectionTitles, "Processing Integrity")
	assert.Contains(t, sectionTitles, "Confidentiality")
	assert.Contains(t, sectionTitles, "Privacy")
}

func TestDefaultRetentionPolicies(t *testing.T) {
	soc2 := DefaultRetentionPolicies[FrameworkSOC2]
	assert.Equal(t, 365, soc2.RetentionDays)
	assert.True(t, soc2.EncryptAtRest)

	gdpr := DefaultRetentionPolicies[FrameworkGDPR]
	assert.Equal(t, 180, gdpr.RetentionDays)

	hipaa := DefaultRetentionPolicies[FrameworkHIPAA]
	assert.Equal(t, 2190, hipaa.RetentionDays)
}
