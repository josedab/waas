package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *SemanticVersion
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.2.3",
			want:  &SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:  "with label",
			input: "2.0.0-beta.1",
			want:  &SemanticVersion{Major: 2, Minor: 0, Patch: 0, Label: "beta.1"},
		},
		{
			name:    "invalid format",
			input:   "not-a-version",
			wantErr: true,
		},
		{
			name:    "incomplete version",
			input:   "1.2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSemanticVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.Major, got.Major)
			assert.Equal(t, tt.want.Minor, got.Minor)
			assert.Equal(t, tt.want.Patch, got.Patch)
			assert.Equal(t, tt.want.Label, got.Label)
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal", "1.0.0", "1.0.0", 0},
		{"major greater", "2.0.0", "1.0.0", 1},
		{"major less", "1.0.0", "2.0.0", -1},
		{"minor greater", "1.2.0", "1.1.0", 1},
		{"patch greater", "1.0.2", "1.0.1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := ParseSemanticVersion(tt.a)
			b, _ := ParseSemanticVersion(tt.b)
			assert.Equal(t, tt.want, CompareVersions(a, b))
		})
	}
}

func TestSchemaVersioning_GenerateTypeScriptTypes(t *testing.T) {
	events := []*EventType{
		{
			Slug: "user-created",
			Name: "User Created",
			Schema: &EventSchema{
				Properties: []SchemaProperty{
					{Name: "id", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true},
					{Name: "name", Type: "string", Required: false},
				},
			},
		},
	}

	result := generateTypeScriptTypes(events)
	assert.Contains(t, result, "export interface UserCreatedEvent")
	assert.Contains(t, result, "id: string;")
	assert.Contains(t, result, "email: string;")
	assert.Contains(t, result, "name?: string;")
}

func TestSchemaVersioning_GenerateGoTypes(t *testing.T) {
	events := []*EventType{
		{
			Slug: "order-placed",
			Name: "Order Placed",
			Schema: &EventSchema{
				Properties: []SchemaProperty{
					{Name: "order_id", Type: "string", Required: true},
					{Name: "amount", Type: "number", Required: true},
				},
			},
		},
	}

	result := generateGoTypes(events)
	assert.Contains(t, result, "type OrderPlacedEvent struct")
	assert.Contains(t, result, "OrderId string")
	assert.Contains(t, result, "Amount float64")
}

func TestSchemaVersioning_GeneratePythonTypes(t *testing.T) {
	events := []*EventType{
		{
			Slug: "payment-received",
			Name: "Payment Received",
			Schema: &EventSchema{
				Properties: []SchemaProperty{
					{Name: "payment_id", Type: "string", Required: true},
					{Name: "refunded", Type: "boolean", Required: false},
				},
			},
		},
	}

	result := generatePythonTypes(events)
	assert.Contains(t, result, "class PaymentReceivedEvent")
	assert.Contains(t, result, "payment_id: str")
	assert.Contains(t, result, "Optional[bool]")
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"user-created", "UserCreated"},
		{"order_placed", "OrderPlaced"},
		{"payment.received", "PaymentReceived"},
		{"simple", "Simple"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, toPascalCase(tt.input))
	}
}
