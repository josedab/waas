package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Webhook model tests ---

func TestWebhookEndpoint_JSONRoundTrip(t *testing.T) {
	original := WebhookEndpoint{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		URL:        "https://example.com/hook",
		SecretHash: "hidden",
		IsActive:   true,
		RetryConfig: RetryConfiguration{
			MaxAttempts:       5,
			InitialDelayMs:    1000,
			MaxDelayMs:        300000,
			BackoffMultiplier: 2,
		},
		CustomHeaders: map[string]string{"X-Custom": "value"},
		CreatedAt:     time.Now().Truncate(time.Second),
		UpdatedAt:     time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded WebhookEndpoint
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.URL, decoded.URL)
	assert.Equal(t, original.IsActive, decoded.IsActive)
	assert.Equal(t, original.RetryConfig.MaxAttempts, decoded.RetryConfig.MaxAttempts)
	assert.Equal(t, original.CustomHeaders["X-Custom"], decoded.CustomHeaders["X-Custom"])
	// SecretHash has json:"-" so it should NOT appear in JSON
	assert.Empty(t, decoded.SecretHash)
}

func TestWebhookEndpoint_SecretHashNotInJSON(t *testing.T) {
	ep := WebhookEndpoint{
		ID:         uuid.New(),
		SecretHash: "super-secret-hash",
	}
	data, _ := json.Marshal(ep)
	assert.NotContains(t, string(data), "super-secret-hash")
}

func TestRetryConfiguration_JSON(t *testing.T) {
	config := RetryConfiguration{
		MaxAttempts:       3,
		InitialDelayMs:    500,
		MaxDelayMs:        30000,
		BackoffMultiplier: 2,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var decoded RetryConfiguration
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, config, decoded)
}

func TestDeliveryAttempt_JSONRoundTrip(t *testing.T) {
	httpStatus := 200
	body := "OK"
	attempt := DeliveryAttempt{
		ID:            uuid.New(),
		EndpointID:    uuid.New(),
		PayloadHash:   "sha256-" + strings.Repeat("a", 64),
		PayloadSize:   1024,
		Status:        "delivered",
		HTTPStatus:    &httpStatus,
		ResponseBody:  &body,
		AttemptNumber: 1,
	}

	data, err := json.Marshal(attempt)
	require.NoError(t, err)

	var decoded DeliveryAttempt
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, attempt.ID, decoded.ID)
	assert.Equal(t, *attempt.HTTPStatus, *decoded.HTTPStatus)
}

// --- Tenant model tests ---

func TestTenant_JSONRoundTrip(t *testing.T) {
	original := Tenant{
		ID:                 uuid.New(),
		Name:               "Test Tenant",
		APIKeyHash:         "should-be-hidden",
		SubscriptionTier:   "premium",
		RateLimitPerMinute: 500,
		MonthlyQuota:       50000,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	// APIKeyHash has json:"-"
	assert.NotContains(t, string(data), "should-be-hidden")

	var decoded Tenant
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Empty(t, decoded.APIKeyHash)
}

func TestTenant_ZeroValues(t *testing.T) {
	tenant := Tenant{}
	assert.Equal(t, uuid.Nil, tenant.ID)
	assert.Empty(t, tenant.Name)
	assert.Equal(t, 0, tenant.RateLimitPerMinute)
	assert.Equal(t, 0, tenant.MonthlyQuota)
}

// --- Transformation model tests ---

func TestTransformation_JSONRoundTrip(t *testing.T) {
	original := Transformation{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Name:        "My Transform",
		Description: "A test transformation",
		Script:      "return payload;",
		Enabled:     true,
		Version:     3,
		Config:      DefaultTransformConfig(),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Transformation
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Script, decoded.Script)
	assert.Equal(t, original.Config.TimeoutMs, decoded.Config.TimeoutMs)
}

func TestDefaultTransformConfig(t *testing.T) {
	config := DefaultTransformConfig()
	assert.Equal(t, 5000, config.TimeoutMs)
	assert.Equal(t, 64, config.MaxMemoryMB)
	assert.False(t, config.AllowHTTP)
	assert.True(t, config.EnableLogging)
}

func TestEndpointTransformation_JSONRoundTrip(t *testing.T) {
	et := EndpointTransformation{
		ID:               uuid.New(),
		EndpointID:       uuid.New(),
		TransformationID: uuid.New(),
		Priority:         1,
		Enabled:          true,
	}

	data, err := json.Marshal(et)
	require.NoError(t, err)

	var decoded EndpointTransformation
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, et.ID, decoded.ID)
	assert.Equal(t, et.Priority, decoded.Priority)
}

