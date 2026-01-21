// Package blockchain provides smart contract event monitoring and webhook triggers
package blockchain

import (
	"encoding/json"
	"time"
)

// ChainType represents supported blockchain networks
type ChainType string

const (
	ChainEthereum ChainType = "ethereum"
	ChainPolygon  ChainType = "polygon"
	ChainArbitrum ChainType = "arbitrum"
	ChainOptimism ChainType = "optimism"
	ChainBase     ChainType = "base"
	ChainBSC      ChainType = "bsc"
	ChainAvalanche ChainType = "avalanche"
	ChainSolana   ChainType = "solana"
)

// EVMChains returns all EVM-compatible chains
func EVMChains() []ChainType {
	return []ChainType{
		ChainEthereum, ChainPolygon, ChainArbitrum, ChainOptimism,
		ChainBase, ChainBSC, ChainAvalanche,
	}
}

// NetworkType represents mainnet/testnet
type NetworkType string

const (
	NetworkMainnet NetworkType = "mainnet"
	NetworkTestnet NetworkType = "testnet"
	NetworkDevnet  NetworkType = "devnet"
)

// ContractMonitorStatus represents monitor lifecycle
type ContractMonitorStatus string

const (
	MonitorStatusActive   ContractMonitorStatus = "active"
	MonitorStatusPaused   ContractMonitorStatus = "paused"
	MonitorStatusError    ContractMonitorStatus = "error"
	MonitorStatusDisabled ContractMonitorStatus = "disabled"
)

// ContractMonitor represents a smart contract event listener
type ContractMonitor struct {
	ID              string                `json:"id"`
	TenantID        string                `json:"tenant_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Chain           ChainType             `json:"chain"`
	Network         NetworkType           `json:"network"`
	ContractAddress string                `json:"contract_address"`
	ABI             string                `json:"abi,omitempty"`
	Events          []EventFilter         `json:"events"`
	Status          ContractMonitorStatus `json:"status"`
	Config          *MonitorConfig        `json:"config"`
	WebhookConfigs  []WebhookConfig       `json:"webhook_configs"`
	Stats           *MonitorStats         `json:"stats,omitempty"`
	LastBlock       uint64                `json:"last_block"`
	LastEventAt     *time.Time            `json:"last_event_at,omitempty"`
	ErrorMessage    string                `json:"error_message,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// EventFilter defines which contract events to monitor
type EventFilter struct {
	EventName  string            `json:"event_name"`
	Topics     []string          `json:"topics,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Conditions []FilterCondition `json:"conditions,omitempty"`
}

// FilterCondition defines event parameter filtering
type FilterCondition struct {
	Parameter string `json:"parameter"`
	Operator  string `json:"operator"` // eq, ne, gt, lt, gte, lte, in, contains
	Value     string `json:"value"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	ConfirmationBlocks int           `json:"confirmation_blocks"` // Blocks to wait before triggering
	BatchSize          int           `json:"batch_size"`          // Events per webhook
	PollIntervalSec    int           `json:"poll_interval_sec"`   // Polling interval
	MaxRetries         int           `json:"max_retries"`
	BackfillFromBlock  uint64        `json:"backfill_from_block,omitempty"`
	EnableReorg        bool          `json:"enable_reorg"`        // Handle chain reorganizations
	ReorgDepth         int           `json:"reorg_depth"`         // Max reorg depth to handle
	Timeout            time.Duration `json:"timeout"`
}

// DefaultMonitorConfig returns sensible defaults
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		ConfirmationBlocks: 12,
		BatchSize:          100,
		PollIntervalSec:    12,
		MaxRetries:         3,
		EnableReorg:        true,
		ReorgDepth:         20,
		Timeout:            30 * time.Second,
	}
}

// WebhookConfig defines where to send events
type WebhookConfig struct {
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers,omitempty"`
	Transform  string            `json:"transform,omitempty"`
	EventTypes []string          `json:"event_types,omitempty"`
	Secret     string            `json:"-"` // For signing
}

// MonitorStats holds runtime statistics
type MonitorStats struct {
	TotalEvents       int64     `json:"total_events"`
	EventsLast24h     int64     `json:"events_last_24h"`
	WebhooksDelivered int64     `json:"webhooks_delivered"`
	WebhooksFailed    int64     `json:"webhooks_failed"`
	AverageLatencyMs  float64   `json:"average_latency_ms"`
	LastCheckedAt     time.Time `json:"last_checked_at"`
}

