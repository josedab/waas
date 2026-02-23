package standardwebhooks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_SignAndVerify(t *testing.T) {
	t.Parallel()
	svc := NewService()

	payload := json.RawMessage(`{"event":"user.created","data":{"id":"123"}}`)
	secret := "whsec_testsecret123"

	signResp, err := svc.Sign("msg_abc123", payload, secret)
	require.NoError(t, err)
	assert.NotEmpty(t, signResp.Headers[HeaderWebhookID])
	assert.NotEmpty(t, signResp.Headers[HeaderWebhookTimestamp])
	assert.NotEmpty(t, signResp.Headers[HeaderWebhookSignature])
	assert.Equal(t, "msg_abc123", signResp.Headers[HeaderWebhookID])

	verifyResp, err := svc.Verify(signResp.Headers, payload, secret, 60)
	require.NoError(t, err)
	assert.True(t, verifyResp.Valid)
}

func TestService_VerifyInvalidSignature(t *testing.T) {
	t.Parallel()
	svc := NewService()

	payload := json.RawMessage(`{"test":true}`)
	signResp, _ := svc.Sign("msg_1", payload, "secret1")

	verifyResp, err := svc.Verify(signResp.Headers, payload, "wrong_secret", 60)
	require.NoError(t, err)
	assert.False(t, verifyResp.Valid)
	assert.Equal(t, "signature mismatch", verifyResp.Reason)
}

func TestService_VerifyExpiredTimestamp(t *testing.T) {
	t.Parallel()
	svc := NewService()

	headers := map[string]string{
		HeaderWebhookID:        "msg_old",
		HeaderWebhookTimestamp: "1000000000",
		HeaderWebhookSignature: "v1,invalid",
	}

	verifyResp, err := svc.Verify(headers, json.RawMessage(`{}`), "secret", 60)
	require.NoError(t, err)
	assert.False(t, verifyResp.Valid)
	assert.Contains(t, verifyResp.Reason, "tolerance")
}

func TestService_DetectFormat_StandardWebhooks(t *testing.T) {
	t.Parallel()
	svc := NewService()

	headers := map[string]string{
		HeaderWebhookID:        "msg_1",
		HeaderWebhookTimestamp: "1700000000",
		HeaderWebhookSignature: "v1,abc",
	}

	result := svc.DetectFormat(headers, json.RawMessage(`{}`))
	assert.Equal(t, FormatStandardWebhooks, result.Format)
	assert.Equal(t, 1.0, result.Confidence)
}

func TestService_DetectFormat_CloudEventsBinary(t *testing.T) {
	t.Parallel()
	svc := NewService()

	headers := map[string]string{
		HeaderCESpecVersion: "1.0",
		HeaderCEID:          "evt-1",
		HeaderCESource:      "test",
		HeaderCEType:        "test.event",
	}

	result := svc.DetectFormat(headers, json.RawMessage(`{"data":"value"}`))
	assert.Equal(t, FormatCloudEvents, result.Format)
	assert.Equal(t, ContentModeBinary, result.ContentMode)
}

func TestService_DetectFormat_CloudEventsStructured(t *testing.T) {
	t.Parallel()
	svc := NewService()

	ce := CloudEvent{
		SpecVersion: "1.0",
		ID:          "evt-1",
		Source:      "test",
		Type:        "test.event",
		Data:        json.RawMessage(`{}`),
	}
	payload, _ := json.Marshal(ce)

	result := svc.DetectFormat(map[string]string{}, payload)
	assert.Equal(t, FormatCloudEvents, result.Format)
	assert.Equal(t, ContentModeStructured, result.ContentMode)
}

func TestService_ConvertToCloudEvents(t *testing.T) {
	t.Parallel()
	svc := NewService()

	headers := map[string]string{HeaderWebhookID: "msg_test"}
	payload := json.RawMessage(`{"user":"alice"}`)

	result, err := svc.ConvertToCloudEvents(headers, payload, "waas-test", "user.created")
	require.NoError(t, err)
	assert.Equal(t, FormatCloudEvents, result.Format)

	var ce CloudEvent
	require.NoError(t, json.Unmarshal(result.Payload, &ce))
	assert.Equal(t, "1.0", ce.SpecVersion)
	assert.Equal(t, "msg_test", ce.ID)
	assert.Equal(t, "user.created", ce.Type)
}

func TestService_ConvertToStandardWebhooks(t *testing.T) {
	t.Parallel()
	svc := NewService()

	ce := CloudEvent{
		SpecVersion: "1.0",
		ID:          "evt-1",
		Source:      "test",
		Type:        "order.completed",
		Data:        json.RawMessage(`{"amount":100}`),
	}
	payload, _ := json.Marshal(ce)

	result, err := svc.ConvertToStandardWebhooks(map[string]string{}, payload)
	require.NoError(t, err)
	assert.Equal(t, FormatStandardWebhooks, result.Format)
	assert.Equal(t, "evt-1", result.Headers[HeaderWebhookID])
}

func TestService_RunConformance_StandardWebhooks(t *testing.T) {
	t.Parallel()
	svc := NewService()

	result := svc.RunConformanceTests(FormatStandardWebhooks)
	assert.Equal(t, FormatStandardWebhooks, result.Format)
	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 5, result.Passed)
	assert.Equal(t, 0, result.Failed)
}

func TestService_RunConformance_CloudEvents(t *testing.T) {
	t.Parallel()
	svc := NewService()

	result := svc.RunConformanceTests(FormatCloudEvents)
	assert.Equal(t, FormatCloudEvents, result.Format)
	assert.Equal(t, 4, result.Total)
	assert.Equal(t, 4, result.Passed)
	assert.Equal(t, 0, result.Failed)
}
