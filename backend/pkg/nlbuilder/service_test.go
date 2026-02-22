package nlbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartConversation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil, nil)

	conv, err := svc.StartConversation("tenant-1")
	require.NoError(t, err)
	assert.NotEmpty(t, conv.ID)
	assert.Equal(t, "active", conv.Status)
	assert.Len(t, conv.Messages, 2) // system + greeting
}

func TestChat(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil, nil)

	// Start a conversation first
	conv, _ := svc.StartConversation("tenant-1")

	resp, err := svc.Chat("tenant-1", &ChatRequest{
		ConversationID: conv.ID,
		Message:        "Send order.created events to https://api.example.com/webhooks",
	})

	require.NoError(t, err)
	assert.Equal(t, conv.ID, resp.ConversationID)
	assert.NotEmpty(t, resp.Reply)
	assert.NotNil(t, resp.Intent)
	assert.Equal(t, "create_endpoint", resp.Intent.Action)
	assert.Equal(t, "https://api.example.com/webhooks", resp.Intent.TargetURL)
	assert.Contains(t, resp.Intent.EventTypes, "order.created")
}

func TestChatWithRetry(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil, nil)

	conv, _ := svc.StartConversation("tenant-1")
	resp, err := svc.Chat("tenant-1", &ChatRequest{
		ConversationID: conv.ID,
		Message:        "Configure exponential retry with 10 retries",
	})

	require.NoError(t, err)
	assert.Equal(t, "configure_retry", resp.Intent.Action)
	assert.NotNil(t, resp.Intent.RetryPolicy)
	assert.Equal(t, 10, resp.Intent.RetryPolicy.MaxRetries)
	assert.Equal(t, "exponential", resp.Intent.RetryPolicy.Strategy)
}

func TestBuiltinIntentParser(t *testing.T) {
	parser := &BuiltinIntentParser{}

	t.Run("URL extraction", func(t *testing.T) {
		intent, err := parser.ParseIntent("Send to https://example.com/hook")
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/hook", intent.TargetURL)
	})

	t.Run("event type extraction", func(t *testing.T) {
		intent, err := parser.ParseIntent("Listen for order.created and payment.completed events")
		require.NoError(t, err)
		assert.Contains(t, intent.EventTypes, "order.created")
		assert.Contains(t, intent.EventTypes, "payment.completed")
	})

	t.Run("retry detection", func(t *testing.T) {
		intent, err := parser.ParseIntent("Use linear retry with 3 attempts")
		require.NoError(t, err)
		assert.Equal(t, "configure_retry", intent.Action)
		assert.Equal(t, "linear", intent.RetryPolicy.Strategy)
		assert.Equal(t, 3, intent.RetryPolicy.MaxRetries)
	})

	t.Run("filter detection", func(t *testing.T) {
		intent, err := parser.ParseIntent("Filter events only when status is active")
		require.NoError(t, err)
		assert.Equal(t, "set_filter", intent.Action)
	})
}

func TestApplyConfig(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil, nil)

	conv, _ := svc.StartConversation("tenant-1")

	// Build config via chat
	svc.Chat("tenant-1", &ChatRequest{
		ConversationID: conv.ID,
		Message:        "Create endpoint at https://api.example.com/hooks for order.created events",
	})

	config, err := svc.ApplyConfig(conv.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/hooks", config.URL)
	assert.True(t, config.Validated)
}
