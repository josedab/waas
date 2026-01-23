package tfprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides Terraform/Pulumi provider API functionality
type Service struct {
	repo Repository
}

// NewService creates a new Terraform provider service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetResourceSchemas returns schemas for all manageable resource types
func (s *Service) GetResourceSchemas() []ResourceSchema {
	return []ResourceSchema{
		{
			Type: ResourceTenant, Description: "A WaaS tenant",
			Attributes: []AttributeSchema{
				{Name: "name", Type: "string", Required: true, Description: "Tenant name"},
				{Name: "email", Type: "string", Required: true, Description: "Contact email"},
				{Name: "plan", Type: "string", Required: false, Default: "free", Description: "Subscription plan"},
			},
			Example: "resource \"waas_tenant\" \"main\" {\n  name  = \"my-app\"\n  email = \"admin@example.com\"\n  plan  = \"pro\"\n}",
		},
		{
			Type: ResourceEndpoint, Description: "A webhook endpoint",
			Attributes: []AttributeSchema{
				{Name: "url", Type: "string", Required: true, Description: "Endpoint URL"},
				{Name: "is_active", Type: "bool", Required: false, Default: "true", Description: "Whether endpoint is active"},
				{Name: "max_retries", Type: "int", Required: false, Default: "5", Description: "Maximum retry attempts"},
			},
			Example: "resource \"waas_endpoint\" \"payments\" {\n  url         = \"https://api.example.com/webhooks\"\n  max_retries = 5\n}",
		},
		{
			Type: ResourceRoute, Description: "An event mesh routing rule",
			Attributes: []AttributeSchema{
				{Name: "name", Type: "string", Required: true, Description: "Route name"},
				{Name: "event_types", Type: "list", Required: false, Description: "Event types to match"},
				{Name: "target_endpoints", Type: "list", Required: true, Description: "Target endpoint IDs"},
				{Name: "priority", Type: "int", Required: false, Default: "0", Description: "Route priority"},
			},
			Example: "resource \"waas_route\" \"payments\" {\n  name           = \"payment-events\"\n  event_types    = [\"payment.*\"]\n  target_endpoints = [waas_endpoint.payments.id]\n}",
		},
		{
			Type: ResourceSLATarget, Description: "An SLA target",
			Attributes: []AttributeSchema{
				{Name: "name", Type: "string", Required: true, Description: "SLA target name"},
				{Name: "delivery_rate_pct", Type: "float", Required: true, Description: "Target delivery rate percentage"},
				{Name: "latency_p99_ms", Type: "int", Required: false, Default: "0", Description: "P99 latency target in ms"},
				{Name: "window_minutes", Type: "int", Required: true, Description: "Measurement window in minutes"},
			},
			Example: "resource \"waas_sla_target\" \"production\" {\n  name              = \"production-sla\"\n  delivery_rate_pct = 99.9\n  latency_p99_ms    = 5000\n  window_minutes    = 60\n}",
		},
		{
			Type: ResourceContract, Description: "A webhook contract",
			Attributes: []AttributeSchema{
				{Name: "name", Type: "string", Required: true, Description: "Contract name"},
				{Name: "version", Type: "string", Required: true, Description: "Schema version"},
				{Name: "event_type", Type: "string", Required: true, Description: "Event type"},
				{Name: "schema", Type: "string", Required: true, Description: "JSON Schema definition"},
			},
			Example: "resource \"waas_contract\" \"order_created\" {\n  name       = \"order-created\"\n  version    = \"1.0.0\"\n  event_type = \"order.created\"\n  schema     = file(\"schemas/order.json\")\n}",
		},
	}
}

// ImportResource imports an existing resource into Terraform state
func (s *Service) ImportResource(ctx context.Context, tenantID string, req *StateImportRequest) (*StateExport, error) {
	resource, err := s.repo.GetResource(ctx, tenantID, req.ResourceType, req.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("resource not found: %w", err)
	}

	hcl := generateHCL(resource)

	return &StateExport{
		ResourceType: resource.ResourceType,
		ResourceID:   resource.ResourceID,
		Attributes:   resource.Attributes,
		HCL:          hcl,
	}, nil
}

// ListManagedResources lists all resources managed by IaC for a tenant
func (s *Service) ListManagedResources(ctx context.Context, tenantID string, resourceType ResourceType) ([]ManagedResource, error) {
	return s.repo.ListResources(ctx, tenantID, resourceType)
}

// RegisterResource records a resource as managed by IaC
func (s *Service) RegisterResource(ctx context.Context, tenantID string, resourceType ResourceType, resourceID string, attrs map[string]string, managedBy string) (*ManagedResource, error) {
	resource := &ManagedResource{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Name:         attrs["name"],
		State:        "managed",
		Attributes:   attrs,
		ManagedBy:    managedBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.SaveResource(ctx, resource); err != nil {
		return nil, fmt.Errorf("failed to register resource: %w", err)
	}

	return resource, nil
}

// DeregisterResource removes a resource from IaC management
func (s *Service) DeregisterResource(ctx context.Context, tenantID string, resourceType ResourceType, resourceID string) error {
	return s.repo.DeleteResource(ctx, tenantID, resourceType, resourceID)
}

func generateHCL(resource *ManagedResource) string {
	var sb strings.Builder
	typeName := string(resource.ResourceType)
	name := resource.Attributes["name"]
	if name == "" {
		name = "imported"
	}

	sb.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", typeName, sanitizeHCLName(name)))
	for k, v := range resource.Attributes {
		sb.WriteString(fmt.Sprintf("  %s = \"%s\"\n", k, v))
	}
	sb.WriteString("}\n")

	return sb.String()
}

func sanitizeHCLName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToLower(name)
}
