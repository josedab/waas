package portal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrOAuthClientNotFound  = errors.New("oauth client not found")
	ErrInvalidRedirectURI   = errors.New("invalid redirect URI")
	ErrInvalidGrantType     = errors.New("invalid grant type")
	ErrAuthorizationExpired = errors.New("authorization code expired")
	ErrTokenExpired         = errors.New("access token expired")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrCustomDomainInUse    = errors.New("custom domain already in use")
)

// OAuthClient represents a registered OAuth 2.0 client
type OAuthClient struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"` // Only returned on creation
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"` // authorization_code, client_credentials
	CreatedAt    time.Time `json:"created_at"`
}

// AuthorizationCode represents an OAuth authorization code
type AuthorizationCode struct {
	Code        string    `json:"code"`
	ClientID    string    `json:"client_id"`
	TenantID    string    `json:"tenant_id"`
	Scopes      []string  `json:"scopes"`
	RedirectURI string    `json:"redirect_uri"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// OAuthToken represents an OAuth access/refresh token pair
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scopes       []string  `json:"scopes"`
	TenantID     string    `json:"tenant_id"`
	ClientID     string    `json:"client_id"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// CustomDomain represents a custom domain for the portal
type CustomDomain struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	PortalID     string    `json:"portal_id"`
	Domain       string    `json:"domain"`
	Status       string    `json:"status"` // pending, active, error
	SSLStatus    string    `json:"ssl_status"` // pending, active, error
	VerifyTXT    string    `json:"verify_txt"`
	Verified     bool      `json:"verified"`
	CreatedAt    time.Time `json:"created_at"`
}

// ThemeConfig represents the portal's visual theme
type ThemeConfig struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	PortalID         string            `json:"portal_id"`
	PrimaryColor     string            `json:"primary_color"`
	SecondaryColor   string            `json:"secondary_color"`
	BackgroundColor  string            `json:"background_color"`
	TextColor        string            `json:"text_color"`
	FontFamily       string            `json:"font_family"`
	BorderRadius     string            `json:"border_radius"`
	LogoURL          string            `json:"logo_url,omitempty"`
	FaviconURL       string            `json:"favicon_url,omitempty"`
	CustomCSS        string            `json:"custom_css,omitempty"`
	DarkMode         bool              `json:"dark_mode"`
	CustomProperties map[string]string `json:"custom_properties,omitempty"`
}

// OAuthProvider manages OAuth 2.0 for embedded portals
type OAuthProvider struct {
	mu             sync.RWMutex
	clients        map[string]*OAuthClient        // clientID -> client
	authCodes      map[string]*AuthorizationCode   // code -> auth
	tokens         map[string]*OAuthToken          // accessToken -> token
	refreshTokens  map[string]*OAuthToken          // refreshToken -> token
	customDomains  map[string]*CustomDomain         // domain -> config
	themes         map[string]*ThemeConfig          // portalID -> theme
}

// NewOAuthProvider creates a new OAuth provider
func NewOAuthProvider() *OAuthProvider {
	return &OAuthProvider{
		clients:       make(map[string]*OAuthClient),
		authCodes:     make(map[string]*AuthorizationCode),
		tokens:        make(map[string]*OAuthToken),
		refreshTokens: make(map[string]*OAuthToken),
		customDomains: make(map[string]*CustomDomain),
		themes:        make(map[string]*ThemeConfig),
	}
}

// RegisterClient registers a new OAuth client
func (p *OAuthProvider) RegisterClient(_ context.Context, tenantID, name string, redirectURIs, scopes []string) (*OAuthClient, error) {
	if name == "" {
		return nil, errors.New("client name is required")
	}
	if len(redirectURIs) == 0 {
		return nil, ErrInvalidRedirectURI
	}

	for _, uri := range redirectURIs {
		if !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "http://localhost") {
			return nil, fmt.Errorf("%w: must use HTTPS (got %s)", ErrInvalidRedirectURI, uri)
		}
	}

	if len(scopes) == 0 {
		scopes = []string{"read:endpoints", "read:deliveries", "write:endpoints"}
	}

	client := &OAuthClient{
		ID:           generateToken(8),
		TenantID:     tenantID,
		Name:         name,
		ClientID:     "waas_" + generateToken(16),
		ClientSecret: generateToken(32),
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   []string{"authorization_code", "client_credentials"},
		CreatedAt:    time.Now(),
	}

	p.mu.Lock()
	p.clients[client.ClientID] = client
	p.mu.Unlock()
	return client, nil
}

