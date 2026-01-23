package tfprovider

import "time"

// ProviderConfig holds Terraform provider configuration
type ProviderConfig struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

// ManagedResource represents a WaaS resource managed by Terraform
type ManagedResource struct {
	ID           string            `json:"id" db:"id"`
	TenantID     string            `json:"tenant_id" db:"tenant_id"`
	ResourceType ResourceType      `json:"resource_type" db:"resource_type"`
	ResourceID   string            `json:"resource_id" db:"resource_id"`
	Name         string            `json:"name" db:"name"`
	State        string            `json:"state" db:"state"`
	Attributes   map[string]string `json:"attributes"`
	AttrsJSON    string            `json:"-" db:"attributes"`
	ManagedBy    string            `json:"managed_by" db:"managed_by"` // terraform, pulumi, manual
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// ResourceType represents types of resources manageable via IaC
type ResourceType string

const (
	ResourceTenant       ResourceType = "waas_tenant"
	ResourceEndpoint     ResourceType = "waas_endpoint"
	ResourceRetryPolicy  ResourceType = "waas_retry_policy"
	ResourceRoute        ResourceType = "waas_route"
	ResourceSLATarget    ResourceType = "waas_sla_target"
	ResourceTLSPolicy    ResourceType = "waas_tls_policy"
	ResourceContract     ResourceType = "waas_contract"
)

// ResourceSchema describes a resource type for documentation and validation
type ResourceSchema struct {
	Type        ResourceType       `json:"type"`
	Description string             `json:"description"`
	Attributes  []AttributeSchema  `json:"attributes"`
	Example     string             `json:"example_hcl"`
}

// AttributeSchema describes a single resource attribute
type AttributeSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, int, bool, list, map
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// StateImportRequest is the request DTO for importing existing resources
type StateImportRequest struct {
	ResourceType ResourceType `json:"resource_type" binding:"required"`
	ResourceID   string       `json:"resource_id" binding:"required"`
}

// StateExport represents the exported state of a resource
type StateExport struct {
	ResourceType ResourceType      `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Attributes   map[string]string `json:"attributes"`
	HCL          string            `json:"hcl"`
}

// PlanChange represents a planned change to a resource
type PlanChange struct {
	Action       string            `json:"action"` // create, update, delete
	ResourceType ResourceType      `json:"resource_type"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Before       map[string]string `json:"before,omitempty"`
	After        map[string]string `json:"after,omitempty"`
}

// ApplyResult represents the result of applying changes
type ApplyResult struct {
	TotalChanges int          `json:"total_changes"`
	Created      int          `json:"created"`
	Updated      int          `json:"updated"`
	Deleted      int          `json:"deleted"`
	Errors       []string     `json:"errors,omitempty"`
}
