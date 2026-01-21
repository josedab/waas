package blockchain

import (
	"context"
	"errors"
	"math/big"
)

var (
	ErrMonitorNotFound     = errors.New("monitor not found")
	ErrEventNotFound       = errors.New("event not found")
	ErrProviderNotFound    = errors.New("RPC provider not found")
	ErrInvalidAddress      = errors.New("invalid contract address")
	ErrInvalidABI          = errors.New("invalid contract ABI")
	ErrChainNotSupported   = errors.New("chain not supported")
	ErrConnectionFailed    = errors.New("RPC connection failed")
	ErrRateLimitExceeded   = errors.New("RPC rate limit exceeded")
)

// Repository defines the interface for blockchain data storage
type Repository interface {
	// Monitors
	CreateMonitor(ctx context.Context, monitor *ContractMonitor) error
	GetMonitor(ctx context.Context, tenantID, monitorID string) (*ContractMonitor, error)
	UpdateMonitor(ctx context.Context, monitor *ContractMonitor) error
	DeleteMonitor(ctx context.Context, tenantID, monitorID string) error
	ListMonitors(ctx context.Context, tenantID string, filters *MonitorFilters) ([]ContractMonitor, int, error)
	ListActiveMonitors(ctx context.Context) ([]ContractMonitor, error)

	// Events
	CreateEvent(ctx context.Context, event *ContractEvent) error
	GetEvent(ctx context.Context, eventID string) (*ContractEvent, error)
	UpdateEvent(ctx context.Context, event *ContractEvent) error
	ListEvents(ctx context.Context, tenantID string, filters *EventFilters) ([]ContractEvent, int, error)
	GetUnconfirmedEvents(ctx context.Context, monitorID string, beforeBlock uint64) ([]ContractEvent, error)
	MarkEventsReorged(ctx context.Context, monitorID string, fromBlock uint64) error

	// Deliveries
	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	GetDelivery(ctx context.Context, deliveryID string) (*WebhookDelivery, error)
	UpdateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	ListDeliveriesByEvent(ctx context.Context, eventID string) ([]WebhookDelivery, error)
	GetPendingDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error)

	// RPC Providers
	CreateProvider(ctx context.Context, provider *RPCProvider) error
	GetProvider(ctx context.Context, providerID string) (*RPCProvider, error)
	UpdateProvider(ctx context.Context, provider *RPCProvider) error
	DeleteProvider(ctx context.Context, providerID string) error
	ListProviders(ctx context.Context, chain ChainType, network NetworkType) ([]RPCProvider, error)

	// ABIs
	SaveABI(ctx context.Context, abi *ContractABI) error
	GetABI(ctx context.Context, chain ChainType, contractAddress string) (*ContractABI, error)
	DeleteABI(ctx context.Context, chain ChainType, contractAddress string) error

	// Checkpoints
	SaveCheckpoint(ctx context.Context, monitorID string, blockNumber uint64) error
	GetCheckpoint(ctx context.Context, monitorID string) (uint64, error)
}

// ChainClient defines the interface for blockchain interaction
type ChainClient interface {
	// Connection
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetChain() ChainType
	GetNetwork() NetworkType

	// Block operations
	GetLatestBlock(ctx context.Context) (uint64, error)
	GetBlockByNumber(ctx context.Context, number uint64) (*Block, error)
	GetBlockTimestamp(ctx context.Context, number uint64) (uint64, error)

	// Log operations
	GetLogs(ctx context.Context, filter *LogFilter) ([]Log, error)
	SubscribeLogs(ctx context.Context, filter *LogFilter, ch chan<- Log) error
	UnsubscribeLogs(subscriptionID string) error

	// Contract operations
	GetCode(ctx context.Context, address string) ([]byte, error)
	CallContract(ctx context.Context, call *ContractCall) ([]byte, error)
}

// Block represents a blockchain block
type Block struct {
	Number       uint64    `json:"number"`
	Hash         string    `json:"hash"`
	ParentHash   string    `json:"parent_hash"`
	Timestamp    uint64    `json:"timestamp"`
	Transactions []string  `json:"transactions"`
}

// Log represents a blockchain event log
type Log struct {
	Address         string   `json:"address"`
	Topics          []string `json:"topics"`
	Data            string   `json:"data"`
	BlockNumber     uint64   `json:"block_number"`
	BlockHash       string   `json:"block_hash"`
	TransactionHash string   `json:"transaction_hash"`
	TransactionIdx  uint     `json:"transaction_idx"`
	LogIndex        uint     `json:"log_index"`
	Removed         bool     `json:"removed"` // True if removed due to reorg
}

// LogFilter for querying logs
type LogFilter struct {
	Addresses []string   `json:"addresses"`
	Topics    [][]string `json:"topics"` // Outer = position, inner = OR
	FromBlock uint64     `json:"from_block"`
	ToBlock   uint64     `json:"to_block"`
}

// ContractCall represents a contract call request
type ContractCall struct {
	To       string `json:"to"`
	Data     string `json:"data"`
	Block    string `json:"block,omitempty"` // Block number or "latest"
}

// ABIDecoder defines the interface for ABI decoding
type ABIDecoder interface {
	// ParseABI parses a JSON ABI string
	ParseABI(abiJSON string) error

	// GetEventSignature returns the topic0 for an event
	GetEventSignature(eventName string) (string, error)

	// DecodeLog decodes a log using the ABI
	DecodeLog(log *Log) (map[string]interface{}, error)

	// GetEventInputs returns the inputs for an event
	GetEventInputs(eventName string) ([]ABIInput, error)

	// GetEvents returns all events in the ABI
	GetEvents() []ABIEvent
}

// WebhookSender defines the interface for sending webhooks
type WebhookSender interface {
	Send(ctx context.Context, delivery *WebhookDelivery) error
}

// EventProcessor defines the interface for processing events
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event *ContractEvent, monitor *ContractMonitor) error
}

// BlockTracker defines the interface for tracking block progress
type BlockTracker interface {
	GetLatestProcessedBlock(monitorID string) (uint64, error)
	SetLatestProcessedBlock(monitorID string, block uint64) error
	HandleReorg(monitorID string, reorgBlock uint64) error
}

// BalanceChecker defines the interface for checking balances (useful for monitoring)
type BalanceChecker interface {
	GetBalance(ctx context.Context, address string) (*big.Int, error)
	GetTokenBalance(ctx context.Context, tokenAddress, holderAddress string) (*big.Int, error)
}

// TransactionTracker defines the interface for tracking specific transactions
type TransactionTracker interface {
	GetTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error)
	WaitForTransaction(ctx context.Context, txHash string, confirmations int) (*TransactionReceipt, error)
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash string `json:"transaction_hash"`
	BlockNumber     uint64 `json:"block_number"`
	BlockHash       string `json:"block_hash"`
	Status          uint64 `json:"status"` // 1 = success, 0 = failure
	GasUsed         uint64 `json:"gas_used"`
	Logs            []Log  `json:"logs"`
}
