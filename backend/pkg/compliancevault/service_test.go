package compliancevault

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorePayload(t *testing.T) {
	svc := NewService(nil, nil)

	req := &StorePayloadRequest{
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		EventType:  "order.created",
		Payload:    json.RawMessage(`{"order_id": "123"}`),
	}

	entry, err := svc.StorePayload(context.Background(), "tenant-1", req)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
	assert.Equal(t, "tenant-1", entry.TenantID)
	assert.Equal(t, "order.created", entry.EventType)
	assert.NotEmpty(t, entry.PayloadHash)
	assert.Equal(t, int64(19), entry.SizeBytes)
}

func TestStorePayloadValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.StorePayload(context.Background(), "t1", &StorePayloadRequest{})
	assert.Error(t, err)

	_, err = svc.StorePayload(context.Background(), "t1", &StorePayloadRequest{
		WebhookID: "w", EndpointID: "e", EventType: "t",
	})
	assert.Error(t, err)
}

func TestStorePayloadWithoutEncryptor(t *testing.T) {
	svc := NewService(nil, nil)

	entry, err := svc.StorePayload(context.Background(), "tenant-1", &StorePayloadRequest{
		WebhookID: "wh-1", EndpointID: "ep-1", EventType: "test",
		Payload: json.RawMessage(`{"data": "value"}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "none", entry.EncryptionAlgo)
}

func TestCreateRetentionPolicy(t *testing.T) {
	svc := NewService(nil, nil)

	policy, err := svc.CreateRetentionPolicy(context.Background(), "tenant-1", &CreateRetentionPolicyRequest{
		Name:          "GDPR 30-day",
		Framework:     FrameworkGDPR,
		RetentionDays: 30,
		Action:        RetentionActionDelete,
	})
	require.NoError(t, err)
	assert.Equal(t, "GDPR 30-day", policy.Name)
	assert.True(t, policy.IsActive)
}

func TestCreateRetentionPolicyInvalidFramework(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.CreateRetentionPolicy(context.Background(), "t1", &CreateRetentionPolicyRequest{
		Name: "test", Framework: "invalid", RetentionDays: 30, Action: RetentionActionDelete,
	})
	assert.Error(t, err)
}

func TestGenerateComplianceReport(t *testing.T) {
	svc := NewService(nil, nil)

	report, err := svc.GenerateComplianceReport(context.Background(), "tenant-1", &GenerateReportRequest{
		Framework: FrameworkGDPR,
	})
	require.NoError(t, err)
	assert.Equal(t, FrameworkGDPR, report.Framework)
	assert.Equal(t, 6, report.TotalControls)
	assert.NotEmpty(t, report.Findings)
}

func TestRequestErasure(t *testing.T) {
	svc := NewService(nil, nil)

	erasure, err := svc.RequestErasure(context.Background(), "tenant-1", &CreateErasureRequest{
		SubjectID:   "user-123",
		SubjectType: "customer",
		Reason:      "GDPR right to erasure",
	})
	require.NoError(t, err)
	assert.Equal(t, "user-123", erasure.SubjectID)
}

func TestRequestErasureValidation(t *testing.T) {
	svc := NewService(nil, nil)

	_, err := svc.RequestErasure(context.Background(), "t1", &CreateErasureRequest{})
	assert.Error(t, err)
}
