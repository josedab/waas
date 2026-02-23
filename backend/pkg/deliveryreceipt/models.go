package deliveryreceipt

import (
	"encoding/json"
	"time"
)

// Receipt status constants
const (
	StatusPending     = "pending"
	StatusAccepted    = "accepted"
	StatusConfirmed   = "confirmed"
	StatusFailed      = "failed"
	StatusExpired     = "expired"
	StatusUnconfirmed = "unconfirmed"
)

// Processing status constants
const (
	ProcessingSuccess = "success"
	ProcessingPartial = "partial"
	ProcessingFailed  = "failed"
	ProcessingRetry   = "retry_requested"
)

// DeliveryReceipt tracks the processing confirmation for a webhook delivery.
type DeliveryReceipt struct {
	ID                string          `json:"id" db:"id"`
	TenantID          string          `json:"tenant_id" db:"tenant_id"`
	DeliveryID        string          `json:"delivery_id" db:"delivery_id"`
	WebhookID         string          `json:"webhook_id" db:"webhook_id"`
	EndpointID        string          `json:"endpoint_id" db:"endpoint_id"`
	ReceiptURL        string          `json:"receipt_url" db:"receipt_url"`
	Status            string          `json:"status" db:"status"`
	HTTPStatus        int             `json:"http_status" db:"http_status"`
	ProcessingStatus  string          `json:"processing_status,omitempty" db:"processing_status"`
	ProcessingDetails json.RawMessage `json:"processing_details,omitempty" db:"processing_details"`
	ConfirmWindowSec  int             `json:"confirm_window_sec" db:"confirm_window_sec"`
	ConfirmedAt       *time.Time      `json:"confirmed_at,omitempty" db:"confirmed_at"`
	ExpiresAt         time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// ReceiptStats holds aggregated receipt metrics.
type ReceiptStats struct {
	TotalReceipts    int64   `json:"total_receipts"`
	ConfirmedCount   int64   `json:"confirmed_count"`
	PendingCount     int64   `json:"pending_count"`
	FailedCount      int64   `json:"failed_count"`
	ExpiredCount     int64   `json:"expired_count"`
	ConfirmationRate float64 `json:"confirmation_rate"`
	AvgConfirmTimeMs int64   `json:"avg_confirm_time_ms"`
}

// Request DTOs

type CreateReceiptRequest struct {
	DeliveryID       string `json:"delivery_id" binding:"required"`
	WebhookID        string `json:"webhook_id" binding:"required"`
	EndpointID       string `json:"endpoint_id" binding:"required"`
	HTTPStatus       int    `json:"http_status" binding:"required"`
	ReceiptURL       string `json:"receipt_url"`
	ConfirmWindowSec int    `json:"confirm_window_sec"`
}

type ConfirmReceiptRequest struct {
	ProcessingStatus  string          `json:"processing_status" binding:"required"`
	ProcessingDetails json.RawMessage `json:"processing_details,omitempty"`
}
