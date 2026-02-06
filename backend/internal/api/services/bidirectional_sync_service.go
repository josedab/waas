package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// BidirectionalSyncService handles bi-directional webhook sync
type BidirectionalSyncService struct {
	repo   repository.BidirectionalSyncRepository
	logger *utils.Logger
}

// NewBidirectionalSyncService creates a new bi-directional sync service
func NewBidirectionalSyncService(repo repository.BidirectionalSyncRepository, logger *utils.Logger) *BidirectionalSyncService {
	return &BidirectionalSyncService{
		repo:   repo,
		logger: logger,
	}
}

// CreateConfig creates a new sync configuration
func (s *BidirectionalSyncService) CreateConfig(ctx context.Context, tenantID uuid.UUID, req *models.CreateSyncConfigRequest) (*models.WebhookSyncConfig, error) {
	validModes := map[string]bool{
		models.SyncModeRequestResponse:    true,
		models.SyncModeEventAcknowledgment: true,
		models.SyncModeStateSync:          true,
	}

	if !validModes[req.SyncMode] {
		return nil, fmt.Errorf("invalid sync mode: %s", req.SyncMode)
	}

	config := &models.WebhookSyncConfig{
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		InboundEventType:  req.InboundEventType,
		SyncMode:          req.SyncMode,
		TimeoutSeconds:    req.TimeoutSeconds,
		RetryOnTimeout:    req.RetryOnTimeout,
		MaxRetries:        req.MaxRetries,
		CorrelationConfig: req.CorrelationConfig,
		Enabled:           true,
	}

	if req.OutboundEndpointID != "" {
		epID, err := uuid.Parse(req.OutboundEndpointID)
		if err == nil {
			config.OutboundEndpointID = &epID
		}
	}

	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if err := s.repo.CreateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return config, nil
}

// GetConfig retrieves a sync configuration
func (s *BidirectionalSyncService) GetConfig(ctx context.Context, tenantID, configID uuid.UUID) (*models.WebhookSyncConfig, error) {
	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, err
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	return config, nil
}

// GetConfigs retrieves all sync configurations for a tenant
func (s *BidirectionalSyncService) GetConfigs(ctx context.Context, tenantID uuid.UUID) ([]*models.WebhookSyncConfig, error) {
	return s.repo.GetConfigsByTenant(ctx, tenantID)
}

// SendSyncRequest sends a synchronous request and waits for response
func (s *BidirectionalSyncService) SendSyncRequest(ctx context.Context, tenantID uuid.UUID, req *models.SendSyncRequestRequest) (*models.SyncTransaction, error) {
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid config_id")
	}

	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil || config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	if !config.Enabled {
		return nil, fmt.Errorf("sync config is disabled")
	}

	// Generate correlation ID
	correlationID := s.generateCorrelationID(config, req.Payload)

	// Calculate timeout
	timeoutAt := time.Now().Add(time.Duration(config.TimeoutSeconds) * time.Second)

	tx := &models.SyncTransaction{
		TenantID:       tenantID,
		ConfigID:       configID,
		CorrelationID:  correlationID,
		State:          models.SyncStatesAwaitingResponse,
		RequestPayload: req.Payload,
		RequestSentAt:  time.Now(),
		TimeoutAt:      &timeoutAt,
		Metadata:       req.Metadata,
	}

	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// In production, would actually send the webhook here
	s.logger.Info("Sync request sent", map[string]interface{}{"transaction_id": tx.ID, "correlation_id": correlationID})

	return tx, nil
}

