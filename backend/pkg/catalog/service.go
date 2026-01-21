package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides event catalog business logic
type Service struct {
	repo *Repository
}

// NewService creates a new catalog service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateEventTypeRequest represents a request to create an event type
type CreateEventTypeRequest struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description,omitempty"`
	Category         string          `json:"category,omitempty"`
	SchemaID         *uuid.UUID      `json:"schema_id,omitempty"`
	ExamplePayload   json.RawMessage `json:"example_payload,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	DocumentationURL string          `json:"documentation_url,omitempty"`
}

// UpdateEventTypeRequest represents a request to update an event type
type UpdateEventTypeRequest struct {
	Name               string          `json:"name,omitempty"`
	Description        string          `json:"description,omitempty"`
	Category           string          `json:"category,omitempty"`
	SchemaID           *uuid.UUID      `json:"schema_id,omitempty"`
	ExamplePayload     json.RawMessage `json:"example_payload,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	DocumentationURL   string          `json:"documentation_url,omitempty"`
}

// DeprecateEventTypeRequest represents a request to deprecate an event type
type DeprecateEventTypeRequest struct {
	Message          string     `json:"message" binding:"required"`
	ReplacementID    *uuid.UUID `json:"replacement_id,omitempty"`
}

// PublishVersionRequest represents a request to publish a new version
type PublishVersionRequest struct {
	Version          string     `json:"version" binding:"required"`
	SchemaID         *uuid.UUID `json:"schema_id,omitempty"`
	Changelog        string     `json:"changelog,omitempty"`
	IsBreakingChange bool       `json:"is_breaking_change"`
}

// CreateCategoryRequest represents a request to create a category
type CreateCategoryRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// SubscribeRequest represents a request to subscribe an endpoint to an event
type SubscribeRequest struct {
	EndpointID       uuid.UUID       `json:"endpoint_id" binding:"required"`
	FilterExpression json.RawMessage `json:"filter_expression,omitempty"`
}

// CreateEventType creates a new event type in the catalog
func (s *Service) CreateEventType(ctx context.Context, tenantID uuid.UUID, req *CreateEventTypeRequest) (*EventType, error) {
	slug := generateSlug(req.Name)

	et := &EventType{
		TenantID:         tenantID,
		Name:             req.Name,
		Slug:             slug,
		Description:      req.Description,
		Category:         req.Category,
		SchemaID:         req.SchemaID,
		Version:          "1.0.0",
		Status:           StatusActive,
		ExamplePayload:   req.ExamplePayload,
		Tags:             req.Tags,
		DocumentationURL: req.DocumentationURL,
	}

	if err := s.repo.CreateEventType(ctx, et); err != nil {
		return nil, err
	}

	// Create initial version record
	ev := &EventVersion{
		EventTypeID:      et.ID,
		Version:          "1.0.0",
		SchemaID:         req.SchemaID,
		Changelog:        "Initial version",
		IsBreakingChange: false,
	}
	s.repo.CreateEventVersion(ctx, ev)

	return et, nil
}

// GetEventType retrieves an event type by ID
func (s *Service) GetEventType(ctx context.Context, id uuid.UUID) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with subscriber count
	count, _ := s.repo.GetSubscriberCount(ctx, id)
	et.Subscribers = count

	// Get versions
	versions, _ := s.repo.ListEventVersions(ctx, id)
	et.Versions = versions

	return et, nil
}

// GetEventTypeBySlug retrieves an event type by slug
func (s *Service) GetEventTypeBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*EventType, error) {
	return s.repo.GetEventTypeBySlug(ctx, tenantID, slug)
}

// UpdateEventType updates an event type
func (s *Service) UpdateEventType(ctx context.Context, id uuid.UUID, req *UpdateEventTypeRequest) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		et.Name = req.Name
	}
	if req.Description != "" {
		et.Description = req.Description
	}
	if req.Category != "" {
		et.Category = req.Category
	}
	if req.SchemaID != nil {
		et.SchemaID = req.SchemaID
	}
	if req.ExamplePayload != nil {
		et.ExamplePayload = req.ExamplePayload
	}
	if req.Tags != nil {
		et.Tags = req.Tags
	}
	if req.DocumentationURL != "" {
		et.DocumentationURL = req.DocumentationURL
	}

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// DeleteEventType deletes an event type
func (s *Service) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEventType(ctx, id)
}