// --- Validation extended tests ---

func TestValidateWebhookEndpoint_ExtendedCases(t *testing.T) {
	validEndpoint := func() *WebhookEndpoint {
		return &WebhookEndpoint{
			TenantID:   uuid.New(),
			URL:        "https://example.com/hook",
			SecretHash: "valid-hash",
			RetryConfig: RetryConfiguration{
				MaxAttempts:       5,
				InitialDelayMs:    1000,
				MaxDelayMs:        300000,
				BackoffMultiplier: 2,
			},
		}
	}

	tests := []struct {
		name      string
		modify    func(ep *WebhookEndpoint)
		expectErr bool
		errMsg    string
	}{
		{"valid endpoint", func(ep *WebhookEndpoint) {}, false, ""},
		{"nil endpoint", nil, true, "cannot be nil"},
		{"nil tenant ID", func(ep *WebhookEndpoint) { ep.TenantID = uuid.Nil }, true, "tenant ID"},
		{"empty URL", func(ep *WebhookEndpoint) { ep.URL = "" }, true, "URL cannot be empty"},
		{"HTTP URL", func(ep *WebhookEndpoint) { ep.URL = "http://example.com" }, true, "HTTPS"},
		{"empty secret hash", func(ep *WebhookEndpoint) { ep.SecretHash = "" }, true, "secret hash"},
		{"zero max attempts", func(ep *WebhookEndpoint) { ep.RetryConfig.MaxAttempts = 0 }, true, "max attempts"},
		{"backoff multiplier = 1", func(ep *WebhookEndpoint) { ep.RetryConfig.BackoffMultiplier = 1 }, true, "backoff multiplier"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ep *WebhookEndpoint
			if tt.modify != nil {
				ep = validEndpoint()
				tt.modify(ep)
			}
			err := ValidateWebhookEndpoint(ep)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDeliveryAttempt_TableDriven(t *testing.T) {
	validHash := "sha256-" + strings.Repeat("a", 64)
	validAttempt := func() *DeliveryAttempt {
		return &DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   validHash,
			PayloadSize:   100,
			Status:        "pending",
			AttemptNumber: 1,
		}
	}

	tests := []struct {
		name      string
		modify    func(a *DeliveryAttempt)
		expectErr bool
	}{
		{"valid attempt", func(a *DeliveryAttempt) {}, false},
		{"nil attempt", nil, true},
		{"nil endpoint ID", func(a *DeliveryAttempt) { a.EndpointID = uuid.Nil }, true},
		{"empty payload hash", func(a *DeliveryAttempt) { a.PayloadHash = "" }, true},
		{"invalid hash format", func(a *DeliveryAttempt) { a.PayloadHash = "md5-abc" }, true},
		{"hash wrong length", func(a *DeliveryAttempt) { a.PayloadHash = "sha256-tooshort" }, true},
		{"zero payload size", func(a *DeliveryAttempt) { a.PayloadSize = 0 }, true},
		{"invalid status", func(a *DeliveryAttempt) { a.Status = "unknown" }, true},
		{"zero attempt number", func(a *DeliveryAttempt) { a.AttemptNumber = 0 }, true},
		{"status=delivered", func(a *DeliveryAttempt) { a.Status = "delivered" }, false},
		{"status=failed", func(a *DeliveryAttempt) { a.Status = "failed" }, false},
		{"status=retrying", func(a *DeliveryAttempt) { a.Status = "retrying" }, false},
		{"status=processing", func(a *DeliveryAttempt) { a.Status = "processing" }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a *DeliveryAttempt
			if tt.modify != nil {
				a = validAttempt()
				tt.modify(a)
			}
			err := ValidateDeliveryAttempt(a)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRetryConfiguration_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		config    *RetryConfiguration
		expectErr bool
	}{
		{"valid config", &RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2}, false},
		{"nil config", nil, true},
		{"zero max attempts", &RetryConfiguration{MaxAttempts: 0, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 2}, true},
		{"negative initial delay", &RetryConfiguration{MaxAttempts: 3, InitialDelayMs: -1, MaxDelayMs: 30000, BackoffMultiplier: 2}, true},
		{"max delay < initial delay", &RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 5000, MaxDelayMs: 1000, BackoffMultiplier: 2}, true},
		{"backoff = 1", &RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 1}, true},
		{"backoff = 0", &RetryConfiguration{MaxAttempts: 3, InitialDelayMs: 1000, MaxDelayMs: 30000, BackoffMultiplier: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRetryConfiguration(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Compliance constants tests ---

func TestComplianceFrameworkConstants(t *testing.T) {
	frameworks := []string{
		ComplianceFrameworkSOC2,
		ComplianceFrameworkHIPAA,
		ComplianceFrameworkGDPR,
		ComplianceFrameworkPCIDSS,
		ComplianceFrameworkCCPA,
	}

	for _, f := range frameworks {
		assert.NotEmpty(t, f)
	}
	assert.Equal(t, "soc2", ComplianceFrameworkSOC2)
	assert.Equal(t, "gdpr", ComplianceFrameworkGDPR)
}

func TestPIICategoryConstants(t *testing.T) {
	categories := []string{
		PIICategoryEmail,
		PIICategoryPhone,
		PIICategorySSN,
		PIICategoryCreditCard,
		PIICategoryName,
		PIICategoryAddress,
		PIICategoryDOB,
		PIICategoryIPAddress,
	}

	for _, c := range categories {
		assert.NotEmpty(t, c)
	}
}

func TestSensitivityLevelConstants(t *testing.T) {
	levels := []string{SensitivityLow, SensitivityMedium, SensitivityHigh, SensitivityCritical}
	for _, l := range levels {
		assert.NotEmpty(t, l)
	}
}

// --- Analytics model tests ---

func TestDeliveryMetric_ZeroValue(t *testing.T) {
	metric := DeliveryMetric{}
	assert.Equal(t, uuid.Nil, metric.ID)
	assert.Nil(t, metric.HTTPStatus)
	assert.Equal(t, 0, metric.LatencyMs)
}

func TestDeliveryMetric_JSONRoundTrip(t *testing.T) {
	httpStatus := 200
	metric := DeliveryMetric{
		ID:            uuid.New(),
		TenantID:      uuid.New(),
		EndpointID:    uuid.New(),
		DeliveryID:    uuid.New(),
		Status:        "delivered",
		HTTPStatus:    &httpStatus,
		LatencyMs:     150,
		AttemptNumber: 1,
		CreatedAt:     time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(metric)
	require.NoError(t, err)

	var decoded DeliveryMetric
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, metric.ID, decoded.ID)
	assert.Equal(t, 200, *decoded.HTTPStatus)
	assert.Equal(t, 150, decoded.LatencyMs)
}

func TestHourlyMetric_OptionalFields(t *testing.T) {
	metric := HourlyMetric{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		TotalDeliveries:      100,
		SuccessfulDeliveries: 95,
		FailedDeliveries:     5,
	}

	assert.Nil(t, metric.EndpointID)
	assert.Nil(t, metric.AvgLatencyMs)
	assert.Nil(t, metric.P95LatencyMs)
}

func TestContainsHelper_Extended(t *testing.T) {
	assert.True(t, contains([]string{"a", "b", "c"}, "b"))
	assert.False(t, contains([]string{"a", "b", "c"}, "d"))
	assert.False(t, contains([]string{}, "a"))
	assert.False(t, contains(nil, "a"))
}
