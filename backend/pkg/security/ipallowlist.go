package security

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// IPAllowlistEntry represents an allowed IP or CIDR range for an endpoint
type IPAllowlistEntry struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	EndpointID string    `json:"endpoint_id" db:"endpoint_id"`
	CIDR       string    `json:"cidr" db:"cidr"`
	Label      string    `json:"label" db:"label"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// IPAllowlistConfig represents allowlist configuration for an endpoint
type IPAllowlistConfig struct {
	EndpointID string             `json:"endpoint_id"`
	Enabled    bool               `json:"enabled"`
	Mode       string             `json:"mode"` // "allowlist" or "denylist"
	Entries    []IPAllowlistEntry `json:"entries"`
}

// IPAllowlistService manages IP allowlists for webhook endpoints
type IPAllowlistService struct {
	mu    sync.RWMutex
	lists map[string]*IPAllowlistConfig // keyed by endpoint_id
}

// NewIPAllowlistService creates a new IP allowlist service
func NewIPAllowlistService() *IPAllowlistService {
	return &IPAllowlistService{
		lists: make(map[string]*IPAllowlistConfig),
	}
}

// SetConfig sets the allowlist configuration for an endpoint
func (s *IPAllowlistService) SetConfig(ctx context.Context, config *IPAllowlistConfig) error {
	if config.EndpointID == "" {
		return fmt.Errorf("endpoint_id is required")
	}

	// Validate all CIDRs
	for _, entry := range config.Entries {
		if !isValidCIDROrIP(entry.CIDR) {
			return fmt.Errorf("invalid CIDR or IP: %s", entry.CIDR)
		}
	}

	s.mu.Lock()
	s.lists[config.EndpointID] = config
	s.mu.Unlock()
	return nil
}

// GetConfig returns the allowlist configuration for an endpoint
func (s *IPAllowlistService) GetConfig(ctx context.Context, endpointID string) (*IPAllowlistConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, ok := s.lists[endpointID]
	if !ok {
		return &IPAllowlistConfig{
			EndpointID: endpointID,
			Enabled:    false,
			Mode:       "allowlist",
			Entries:    []IPAllowlistEntry{},
		}, nil
	}
	return config, nil
}

// AddEntry adds an IP or CIDR to the allowlist
func (s *IPAllowlistService) AddEntry(ctx context.Context, tenantID, endpointID, cidr, label string) (*IPAllowlistEntry, error) {
	if !isValidCIDROrIP(cidr) {
		return nil, fmt.Errorf("invalid CIDR or IP: %s", cidr)
	}

	// Normalize single IPs to CIDR
	if !strings.Contains(cidr, "/") {
		if strings.Contains(cidr, ":") {
			cidr = cidr + "/128"
		} else {
			cidr = cidr + "/32"
		}
	}

	entry := &IPAllowlistEntry{
		ID:         fmt.Sprintf("ipal_%d", time.Now().UnixNano()),
		TenantID:   tenantID,
		EndpointID: endpointID,
		CIDR:       cidr,
		Label:      label,
		CreatedAt:  time.Now(),
	}

	s.mu.Lock()
	config, ok := s.lists[endpointID]
	if !ok {
		config = &IPAllowlistConfig{
			EndpointID: endpointID,
			Enabled:    true,
			Mode:       "allowlist",
			Entries:    []IPAllowlistEntry{},
		}
		s.lists[endpointID] = config
	}
	config.Entries = append(config.Entries, *entry)
	s.mu.Unlock()

	return entry, nil
}

// RemoveEntry removes an entry from the allowlist
func (s *IPAllowlistService) RemoveEntry(ctx context.Context, endpointID, entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, ok := s.lists[endpointID]
	if !ok {
		return fmt.Errorf("endpoint not found")
	}

	for i, entry := range config.Entries {
		if entry.ID == entryID {
			config.Entries = append(config.Entries[:i], config.Entries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("entry not found")
}

// CheckIP verifies if an IP is allowed for an endpoint
func (s *IPAllowlistService) CheckIP(ctx context.Context, endpointID, clientIP string) (bool, error) {
	s.mu.RLock()
	config, ok := s.lists[endpointID]
	s.mu.RUnlock()

	if !ok || !config.Enabled {
		return true, nil // No allowlist = all allowed
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", clientIP)
	}

	for _, entry := range config.Entries {
		_, network, err := net.ParseCIDR(entry.CIDR)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			if config.Mode == "denylist" {
				return false, nil
			}
			return true, nil
		}
	}

	// If allowlist mode and no match, deny
	if config.Mode == "allowlist" {
		return false, nil
	}
	// If denylist mode and no match, allow
	return true, nil
}

func isValidCIDROrIP(s string) bool {
	if strings.Contains(s, "/") {
		_, _, err := net.ParseCIDR(s)
		return err == nil
	}
	return net.ParseIP(s) != nil
}
