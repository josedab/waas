package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides blockchain monitoring operations
type Service struct {
	repo           Repository
	clients        map[string]ChainClient // key: chain-network
	decoder        ABIDecoder
	webhookSender  WebhookSender
	processor      EventProcessor
	chainConfigs   map[ChainType]ChainConfig
	mu             sync.RWMutex
	monitors       map[string]*monitorWorker
	shutdownCh     chan struct{}
	config         *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxMonitorsPerTenant int
	DefaultPollInterval  time.Duration
	MaxEventsPerBatch    int
	EnableAutoBackfill   bool
	MaxBackfillBlocks    uint64
	WorkerPoolSize       int
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxMonitorsPerTenant: 50,
		DefaultPollInterval:  12 * time.Second,
		MaxEventsPerBatch:    1000,
		EnableAutoBackfill:   true,
		MaxBackfillBlocks:    10000,
		WorkerPoolSize:       10,
	}
}

// monitorWorker manages a single monitor's event processing
type monitorWorker struct {
	monitor    *ContractMonitor
	client     ChainClient
	stopCh     chan struct{}
	running    bool
	lastBlock  uint64
	mu         sync.Mutex
}

// NewService creates a new blockchain service
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}

	chainConfigs := make(map[ChainType]ChainConfig)
	for _, cfg := range DefaultChainConfigs() {
		chainConfigs[cfg.Chain] = cfg
	}

	return &Service{
		repo:         repo,
		clients:      make(map[string]ChainClient),
		chainConfigs: chainConfigs,
		monitors:     make(map[string]*monitorWorker),
		shutdownCh:   make(chan struct{}),
		config:       config,
	}
}

// RegisterClient registers a chain client
func (s *Service) RegisterClient(client ChainClient) {
	key := fmt.Sprintf("%s-%s", client.GetChain(), client.GetNetwork())
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[key] = client
}

// SetDecoder sets the ABI decoder
func (s *Service) SetDecoder(decoder ABIDecoder) {
	s.decoder = decoder
}

// SetWebhookSender sets the webhook sender
func (s *Service) SetWebhookSender(sender WebhookSender) {
	s.webhookSender = sender
}

// SetProcessor sets the event processor
func (s *Service) SetProcessor(processor EventProcessor) {
	s.processor = processor
}

// getClient returns a client for a chain/network
func (s *Service) getClient(chain ChainType, network NetworkType) (ChainClient, error) {
	key := fmt.Sprintf("%s-%s", chain, network)
	s.mu.RLock()
	client, ok := s.clients[key]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrChainNotSupported, key)
	}
	return client, nil
}

// CreateMonitor creates a new contract monitor
func (s *Service) CreateMonitor(ctx context.Context, tenantID string, req *CreateMonitorRequest) (*ContractMonitor, error) {
	// Validate chain
	if _, ok := s.chainConfigs[req.Chain]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrChainNotSupported, req.Chain)
	}

	// Validate address
	if !isValidAddress(req.ContractAddress) {
		return nil, ErrInvalidAddress
	}

	// Check monitor limit
	existing, _, err := s.repo.ListMonitors(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing monitors: %w", err)
	}
	if len(existing) >= s.config.MaxMonitorsPerTenant {
		return nil, fmt.Errorf("maximum monitors reached: %d", s.config.MaxMonitorsPerTenant)
	}

	// Set defaults
	config := req.Config
	if config == nil {
		config = DefaultMonitorConfig()
	}

	now := time.Now()
	monitor := &ContractMonitor{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		Chain:           req.Chain,
		Network:         req.Network,
		ContractAddress: normalizeAddress(req.ContractAddress),
		ABI:             req.ABI,
		Events:          req.Events,
		Status:          MonitorStatusActive,
		Config:          config,
		WebhookConfigs:  req.WebhookConfigs,
		Stats:           &MonitorStats{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Get starting block if not specified
	if config.BackfillFromBlock == 0 {
		client, err := s.getClient(req.Chain, req.Network)
		if err == nil {
			if latestBlock, err := client.GetLatestBlock(ctx); err == nil {
				monitor.LastBlock = latestBlock
			}
		}
	} else {
		monitor.LastBlock = config.BackfillFromBlock
	}

	// Parse and save ABI if provided
	if req.ABI != "" && s.decoder != nil {
		if err := s.decoder.ParseABI(req.ABI); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidABI, err)
		}

		contractABI := &ContractABI{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			ContractAddress: monitor.ContractAddress,
			Chain:           req.Chain,
			ABI:             json.RawMessage(req.ABI),
			Events:          s.decoder.GetEvents(),
			Verified:        false,
			Source:          "user",
			CreatedAt:       now,
		}
		if err := s.repo.SaveABI(ctx, contractABI); err != nil {
			stdlog.Printf("failed to save contract ABI: %v", err)
		}
	}

	// Save monitor
	if err := s.repo.CreateMonitor(ctx, monitor); err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	// Start monitoring
	go s.startMonitorWorker(monitor)

	return monitor, nil
}

