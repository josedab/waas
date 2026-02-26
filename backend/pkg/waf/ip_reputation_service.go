package waf

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// CheckIPReputation checks the reputation of an IP address
func (s *Service) CheckIPReputation(ctx context.Context, ip string) (*IPReputation, error) {
	reputation, err := s.repo.GetIPReputation(ctx, ip)
	if errors.Is(err, ErrIPReputationNotFound) {
		return &IPReputation{
			IP:          ip,
			ThreatScore: 0,
			LastSeen:    time.Now(),
			ReportCount: 0,
			Blocked:     false,
		}, nil
	}
	return reputation, err
}

// ReportIP reports a malicious IP address
func (s *Service) ReportIP(ctx context.Context, req *ReportIPRequest) (*IPReputation, error) {
	existing, err := s.repo.GetIPReputation(ctx, req.IP)
	if errors.Is(err, ErrIPReputationNotFound) {
		existing = &IPReputation{
			IP:          req.IP,
			ThreatScore: 0,
			ReportCount: 0,
			Categories:  []string{},
		}
	} else if err != nil {
		return nil, err
	}

	existing.ReportCount++
	existing.LastSeen = time.Now()

	// Merge categories
	categorySet := make(map[string]bool)
	for _, cat := range existing.Categories {
		categorySet[cat] = true
	}
	for _, cat := range req.Categories {
		categorySet[cat] = true
	}
	existing.Categories = make([]string, 0, len(categorySet))
	for cat := range categorySet {
		existing.Categories = append(existing.Categories, cat)
	}

	// Update threat score based on report count
	existing.ThreatScore = float64(existing.ReportCount) * 10
	if existing.ThreatScore > 100 {
		existing.ThreatScore = 100
	}

	// Auto-block at high threat score
	if existing.ThreatScore >= 80 {
		existing.Blocked = true
	}

	if err := s.repo.UpsertIPReputation(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update IP reputation: %w", err)
	}

	return existing, nil
}

// ListBlockedIPs lists all blocked IP addresses
func (s *Service) ListBlockedIPs(ctx context.Context, limit, offset int) ([]IPReputation, int, error) {
	return s.repo.ListBlockedIPs(ctx, limit, offset)
}
