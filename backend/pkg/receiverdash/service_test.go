package receiverdash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	token, err := svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1", "ep-2"},
		Label:       "My Dashboard",
		Scopes:      []string{"read:deliveries", "read:health"},
		ExpiresIn:   "720h",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, token.ID)
	assert.Equal(t, "tenant-1", token.TenantID)
	assert.Contains(t, token.Token, "rcv_")
	assert.Len(t, token.EndpointIDs, 2)
	assert.Len(t, token.Scopes, 2)
}

func TestCreateTokenValidation(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Missing tenant
	_, err := svc.CreateToken("", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1"},
		Label:       "test",
		Scopes:      []string{"read:deliveries"},
	})
	assert.Error(t, err)

	// Missing endpoints
	_, err = svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{},
		Label:       "test",
		Scopes:      []string{"read:deliveries"},
	})
	assert.Error(t, err)

	// Invalid scope
	_, err = svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1"},
		Label:       "test",
		Scopes:      []string{"write:everything"},
	})
	assert.Error(t, err)
}

func TestValidateToken(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	created, err := svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1"},
		Label:       "test",
		Scopes:      []string{"read:deliveries"},
	})
	require.NoError(t, err)

	validated, err := svc.ValidateToken(created.Token)
	require.NoError(t, err)
	assert.Equal(t, created.ID, validated.ID)

	// Invalid token
	_, err = svc.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestRevokeToken(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	created, _ := svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1"},
		Label:       "test",
		Scopes:      []string{"read:deliveries"},
	})

	err := svc.RevokeToken(created.ID)
	require.NoError(t, err)

	_, err = svc.ValidateToken(created.Token)
	assert.Error(t, err)
}

func TestScopeEnforcement(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	token, _ := svc.CreateToken("tenant-1", &CreateTokenRequest{
		EndpointIDs: []string{"ep-1"},
		Label:       "limited",
		Scopes:      []string{"read:health"},
	})
	validated, _ := svc.ValidateToken(token.Token)

	// Should fail - no read:deliveries scope
	_, _, err := svc.GetDeliveryHistory(validated, &DeliveryHistoryRequest{})
	assert.Error(t, err)

	// Should succeed - has read:health scope
	_, err = svc.GetHealthSummary(validated, "24h")
	assert.NoError(t, err)
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()
	assert.Equal(t, 50, config.MaxTokensPerTenant)
	assert.Equal(t, 20, config.MaxEndpointsPerToken)
	assert.Equal(t, 30*24*time.Hour, config.DefaultTokenExpiry)
}