// GetMonitor retrieves a monitor
func (s *Service) GetMonitor(ctx context.Context, tenantID, monitorID string) (*ContractMonitor, error) {
	return s.repo.GetMonitor(ctx, tenantID, monitorID)
}

// UpdateMonitor updates a monitor
func (s *Service) UpdateMonitor(ctx context.Context, tenantID, monitorID string, req *UpdateMonitorRequest) (*ContractMonitor, error) {
	monitor, err := s.repo.GetMonitor(ctx, tenantID, monitorID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		monitor.Name = *req.Name
	}
	if req.Description != nil {
		monitor.Description = *req.Description
	}
	if req.Events != nil {
		monitor.Events = req.Events
	}
	if req.Config != nil {
		monitor.Config = req.Config
	}
	if req.WebhookConfigs != nil {
		monitor.WebhookConfigs = req.WebhookConfigs
	}
	if req.Status != nil {
		oldStatus := monitor.Status
		monitor.Status = *req.Status

		// Handle status changes
		if oldStatus != *req.Status {
			s.handleStatusChange(monitor, oldStatus, *req.Status)
		}
	}

	monitor.UpdatedAt = time.Now()

	if err := s.repo.UpdateMonitor(ctx, monitor); err != nil {
		return nil, err
	}

	return monitor, nil
}

// handleStatusChange handles monitor status transitions
func (s *Service) handleStatusChange(monitor *ContractMonitor, oldStatus, newStatus ContractMonitorStatus) {
	s.mu.Lock()
	worker, exists := s.monitors[monitor.ID]
	s.mu.Unlock()

	if newStatus == MonitorStatusActive && (oldStatus == MonitorStatusPaused || oldStatus == MonitorStatusDisabled) {
		if !exists || !worker.running {
			go s.startMonitorWorker(monitor)
		}
	} else if (newStatus == MonitorStatusPaused || newStatus == MonitorStatusDisabled) && exists && worker.running {
		s.stopMonitorWorker(monitor.ID)
	}
}

// DeleteMonitor deletes a monitor
func (s *Service) DeleteMonitor(ctx context.Context, tenantID, monitorID string) error {
	// Stop worker
	s.stopMonitorWorker(monitorID)

	return s.repo.DeleteMonitor(ctx, tenantID, monitorID)
}