// ContractEvent represents a captured blockchain event
type ContractEvent struct {
	ID              string          `json:"id"`
	MonitorID       string          `json:"monitor_id"`
	TenantID        string          `json:"tenant_id"`
	Chain           ChainType       `json:"chain"`
	Network         NetworkType     `json:"network"`
	ContractAddress string          `json:"contract_address"`
	EventName       string          `json:"event_name"`
	BlockNumber     uint64          `json:"block_number"`
	BlockHash       string          `json:"block_hash"`
	TransactionHash string          `json:"transaction_hash"`
	TransactionIdx  uint            `json:"transaction_idx"`
	LogIndex        uint            `json:"log_index"`
	Topics          []string        `json:"topics"`
	Data            string          `json:"data"`
	DecodedData     json.RawMessage `json:"decoded_data,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
	Confirmed       bool            `json:"confirmed"`
	Reorged         bool            `json:"reorged"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// WebhookDelivery represents a webhook sent for a contract event
type WebhookDelivery struct {
	ID           string         `json:"id"`
	EventID      string         `json:"event_id"`
	MonitorID    string         `json:"monitor_id"`
	TenantID     string         `json:"tenant_id"`
	WebhookURL   string         `json:"webhook_url"`
	Status       DeliveryStatus `json:"status"`
	Payload      json.RawMessage `json:"payload"`
	Response     string         `json:"response,omitempty"`
	StatusCode   int            `json:"status_code,omitempty"`
	Attempts     int            `json:"attempts"`
	LatencyMs    int64          `json:"latency_ms"`
	ErrorMessage string         `json:"error_message,omitempty"`
	DeliveredAt  *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// DeliveryStatus represents webhook delivery status
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

// RPCProvider represents a blockchain RPC endpoint configuration
type RPCProvider struct {
	ID        string      `json:"id"`
	TenantID  string      `json:"tenant_id"`
	Name      string      `json:"name"`
	Chain     ChainType   `json:"chain"`
	Network   NetworkType `json:"network"`
	URL       string      `json:"url"`
	WSURL     string      `json:"ws_url,omitempty"`
	APIKey    string      `json:"-"`
	Priority  int         `json:"priority"`
	RateLimit int         `json:"rate_limit"`
	Healthy   bool        `json:"healthy"`
	CreatedAt time.Time   `json:"created_at"`
}

// ContractABI represents a parsed contract ABI
type ContractABI struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	ContractAddress string          `json:"contract_address"`
	Chain           ChainType       `json:"chain"`
	Name            string          `json:"name,omitempty"`
	ABI             json.RawMessage `json:"abi"`
	Events          []ABIEvent      `json:"events"`
	Verified        bool            `json:"verified"`
	Source          string          `json:"source,omitempty"` // etherscan, user, etc.
	CreatedAt       time.Time       `json:"created_at"`
}

// ABIEvent represents a decoded event signature from ABI
type ABIEvent struct {
	Name      string        `json:"name"`
	Signature string        `json:"signature"`
	Topic0    string        `json:"topic0"`
	Inputs    []ABIInput    `json:"inputs"`
}

// ABIInput represents an event input parameter
type ABIInput struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed"`
}

// ChainConfig holds per-chain configuration
type ChainConfig struct {
	Chain              ChainType   `json:"chain"`
	Network            NetworkType `json:"network"`
	ChainID            int64       `json:"chain_id"`
	BlockTime          int         `json:"block_time_sec"`
	ConfirmationBlocks int         `json:"confirmation_blocks"`
	ExplorerURL        string      `json:"explorer_url"`
	NativeCurrency     string      `json:"native_currency"`
}

