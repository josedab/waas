package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWellKnownEventTypes(t *testing.T) {
	types := WellKnownEventTypes()
	assert.NotEmpty(t, types)
	assert.Contains(t, types, EventOrderCreated)
	assert.Contains(t, types, EventPing)
}

func TestIsWellKnownEventType(t *testing.T) {
	assert.True(t, IsWellKnownEventType("order.created"))
	assert.True(t, IsWellKnownEventType("ping"))
	assert.False(t, IsWellKnownEventType("custom.event"))
	assert.False(t, IsWellKnownEventType(""))
}

func TestValidateEventType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid dotted", "order.created", false},
		{"valid custom", "my-app.user.signup", false},
		{"valid single word", "ping", false},
		{"empty", "", true},
		{"whitespace", "order created", true},
		{"tab", "order\tcreated", true},
		{"newline", "order\ncreated", true},
		{"too long", strings.Repeat("a", 256), true},
		{"max length", strings.Repeat("a", 255), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
