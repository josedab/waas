package portalsdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateConfig(t *testing.T) {
	svc := NewService(nil)

	config, err := svc.CreateConfig(context.Background(), "tenant-1", &CreatePortalConfigRequest{
		Name:           "My Portal",
		AllowedOrigins: []string{"https://app.example.com"},
	})
	require.NoError(t, err)
	assert.Equal(t, "My Portal", config.Name)
	assert.True(t, config.IsActive)
	assert.NotEmpty(t, config.Components)
	assert.NotEmpty(t, config.Theme.PrimaryColor)
	assert.True(t, config.Features.EndpointManagement)
}

func TestCreateConfigValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.CreateConfig(context.Background(), "t1", &CreatePortalConfigRequest{})
	assert.Error(t, err)

	_, err = svc.CreateConfig(context.Background(), "t1", &CreatePortalConfigRequest{Name: "test"})
	assert.Error(t, err)
}

func TestCreateConfigInvalidComponent(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.CreateConfig(context.Background(), "t1", &CreatePortalConfigRequest{
		Name:           "test",
		AllowedOrigins: []string{"https://example.com"},
		Components:     []string{"invalid_component"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid component")
}

func TestCreateConfigValidComponents(t *testing.T) {
	svc := NewService(nil)

	config, err := svc.CreateConfig(context.Background(), "t1", &CreatePortalConfigRequest{
		Name:           "test",
		AllowedOrigins: []string{"https://example.com"},
		Components:     []string{ComponentEndpointManager, ComponentDeliveryLog},
	})
	assert.NoError(t, err)
	assert.Len(t, config.Components, 2)
}

func TestCreateSession(t *testing.T) {
	svc := NewService(nil)

	session, err := svc.CreateSession(context.Background(), "tenant-1", &CreateSessionRequest{
		ConfigID:   "cfg-1",
		CustomerID: "cust-1",
		ExpiresIn:  "2h",
	})
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", session.TenantID)
	assert.Equal(t, "cust-1", session.CustomerID)
	assert.Equal(t, SessionStatusActive, session.Status)
	assert.Contains(t, session.Token, "psk_")
	assert.NotEmpty(t, session.Permissions)
}

func TestCreateSessionValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.CreateSession(context.Background(), "t1", &CreateSessionRequest{})
	assert.Error(t, err)
}

func TestGenerateSDKSnippet(t *testing.T) {
	svc := NewService(nil)

	frameworks := []string{"react", "vue", "vanilla"}
	for _, fw := range frameworks {
		t.Run(fw, func(t *testing.T) {
			snippet, err := svc.GenerateSDKSnippet(context.Background(), "tenant-1", &GenerateSDKRequest{
				ConfigID:  "cfg-1",
				Framework: fw,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, snippet)
		})
	}

	_, err := svc.GenerateSDKSnippet(context.Background(), "t1", &GenerateSDKRequest{
		ConfigID: "cfg-1", Framework: "angular",
	})
	assert.Error(t, err)
}