// DefaultChainConfigs returns default configurations for supported chains
func DefaultChainConfigs() []ChainConfig {
	return []ChainConfig{
		{
			Chain: ChainEthereum, Network: NetworkMainnet, ChainID: 1,
			BlockTime: 12, ConfirmationBlocks: 12,
			ExplorerURL: "https://etherscan.io", NativeCurrency: "ETH",
		},
		{
			Chain: ChainPolygon, Network: NetworkMainnet, ChainID: 137,
			BlockTime: 2, ConfirmationBlocks: 32,
			ExplorerURL: "https://polygonscan.com", NativeCurrency: "MATIC",
		},
		{
			Chain: ChainArbitrum, Network: NetworkMainnet, ChainID: 42161,
			BlockTime: 1, ConfirmationBlocks: 1,
			ExplorerURL: "https://arbiscan.io", NativeCurrency: "ETH",
		},
		{
			Chain: ChainOptimism, Network: NetworkMainnet, ChainID: 10,
			BlockTime: 2, ConfirmationBlocks: 1,
			ExplorerURL: "https://optimistic.etherscan.io", NativeCurrency: "ETH",
		},
		{
			Chain: ChainBase, Network: NetworkMainnet, ChainID: 8453,
			BlockTime: 2, ConfirmationBlocks: 1,
			ExplorerURL: "https://basescan.org", NativeCurrency: "ETH",
		},
		{
			Chain: ChainBSC, Network: NetworkMainnet, ChainID: 56,
			BlockTime: 3, ConfirmationBlocks: 15,
			ExplorerURL: "https://bscscan.com", NativeCurrency: "BNB",
		},
		{
			Chain: ChainAvalanche, Network: NetworkMainnet, ChainID: 43114,
			BlockTime: 2, ConfirmationBlocks: 1,
			ExplorerURL: "https://snowtrace.io", NativeCurrency: "AVAX",
		},
	}
}

// CreateMonitorRequest represents a request to create a monitor
type CreateMonitorRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	Chain           ChainType       `json:"chain" binding:"required"`
	Network         NetworkType     `json:"network" binding:"required"`
	ContractAddress string          `json:"contract_address" binding:"required"`
	ABI             string          `json:"abi"`
	Events          []EventFilter   `json:"events" binding:"required"`
	Config          *MonitorConfig  `json:"config"`
	WebhookConfigs  []WebhookConfig `json:"webhook_configs" binding:"required"`
}

// UpdateMonitorRequest represents a request to update a monitor
type UpdateMonitorRequest struct {
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Events         []EventFilter          `json:"events,omitempty"`
	Config         *MonitorConfig         `json:"config,omitempty"`
	WebhookConfigs []WebhookConfig        `json:"webhook_configs,omitempty"`
	Status         *ContractMonitorStatus `json:"status,omitempty"`
}

// MonitorFilters for listing monitors
type MonitorFilters struct {
	Chain    *ChainType             `json:"chain,omitempty"`
	Network  *NetworkType           `json:"network,omitempty"`
	Status   *ContractMonitorStatus `json:"status,omitempty"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// EventFilters for listing events
type EventFilters struct {
	MonitorID   string    `json:"monitor_id,omitempty"`
	EventName   string    `json:"event_name,omitempty"`
	FromBlock   uint64    `json:"from_block,omitempty"`
	ToBlock     uint64    `json:"to_block,omitempty"`
	Confirmed   *bool     `json:"confirmed,omitempty"`
	Since       time.Time `json:"since,omitempty"`
	Page        int       `json:"page"`
	PageSize    int       `json:"page_size"`
}

// ListMonitorsResponse represents paginated monitors
type ListMonitorsResponse struct {
	Monitors   []ContractMonitor `json:"monitors"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// ListEventsResponse represents paginated events
type ListEventsResponse struct {
	Events     []ContractEvent `json:"events"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// WebhookPayload represents the webhook payload for a contract event
type WebhookPayload struct {
	Type            string          `json:"type"`
	Chain           ChainType       `json:"chain"`
	Network         NetworkType     `json:"network"`
	ContractAddress string          `json:"contract_address"`
	EventName       string          `json:"event_name"`
	BlockNumber     uint64          `json:"block_number"`
	BlockHash       string          `json:"block_hash"`
	TransactionHash string          `json:"transaction_hash"`
	LogIndex        uint            `json:"log_index"`
	Timestamp       time.Time       `json:"timestamp"`
	DecodedData     json.RawMessage `json:"decoded_data"`
	RawData         string          `json:"raw_data,omitempty"`
	Confirmed       bool            `json:"confirmed"`
	Confirmations   int             `json:"confirmations"`
}
