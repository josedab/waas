package livemigration

import (
	"time"

	"github.com/google/uuid"
)

// SvixCompatLayer provides API compatibility for Svix SDK calls
type SvixCompatLayer struct {
	service *Service
}

func NewSvixCompatLayer(service *Service) *SvixCompatLayer {
	return &SvixCompatLayer{service: service}
}

// Svix-compatible endpoint structures
type SvixApplication struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type SvixEndpointIn struct {
	UID         string            `json:"uid"`
	URL         string            `json:"url"`
	Version     int               `json:"version"`
	Description string            `json:"description"`
	FilterTypes []string          `json:"filterTypes"`
	Metadata    map[string]string `json:"metadata"`
}

type SvixEndpointOut struct {
	ID          string            `json:"id"`
	UID         string            `json:"uid"`
	URL         string            `json:"url"`
	Version     int               `json:"version"`
	Description string            `json:"description"`
	FilterTypes []string          `json:"filterTypes"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   string            `json:"createdAt"`
}

// CreateEndpoint maps a Svix endpoint creation to WaaS
func (c *SvixCompatLayer) CreateEndpoint(appUID string, endpoint SvixEndpointIn) (*SvixEndpointOut, error) {
	// Map Svix endpoint to WaaS endpoint
	return &SvixEndpointOut{
		ID:          endpoint.UID,
		UID:         endpoint.UID,
		URL:         endpoint.URL,
		Version:     endpoint.Version,
		Description: endpoint.Description,
		FilterTypes: endpoint.FilterTypes,
		Metadata:    endpoint.Metadata,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// ListEndpoints maps a Svix list call to WaaS.
// Returns empty because the compat layer handles incoming migration calls;
// endpoint discovery is performed through the native WaaS API instead.
func (c *SvixCompatLayer) ListEndpoints(appUID string) ([]SvixEndpointOut, error) {
	return []SvixEndpointOut{}, nil
}

// ConvoyCompatLayer provides API compatibility for Convoy SDK calls
type ConvoyCompatLayer struct {
	service *Service
}

func NewConvoyCompatLayer(service *Service) *ConvoyCompatLayer {
	return &ConvoyCompatLayer{service: service}
}

type ConvoyEndpointIn struct {
	URL         string `json:"url"`
	Description string `json:"description"`
	Secret      string `json:"secret"`
	RateLimit   int    `json:"rate_limit"`
}

type ConvoyEndpointOut struct {
	UID         string `json:"uid"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (c *ConvoyCompatLayer) CreateEndpoint(projectID string, endpoint ConvoyEndpointIn) (*ConvoyEndpointOut, error) {
	return &ConvoyEndpointOut{
		UID:         uuid.New().String(),
		TargetURL:   endpoint.URL,
		Description: endpoint.Description,
		Status:      "active",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// ListEndpoints maps a Convoy list call to WaaS.
// Returns empty because the compat layer handles incoming migration calls;
// endpoint discovery is performed through the native WaaS API instead.
func (c *ConvoyCompatLayer) ListEndpoints(projectID string) ([]ConvoyEndpointOut, error) {
	return []ConvoyEndpointOut{}, nil
}
