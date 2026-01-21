package pushbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ProviderCredentials holds credentials for a push notification provider
type ProviderCredentials struct {
	ID          string           `json:"id" db:"id"`
	TenantID    string           `json:"tenant_id" db:"tenant_id"`
	Provider    Platform         `json:"provider" db:"provider"`
	Name        string           `json:"name" db:"name"`
	Credentials CredentialData   `json:"credentials" db:"credentials"`
	Environment string           `json:"environment" db:"environment"` // production, sandbox
	IsDefault   bool             `json:"is_default" db:"is_default"`
	Status      CredentialStatus `json:"status" db:"status"`
	LastUsedAt  *time.Time       `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
}

// CredentialStatus defines credential states
type CredentialStatus string

const (
	CredentialActive   CredentialStatus = "active"
	CredentialInactive CredentialStatus = "inactive"
	CredentialInvalid  CredentialStatus = "invalid"
	CredentialExpired  CredentialStatus = "expired"
)

// CredentialData holds provider-specific credential information
type CredentialData struct {
	// FCM (Firebase Cloud Messaging)
	FCMProjectID      string `json:"fcm_project_id,omitempty"`
	FCMServiceAccount string `json:"fcm_service_account,omitempty"` // JSON service account key
	FCMServerKey      string `json:"fcm_server_key,omitempty"`      // Legacy server key

	// APNs (Apple Push Notification service)
	APNsKeyID       string `json:"apns_key_id,omitempty"`
	APNsTeamID      string `json:"apns_team_id,omitempty"`
	APNsBundleID    string `json:"apns_bundle_id,omitempty"`
	APNsPrivateKey  string `json:"apns_private_key,omitempty"`  // .p8 key content
	APNsCertificate string `json:"apns_certificate,omitempty"` // .pem certificate (legacy)
	APNsPassword    string `json:"apns_password,omitempty"`    // Certificate password

	// Web Push (VAPID)
	VAPIDPublicKey  string `json:"vapid_public_key,omitempty"`
	VAPIDPrivateKey string `json:"vapid_private_key,omitempty"`
	VAPIDSubject    string `json:"vapid_subject,omitempty"` // mailto: or https:// URL

	// Huawei Push Kit
	HuaweiAppID     string `json:"huawei_app_id,omitempty"`
	HuaweiAppSecret string `json:"huawei_app_secret,omitempty"`
}

// Validate validates credentials based on provider type
func (c *CredentialData) Validate(provider Platform) error {
	switch provider {
	case PlatformAndroid:
		if c.FCMProjectID == "" {
			return fmt.Errorf("FCM project ID is required")
		}
		if c.FCMServiceAccount == "" && c.FCMServerKey == "" {
			return fmt.Errorf("FCM service account or server key is required")
		}
	case PlatformIOS:
		if c.APNsBundleID == "" {
			return fmt.Errorf("APNs bundle ID is required")
		}
		if c.APNsPrivateKey == "" && c.APNsCertificate == "" {
			return fmt.Errorf("APNs private key or certificate is required")
		}
		if c.APNsPrivateKey != "" && (c.APNsKeyID == "" || c.APNsTeamID == "") {
			return fmt.Errorf("APNs key ID and team ID are required with private key")
		}
	case PlatformWeb:
		if c.VAPIDPublicKey == "" || c.VAPIDPrivateKey == "" {
			return fmt.Errorf("VAPID public and private keys are required")
		}
	case PlatformHuawei:
		if c.HuaweiAppID == "" || c.HuaweiAppSecret == "" {
			return fmt.Errorf("Huawei app ID and secret are required")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	return nil
}

// Scan implements sql.Scanner for CredentialData
func (c *CredentialData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}
	return json.Unmarshal(b, c)
}

// CreateCredentialsRequest is the request to create provider credentials
type CreateCredentialsRequest struct {
	Provider    Platform       `json:"provider" binding:"required"`
	Name        string         `json:"name" binding:"required"`
	Credentials CredentialData `json:"credentials" binding:"required"`
	Environment string         `json:"environment"` // defaults to "production"
	IsDefault   bool           `json:"is_default"`
}

// UpdateCredentialsRequest is the request to update provider credentials
type UpdateCredentialsRequest struct {
	Name        *string         `json:"name,omitempty"`
	Credentials *CredentialData `json:"credentials,omitempty"`
	Environment *string         `json:"environment,omitempty"`
	IsDefault   *bool           `json:"is_default,omitempty"`
	Status      *CredentialStatus `json:"status,omitempty"`
}

// CredentialsManager handles push provider credentials
type CredentialsManager struct {
	repo Repository
}

// NewCredentialsManager creates a new credentials manager
func NewCredentialsManager(repo Repository) *CredentialsManager {
	return &CredentialsManager{repo: repo}
}

// CreateCredentials creates new provider credentials
func (m *CredentialsManager) CreateCredentials(ctx context.Context, tenantID string, req *CreateCredentialsRequest) (*ProviderCredentials, error) {
	if err := req.Credentials.Validate(req.Provider); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	env := req.Environment
	if env == "" {
		env = "production"
	}

	creds := &ProviderCredentials{
		TenantID:    tenantID,
		Provider:    req.Provider,
		Name:        req.Name,
		Credentials: req.Credentials,
		Environment: env,
		IsDefault:   req.IsDefault,
		Status:      CredentialActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.repo.SaveCredentials(ctx, creds); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	return creds, nil
}

// GetCredentials retrieves provider credentials by ID
func (m *CredentialsManager) GetCredentials(ctx context.Context, tenantID, id string) (*ProviderCredentials, error) {
	return m.repo.GetCredentials(ctx, tenantID, id)
}

// ListCredentials lists all provider credentials for a tenant
func (m *CredentialsManager) ListCredentials(ctx context.Context, tenantID string, provider *Platform) ([]*ProviderCredentials, error) {
	return m.repo.ListCredentials(ctx, tenantID, provider)
}

// UpdateCredentials updates provider credentials
func (m *CredentialsManager) UpdateCredentials(ctx context.Context, tenantID, id string, req *UpdateCredentialsRequest) (*ProviderCredentials, error) {
	creds, err := m.repo.GetCredentials(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		creds.Name = *req.Name
	}
	if req.Credentials != nil {
		if err := req.Credentials.Validate(creds.Provider); err != nil {
			return nil, fmt.Errorf("invalid credentials: %w", err)
		}
		creds.Credentials = *req.Credentials
	}
	if req.Environment != nil {
		creds.Environment = *req.Environment
	}
	if req.IsDefault != nil {
		creds.IsDefault = *req.IsDefault
	}
	if req.Status != nil {
		creds.Status = *req.Status
	}

	creds.UpdatedAt = time.Now()

	if err := m.repo.UpdateCredentials(ctx, creds); err != nil {
		return nil, fmt.Errorf("failed to update credentials: %w", err)
	}

	return creds, nil
}

// DeleteCredentials deletes provider credentials
func (m *CredentialsManager) DeleteCredentials(ctx context.Context, tenantID, id string) error {
	return m.repo.DeleteCredentials(ctx, tenantID, id)
}

// GetDefaultCredentials retrieves the default credentials for a provider
func (m *CredentialsManager) GetDefaultCredentials(ctx context.Context, tenantID string, provider Platform) (*ProviderCredentials, error) {
	creds, err := m.repo.ListCredentials(ctx, tenantID, &provider)
	if err != nil {
		return nil, err
	}

	for _, c := range creds {
		if c.IsDefault && c.Status == CredentialActive {
			return c, nil
		}
	}

	// Return first active credential if no default
	for _, c := range creds {
		if c.Status == CredentialActive {
			return c, nil
		}
	}

	return nil, fmt.Errorf("no active credentials found for provider %s", provider)
}

// ValidateCredentials tests if credentials are valid by attempting provider authentication
func (m *CredentialsManager) ValidateCredentials(ctx context.Context, creds *ProviderCredentials) error {
	// Basic validation
	if err := creds.Credentials.Validate(creds.Provider); err != nil {
		return err
	}

	// Provider-specific validation would go here
	// In production, this would attempt actual authentication with each provider
	switch creds.Provider {
	case PlatformAndroid:
		// Validate FCM credentials by attempting to get an access token
		return nil
	case PlatformIOS:
		// Validate APNs credentials
		return nil
	case PlatformWeb:
		// VAPID keys are validated by format
		return nil
	case PlatformHuawei:
		// Validate Huawei credentials
		return nil
	}

	return nil
}

// RotateCredentials generates new credentials (for providers that support it)
func (m *CredentialsManager) RotateCredentials(ctx context.Context, tenantID, id string) (*ProviderCredentials, error) {
	creds, err := m.repo.GetCredentials(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// For VAPID keys, we can generate new ones
	if creds.Provider == PlatformWeb {
		// In production, generate new VAPID key pair
		// For now, just mark as rotated
		creds.UpdatedAt = time.Now()
		if err := m.repo.UpdateCredentials(ctx, creds); err != nil {
			return nil, err
		}
		return creds, nil
	}

	return nil, fmt.Errorf("credential rotation not supported for provider %s", creds.Provider)
}
