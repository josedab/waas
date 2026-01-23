package portal

import (
	"context"
	"testing"
)

func TestOAuthRegisterClient(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	client, err := provider.RegisterClient(ctx, "tenant-1", "My App", []string{"https://example.com/callback"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.ClientID == "" {
		t.Error("expected non-empty client ID")
	}
	if client.ClientSecret == "" {
		t.Error("expected non-empty client secret")
	}
	if len(client.Scopes) == 0 {
		t.Error("expected default scopes")
	}
}

func TestOAuthInvalidRedirectURI(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	_, err := provider.RegisterClient(ctx, "tenant-1", "Test", []string{"http://evil.com/callback"}, nil)
	if err == nil {
		t.Error("expected error for non-HTTPS redirect URI")
	}
}

func TestOAuthLocalhostAllowed(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	_, err := provider.RegisterClient(ctx, "tenant-1", "Dev", []string{"http://localhost:3000/callback"}, nil)
	if err != nil {
		t.Errorf("localhost should be allowed: %v", err)
	}
}

func TestOAuthFullFlow(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	// Register client
	client, _ := provider.RegisterClient(ctx, "tenant-1", "Test App",
		[]string{"https://app.example.com/callback"},
		[]string{"read:endpoints"})

	// Authorize
	auth, err := provider.Authorize(ctx, client.ClientID, "https://app.example.com/callback", []string{"read:endpoints"})
	if err != nil {
		t.Fatalf("authorize error: %v", err)
	}
	if auth.Code == "" {
		t.Error("expected authorization code")
	}

	// Exchange code for token
	token, err := provider.ExchangeCode(ctx, auth.Code, client.ClientID, client.ClientSecret)
	if err != nil {
		t.Fatalf("exchange error: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected access token")
	}
	if token.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if token.TokenType != "Bearer" {
		t.Errorf("expected Bearer token type, got %s", token.TokenType)
	}

	// Validate token
	validated, err := provider.ValidateToken(ctx, token.AccessToken)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if validated.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", validated.TenantID)
	}

	// Refresh token
	newToken, err := provider.RefreshAccessToken(ctx, token.RefreshToken)
	if err != nil {
		t.Fatalf("refresh error: %v", err)
	}
	if newToken.AccessToken == token.AccessToken {
		t.Error("expected new access token")
	}

	// Old token should be invalid
	_, err = provider.ValidateToken(ctx, token.AccessToken)
	if err != ErrTokenExpired {
		t.Error("expected old token to be invalid")
	}
}

func TestOAuthInvalidClient(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	_, err := provider.Authorize(ctx, "nonexistent", "https://example.com/cb", nil)
	if err != ErrOAuthClientNotFound {
		t.Errorf("expected ErrOAuthClientNotFound, got %v", err)
	}
}

func TestCustomDomain(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	cd, err := provider.SetCustomDomain(ctx, "tenant-1", "portal-1", "webhooks.myapp.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cd.Domain != "webhooks.myapp.com" {
		t.Error("expected domain to be set")
	}
	if cd.VerifyTXT == "" {
		t.Error("expected verification TXT record")
	}
	if cd.Status != "pending" {
		t.Error("expected pending status")
	}

	// Verify
	verified, _ := provider.VerifyDomain(ctx, "webhooks.myapp.com")
	if !verified.Verified {
		t.Error("expected domain to be verified")
	}
	if verified.Status != "active" {
		t.Error("expected active status after verification")
	}
}

func TestCustomDomainInUse(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	_, _ = provider.SetCustomDomain(ctx, "tenant-1", "portal-1", "shared.example.com")
	_, err := provider.SetCustomDomain(ctx, "tenant-2", "portal-2", "shared.example.com")
	if err != ErrCustomDomainInUse {
		t.Errorf("expected ErrCustomDomainInUse, got %v", err)
	}
}

func TestThemeConfig(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	theme, err := provider.SetTheme(ctx, "tenant-1", "portal-1", &ThemeConfig{
		PrimaryColor: "#FF6B6B",
		DarkMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.PrimaryColor != "#FF6B6B" {
		t.Error("expected custom primary color")
	}
	if theme.BackgroundColor == "" {
		t.Error("expected default background color")
	}
	if !theme.DarkMode {
		t.Error("expected dark mode to be set")
	}

	// Retrieve
	got, _ := provider.GetTheme(ctx, "portal-1")
	if got.PrimaryColor != "#FF6B6B" {
		t.Error("expected to retrieve saved theme")
	}
}

func TestDefaultTheme(t *testing.T) {
	provider := NewOAuthProvider()
	ctx := context.Background()

	theme, err := provider.GetTheme(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.PrimaryColor == "" {
		t.Error("expected default theme with colors")
	}
}
