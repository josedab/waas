package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// WaaSConfig represents the top-level YAML/JSON configuration file
type WaaSConfig struct {
	APIVersion string         `json:"apiVersion" yaml:"apiVersion"`
	Kind       string         `json:"kind" yaml:"kind"` // WaaSConfig
	Metadata   ConfigMetadata `json:"metadata" yaml:"metadata"`
	Spec       WaaSConfigSpec `json:"spec" yaml:"spec"`
}

// ConfigMetadata holds metadata for the configuration
type ConfigMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// WaaSConfigSpec holds the configuration specification
type WaaSConfigSpec struct {
	Tenant     *TenantSpec     `json:"tenant,omitempty" yaml:"tenant,omitempty"`
	Endpoints  []EndpointSpec  `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	EventTypes []EventTypeSpec `json:"eventTypes,omitempty" yaml:"eventTypes,omitempty"`
	Transforms []TransformSpec `json:"transforms,omitempty" yaml:"transforms,omitempty"`
	Alerts     []AlertSpec     `json:"alerts,omitempty" yaml:"alerts,omitempty"`
	Pipelines  []PipelineSpec  `json:"pipelines,omitempty" yaml:"pipelines,omitempty"`
}

// TenantSpec defines tenant configuration
type TenantSpec struct {
	Name        string            `json:"name" yaml:"name"`
	Environment string            `json:"environment,omitempty" yaml:"environment,omitempty"`
	Settings    map[string]string `json:"settings,omitempty" yaml:"settings,omitempty"`
}

// EndpointSpec defines a webhook endpoint
type EndpointSpec struct {
	Name        string            `json:"name" yaml:"name"`
	URL         string            `json:"url" yaml:"url"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	EventTypes  []string          `json:"eventTypes" yaml:"eventTypes"`
	RateLimit   int               `json:"rateLimit,omitempty" yaml:"rateLimit,omitempty"`
	Timeout     int               `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	RetryPolicy *RetryPolicySpec  `json:"retryPolicy,omitempty" yaml:"retryPolicy,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

// RetryPolicySpec defines retry behavior
type RetryPolicySpec struct {
	MaxRetries   int    `json:"maxRetries" yaml:"maxRetries"`
	Strategy     string `json:"strategy" yaml:"strategy"` // exponential, fixed, adaptive
	InitialDelay string `json:"initialDelay,omitempty" yaml:"initialDelay,omitempty"`
	MaxDelay     string `json:"maxDelay,omitempty" yaml:"maxDelay,omitempty"`
}

// EventTypeSpec defines an event type
type EventTypeSpec struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// TransformSpec defines a payload transformation
type TransformSpec struct {
	Name       string `json:"name" yaml:"name"`
	EventType  string `json:"eventType" yaml:"eventType"`
	Expression string `json:"expression" yaml:"expression"`
}

// AlertSpec defines an alert rule
type AlertSpec struct {
	Name      string   `json:"name" yaml:"name"`
	Metric    string   `json:"metric" yaml:"metric"`
	Threshold float64  `json:"threshold" yaml:"threshold"`
	Operator  string   `json:"operator" yaml:"operator"`
	Channels  []string `json:"channels" yaml:"channels"`
}

// PipelineSpec defines a webhook delivery pipeline
type PipelineSpec struct {
	Name   string         `json:"name" yaml:"name"`
	Source string         `json:"source" yaml:"source"`
	Steps  []PipelineStep `json:"steps" yaml:"steps"`
}

// PipelineStep defines a step in a pipeline
type PipelineStep struct {
	Name   string            `json:"name" yaml:"name"`
	Type   string            `json:"type" yaml:"type"` // transform, filter, route, enrich
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// RollbackRequest is the request for rolling back a manifest
type RollbackRequest struct {
	ManifestID    string `json:"manifest_id" binding:"required"`
	TargetVersion int    `json:"target_version"`
	Reason        string `json:"reason,omitempty"`
}

// RollbackResult describes the outcome of a rollback
type RollbackResult struct {
	ManifestID    string    `json:"manifest_id"`
	FromVersion   int       `json:"from_version"`
	ToVersion     int       `json:"to_version"`
	Status        string    `json:"status"`
	ResourceCount int       `json:"resource_count"`
	Errors        []string  `json:"errors,omitempty"`
	RolledBackAt  time.Time `json:"rolled_back_at"`
}

// ManifestHistory represents the version history of a manifest
type ManifestHistory struct {
	ManifestID string            `json:"manifest_id"`
	Versions   []ManifestVersion `json:"versions"`
}

// ManifestVersion represents a single version of a manifest
type ManifestVersion struct {
	Version   int        `json:"version"`
	Checksum  string     `json:"checksum"`
	Status    string     `json:"status"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Changes   int        `json:"changes"`
}

// ParseConfig parses a YAML or JSON WaaS configuration
func ParseConfig(content []byte) (*WaaSConfig, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("empty configuration content")
	}

	var config WaaSConfig

	// Try YAML first (superset of JSON)
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if config.APIVersion == "" {
		config.APIVersion = "waas.dev/v1"
	}
	if config.Kind == "" {
		config.Kind = "WaaSConfig"
	}

	return &config, nil
}

// ValidateConfig validates a WaaS configuration for correctness
func ValidateConfig(config *WaaSConfig) []string {
	var errors []string

	if config.Metadata.Name == "" {
		errors = append(errors, "metadata.name is required")
	}

	for i, ep := range config.Spec.Endpoints {
		if ep.Name == "" {
			errors = append(errors, fmt.Sprintf("spec.endpoints[%d].name is required", i))
		}
		if ep.URL == "" {
			errors = append(errors, fmt.Sprintf("spec.endpoints[%d].url is required", i))
		}
		if !strings.HasPrefix(ep.URL, "http://") && !strings.HasPrefix(ep.URL, "https://") {
			errors = append(errors, fmt.Sprintf("spec.endpoints[%d].url must start with http:// or https://", i))
		}
		if len(ep.EventTypes) == 0 {
			errors = append(errors, fmt.Sprintf("spec.endpoints[%d].eventTypes must have at least one entry", i))
		}
	}

	for i, et := range config.Spec.EventTypes {
		if et.Name == "" {
			errors = append(errors, fmt.Sprintf("spec.eventTypes[%d].name is required", i))
		}
	}

	for i, t := range config.Spec.Transforms {
		if t.Name == "" {
			errors = append(errors, fmt.Sprintf("spec.transforms[%d].name is required", i))
		}
		if t.Expression == "" {
			errors = append(errors, fmt.Sprintf("spec.transforms[%d].expression is required", i))
		}
	}

	return errors
}

// GenerateApplyPlan creates an execution plan from a config diff
func GenerateApplyPlan(config *WaaSConfig, existingResources map[string]string) *ApplyPlan {
	plan := &ApplyPlan{
		ManifestID: uuid.New().String(),
	}

	// Plan endpoint changes
	for _, ep := range config.Spec.Endpoints {
		action := ActionCreate
		if _, exists := existingResources["endpoint:"+ep.Name]; exists {
			action = ActionUpdate
		}

		isDestructive := false
		if action == ActionUpdate {
			// URL changes are destructive
			existing := existingResources["endpoint:"+ep.Name]
			if existing != ep.URL {
				isDestructive = true
			}
		}

		pa := PlannedAction{
			ResourceType:  ResourceTypeEndpoint,
			ResourceID:    ep.Name,
			Action:        action,
			Description:   fmt.Sprintf("%s endpoint %q -> %s", action, ep.Name, ep.URL),
			IsDestructive: isDestructive,
		}
		plan.Resources = append(plan.Resources, pa)

		switch action {
		case ActionCreate:
			plan.TotalCreates++
		case ActionUpdate:
			plan.TotalUpdates++
		}
		if isDestructive {
			plan.HasDestructive = true
		}
	}

	// Plan event type changes
	for _, et := range config.Spec.EventTypes {
		action := ActionCreate
		if _, exists := existingResources["eventtype:"+et.Name]; exists {
			action = ActionUpdate
		}
		plan.Resources = append(plan.Resources, PlannedAction{
			ResourceType: "event_type",
			ResourceID:   et.Name,
			Action:       action,
			Description:  fmt.Sprintf("%s event type %q", action, et.Name),
		})
	}

	// Plan transform changes
	for _, t := range config.Spec.Transforms {
		action := ActionCreate
		if _, exists := existingResources["transform:"+t.Name]; exists {
			action = ActionUpdate
		}
		plan.Resources = append(plan.Resources, PlannedAction{
			ResourceType: ResourceTypeTransform,
			ResourceID:   t.Name,
			Action:       action,
			Description:  fmt.Sprintf("%s transform %q", action, t.Name),
		})
	}

	return plan
}

// DetectConfigDrift compares desired state from a WaaSConfig with actual state
func (s *Service) DetectConfigDrift(ctx context.Context, tenantID string, config *WaaSConfig) (*DriftReport, error) {
	report := &DriftReport{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		DetectedAt: time.Now(),
		Status:     DriftStatusDetected,
	}

	// Count all resources
	report.ResourceCount = len(config.Spec.Endpoints) + len(config.Spec.EventTypes) + len(config.Spec.Transforms)

	// Check for endpoint drift by comparing with repo state
	if s.repo != nil {
		manifests, err := s.repo.ListManifests(ctx, tenantID)
		if err == nil {
			for _, ep := range config.Spec.Endpoints {
				drifted := false
				for _, m := range manifests {
					if m.Status != ManifestStatusApplied {
						continue
					}
					var existingConfig WaaSConfig
					if yaml.Unmarshal([]byte(m.Content), &existingConfig) == nil {
						for _, existingEp := range existingConfig.Spec.Endpoints {
							if existingEp.Name == ep.Name && existingEp.URL != ep.URL {
								report.Details = append(report.Details, DriftDetail{
									ResourceType:  ResourceTypeEndpoint,
									ResourceID:    ep.Name,
									Field:         "url",
									ExpectedValue: ep.URL,
									ActualValue:   existingEp.URL,
									Severity:      DriftSeverityWarning,
								})
								drifted = true
							}
						}
					}
				}
				if drifted {
					report.DriftedCount++
				}
			}
		}
	}

	if report.DriftedCount == 0 {
		report.Status = DriftStatusResolved
	}

	return report, nil
}

// Rollback reverts to a previous manifest version
func (s *Service) Rollback(ctx context.Context, tenantID string, req *RollbackRequest) (*RollbackResult, error) {
	result := &RollbackResult{
		ManifestID:   req.ManifestID,
		Status:       "success",
		RolledBackAt: time.Now(),
	}

	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	manifest, err := s.repo.GetManifest(ctx, tenantID, req.ManifestID)
	if err != nil {
		return nil, fmt.Errorf("manifest not found: %w", err)
	}

	result.FromVersion = manifest.Version

	if req.TargetVersion > 0 {
		result.ToVersion = req.TargetVersion
	} else {
		result.ToVersion = manifest.Version - 1
	}

	if result.ToVersion < 1 {
		return nil, fmt.Errorf("cannot rollback to version %d", result.ToVersion)
	}

	// Update manifest status
	manifest.Status = ManifestStatusRolledBack
	manifest.UpdatedAt = time.Now()
	if err := s.repo.UpdateManifest(ctx, manifest); err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}

// ComputeChecksum generates a SHA256 checksum of content
func ComputeChecksum(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GenerateExampleConfig returns an example WaaS YAML configuration
func GenerateExampleConfig() string {
	return `apiVersion: waas.dev/v1
kind: WaaSConfig
metadata:
  name: my-webhooks
  namespace: production
  labels:
    team: platform
    env: production

spec:
  tenant:
    name: My Application
    environment: production
    settings:
      max_retry_attempts: "5"
      default_timeout: "30s"

  endpoints:
    - name: order-processor
      url: https://api.example.com/webhooks/orders
      description: Handles order lifecycle events
      eventTypes:
        - order.created
        - order.updated
        - order.completed
      rateLimit: 100
      timeoutSeconds: 30
      retryPolicy:
        maxRetries: 5
        strategy: exponential
        initialDelay: 5s
        maxDelay: 1h
      headers:
        X-Custom-Header: my-value

    - name: notification-service
      url: https://notifications.example.com/webhook
      eventTypes:
        - user.signup
        - order.completed
      retryPolicy:
        maxRetries: 3
        strategy: fixed
        initialDelay: 10s

  eventTypes:
    - name: order.created
      description: Fired when a new order is placed
    - name: order.updated
      description: Fired when an order is modified
    - name: order.completed
      description: Fired when an order is fulfilled

  transforms:
    - name: flatten-order
      eventType: order.created
      expression: "{ order_id: payload.id, total: payload.amount }"

  alerts:
    - name: high-failure-rate
      metric: error_rate
      threshold: 5.0
      operator: gt
      channels:
        - slack
        - email

  pipelines:
    - name: order-pipeline
      source: stripe
      steps:
        - name: validate
          type: filter
          config:
            expression: "payload.type == 'order.created'"
        - name: transform
          type: transform
          config:
            expression: "{ id: payload.id, amount: payload.amount }"
        - name: deliver
          type: route
          config:
            endpoint: order-processor
`
}
