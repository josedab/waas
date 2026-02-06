package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DeliveryPublisher defines the interface for publishing deliveries
type DeliveryPublisher interface {
	Publish(ctx context.Context, tenantID, endpointID string, payload []byte, headers map[string]string) (string, error)
}

// Service provides replay functionality
type Service struct {
	repo      Repository
	publisher DeliveryPublisher
}

// NewService creates a new replay service
func NewService(repo Repository, publisher DeliveryPublisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}

// ReplaySingle replays a single delivery
func (s *Service) ReplaySingle(ctx context.Context, tenantID string, req *ReplayRequest) (*ReplayResult, error) {
	// Get the original delivery
	archive, err := s.repo.GetDeliveryArchive(ctx, tenantID, req.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}
	if archive == nil {
		return nil, fmt.Errorf("delivery not found: %s", req.DeliveryID)
	}

	// Determine endpoint and payload
	endpointID := archive.EndpointID
	if req.EndpointID != "" {
		endpointID = req.EndpointID
	}

	payload := archive.Payload
	if req.ModifyPayload && req.Payload != nil {
		payload = req.Payload
	}

	// Publish the new delivery
	newDeliveryID, err := s.publisher.Publish(ctx, tenantID, endpointID, payload, archive.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to queue replay: %w", err)
	}

	return &ReplayResult{
		OriginalDeliveryID: req.DeliveryID,
		NewDeliveryID:      newDeliveryID,
		EndpointID:         endpointID,
		Status:             "queued",
		ReplayedAt:         time.Now(),
	}, nil
}

// ReplayBulk replays multiple deliveries
func (s *Service) ReplayBulk(ctx context.Context, tenantID string, req *BulkReplayRequest) (*BulkReplayResult, error) {
	result := &BulkReplayResult{
		DryRun: req.DryRun,
	}

	var archives []DeliveryArchive
	var err error

	if len(req.DeliveryIDs) > 0 {
		// Replay specific deliveries
		for _, id := range req.DeliveryIDs {
			archive, err := s.repo.GetDeliveryArchive(ctx, tenantID, id)
			if err != nil {
				result.Errors = append(result.Errors, ReplayError{
					DeliveryID: id,
					Error:      err.Error(),
				})
				continue
			}
			if archive != nil {
				archives = append(archives, *archive)
			}
		}
	} else {
		// Query deliveries based on filters
		archives, result.TotalFound, err = s.repo.ListDeliveryArchives(ctx, tenantID, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list deliveries: %w", err)
		}
	}

	result.TotalFound = len(archives)

	if req.DryRun {
		// Return preview without actually replaying
		for _, archive := range archives {
			result.Results = append(result.Results, ReplayResult{
				OriginalDeliveryID: archive.ID,
				EndpointID:         archive.EndpointID,
				Status:             "would_replay",
			})
		}
		return result, nil
	}

	// Rate limiting
	rateLimit := req.RateLimit
	if rateLimit <= 0 {
		rateLimit = 10 // Default 10 per second
	}
	ticker := time.NewTicker(time.Second / time.Duration(rateLimit))
	defer ticker.Stop()

	for _, archive := range archives {
		<-ticker.C // Rate limit

		newDeliveryID, err := s.publisher.Publish(ctx, tenantID, archive.EndpointID, archive.Payload, archive.Headers)
		if err != nil {
			result.TotalFailed++
			result.Errors = append(result.Errors, ReplayError{
				DeliveryID: archive.ID,
				Error:      err.Error(),
			})
			continue
		}

		result.TotalReplayed++
		result.Results = append(result.Results, ReplayResult{
			OriginalDeliveryID: archive.ID,
			NewDeliveryID:      newDeliveryID,
			EndpointID:         archive.EndpointID,
			Status:             "queued",
			ReplayedAt:         time.Now(),
		})
	}

	return result, nil
}

// CreateSnapshot creates a point-in-time snapshot of deliveries
func (s *Service) CreateSnapshot(ctx context.Context, tenantID string, req *CreateSnapshotRequest) (*Snapshot, error) {
	// Build filters for query
	filters := &BulkReplayRequest{
		EndpointID: req.Filters.EndpointID,
		Status:     req.Filters.Status,
		StartTime:  req.Filters.StartTime,
		EndTime:    req.Filters.EndTime,
		Limit:      10000, // Max deliveries per snapshot
	}

	// Get matching deliveries
	archives, _, err := s.repo.ListDeliveryArchives(ctx, tenantID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query deliveries: %w", err)
	}

	if len(archives) == 0 {
		return nil, fmt.Errorf("no deliveries match the specified filters")
	}

	// Extract delivery IDs
	deliveryIDs := make([]string, len(archives))
	for i, archive := range archives {
		deliveryIDs[i] = archive.ID
	}

	// Create snapshot
	filtersJSON, _ := json.Marshal(req.Filters)
	snapshot := &Snapshot{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Filters:     filtersJSON,
	}

	if req.TTLDays > 0 {
		snapshot.ExpiresAt = time.Now().AddDate(0, 0, req.TTLDays)
	}

	if err := s.repo.CreateSnapshot(ctx, snapshot, deliveryIDs); err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	snapshot.DeliveryIDs = deliveryIDs
	return snapshot, nil
}

// GetSnapshot retrieves a snapshot
func (s *Service) GetSnapshot(ctx context.Context, tenantID, snapshotID string) (*Snapshot, error) {
	snapshot, err := s.repo.GetSnapshot(ctx, tenantID, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found")
	}

	// Get delivery IDs
	ids, err := s.repo.GetSnapshotDeliveryIDs(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot deliveries: %w", err)
	}
	snapshot.DeliveryIDs = ids

	return snapshot, nil
}