// generateCorrelationID generates a correlation ID for tracking
func (s *BidirectionalSyncService) generateCorrelationID(config *models.WebhookSyncConfig, payload map[string]interface{}) string {
	// Check if custom correlation field is configured
	if field, ok := config.CorrelationConfig["field"].(string); ok {
		if val, ok := payload[field]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	// Default to UUID
	return uuid.New().String()
}

// ReceiveSyncResponse processes an incoming sync response
func (s *BidirectionalSyncService) ReceiveSyncResponse(ctx context.Context, tenantID uuid.UUID, req *models.ReceiveSyncResponseRequest) (*models.SyncTransaction, error) {
	tx, err := s.repo.GetTransactionByCorrelation(ctx, req.CorrelationID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found")
	}

	if tx.TenantID != tenantID {
		return nil, fmt.Errorf("transaction not found")
	}

	if tx.State != models.SyncStatesAwaitingResponse {
		return nil, fmt.Errorf("transaction not awaiting response (current state: %s)", tx.State)
	}

	// Complete the transaction
	if err := s.repo.CompleteTransaction(ctx, tx.ID, req.Payload); err != nil {
		return nil, fmt.Errorf("failed to complete transaction: %w", err)
	}

	// Fetch updated transaction
	tx, err = s.repo.GetTransaction(ctx, tx.ID)
	if err != nil {
		s.logger.Error("Failed to fetch updated transaction", map[string]interface{}{"transaction_id": tx.ID, "error": err.Error()})
		return nil, fmt.Errorf("failed to fetch updated transaction: %w", err)
	}

	s.logger.Info("Sync response received", map[string]interface{}{"transaction_id": tx.ID, "correlation_id": req.CorrelationID})

	return tx, nil
}

// GetTransaction retrieves a sync transaction
func (s *BidirectionalSyncService) GetTransaction(ctx context.Context, tenantID, txID uuid.UUID) (*models.SyncTransaction, error) {
	tx, err := s.repo.GetTransaction(ctx, txID)
	if err != nil {
		return nil, err
	}

	if tx.TenantID != tenantID {
		return nil, fmt.Errorf("transaction not found")
	}

	return tx, nil
}

// GetTransactions retrieves transactions for a config
func (s *BidirectionalSyncService) GetTransactions(ctx context.Context, tenantID, configID uuid.UUID, limit int) ([]*models.SyncTransaction, error) {
	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil || config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	if limit <= 0 {
		limit = 20
	}

	return s.repo.GetTransactionsByConfig(ctx, configID, limit)
}

// SendAcknowledgment sends an acknowledgment for an event
func (s *BidirectionalSyncService) SendAcknowledgment(ctx context.Context, tenantID uuid.UUID, req *models.SendAcknowledgmentRequest) (*models.SyncAcknowledgment, error) {
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid config_id")
	}

	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return nil, fmt.Errorf("invalid event_id")
	}

	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil || config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	if config.SyncMode != models.SyncModeEventAcknowledgment {
		return nil, fmt.Errorf("config is not in event_acknowledgment mode")
	}

	validAckTypes := map[string]bool{
		models.AckTypeReceived:  true,
		models.AckTypeProcessed: true,
		models.AckTypeRejected:  true,
	}

	if !validAckTypes[req.AckType] {
		return nil, fmt.Errorf("invalid ack_type: %s", req.AckType)
	}

	correlationID := uuid.New().String()
	timeoutAt := time.Now().Add(time.Duration(config.TimeoutSeconds) * time.Second)

	ack := &models.SyncAcknowledgment{
		TenantID:      tenantID,
		ConfigID:      configID,
		EventID:       eventID,
		CorrelationID: correlationID,
		AckType:       req.AckType,
		AckPayload:    req.AckPayload,
		SentAt:        time.Now(),
		TimeoutAt:     &timeoutAt,
	}

	if err := s.repo.CreateAcknowledgment(ctx, ack); err != nil {
		return nil, fmt.Errorf("failed to create acknowledgment: %w", err)
	}

	s.logger.Info("Acknowledgment sent", map[string]interface{}{"ack_id": ack.ID, "event_id": eventID, "ack_type": req.AckType})

	return ack, nil
}

// ConfirmAcknowledgment confirms receipt of an acknowledgment
func (s *BidirectionalSyncService) ConfirmAcknowledgment(ctx context.Context, correlationID string) error {
	ack, err := s.repo.GetAcknowledgmentByCorrelation(ctx, correlationID)
	if err != nil {
		return fmt.Errorf("acknowledgment not found")
	}

	return s.repo.MarkAcknowledged(ctx, ack.ID)
}