// ListMonitors lists monitors
func (s *Service) ListMonitors(ctx context.Context, tenantID string, filters *MonitorFilters) (*ListMonitorsResponse, error) {
	monitors, total, err := s.repo.ListMonitors(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := 20
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	return &ListMonitorsResponse{
		Monitors:   monitors,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetEvents retrieves events for a monitor
func (s *Service) GetEvents(ctx context.Context, tenantID string, filters *EventFilters) (*ListEventsResponse, error) {
	events, total, err := s.repo.ListEvents(ctx, tenantID, filters)
	if err != nil {
		return nil, err
	}

	page := 1
	pageSize := 50
	if filters != nil {
		if filters.Page > 0 {
			page = filters.Page
		}
		if filters.PageSize > 0 {
			pageSize = filters.PageSize
		}
	}

	return &ListEventsResponse{
		Events:     events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetMonitorStats retrieves monitor statistics
func (s *Service) GetMonitorStats(ctx context.Context, tenantID, monitorID string) (*MonitorStats, error) {
	monitor, err := s.repo.GetMonitor(ctx, tenantID, monitorID)
	if err != nil {
		return nil, err
	}
	return monitor.Stats, nil
}

// startMonitorWorker starts event monitoring for a contract
func (s *Service) startMonitorWorker(monitor *ContractMonitor) {
	client, err := s.getClient(monitor.Chain, monitor.Network)
	if err != nil {
		monitor.Status = MonitorStatusError
		monitor.ErrorMessage = err.Error()
		if err := s.repo.UpdateMonitor(context.Background(), monitor); err != nil {
			stdlog.Printf("failed to update monitor status to error: %v", err)
		}
		return
	}

	worker := &monitorWorker{
		monitor:   monitor,
		client:    client,
		stopCh:    make(chan struct{}),
		running:   true,
		lastBlock: monitor.LastBlock,
	}

	s.mu.Lock()
	s.monitors[monitor.ID] = worker
	s.mu.Unlock()

	go worker.run(s)
}

// stopMonitorWorker stops a monitor worker
func (s *Service) stopMonitorWorker(monitorID string) {
	s.mu.Lock()
	worker, exists := s.monitors[monitorID]
	if exists {
		close(worker.stopCh)
		worker.running = false
		delete(s.monitors, monitorID)
	}
	s.mu.Unlock()
}

// run executes the monitoring loop
func (w *monitorWorker) run(s *Service) {
	pollInterval := time.Duration(w.monitor.Config.PollIntervalSec) * time.Second
	if pollInterval == 0 {
		pollInterval = s.config.DefaultPollInterval
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-s.shutdownCh:
			return
		case <-ticker.C:
			w.processBlocks(s)
		}
	}
}

// processBlocks processes new blocks for events
func (w *monitorWorker) processBlocks(s *Service) {
	ctx := context.Background()

	// Get latest block
	latestBlock, err := w.client.GetLatestBlock(ctx)
	if err != nil {
		return
	}

	// Calculate safe block (with confirmations)
	safeBlock := latestBlock
	if w.monitor.Config.ConfirmationBlocks > 0 {
		confirmations := uint64(w.monitor.Config.ConfirmationBlocks)
		if latestBlock > confirmations {
			safeBlock = latestBlock - confirmations
		}
	}

	w.mu.Lock()
	fromBlock := w.lastBlock + 1
	w.mu.Unlock()

	if fromBlock > safeBlock {
		return // Already caught up
	}

	// Limit batch size
	toBlock := safeBlock
	batchSize := uint64(s.config.MaxEventsPerBatch)
	if toBlock-fromBlock > batchSize {
		toBlock = fromBlock + batchSize
	}

	// Build topic filters from event configurations
	topics := buildTopicFilters(w.monitor.Events, s.decoder)

	// Query logs
	filter := &LogFilter{
		Addresses: []string{w.monitor.ContractAddress},
		Topics:    topics,
		FromBlock: fromBlock,
		ToBlock:   toBlock,
	}

	logs, err := w.client.GetLogs(ctx, filter)
	if err != nil {
		return
	}

	// Process logs
	for _, log := range logs {
		event := s.logToEvent(log, w.monitor)
		if event == nil {
			continue
		}

		// Check if confirmed
		if log.BlockNumber <= safeBlock {
			event.Confirmed = true
		}

		// Save event
		if err := s.repo.CreateEvent(ctx, event); err != nil {
			continue
		}

		// Process event (send webhooks)
		if s.processor != nil {
			if err := s.processor.ProcessEvent(ctx, event, w.monitor); err != nil {
				stdlog.Printf("failed to process blockchain event %s: %v", event.ID, err)
			}
		} else {
			s.sendWebhooks(ctx, event, w.monitor)
		}

		// Update stats
		w.monitor.Stats.TotalEvents++
	}

	// Update checkpoint
	w.mu.Lock()
	w.lastBlock = toBlock
	w.mu.Unlock()

	w.monitor.LastBlock = toBlock
	now := time.Now()
	w.monitor.Stats.LastCheckedAt = now
	if len(logs) > 0 {
		w.monitor.LastEventAt = &now
	}
	if err := s.repo.UpdateMonitor(ctx, w.monitor); err != nil {
		stdlog.Printf("failed to update monitor checkpoint: %v", err)
	}
	if err := s.repo.SaveCheckpoint(ctx, w.monitor.ID, toBlock); err != nil {
		stdlog.Printf("failed to save block checkpoint: %v", err)
	}
}

// buildTopicFilters builds EVM topic filters from event configurations
func buildTopicFilters(events []EventFilter, decoder ABIDecoder) [][]string {
	if len(events) == 0 || decoder == nil {
		return nil
	}

	var topic0s []string
	for _, event := range events {
		sig, err := decoder.GetEventSignature(event.EventName)
		if err == nil {
			topic0s = append(topic0s, sig)
		}
	}

	if len(topic0s) == 0 {
		return nil
	}

	return [][]string{topic0s}
}

// logToEvent converts a blockchain log to a ContractEvent
func (s *Service) logToEvent(log Log, monitor *ContractMonitor) *ContractEvent {
	// Decode event data if decoder available
	var decodedData json.RawMessage
	if s.decoder != nil {
		if decoded, err := s.decoder.DecodeLog(&log); err == nil {
			decodedData, _ = json.Marshal(decoded)
		}
	}

	// Determine event name from topic0
	eventName := "Unknown"
	if len(log.Topics) > 0 && s.decoder != nil {
		for _, eventFilter := range monitor.Events {
			if sig, err := s.decoder.GetEventSignature(eventFilter.EventName); err == nil && sig == log.Topics[0] {
				eventName = eventFilter.EventName
				break
			}
		}
	}

	return &ContractEvent{
		ID:              uuid.New().String(),
		MonitorID:       monitor.ID,
		TenantID:        monitor.TenantID,
		Chain:           monitor.Chain,
		Network:         monitor.Network,
		ContractAddress: log.Address,
		EventName:       eventName,
		BlockNumber:     log.BlockNumber,
		BlockHash:       log.BlockHash,
		TransactionHash: log.TransactionHash,
		TransactionIdx:  log.TransactionIdx,
		LogIndex:        log.LogIndex,
		Topics:          log.Topics,
		Data:            log.Data,
		DecodedData:     decodedData,
		Confirmed:       false,
		Reorged:         log.Removed,
		CreatedAt:       time.Now(),
	}
}

// sendWebhooks sends webhooks for an event
func (s *Service) sendWebhooks(ctx context.Context, event *ContractEvent, monitor *ContractMonitor) {
	for _, webhookConfig := range monitor.WebhookConfigs {
		// Check if this webhook should receive this event type
		if len(webhookConfig.EventTypes) > 0 {
			found := false
			for _, et := range webhookConfig.EventTypes {
				if et == event.EventName || et == "*" {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Build payload
		payload := WebhookPayload{
			Type:            "blockchain.event",
			Chain:           event.Chain,
			Network:         event.Network,
			ContractAddress: event.ContractAddress,
			EventName:       event.EventName,
			BlockNumber:     event.BlockNumber,
			BlockHash:       event.BlockHash,
			TransactionHash: event.TransactionHash,
			LogIndex:        event.LogIndex,
			Timestamp:       event.CreatedAt,
			DecodedData:     event.DecodedData,
			RawData:         event.Data,
			Confirmed:       event.Confirmed,
			Confirmations:   monitor.Config.ConfirmationBlocks,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			stdlog.Printf("failed to marshal webhook payload for event %s: %v", event.ID, err)
			continue
		}

		// Create delivery record
		delivery := &WebhookDelivery{
			ID:         uuid.New().String(),
			EventID:    event.ID,
			MonitorID:  monitor.ID,
			TenantID:   monitor.TenantID,
			WebhookURL: webhookConfig.URL,
			Status:     DeliveryStatusPending,
			Payload:    payloadBytes,
			Attempts:   0,
			CreatedAt:  time.Now(),
		}

		if err := s.repo.CreateDelivery(ctx, delivery); err != nil {
			stdlog.Printf("failed to create webhook delivery record: %v", err)
		}

		// Send webhook
		if s.webhookSender != nil {
			if err := s.webhookSender.Send(ctx, delivery); err != nil {
				delivery.Status = DeliveryStatusFailed
				delivery.ErrorMessage = err.Error()
				monitor.Stats.WebhooksFailed++
			} else {
				delivery.Status = DeliveryStatusDelivered
				now := time.Now()
				delivery.DeliveredAt = &now
				monitor.Stats.WebhooksDelivered++
			}
			if err := s.repo.UpdateDelivery(ctx, delivery); err != nil {
				stdlog.Printf("failed to update webhook delivery status: %v", err)
			}
		}
	}
}

// Shutdown gracefully shuts down the service
func (s *Service) Shutdown() {
	close(s.shutdownCh)

	s.mu.Lock()
	for id, worker := range s.monitors {
		if worker.running {
			close(worker.stopCh)
			worker.running = false
		}
		delete(s.monitors, id)
	}
	s.mu.Unlock()
}

// AddRPCProvider adds an RPC provider
func (s *Service) AddRPCProvider(ctx context.Context, tenantID string, provider *RPCProvider) error {
	provider.ID = uuid.New().String()
	provider.TenantID = tenantID
	provider.CreatedAt = time.Now()
	return s.repo.CreateProvider(ctx, provider)
}

// ListChains returns supported chains
func (s *Service) ListChains() []ChainConfig {
	return DefaultChainConfigs()
}

// Helper functions

func isValidAddress(address string) bool {
	// Basic Ethereum address validation
	if !strings.HasPrefix(address, "0x") {
		return false
	}
	if len(address) != 42 {
		return false
	}
	return true
}

func normalizeAddress(address string) string {
	return strings.ToLower(address)
}
