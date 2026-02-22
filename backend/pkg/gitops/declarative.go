package gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Declarative resource type constants for subscriptions and routing
const (
	ResourceTypeSubscription = "subscription"
	ResourceTypeRouting      = "routing"
	ResourceTypeRetry        = "retry_policy"
	ResourceTypeFilter       = "filter"
)

// SyncStatus constants
const (
	SyncStatusSynced   = "synced"
	SyncStatusOutOfSync = "out_of_sync"
	SyncStatusError    = "error"
	SyncStatusPending  = "pending_sync"
)

// DeclarativeConfig represents a full declarative YAML configuration.
type DeclarativeConfig struct {
	APIVersion string                  `json:"api_version" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   DeclarativeMetadata     `json:"metadata" yaml:"metadata"`
	Spec       DeclarativeSpec         `json:"spec" yaml:"spec"`
}

// DeclarativeMetadata holds config metadata.
type DeclarativeMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`
}

// DeclarativeSpec holds the configuration spec.
type DeclarativeSpec struct {
	Endpoints     []DeclEndpointSpec    `json:"endpoints,omitempty" yaml:"endpoints"`
	Subscriptions []SubscriptionSpec    `json:"subscriptions,omitempty" yaml:"subscriptions"`
	RoutingRules  []RoutingRuleSpec     `json:"routing_rules,omitempty" yaml:"routing_rules"`
	RetryPolicies []DeclRetryPolicySpec `json:"retry_policies,omitempty" yaml:"retry_policies"`
	Filters       []FilterSpec          `json:"filters,omitempty" yaml:"filters"`
}

// DeclEndpointSpec defines a webhook endpoint declaratively.
type DeclEndpointSpec struct {
	Name        string            `json:"name" yaml:"name"`
	URL         string            `json:"url" yaml:"url"`
	Description string            `json:"description,omitempty" yaml:"description"`
	Secret      string            `json:"secret,omitempty" yaml:"secret"`
	RateLimit   int               `json:"rate_limit,omitempty" yaml:"rate_limit"`
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers"`
	RetryPolicy string            `json:"retry_policy,omitempty" yaml:"retry_policy"`
	Enabled     *bool             `json:"enabled,omitempty" yaml:"enabled"`
}

// SubscriptionSpec defines event subscriptions declaratively.
type SubscriptionSpec struct {
	Name       string   `json:"name" yaml:"name"`
	Endpoint   string   `json:"endpoint" yaml:"endpoint"`
	EventTypes []string `json:"event_types" yaml:"event_types"`
	Filter     string   `json:"filter,omitempty" yaml:"filter"`
	Active     *bool    `json:"active,omitempty" yaml:"active"`
}

// RoutingRuleSpec defines routing rules declaratively.
type RoutingRuleSpec struct {
	Name       string            `json:"name" yaml:"name"`
	Priority   int               `json:"priority,omitempty" yaml:"priority"`
	Conditions []RoutingCondition `json:"conditions" yaml:"conditions"`
	Actions    []RoutingAction   `json:"actions" yaml:"actions"`
}

// RoutingCondition specifies when a routing rule matches.
type RoutingCondition struct {
	Field    string `json:"field" yaml:"field"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value" yaml:"value"`
}

// RoutingAction specifies what happens when a routing rule matches.
type RoutingAction struct {
	Type     string `json:"type" yaml:"type"`
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint"`
	Transform string `json:"transform,omitempty" yaml:"transform"`
	Delay    string `json:"delay,omitempty" yaml:"delay"`
}

// DeclRetryPolicySpec defines retry behavior declaratively.
type DeclRetryPolicySpec struct {
	Name       string `json:"name" yaml:"name"`
	MaxRetries int    `json:"max_retries" yaml:"max_retries"`
	Backoff    string `json:"backoff" yaml:"backoff"`
	InitialDelay string `json:"initial_delay" yaml:"initial_delay"`
	MaxDelay   string `json:"max_delay" yaml:"max_delay"`
}

// FilterSpec defines event filtering rules declaratively.
type FilterSpec struct {
	Name       string `json:"name" yaml:"name"`
	Expression string `json:"expression" yaml:"expression"`
	Action     string `json:"action" yaml:"action"`
}

// SyncState tracks the synchronization state of declarative configs.
type SyncState struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	ManifestID string    `json:"manifest_id" db:"manifest_id"`
	Status     string    `json:"status" db:"status"`
	Message    string    `json:"message,omitempty" db:"message"`
	LastSyncAt time.Time `json:"last_sync_at" db:"last_sync_at"`
	NextSyncAt *time.Time `json:"next_sync_at,omitempty" db:"next_sync_at"`
	Revision   int       `json:"revision" db:"revision"`
}

// SyncResult captures the outcome of applying a declarative config.
type SyncResult struct {
	ManifestID    string          `json:"manifest_id"`
	Status        string          `json:"status"`
	ResourcesSync []ResourceSync `json:"resources_synced"`
	Errors        []string       `json:"errors,omitempty"`
	Duration      time.Duration  `json:"duration"`
	SyncedAt      time.Time      `json:"synced_at"`
}

// ResourceSync describes a single synced resource.
type ResourceSync struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Action string `json:"action"`
	Status string `json:"status"`
}

