package piidetection

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetector_ScanAndMask_Email(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{CategoryEmail},
		MaskingAction: ActionRedact,
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"user":"test@example.com","name":"Alice"}`)
	masked, detections, scanned, maskedCount := detector.ScanAndMask(payload)

	assert.Equal(t, 1, len(detections))
	assert.Equal(t, CategoryEmail, detections[0].Category)
	assert.Equal(t, "user", detections[0].FieldPath)
	assert.Equal(t, 2, scanned)
	assert.Equal(t, 1, maskedCount)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(masked, &result))
	assert.Equal(t, "[REDACTED]", result["user"])
	assert.Equal(t, "Alice", result["name"])
}

func TestDetector_ScanAndMask_SSN(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{CategorySSN},
		MaskingAction: ActionMask,
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"ssn":"123-45-6789"}`)
	masked, detections, _, _ := detector.ScanAndMask(payload)

	assert.Equal(t, 1, len(detections))
	assert.Equal(t, CategorySSN, detections[0].Category)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(masked, &result))
	assert.NotEqual(t, "123-45-6789", result["ssn"])
}

func TestDetector_ScanAndMask_Hash(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{CategoryEmail},
		MaskingAction: ActionHash,
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"email":"test@example.com"}`)
	masked, _, _, _ := detector.ScanAndMask(payload)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(masked, &result))
	assert.NotEqual(t, "test@example.com", result["email"])
	assert.NotContains(t, result["email"], "@")
}

func TestDetector_ScanAndMask_NestedPayload(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{CategoryEmail, CategorySSN},
		MaskingAction: ActionRedact,
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"customer":{"email":"test@example.com","details":{"ssn":"123-45-6789"}}}`)
	_, detections, _, maskedCount := detector.ScanAndMask(payload)

	assert.Equal(t, 2, len(detections))
	assert.Equal(t, 2, maskedCount)
}

func TestDetector_ScanAndMask_NoDetections(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{CategorySSN},
		MaskingAction: ActionMask,
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"name":"Alice","age":30}`)
	masked, detections, _, maskedCount := detector.ScanAndMask(payload)

	assert.Equal(t, 0, len(detections))
	assert.Equal(t, 0, maskedCount)
	assert.JSONEq(t, `{"name":"Alice","age":30}`, string(masked))
}

func TestDetector_CustomPattern(t *testing.T) {
	t.Parallel()
	policy := &Policy{
		Categories:    []string{},
		MaskingAction: ActionRedact,
		CustomPatterns: []CustomPattern{
			{Name: "Employee ID", Pattern: `EMP-\d{6}`, Label: "employee_id"},
		},
	}
	detector := NewDetector(policy)

	payload := json.RawMessage(`{"employee":"EMP-123456"}`)
	_, detections, _, maskedCount := detector.ScanAndMask(payload)

	assert.Equal(t, 1, len(detections))
	assert.Equal(t, "employee_id", detections[0].Category)
	assert.Equal(t, 1, maskedCount)
}

func TestService_CreatePolicy_Validation(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)

	tests := []struct {
		name    string
		req     CreatePolicyRequest
		wantErr bool
	}{
		{
			name: "valid policy",
			req: CreatePolicyRequest{
				Name:          "Test Policy",
				Sensitivity:   SensitivityMedium,
				Categories:    []string{CategoryEmail},
				MaskingAction: ActionMask,
			},
			wantErr: false,
		},
		{
			name: "invalid sensitivity",
			req: CreatePolicyRequest{
				Name:          "Bad",
				Sensitivity:   "ultra",
				Categories:    []string{CategoryEmail},
				MaskingAction: ActionMask,
			},
			wantErr: true,
		},
		{
			name: "invalid masking action",
			req: CreatePolicyRequest{
				Name:          "Bad",
				Sensitivity:   SensitivityLow,
				Categories:    []string{CategoryEmail},
				MaskingAction: "encrypt",
			},
			wantErr: true,
		},
		{
			name: "empty categories",
			req: CreatePolicyRequest{
				Name:          "Bad",
				Sensitivity:   SensitivityLow,
				Categories:    []string{},
				MaskingAction: ActionMask,
			},
			wantErr: true,
		},
		{
			name: "invalid category",
			req: CreatePolicyRequest{
				Name:          "Bad",
				Sensitivity:   SensitivityLow,
				Categories:    []string{"passport"},
				MaskingAction: ActionMask,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreatePolicy(context.Background(), "tenant-1", &tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_ScanPayload_NoPolicies(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	resp, err := svc.ScanPayload(context.Background(), "tenant-1", &ScanRequest{
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		Payload:    json.RawMessage(`{"data":"hello"}`),
	})

	require.NoError(t, err)
	assert.JSONEq(t, `{"data":"hello"}`, string(resp.MaskedPayload))
}
