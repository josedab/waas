package sdkgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSDK_TypeScript(t *testing.T) {
	t.Parallel()
	svc := NewService()

	sdk, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:   LangTypeScript,
		EventTypes: []string{"order.created"},
		Schemas: []SchemaDefinition{
			{
				EventType: "order.created",
				Properties: map[string]PropertyDef{
					"order_id": {Type: "string", Required: true},
					"amount":   {Type: "number", Required: true},
					"notes":    {Type: "string", Required: false},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, LangTypeScript, sdk.Language)
	assert.Contains(t, sdk.Files, "order_created.ts")
	assert.Contains(t, sdk.Files, "index.ts")
	assert.Contains(t, sdk.Files["order_created.ts"], "interface OrderCreated")
}

func TestGenerateSDK_Python(t *testing.T) {
	t.Parallel()
	svc := NewService()

	sdk, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:   LangPython,
		EventTypes: []string{"user.updated"},
		Schemas: []SchemaDefinition{
			{
				EventType: "user.updated",
				Properties: map[string]PropertyDef{
					"user_id": {Type: "string", Required: true},
					"active":  {Type: "boolean", Required: false},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Contains(t, sdk.Files, "user_updated.py")
	assert.Contains(t, sdk.Files["user_updated.py"], "class UserUpdated")
}

func TestGenerateSDK_Go(t *testing.T) {
	t.Parallel()
	svc := NewService()

	sdk, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:    LangGo,
		EventTypes:  []string{"payment.completed"},
		PackageName: "events",
		Schemas: []SchemaDefinition{
			{
				EventType: "payment.completed",
				Properties: map[string]PropertyDef{
					"payment_id": {Type: "string", Required: true},
					"amount":     {Type: "number", Required: true},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Contains(t, sdk.Files, "payment_completed.go")
	assert.Contains(t, sdk.Files["payment_completed.go"], "type PaymentCompleted struct")
}

func TestGenerateSDK_Java(t *testing.T) {
	t.Parallel()
	svc := NewService()

	sdk, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:   LangJava,
		EventTypes: []string{"invoice.sent"},
		Schemas: []SchemaDefinition{
			{
				EventType: "invoice.sent",
				Properties: map[string]PropertyDef{
					"invoice_id": {Type: "string", Required: true},
					"total":      {Type: "integer", Required: true},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Contains(t, sdk.Files, "InvoiceSent.java")
	assert.Contains(t, sdk.Files["InvoiceSent.java"], "public class InvoiceSent")
}

func TestGenerateSDK_InvalidLanguage(t *testing.T) {
	t.Parallel()
	svc := NewService()

	_, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:   "rust",
		EventTypes: []string{"test"},
		Schemas:    []SchemaDefinition{{EventType: "test"}},
	})

	assert.Error(t, err)
}

func TestGenerateSDK_EmptySchemas(t *testing.T) {
	t.Parallel()
	svc := NewService()

	_, err := svc.GenerateSDK("tenant-1", &GenerateSDKRequest{
		Language:   LangTypeScript,
		EventTypes: []string{"test"},
		Schemas:    []SchemaDefinition{},
	})

	assert.Error(t, err)
}