// DeprecateEventType marks an event type as deprecated
func (s *Service) DeprecateEventType(ctx context.Context, id uuid.UUID, req *DeprecateEventTypeRequest) (*EventType, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	et.Status = StatusDeprecated
	et.DeprecationMessage = req.Message
	et.DeprecatedAt = &now
	et.ReplacementEventID = req.ReplacementID

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// PublishVersion publishes a new version of an event type
func (s *Service) PublishVersion(ctx context.Context, id uuid.UUID, req *PublishVersionRequest, publisherID *uuid.UUID) (*EventVersion, error) {
	// Validate version format
	if !isValidSemver(req.Version) {
		return nil, fmt.Errorf("invalid version format, must be semver (e.g., 1.2.3)")
	}

	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update the event type's current version
	et.Version = req.Version
	if req.SchemaID != nil {
		et.SchemaID = req.SchemaID
	}

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	// Create version record
	ev := &EventVersion{
		EventTypeID:      id,
		Version:          req.Version,
		SchemaID:         req.SchemaID,
		Changelog:        req.Changelog,
		IsBreakingChange: req.IsBreakingChange,
		PublishedBy:      publisherID,
	}

	if err := s.repo.CreateEventVersion(ctx, ev); err != nil {
		return nil, err
	}

	return ev, nil
}

// SearchEventTypes searches the event catalog
func (s *Service) SearchEventTypes(ctx context.Context, params *CatalogSearchParams) (*CatalogSearchResult, error) {
	return s.repo.SearchEventTypes(ctx, params)
}

// ListCategories lists all event categories
func (s *Service) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]*EventCategory, error) {
	categories, err := s.repo.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Build hierarchy
	categoryMap := make(map[uuid.UUID]*EventCategory)
	var roots []*EventCategory

	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	for _, cat := range categories {
		if cat.ParentID == nil {
			roots = append(roots, cat)
		} else if parent, ok := categoryMap[*cat.ParentID]; ok {
			parent.Children = append(parent.Children, cat)
		}
	}

	return roots, nil
}

// CreateCategory creates a new event category
func (s *Service) CreateCategory(ctx context.Context, tenantID uuid.UUID, req *CreateCategoryRequest) (*EventCategory, error) {
	cat := &EventCategory{
		TenantID:    tenantID,
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		ParentID:    req.ParentID,
	}

	if err := s.repo.CreateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

// Subscribe subscribes an endpoint to an event type
func (s *Service) Subscribe(ctx context.Context, eventTypeID uuid.UUID, req *SubscribeRequest) (*EventSubscription, error) {
	sub := &EventSubscription{
		EndpointID:       req.EndpointID,
		EventTypeID:      eventTypeID,
		FilterExpression: req.FilterExpression,
		IsActive:         true,
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// Unsubscribe removes an endpoint subscription from an event type
func (s *Service) Unsubscribe(ctx context.Context, eventTypeID, endpointID uuid.UUID) error {
	return s.repo.DeleteSubscription(ctx, endpointID, eventTypeID)
}

// GetEndpointSubscriptions returns all event subscriptions for an endpoint
func (s *Service) GetEndpointSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]*EventSubscription, error) {
	return s.repo.ListEndpointSubscriptions(ctx, endpointID)
}

// GetVersions returns all versions of an event type
func (s *Service) GetVersions(ctx context.Context, eventTypeID uuid.UUID) ([]*EventVersion, error) {
	return s.repo.ListEventVersions(ctx, eventTypeID)
}

// SaveDocumentation saves documentation for an event type
func (s *Service) SaveDocumentation(ctx context.Context, eventTypeID uuid.UUID, section, content string) error {
	doc := &EventDocumentation{
		EventTypeID: eventTypeID,
		ContentType: "markdown",
		Content:     content,
		Section:     section,
	}
	return s.repo.SaveDocumentation(ctx, doc)
}

// GetDocumentation retrieves all documentation for an event type
func (s *Service) GetDocumentation(ctx context.Context, eventTypeID uuid.UUID) ([]*EventDocumentation, error) {
	return s.repo.GetDocumentation(ctx, eventTypeID)
}

// GenerateOpenAPISpec generates an OpenAPI spec for the event catalog
func (s *Service) GenerateOpenAPISpec(ctx context.Context, tenantID uuid.UUID) (json.RawMessage, error) {
	result, err := s.repo.SearchEventTypes(ctx, &CatalogSearchParams{
		TenantID: tenantID,
		Status:   StatusActive,
		Limit:    1000,
	})
	if err != nil {
		return nil, err
	}

	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Webhook Events API",
			"version":     "1.0.0",
			"description": "Event types and schemas for webhooks",
		},
		"webhooks": make(map[string]interface{}),
	}

	webhooks := spec["webhooks"].(map[string]interface{})
	for _, et := range result.EventTypes {
		webhooks[et.Slug] = map[string]interface{}{
			"post": map[string]interface{}{
				"summary":     et.Name,
				"description": et.Description,
				"operationId": et.Slug,
				"tags":        et.Tags,
				"requestBody": map[string]interface{}{
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"example": et.ExamplePayload,
						},
					},
				},
			},
		}
	}

	return json.MarshalIndent(spec, "", "  ")
}

// Helper functions

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", ".")
	re := regexp.MustCompile(`[^a-z0-9.]`)
	slug = re.ReplaceAllString(slug, "")
	return slug
}

func isValidSemver(version string) bool {
	re := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`)
	return re.MatchString(version)
}
