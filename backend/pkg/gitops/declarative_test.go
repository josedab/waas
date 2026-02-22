package gitops

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validDeclarativeYAML = `
apiVersion: waas/v1
kind: WebhookConfig
metadata:
  name: my-webhooks
  labels:
    env: production
spec:
  endpoints:
    - name: orders-endpoint
      url: https://api.example.com/webhooks/orders
      rate_limit: 100
      timeout: 30s
    - name: payments-endpoint
      url: https://api.example.com/webhooks/payments
  subscriptions:
    - name: order-events
      endpoint: orders-endpoint
      event_types:
        - order.created
        - order.updated
    - name: payment-events
      endpoint: payments-endpoint
      event_types:
        - payment.completed
      filter: amount_filter
  routing_rules:
    - name: high-value-routing
      priority: 1
      conditions:
        - field: payload.amount
          operator: gt
          value: "1000"
      actions:
        - type: route
          endpoint: payments-endpoint
  retry_policies:
    - name: default-retry
      max_retries: 5
      backoff: exponential
      initial_delay: 1s
      max_delay: 60s
  filters:
    - name: amount_filter
      expression: "payload.amount > 0"
      action: include
`

func TestParseDeclarativeConfig(t *testing.T) {
	config, errors := ParseDeclarativeConfig(validDeclarativeYAML)
	assert.Empty(t, errors)
	require.NotNil(t, config)

	assert.Equal(t, "waas/v1", config.APIVersion)
	assert.Equal(t, "WebhookConfig", config.Kind)
	assert.Equal(t, "my-webhooks", config.Metadata.Name)
	assert.Len(t, config.Spec.Endpoints, 2)
	assert.Len(t, config.Spec.Subscriptions, 2)
	assert.Len(t, config.Spec.RoutingRules, 1)
	assert.Len(t, config.Spec.RetryPolicies, 1)
	assert.Len(t, config.Spec.Filters, 1)

	assert.Equal(t, "orders-endpoint", config.Spec.Endpoints[0].Name)
	assert.Equal(t, 100, config.Spec.Endpoints[0].RateLimit)
	assert.Equal(t, "order-events", config.Spec.Subscriptions[0].Name)
	assert.Equal(t, 2, len(config.Spec.Subscriptions[0].EventTypes))
}

func TestParseDeclarativeConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		errCount int
	}{
		{"missing apiVersion", `kind: WebhookConfig
metadata:
  name: test
spec: {}`, 1},
		{"missing endpoint name", `apiVersion: v1
kind: WebhookConfig
metadata:
  name: test
spec:
  endpoints:
    - url: http://test.com`, 1},
		{"missing subscription endpoint", `apiVersion: v1
kind: WebhookConfig
metadata:
  name: test
spec:
  subscriptions:
    - name: test
      event_types:
        - test.event`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := ParseDeclarativeConfig(tt.yaml)
			assert.GreaterOrEqual(t, len(errors), tt.errCount)
		})
	}
}

func TestApplyDeclarativeConfigDryRun(t *testing.T) {
	svc := NewService(nil)

	result, err := svc.ApplyDeclarativeConfig(context.Background(), "tenant-1", validDeclarativeYAML, true)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ManifestID)
	assert.Equal(t, SyncStatusSynced, result.Status)
	// 2 endpoints + 2 subscriptions + 1 routing rule + 1 retry policy + 1 filter = 7
	assert.Len(t, result.ResourcesSync, 7)

	for _, rs := range result.ResourcesSync {
		assert.Contains(t, rs.Action, "dry_run:")
	}
}

func TestValidateDeclarativeConfig(t *testing.T) {
	svc := NewService(nil)

	errors, err := svc.ValidateDeclarativeConfig(context.Background(), validDeclarativeYAML)
	assert.NoError(t, err)
	assert.Empty(t, errors)

	errors, err = svc.ValidateDeclarativeConfig(context.Background(), "invalid yaml: [")
	assert.Error(t, err)
	assert.NotEmpty(t, errors)
}
