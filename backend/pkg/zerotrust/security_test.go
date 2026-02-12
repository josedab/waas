package zerotrust

import "testing"

func TestNewCertificateManager(t *testing.T) {
	// Verify constructor doesn't panic
	cm := NewCertificateManager()
	if cm == nil {
		t.Fatal("NewCertificateManager returned nil")
	}
}

func TestNewRequestSigner(t *testing.T) {
	// Verify constructor doesn't panic
	rs := NewRequestSigner()
	if rs == nil {
		t.Fatal("NewRequestSigner returned nil")
	}
}

func TestNewAsymmetricSigner(t *testing.T) {
	// Verify constructor doesn't panic
	as := NewAsymmetricSigner()
	if as == nil {
		t.Fatal("NewAsymmetricSigner returned nil")
	}
}

func TestNewWebhookSecurityMiddleware(t *testing.T) {
	// Verify constructor doesn't panic
	wm := NewWebhookSecurityMiddleware()
	if wm == nil {
		t.Fatal("NewWebhookSecurityMiddleware returned nil")
	}
}
