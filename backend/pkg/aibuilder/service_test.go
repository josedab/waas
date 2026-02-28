package aibuilder

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSendMessage_NewConversation(t *testing.T) {
	svc := NewService(nil, nil)
	req := &SendMessageRequest{Message: "Create a new webhook endpoint"}
	resp, err := svc.SendMessage(context.Background(), "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConversationID == "" {
		t.Error("expected conversation ID")
	}
	if resp.Reply == nil {
		t.Error("expected reply message")
	}
	if resp.Reply.Role != RoleAssistant {
		t.Errorf("expected assistant role, got %s", resp.Reply.Role)
	}
}

func TestSendMessage_ContinueConversation(t *testing.T) {
	svc := NewService(nil, nil)
	resp1, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{Message: "Create endpoint"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp2, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{
		Message:        "Use https://example.com/hooks",
		ConversationID: resp1.ConversationID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.ConversationID != resp1.ConversationID {
		t.Error("expected same conversation ID")
	}
}

func TestSendMessage_EmptyMessage(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{Message: ""})
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestSendMessage_MessageTooLong(t *testing.T) {
	svc := NewService(nil, &ServiceConfig{MaxMessageLength: 10, ConversationTTL: DefaultServiceConfig().ConversationTTL, MaxConversationsPerTenant: 50, MaxMessagesPerConversation: 200})
	_, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{Message: "this message is way too long"})
	if err == nil {
		t.Error("expected error for long message")
	}
}

func TestClassifyIntent(t *testing.T) {
	svc := NewService(nil, nil)
	tests := []struct {
		msg    string
		expect IntentType
	}{
		{"Create a new endpoint", IntentCreateEndpoint},
		{"configure retry policy", IntentConfigureRetry},
		{"setup authentication", IntentSetupAuth},
		{"my webhook is failing", IntentDebugDelivery},
		{"list my endpoints", IntentListEndpoints},
		{"explain this error", IntentExplainError},
		{"transform the payload", IntentTransformSetup},
		{"hello there", IntentGeneral},
	}
	for _, tt := range tests {
		got := svc.classifyIntent(tt.msg)
		if got != tt.expect {
			t.Errorf("classifyIntent(%q) = %s, want %s", tt.msg, got, tt.expect)
		}
	}
}

func TestDebugDelivery(t *testing.T) {
	svc := NewService(nil, nil)
	resp, err := svc.DebugDelivery(context.Background(), "tenant-1", &DebugRequest{DeliveryID: "del-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DeliveryID != "del-123" {
		t.Errorf("expected del-123, got %s", resp.DeliveryID)
	}
	if len(resp.Suggestions) == 0 {
		t.Error("expected suggestions")
	}
}

func TestDebugDelivery_EmptyID(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.DebugDelivery(context.Background(), "tenant-1", &DebugRequest{})
	if err == nil {
		t.Error("expected error for empty delivery ID")
	}
}

func TestListConversations(t *testing.T) {
	svc := NewService(nil, nil)
	// Create a conversation first
	_, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{Message: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	convs, err := svc.ListConversations(context.Background(), "tenant-1", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(convs) == 0 {
		t.Error("expected at least one conversation")
	}
}

func TestDeleteConversation(t *testing.T) {
	svc := NewService(nil, nil)
	resp, err := svc.SendMessage(context.Background(), "tenant-1", &SendMessageRequest{Message: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = svc.DeleteConversation(context.Background(), "tenant-1", resp.ConversationID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetConversation(context.Background(), "tenant-1", resp.ConversationID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}
