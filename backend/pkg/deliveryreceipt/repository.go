package deliveryreceipt

import "context"

// Repository defines the data access interface for delivery receipts.
type Repository interface {
	CreateReceipt(ctx context.Context, receipt *DeliveryReceipt) error
	GetReceipt(ctx context.Context, tenantID, receiptID string) (*DeliveryReceipt, error)
	GetReceiptByDelivery(ctx context.Context, tenantID, deliveryID string) (*DeliveryReceipt, error)
	ListReceipts(ctx context.Context, tenantID string, limit, offset int) ([]DeliveryReceipt, error)
	UpdateReceipt(ctx context.Context, receipt *DeliveryReceipt) error
	GetExpiredReceipts(ctx context.Context, limit int) ([]DeliveryReceipt, error)
	GetStats(ctx context.Context, tenantID string) (*ReceiptStats, error)
}