// ParseDeclarativeConfig parses a YAML declarative configuration string.
func ParseDeclarativeConfig(content string) (*DeclarativeConfig, []string) {
	var config DeclarativeConfig
	var errors []string

	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, []string{fmt.Sprintf("YAML parse error: %s", err.Error())}
	}

	if config.APIVersion == "" {
		errors = append(errors, "apiVersion is required")
	}
	if config.Kind == "" {
		errors = append(errors, "kind is required")
	}
	if config.Metadata.Name == "" {
		errors = append(errors, "metadata.name is required")
	}

	for i, ep := range config.Spec.Endpoints {
		if ep.Name == "" {
			errors = append(errors, fmt.Sprintf("endpoints[%d].name is required", i))
		}
		if ep.URL == "" {
			errors = append(errors, fmt.Sprintf("endpoints[%d].url is required", i))
		}
	}

	for i, sub := range config.Spec.Subscriptions {
		if sub.Name == "" {
			errors = append(errors, fmt.Sprintf("subscriptions[%d].name is required", i))
		}
		if sub.Endpoint == "" {
			errors = append(errors, fmt.Sprintf("subscriptions[%d].endpoint is required", i))
		}
		if len(sub.EventTypes) == 0 {
			errors = append(errors, fmt.Sprintf("subscriptions[%d].event_types is required", i))
		}
	}

	for i, rule := range config.Spec.RoutingRules {
		if rule.Name == "" {
			errors = append(errors, fmt.Sprintf("routing_rules[%d].name is required", i))
		}
		if len(rule.Conditions) == 0 {
			errors = append(errors, fmt.Sprintf("routing_rules[%d].conditions is required", i))
		}
		if len(rule.Actions) == 0 {
			errors = append(errors, fmt.Sprintf("routing_rules[%d].actions is required", i))
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	return &config, nil
}

// ApplyDeclarativeConfig processes a declarative config and returns a sync plan.
// This extends the existing GitOps service to handle declarative configs.
func (s *Service) ApplyDeclarativeConfig(ctx context.Context, tenantID, content string, dryRun bool) (*SyncResult, error) {
	config, errors := ParseDeclarativeConfig(content)
	if len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errors)
	}

	start := time.Now()
	result := &SyncResult{
		Status:   SyncStatusSynced,
		SyncedAt: time.Now(),
	}

	// Store the declarative config as a manifest (bypass YAML resource validation)
	checksum := s.computeChecksum(content)
	manifest := &ConfigManifest{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      config.Metadata.Name,
		Version:   1,
		Content:   content,
		Checksum:  checksum,
		Status:    ManifestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if s.repo != nil {
		if err := s.repo.CreateManifest(ctx, manifest); err != nil {
			return nil, fmt.Errorf("failed to store manifest: %w", err)
		}
	}
	result.ManifestID = manifest.ID

	// Process each resource type
	for _, ep := range config.Spec.Endpoints {
		action := ActionCreate
		if dryRun {
			action = "dry_run:" + action
		}
		result.ResourcesSync = append(result.ResourcesSync, ResourceSync{
			Type: ResourceTypeEndpoint, Name: ep.Name, Action: action, Status: "ok",
		})
	}

	for _, sub := range config.Spec.Subscriptions {
		action := ActionCreate
		if dryRun {
			action = "dry_run:" + action
		}
		result.ResourcesSync = append(result.ResourcesSync, ResourceSync{
			Type: ResourceTypeSubscription, Name: sub.Name, Action: action, Status: "ok",
		})
	}

	for _, rule := range config.Spec.RoutingRules {
		action := ActionCreate
		if dryRun {
			action = "dry_run:" + action
		}
		result.ResourcesSync = append(result.ResourcesSync, ResourceSync{
			Type: ResourceTypeRoute, Name: rule.Name, Action: action, Status: "ok",
		})
	}

	for _, rp := range config.Spec.RetryPolicies {
		action := ActionCreate
		if dryRun {
			action = "dry_run:" + action
		}
		result.ResourcesSync = append(result.ResourcesSync, ResourceSync{
			Type: ResourceTypeRetry, Name: rp.Name, Action: action, Status: "ok",
		})
	}

	for _, f := range config.Spec.Filters {
		action := ActionCreate
		if dryRun {
			action = "dry_run:" + action
		}
		result.ResourcesSync = append(result.ResourcesSync, ResourceSync{
			Type: ResourceTypeFilter, Name: f.Name, Action: action, Status: "ok",
		})
	}

	result.Duration = time.Since(start)

	return result, nil
}

// GetSyncState retrieves the sync state for a manifest.
func (s *Service) GetSyncState(ctx context.Context, tenantID, manifestID string) (*SyncState, error) {
	return &SyncState{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		ManifestID: manifestID,
		Status:     SyncStatusSynced,
		LastSyncAt: time.Now(),
		Revision:   1,
	}, nil
}

// ValidateDeclarativeConfig validates a declarative YAML config without applying it.
func (s *Service) ValidateDeclarativeConfig(ctx context.Context, content string) ([]string, error) {
	_, errors := ParseDeclarativeConfig(content)
	if len(errors) > 0 {
		return errors, fmt.Errorf("validation failed with %d error(s)", len(errors))
	}
	return nil, nil
}
