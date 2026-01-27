package openapigen

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test JSON spec with webhooks
var testJSONSpec = []byte(`{
    "openapi": "3.0.0",
    "info": {"title": "Test API", "version": "1.0.0"},
    "webhooks": {
        "order.created": {
            "post": {
                "summary": "Order Created",
                "description": "Fired when an order is created",
                "operationId": "orderCreated",
                "tags": ["orders"],
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": {"type": "object", "properties": {"id": {"type": "string"}, "amount": {"type": "number"}}},
                            "example": {"id": "ord_123", "amount": 99.99}
                        }
                    }
                }
            }
        },
        "order.cancelled": {
            "post": {
                "summary": "Order Cancelled",
                "operationId": "orderCancelled"
            }
        }
    }
}`)

// Test YAML spec
var testYAMLSpec = []byte(`
openapi: "3.0.0"
info:
  title: YAML Test API
  version: "2.0.0"
webhooks:
  payment.completed:
    post:
      summary: Payment Completed
      operationId: paymentCompleted
      tags:
        - payments
`)

// Test spec with callbacks
var testCallbackSpec = []byte(`{
    "openapi": "3.0.0",
    "info": {"title": "Callback API", "version": "1.0.0"},
    "paths": {
        "/subscriptions": {
            "post": {
                "summary": "Create subscription",
                "operationId": "createSubscription",
                "callbacks": {
                    "onEvent": {
                        "{$request.body#/callbackUrl}": {
                            "post": {
                                "summary": "Subscription Event",
                                "operationId": "subscriptionEvent",
                                "requestBody": {
                                    "content": {
                                        "application/json": {
                                            "schema": {"type": "object"}
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}`)

func TestParseSpec_JSON(t *testing.T) {
	svc := NewService()
	spec, err := svc.ParseSpec(testJSONSpec)
	require.NoError(t, err)
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "Test API", spec.Info.Title)
	assert.Len(t, spec.Webhooks, 2)
}

func TestParseSpec_YAML(t *testing.T) {
	svc := NewService()
	spec, err := svc.ParseSpec(testYAMLSpec)
	require.NoError(t, err)
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "YAML Test API", spec.Info.Title)
	assert.Len(t, spec.Webhooks, 1)
}

func TestParseSpec_Empty(t *testing.T) {
	svc := NewService()
	_, err := svc.ParseSpec([]byte{})
	assert.Error(t, err)
}

func TestParseSpec_InvalidJSON(t *testing.T) {
	svc := NewService()
	_, err := svc.ParseSpec([]byte(`{invalid`))
	assert.Error(t, err)
}

func TestParseSpec_WrongVersion(t *testing.T) {
	svc := NewService()
	_, err := svc.ParseSpec([]byte(`{"openapi": "2.0", "info": {"title": "test", "version": "1.0"}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestParseSpec_MissingVersion(t *testing.T) {
	svc := NewService()
	_, err := svc.ParseSpec([]byte(`{"info": {"title": "test"}}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing openapi version")
}

func TestGenerateFromSpec_Webhooks(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, err := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{IncludeTests: true})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(config.EventTypes), 2)
	// Check that test fixtures were generated
	assert.NotEmpty(t, config.TestFixtures)
}

func TestGenerateFromSpec_WithNamespace(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, err := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{Namespace: "myapp"})
	require.NoError(t, err)
	for _, et := range config.EventTypes {
		assert.True(t, len(et.Slug) > 0)
	}
}

func TestGenerateFromSpec_Callbacks(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testCallbackSpec)
	config, err := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(config.EventTypes), 1)
}

func TestGenerateFromSpec_NilSpec(t *testing.T) {
	svc := NewService()
	_, err := svc.GenerateFromSpec(context.Background(), nil, GenerateOptions{})
	assert.Error(t, err)
}

func TestGenerateSDKClient_Go(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, _ := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	sdk, err := svc.GenerateSDKClient(context.Background(), config, "go")
	require.NoError(t, err)
	assert.Equal(t, "go", sdk.Language)
	assert.NotEmpty(t, sdk.Code)
}

func TestGenerateSDKClient_Python(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, _ := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	sdk, err := svc.GenerateSDKClient(context.Background(), config, "python")
	require.NoError(t, err)
	assert.Equal(t, "python", sdk.Language)
	assert.NotEmpty(t, sdk.Code)
}

func TestGenerateSDKClient_TypeScript(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, _ := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	sdk, err := svc.GenerateSDKClient(context.Background(), config, "typescript")
	require.NoError(t, err)
	assert.Equal(t, "typescript", sdk.Language)
	assert.NotEmpty(t, sdk.Code)
}

func TestGenerateSDKClient_UnsupportedLang(t *testing.T) {
	svc := NewService()
	config := &GeneratedConfig{}
	_, err := svc.GenerateSDKClient(context.Background(), config, "rust")
	assert.Error(t, err)
}

func TestGenerateContractTests(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, _ := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	suite, err := svc.GenerateContractTests(context.Background(), config)
	require.NoError(t, err)
	assert.NotEmpty(t, suite.Name)
	assert.NotEmpty(t, suite.Tests)
}

func TestToSlug(t *testing.T) {
	assert.Equal(t, "order.created", toSlug("order.created"))
	assert.Equal(t, "my.event", toSlug("My Event"))
}

func TestGenerateTransformTemplates(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, _ := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{})
	transforms := svc.GenerateTransformTemplates(config)
	assert.Len(t, transforms, len(config.EventTypes))
	for _, tr := range transforms {
		assert.Contains(t, tr.Name, "flatten_")
		assert.Contains(t, tr.Target, ".flat")
	}
}

func TestGenerateFromSpec_IncludeTransforms(t *testing.T) {
	svc := NewService()
	spec, _ := svc.ParseSpec(testJSONSpec)
	config, err := svc.GenerateFromSpec(context.Background(), spec, GenerateOptions{IncludeTransforms: true})
	require.NoError(t, err)
	assert.NotEmpty(t, config.Transforms)
	for _, tr := range config.Transforms {
		assert.Contains(t, tr.Name, "flatten_")
		assert.Contains(t, tr.Target, ".flat")
	}
}