// Authorize creates an authorization code
func (p *OAuthProvider) Authorize(_ context.Context, clientID, redirectURI string, scopes []string) (*AuthorizationCode, error) {
	p.mu.RLock()
	client, ok := p.clients[clientID]
	p.mu.RUnlock()
	if !ok {
		return nil, ErrOAuthClientNotFound
	}

	validURI := false
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			validURI = true
			break
		}
	}
	if !validURI {
		return nil, ErrInvalidRedirectURI
	}

	code := &AuthorizationCode{
		Code:        generateToken(32),
		ClientID:    clientID,
		TenantID:    client.TenantID,
		Scopes:      scopes,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	p.mu.Lock()
	p.authCodes[code.Code] = code
	p.mu.Unlock()
	return code, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (p *OAuthProvider) ExchangeCode(_ context.Context, code, clientID, clientSecret string) (*OAuthToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	auth, ok := p.authCodes[code]
	if !ok {
		return nil, ErrAuthorizationExpired
	}

	if time.Now().After(auth.ExpiresAt) {
		delete(p.authCodes, code)
		return nil, ErrAuthorizationExpired
	}

	client, ok := p.clients[clientID]
	if !ok || client.ClientSecret != clientSecret {
		return nil, ErrOAuthClientNotFound
	}

	// Create token pair
	token := &OAuthToken{
		AccessToken:  generateToken(32),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: generateToken(32),
		Scopes:       auth.Scopes,
		TenantID:     auth.TenantID,
		ClientID:     clientID,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	p.tokens[token.AccessToken] = token
	p.refreshTokens[token.RefreshToken] = token
	delete(p.authCodes, code)

	return token, nil
}

// ValidateToken validates an access token
func (p *OAuthProvider) ValidateToken(_ context.Context, accessToken string) (*OAuthToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, ok := p.tokens[accessToken]
	if !ok {
		return nil, ErrTokenExpired
	}
	if time.Now().After(token.ExpiresAt) {
		delete(p.tokens, accessToken)
		return nil, ErrTokenExpired
	}
	return token, nil
}

// RefreshAccessToken refreshes an access token
func (p *OAuthProvider) RefreshAccessToken(_ context.Context, refreshToken string) (*OAuthToken, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	oldToken, ok := p.refreshTokens[refreshToken]
	if !ok {
		return nil, ErrInvalidRefreshToken
	}

	// Create new token pair
	newToken := &OAuthToken{
		AccessToken:  generateToken(32),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: generateToken(32),
		Scopes:       oldToken.Scopes,
		TenantID:     oldToken.TenantID,
		ClientID:     oldToken.ClientID,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	// Revoke old tokens
	delete(p.tokens, oldToken.AccessToken)
	delete(p.refreshTokens, refreshToken)

	// Store new tokens
	p.tokens[newToken.AccessToken] = newToken
	p.refreshTokens[newToken.RefreshToken] = newToken

	return newToken, nil
}

// SetCustomDomain configures a custom domain for a portal
func (p *OAuthProvider) SetCustomDomain(_ context.Context, tenantID, portalID, domain string) (*CustomDomain, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if existing, ok := p.customDomains[domain]; ok && existing.TenantID != tenantID {
		return nil, ErrCustomDomainInUse
	}

	cd := &CustomDomain{
		ID:        generateToken(8),
		TenantID:  tenantID,
		PortalID:  portalID,
		Domain:    domain,
		Status:    "pending",
		SSLStatus: "pending",
		VerifyTXT: "waas-verify=" + generateToken(16),
		CreatedAt: time.Now(),
	}

	p.customDomains[domain] = cd
	return cd, nil
}

// VerifyDomain verifies domain ownership
func (p *OAuthProvider) VerifyDomain(_ context.Context, domain string) (*CustomDomain, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cd, ok := p.customDomains[domain]
	if !ok {
		return nil, errors.New("domain not found")
	}
	cd.Verified = true
	cd.Status = "active"
	cd.SSLStatus = "active"
	return cd, nil
}

// SetTheme configures the visual theme for a portal
func (p *OAuthProvider) SetTheme(_ context.Context, tenantID, portalID string, theme *ThemeConfig) (*ThemeConfig, error) {
	if theme.PrimaryColor == "" {
		theme.PrimaryColor = "#6366F1" // Default indigo
	}
	if theme.BackgroundColor == "" {
		theme.BackgroundColor = "#FFFFFF"
	}
	if theme.TextColor == "" {
		theme.TextColor = "#1F2937"
	}
	if theme.FontFamily == "" {
		theme.FontFamily = "Inter, system-ui, sans-serif"
	}
	if theme.BorderRadius == "" {
		theme.BorderRadius = "8px"
	}

	theme.ID = generateToken(8)
	theme.TenantID = tenantID
	theme.PortalID = portalID

	p.mu.Lock()
	p.themes[portalID] = theme
	p.mu.Unlock()
	return theme, nil
}

// GetTheme retrieves the theme for a portal
func (p *OAuthProvider) GetTheme(_ context.Context, portalID string) (*ThemeConfig, error) {
	p.mu.RLock()
	theme, ok := p.themes[portalID]
	p.mu.RUnlock()
	if !ok {
		// Return default theme
		return &ThemeConfig{
			PrimaryColor:    "#6366F1",
			SecondaryColor:  "#8B5CF6",
			BackgroundColor: "#FFFFFF",
			TextColor:       "#1F2937",
			FontFamily:      "Inter, system-ui, sans-serif",
			BorderRadius:    "8px",
		}, nil
	}
	return theme, nil
}

func generateToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