// UpdateState updates local state for state sync mode
func (s *BidirectionalSyncService) UpdateState(ctx context.Context, tenantID uuid.UUID, req *models.UpdateStateRequest) (*models.SyncStateRecord, error) {
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid config_id")
	}

	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil || config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	if config.SyncMode != models.SyncModeStateSync {
		return nil, fmt.Errorf("config is not in state_sync mode")
	}

	record := &models.SyncStateRecord{
		TenantID:     tenantID,
		ConfigID:     configID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		LocalState:   req.State,
		SyncStatus:   models.SyncStatusPendingPush,
	}

	if err := s.repo.CreateStateRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	// Fetch the record
	return s.repo.GetStateRecordByResource(ctx, configID, req.ResourceType, req.ResourceID)
}

// ReceiveRemoteState receives state update from remote
func (s *BidirectionalSyncService) ReceiveRemoteState(ctx context.Context, tenantID, configID uuid.UUID, resourceType, resourceID string, remoteState map[string]interface{}) (*models.SyncStateRecord, error) {
	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil || config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	record, err := s.repo.GetStateRecordByResource(ctx, configID, resourceType, resourceID)
	if err != nil {
		// Create new record
		record = &models.SyncStateRecord{
			TenantID:     tenantID,
			ConfigID:     configID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			LocalState:   remoteState,
			RemoteState:  remoteState,
			SyncStatus:   models.SyncStatusSynced,
		}

		if err := s.repo.CreateStateRecord(ctx, record); err != nil {
			return nil, err
		}

		return s.repo.GetStateRecordByResource(ctx, configID, resourceType, resourceID)
	}

	// Check for conflict
	if s.hasConflict(record.LocalState, remoteState) {
		conflictData := map[string]interface{}{
			"local":  record.LocalState,
			"remote": remoteState,
		}
		s.repo.SetConflict(ctx, record.ID, conflictData)
		return s.repo.GetStateRecord(ctx, record.ID)
	}

	// Update remote state
	if err := s.repo.UpdateRemoteState(ctx, record.ID, remoteState); err != nil {
		return nil, err
	}

	return s.repo.GetStateRecord(ctx, record.ID)
}

// hasConflict checks if there's a conflict between local and remote state
func (s *BidirectionalSyncService) hasConflict(local, remote map[string]interface{}) bool {
	// Simple conflict detection - check if any common keys have different values
	for key, localVal := range local {
		if remoteVal, ok := remote[key]; ok {
			if fmt.Sprintf("%v", localVal) != fmt.Sprintf("%v", remoteVal) {
				return true
			}
		}
	}
	return false
}

// ResolveConflict resolves a state conflict
func (s *BidirectionalSyncService) ResolveConflict(ctx context.Context, tenantID uuid.UUID, req *models.ResolveConflictRequest) (*models.SyncStateRecord, error) {
	recordID, err := uuid.Parse(req.StateRecordID)
	if err != nil {
		return nil, fmt.Errorf("invalid state_record_id")
	}

	record, err := s.repo.GetStateRecord(ctx, recordID)
	if err != nil || record.TenantID != tenantID {
		return nil, fmt.Errorf("state record not found")
	}

	if record.SyncStatus != models.SyncStatusConflict {
		return nil, fmt.Errorf("no conflict to resolve")
	}

	validStrategies := map[string]bool{
		models.ConflictStrategyLocalWins:  true,
		models.ConflictStrategyRemoteWins: true,
		models.ConflictStrategyMerge:      true,
		models.ConflictStrategyManual:     true,
	}

	if !validStrategies[req.ResolutionStrategy] {
		return nil, fmt.Errorf("invalid resolution strategy: %s", req.ResolutionStrategy)
	}

	var resolvedState map[string]interface{}

	switch req.ResolutionStrategy {
	case models.ConflictStrategyLocalWins:
		resolvedState = record.LocalState
	case models.ConflictStrategyRemoteWins:
		resolvedState = record.RemoteState
	case models.ConflictStrategyMerge:
		resolvedState = s.mergeStates(record.LocalState, record.RemoteState)
	case models.ConflictStrategyManual:
		if req.ResolvedState == nil {
			return nil, fmt.Errorf("resolved_state required for manual resolution")
		}
		resolvedState = req.ResolvedState
	}

	// Record conflict history
	history := &models.SyncConflictHistory{
		TenantID:           tenantID,
		StateRecordID:      recordID,
		LocalState:         record.LocalState,
		RemoteState:        record.RemoteState,
		ResolutionStrategy: req.ResolutionStrategy,
		ResolvedState:      resolvedState,
	}

	s.repo.CreateConflictHistory(ctx, history)

	// Resolve the conflict
	if err := s.repo.ResolveConflict(ctx, recordID, resolvedState); err != nil {
		return nil, err
	}

	return s.repo.GetStateRecord(ctx, recordID)
}

