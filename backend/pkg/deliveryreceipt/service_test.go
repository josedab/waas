package deliveryreceipt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReceipt_202Accepted(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	receipt, err := svc.CreateReceipt(context.Background(), "tenant-1", &CreateReceiptRequest{
		DeliveryID: "del-1",
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		HTTPStatus: 202,
		ReceiptURL: "https://example.com/receipt/abc",
	})

	require.NoError(t, err)
	assert.Equal(t, StatusPending, receipt.Status)
	assert.Equal(t, 300, receipt.ConfirmWindowSec)
}

func TestCreateReceipt_200OK(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	receipt, err := svc.CreateReceipt(context.Background(), "tenant-1", &CreateReceiptRequest{
		DeliveryID: "del-2",
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		HTTPStatus: 200,
	})

	require.NoError(t, err)
	assert.Equal(t, StatusConfirmed, receipt.Status)
	assert.NotNil(t, receipt.ConfirmedAt)
}

func TestCreateReceipt_500Error(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	receipt, err := svc.CreateReceipt(context.Background(), "tenant-1", &CreateReceiptRequest{
		DeliveryID: "del-3",
		WebhookID:  "wh-1",
		EndpointID: "ep-1",
		HTTPStatus: 500,
	})

	require.NoError(t, err)
	assert.Equal(t, StatusFailed, receipt.Status)
}

func TestCreateReceipt_ExcessiveWindow(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CreateReceipt(context.Background(), "tenant-1", &CreateReceiptRequest{
		DeliveryID:       "del-4",
		WebhookID:        "wh-1",
		EndpointID:       "ep-1",
		HTTPStatus:       202,
		ConfirmWindowSec: 100000,
	})

	assert.Error(t, err)
}

func TestValidateProcessingStatus(t *testing.T) {
	t.Parallel()

	assert.NoError(t, validateProcessingStatus(ProcessingSuccess))
	assert.NoError(t, validateProcessingStatus(ProcessingFailed))
	assert.Error(t, validateProcessingStatus("invalid"))
}