// ListSnapshots lists all snapshots for a tenant
func (s *Service) ListSnapshots(ctx context.Context, tenantID string, limit, offset int) ([]Snapshot, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListSnapshots(ctx, tenantID, limit, offset)
}

// DeleteSnapshot deletes a snapshot
func (s *Service) DeleteSnapshot(ctx context.Context, tenantID, snapshotID string) error {
	return s.repo.DeleteSnapshot(ctx, tenantID, snapshotID)
}

// ReplayFromSnapshot replays deliveries from a snapshot
func (s *Service) ReplayFromSnapshot(ctx context.Context, tenantID string, req *ReplayFromSnapshotRequest) (*BulkReplayResult, error) {
	// Get snapshot
	snapshot, err := s.repo.GetSnapshot(ctx, tenantID, req.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found")
	}

	// Get delivery IDs
	deliveryIDs, err := s.repo.GetSnapshotDeliveryIDs(ctx, req.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot deliveries: %w", err)
	}

	// Apply limit
	limit := req.Limit
	if limit <= 0 || limit > len(deliveryIDs) {
		limit = len(deliveryIDs)
	}
	deliveryIDs = deliveryIDs[:limit]

	// Replay using bulk replay
	bulkReq := &BulkReplayRequest{
		DeliveryIDs: deliveryIDs,
		DryRun:      req.DryRun,
	}

	return s.ReplayBulk(ctx, tenantID, bulkReq)
}

// GetDeliveryForReplay retrieves a delivery with full payload for replay
func (s *Service) GetDeliveryForReplay(ctx context.Context, tenantID, deliveryID string) (*DeliveryArchive, error) {
	return s.repo.GetDeliveryArchive(ctx, tenantID, deliveryID)
}

// Cleanup removes expired snapshots
func (s *Service) Cleanup(ctx context.Context) (int64, error) {
	return s.repo.CleanupExpiredSnapshots(ctx)
}

// RunWhatIf simulates a delivery replay with modifications without actually sending
func (s *Service) RunWhatIf(ctx context.Context, tenantID string, req *WhatIfRequest) (*WhatIfResult, error) {
	archive, err := s.repo.GetDeliveryArchive(ctx, tenantID, req.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}
	if archive == nil {
		return nil, fmt.Errorf("delivery not found: %s", req.DeliveryID)
	}

	result := &WhatIfResult{
		OriginalDeliveryID: req.DeliveryID,
		Original: &WhatIfDelivery{
			EndpointID:  archive.EndpointID,
			EndpointURL: archive.EndpointURL,
			Payload:     archive.Payload,
			Headers:     archive.Headers,
			PayloadSize: len(archive.Payload),
		},
	}

	// Build simulated delivery
	simulatedPayload := archive.Payload
	if req.ModifiedPayload != nil {
		simulatedPayload = req.ModifiedPayload
	}

	simulatedHeaders := make(map[string]string)
	for k, v := range archive.Headers {
		simulatedHeaders[k] = v
	}
	for k, v := range req.ModifiedHeaders {
		simulatedHeaders[k] = v
	}

	simulatedEndpoint := archive.EndpointID
	simulatedURL := archive.EndpointURL
	if len(req.TargetEndpoints) > 0 {
		simulatedEndpoint = req.TargetEndpoints[0]
		simulatedURL = "endpoint:" + simulatedEndpoint
		for _, ep := range req.TargetEndpoints {
			result.EndpointChanges = append(result.EndpointChanges, ep)
		}
	}

	result.Simulated = &WhatIfDelivery{
		EndpointID:  simulatedEndpoint,
		EndpointURL: simulatedURL,
		Payload:     simulatedPayload,
		Headers:     simulatedHeaders,
		PayloadSize: len(simulatedPayload),
	}

	// Compute payload diff
	if req.ModifiedPayload != nil {
		result.PayloadDiff = computePayloadDiff(archive.Payload, req.ModifiedPayload)
	}

	// Generate analysis
	result.Analysis = generateWhatIfAnalysis(result)

	return result, nil
}

func computePayloadDiff(oldPayload, newPayload json.RawMessage) []PayloadDiffItem {
	var oldMap, newMap map[string]interface{}
	json.Unmarshal(oldPayload, &oldMap)
	json.Unmarshal(newPayload, &newMap)

	var diffs []PayloadDiffItem

	// Check for changed and removed fields
	for key, oldVal := range oldMap {
		newVal, exists := newMap[key]
		if !exists {
			diffs = append(diffs, PayloadDiffItem{
				Path: key, Type: "removed", OldValue: oldVal,
			})
		} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			diffs = append(diffs, PayloadDiffItem{
				Path: key, Type: "changed", OldValue: oldVal, NewValue: newVal,
			})
		}
	}

	// Check for added fields
	for key, newVal := range newMap {
		if _, exists := oldMap[key]; !exists {
			diffs = append(diffs, PayloadDiffItem{
				Path: key, Type: "added", NewValue: newVal,
			})
		}
	}

	return diffs
}

func generateWhatIfAnalysis(result *WhatIfResult) string {
	changes := len(result.PayloadDiff)
	endpoints := len(result.EndpointChanges)
	sizeDiff := result.Simulated.PayloadSize - result.Original.PayloadSize

	analysis := fmt.Sprintf("What-if simulation: %d payload field(s) changed", changes)
	if endpoints > 0 {
		analysis += fmt.Sprintf(", %d endpoint(s) retargeted", endpoints)
	}
	if sizeDiff != 0 {
		analysis += fmt.Sprintf(", payload size delta: %+d bytes", sizeDiff)
	}
	return analysis
}