// mergeStates merges two states (remote values override local)
func (s *BidirectionalSyncService) mergeStates(local, remote map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy all local values
	for k, v := range local {
		merged[k] = v
	}

	// Override with remote values
	for k, v := range remote {
		merged[k] = v
	}

	return merged
}

// GetConflicts retrieves conflicted state records
func (s *BidirectionalSyncService) GetConflicts(ctx context.Context, tenantID uuid.UUID) ([]*models.SyncStateRecord, error) {
	return s.repo.GetConflictedRecords(ctx, tenantID)
}

// ProcessTimeouts handles timed out transactions
func (s *BidirectionalSyncService) ProcessTimeouts(ctx context.Context) error {
	transactions, err := s.repo.GetTimedOutTransactions(ctx)
	if err != nil {
		return err
	}

	for _, tx := range transactions {
		config, err := s.repo.GetConfig(ctx, tx.ConfigID)
		if err != nil {
			continue
		}

		if config.RetryOnTimeout && tx.RetryCount < config.MaxRetries {
			// Retry
			s.repo.IncrementTransactionRetry(ctx, tx.ID)
			// Reset timeout
			newTimeout := time.Now().Add(time.Duration(config.TimeoutSeconds) * time.Second)
			tx.TimeoutAt = &newTimeout
			s.logger.Info("Retrying timed out transaction", map[string]interface{}{"transaction_id": tx.ID, "retry": tx.RetryCount+1})
		} else {
			// Mark as timeout
			s.repo.UpdateTransactionState(ctx, tx.ID, models.SyncStatesTimeout, "Transaction timed out")
			s.logger.Info("Transaction timed out", map[string]interface{}{"transaction_id": tx.ID})
		}
	}

	return nil
}

// GetDashboard retrieves the sync dashboard data
func (s *BidirectionalSyncService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*models.SyncDashboard, error) {
	dashboard := &models.SyncDashboard{}

	configs, _ := s.repo.GetConfigsByTenant(ctx, tenantID)
	activeConfigs := 0
	for _, c := range configs {
		if c.Enabled {
			activeConfigs++
		}
	}
	dashboard.ActiveConfigs = activeConfigs

	pendingTx, _ := s.repo.GetPendingTransactions(ctx, tenantID)
	dashboard.PendingTransactions = len(pendingTx)
	dashboard.RecentTransactions = pendingTx

	completedToday, _ := s.repo.CountTransactionsToday(ctx, tenantID, models.SyncStatesCompleted)
	timeoutsToday, _ := s.repo.CountTransactionsToday(ctx, tenantID, models.SyncStatesTimeout)
	dashboard.CompletedToday = completedToday
	dashboard.TimeoutsToday = timeoutsToday

	activeConflicts, _ := s.repo.CountActiveConflicts(ctx, tenantID)
	dashboard.ActiveConflicts = activeConflicts

	pendingAcks, _ := s.repo.CountPendingAcks(ctx, tenantID)
	dashboard.PendingAcks = pendingAcks

	conflicts, _ := s.repo.GetConflictedRecords(ctx, tenantID)
	dashboard.RecentConflicts = conflicts

	return dashboard, nil
}
