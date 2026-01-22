package client

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TenantService handles tenant-related API operations
type TenantService struct {
	client *Client
}

// Tenant represents a tenant account
type Tenant struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	SubscriptionTier   string    `json:"subscription_tier"`
	RateLimitPerMinute int       `json:"rate_limit_per_minute"`
	MonthlyQuota       int       `json:"monthly_quota"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateTenantRequest represents a tenant creation request
type CreateTenantRequest struct {
	Name               string `json:"name"`
	SubscriptionTier   string `json:"subscription_tier"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute,omitempty"`
	MonthlyQuota       int    `json:"monthly_quota,omitempty"`
}

// CreateTenantResponse represents the response for tenant creation
type CreateTenantResponse struct {
	Tenant *Tenant `json:"tenant"`
	APIKey string  `json:"api_key"`
}

// UpdateTenantRequest represents a tenant update request
type UpdateTenantRequest struct {
	Name               string `json:"name,omitempty"`
	SubscriptionTier   string `json:"subscription_tier,omitempty"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute,omitempty"`
	MonthlyQuota       int    `json:"monthly_quota,omitempty"`
}

// GetTenantResponse represents the response for getting tenant info
type GetTenantResponse struct {
	Tenant *Tenant `json:"tenant"`
}

// UpdateTenantResponse represents the response for updating tenant info
type UpdateTenantResponse struct {
	Tenant *Tenant `json:"tenant"`
}

// RegenerateAPIKeyResponse represents the response for API key regeneration
type RegenerateAPIKeyResponse struct {
	APIKey  string `json:"api_key"`
	Message string `json:"message"`
}

// CreateTenant creates a new tenant account (public endpoint, no auth required)
// Note: This method creates a new client instance since it doesn't require authentication
func CreateTenant(ctx context.Context, req *CreateTenantRequest, baseURL ...string) (*CreateTenantResponse, error) {
	// Create a temporary client without API key for this public endpoint
	url := DefaultBaseURL
	if len(baseURL) > 0 && baseURL[0] != "" {
		url = baseURL[0]
	}
	
	tempClient := &Client{
		config: &Config{BaseURL: url},
		http:   &http.Client{Timeout: DefaultTimeout},
	}
	
	var result CreateTenantResponse
	err := tempClient.post(ctx, "/tenants", req, &result)
	return &result, err
}

// GetTenant retrieves current tenant information
func (s *TenantService) GetTenant(ctx context.Context) (*Tenant, error) {
	var result GetTenantResponse
	err := s.client.get(ctx, "/tenant", &result)
	if err != nil {
		return nil, err
	}
	return result.Tenant, nil
}

// UpdateTenant updates tenant information
func (s *TenantService) UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*Tenant, error) {
	var result UpdateTenantResponse
	err := s.client.put(ctx, "/tenant", req, &result)
	if err != nil {
		return nil, err
	}
	return result.Tenant, nil
}

// RegenerateAPIKey generates a new API key for the tenant
func (s *TenantService) RegenerateAPIKey(ctx context.Context) (string, error) {
	var result RegenerateAPIKeyResponse
	err := s.client.post(ctx, "/tenant/regenerate-key", nil, &result)
	if err != nil {
		return "", err
	}
	return result.APIKey, nil
}