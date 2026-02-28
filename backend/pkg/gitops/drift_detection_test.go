package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffConfigs(t *testing.T) {
	oldConfig := `apiVersion: waas/v1
kind: WebhookConfig
metadata:
  name: test-config
spec:
  endpoints:
    - name: billing
      url: https://billing.example.com/webhooks
    - name: notifications
      url: https://notify.example.com/webhooks
`
	newConfig := `apiVersion: waas/v1
kind: WebhookConfig
metadata:
  name: test-config
spec:
  endpoints:
    - name: billing
      url: https://billing-v2.example.com/webhooks
    - name: analytics
      url: https://analytics.example.com/webhooks
`

	diffs, err := DiffConfigs(oldConfig, newConfig)
	assert.NoError(t, err)
	assert.Len(t, diffs, 3, "should detect 3 changes")

	// Find specific diffs
	var added, modified, removed int
	for _, d := range diffs {
		switch d.DriftType {
		case "added":
			added++
			assert.Equal(t, "analytics", d.ResourceName)
		case "modified":
			modified++
			assert.Equal(t, "billing", d.ResourceName)
		case "removed":
			removed++
			assert.Equal(t, "notifications", d.ResourceName)
		}
	}
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, modified)
	assert.Equal(t, 1, removed)
}

func TestDiffConfigs_NoChanges(t *testing.T) {
	config := `apiVersion: waas/v1
kind: WebhookConfig
metadata:
  name: test-config
spec:
  endpoints:
    - name: billing
      url: https://billing.example.com/webhooks
`
	diffs, err := DiffConfigs(config, config)
	assert.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestGenerateCICDTemplate(t *testing.T) {
	tests := []struct {
		platform string
		wantErr  bool
		contains string
	}{
		{"github_actions", false, "waas gitops apply"},
		{"gitlab_ci", false, "waas gitops apply"},
		{"unsupported", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			tpl, err := GenerateCICDTemplate(tt.platform, "tenant-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Contains(t, tpl.Content, tt.contains)
		})
	}
}

func TestEnvironmentPromotion_InvalidChain(t *testing.T) {
	svc := NewService(nil)

	// Promoting backward should fail
	_, err := svc.PromoteConfig(nil, "tenant-1", &PromotionRequest{
		SourceEnv:  "prod",
		TargetEnv:  "dev",
		ManifestID: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only promote forward")
}

func TestEnvironmentPromotion_InvalidEnv(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.PromoteConfig(nil, "tenant-1", &PromotionRequest{
		SourceEnv:  "nonexistent",
		TargetEnv:  "prod",
		ManifestID: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid environment")
}

func TestParseSemanticVersion_EnvIndex(t *testing.T) {
	assert.Equal(t, 0, envIndex("dev"))
	assert.Equal(t, 1, envIndex("staging"))
	assert.Equal(t, 2, envIndex("prod"))
	assert.Equal(t, -1, envIndex("nonexistent"))
}
