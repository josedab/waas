package cloudmanaged

import (
"context"
"crypto/sha256"
"encoding/hex"
"fmt"
"time"

"github.com/google/uuid"
)

// APIKeyInfo represents a generated API key for a cloud tenant.
type APIKeyInfo struct {
ID         string     `json:"id" db:"id"`
TenantID   string     `json:"tenant_id" db:"tenant_id"`
KeyPrefix  string     `json:"key_prefix" db:"key_prefix"`
KeyHash    string     `json:"-" db:"key_hash"`
Name       string     `json:"name" db:"name"`
Scopes     []string   `json:"scopes"`
ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
IsActive   bool       `json:"is_active" db:"is_active"`
LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// APIKeyCreateRequest is the request to generate a new API key.
type APIKeyCreateRequest struct {
Name      string   `json:"name" binding:"required"`
Scopes    []string `json:"scopes,omitempty"`
ExpiresIn string   `json:"expires_in,omitempty"`
}

// APIKeyCreateResponse includes the full key (only returned once).
type APIKeyCreateResponse struct {
Key  string      `json:"key"`
Info *APIKeyInfo `json:"info"`
}

// SelfServiceProvisionResult represents the result of self-service provisioning.
type SelfServiceProvisionResult struct {
Tenant       *CloudTenant         `json:"tenant"`
APIKey       string               `json:"api_key"`
DashboardURL string               `json:"dashboard_url"`
Region       string               `json:"region"`
Endpoints    ProvisionedEndpoints `json:"endpoints"`
}

// ProvisionedEndpoints lists the API endpoints assigned to the tenant.
type ProvisionedEndpoints struct {
API     string `json:"api"`
Inbound string `json:"inbound"`
Portal  string `json:"portal"`
}

// PlanResourceQuota represents enforced resource limits based on plan.
type PlanResourceQuota struct {
TenantID         string `json:"tenant_id"`
WebhooksPerMonth int64  `json:"webhooks_per_month"`
EndpointsMax     int    `json:"endpoints_max"`
RetentionDays    int    `json:"retention_days"`
PayloadMaxSizeKB int    `json:"payload_max_size_kb"`
RatePerMinute    int    `json:"rate_per_minute"`
TeamMembersMax   int    `json:"team_members_max"`
}

// SelfServiceProvision performs self-service tenant provisioning.
func (s *Service) SelfServiceProvision(ctx context.Context, req *SignupRequest) (*SelfServiceProvisionResult, error) {
tenant, err := s.Signup(ctx, req)
if err != nil {
return nil, fmt.Errorf("signup failed: %w", err)
}

apiKey, err := generateAPIKey()
if err != nil {
return nil, fmt.Errorf("API key generation failed: %w", err)
}

region := tenant.Region
baseURL := fmt.Sprintf("https://%s.waas.cloud", region)

return &SelfServiceProvisionResult{
Tenant:       tenant,
APIKey:       apiKey,
DashboardURL: fmt.Sprintf("https://app.waas.cloud/tenants/%s", tenant.TenantID),
Region:       region,
Endpoints: ProvisionedEndpoints{
API:     fmt.Sprintf("%s/api/v1", baseURL),
Inbound: fmt.Sprintf("%s/inbound/%s", baseURL, tenant.TenantID),
Portal:  fmt.Sprintf("%s/portal/%s", baseURL, tenant.TenantID),
},
}, nil
}

// CreateAPIKey creates a new API key for an existing tenant.
func (s *Service) CreateAPIKey(ctx context.Context, tenantID string, req *APIKeyCreateRequest) (*APIKeyCreateResponse, error) {
_, err := s.repo.GetCloudTenant(ctx, tenantID)
if err != nil {
return nil, fmt.Errorf("tenant not found: %w", err)
}

key, err := generateAPIKey()
if err != nil {
return nil, fmt.Errorf("key generation failed: %w", err)
}

hash := sha256.Sum256([]byte(key))
keyHash := hex.EncodeToString(hash[:])

scopes := req.Scopes
if len(scopes) == 0 {
scopes = []string{"read", "write"}
}

var expiresAt *time.Time
if req.ExpiresIn != "" && req.ExpiresIn != "never" {
var t time.Time
switch req.ExpiresIn {
case "30d":
t = time.Now().AddDate(0, 0, 30)
case "90d":
t = time.Now().AddDate(0, 0, 90)
case "365d":
t = time.Now().AddDate(1, 0, 0)
}
if !t.IsZero() {
expiresAt = &t
}
}

return &APIKeyCreateResponse{
Key: key,
Info: &APIKeyInfo{
ID: uuid.New().String(), TenantID: tenantID, KeyPrefix: key[:12],
KeyHash: keyHash, Name: req.Name, Scopes: scopes,
ExpiresAt: expiresAt, IsActive: true, CreatedAt: time.Now(),
},
}, nil
}

// GetPlanResourceQuota returns enforced resource limits based on plan.
func (s *Service) GetPlanResourceQuota(ctx context.Context, tenantID string) (*PlanResourceQuota, error) {
tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
if err != nil {
return nil, err
}

q := &PlanResourceQuota{TenantID: tenantID}
switch tenant.Plan {
case PlanTierFree:
q.WebhooksPerMonth = 1000; q.EndpointsMax = 3; q.RetentionDays = 7
q.PayloadMaxSizeKB = 256; q.RatePerMinute = 60; q.TeamMembersMax = 1
case PlanTierStarter:
q.WebhooksPerMonth = 50000; q.EndpointsMax = 25; q.RetentionDays = 30
q.PayloadMaxSizeKB = 1024; q.RatePerMinute = 300; q.TeamMembersMax = 5
case PlanTierPro:
q.WebhooksPerMonth = 500000; q.EndpointsMax = 100; q.RetentionDays = 90
q.PayloadMaxSizeKB = 5120; q.RatePerMinute = 1000; q.TeamMembersMax = 25
case PlanTierEnterprise:
q.WebhooksPerMonth = 0; q.EndpointsMax = 0; q.RetentionDays = 365
q.PayloadMaxSizeKB = 10240; q.RatePerMinute = 10000; q.TeamMembersMax = 0
}
return q, nil
}

// EnforceQuota checks and enforces resource quotas.
func (s *Service) EnforceQuota(ctx context.Context, tenantID, metricType string, amount int64) error {
tenant, err := s.repo.GetCloudTenant(ctx, tenantID)
if err != nil {
return err
}
switch metricType {
case "webhooks":
if tenant.WebhooksLimit > 0 && tenant.WebhooksUsed+amount > tenant.WebhooksLimit {
return fmt.Errorf("webhook quota exceeded: %d/%d", tenant.WebhooksUsed+amount, tenant.WebhooksLimit)
}
case "storage":
if tenant.StorageLimit > 0 && tenant.StorageUsed+amount > tenant.StorageLimit {
return fmt.Errorf("storage quota exceeded: %d/%d", tenant.StorageUsed+amount, tenant.StorageLimit)
}
}
return nil
}
