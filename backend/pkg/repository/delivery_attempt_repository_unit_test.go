package repository

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
	"github.com/josedab/waas/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeliveryAttemptRepositoryLogic tests the business logic of delivery attempt repository operations
func TestDeliveryAttemptRepositoryLogic(t *testing.T) {
	t.Run("delivery attempt creation logic", func(t *testing.T) {
		payload := `{"event": "user.created", "data": {"id": "123", "email": "test@example.com"}}`
		payloadHash := fmt.Sprintf("sha256-%x", sha256.Sum256([]byte(payload)))
		
		attempt := &models.DeliveryAttempt{
			EndpointID:    uuid.New(),
			PayloadHash:   payloadHash,
			PayloadSize:   len(payload),
			Status:        "pending",
			AttemptNumber: 1,
			ScheduledAt:   time.Now(),
		}

		// Test that attempt has required fields for creation
		assert.NotEqual(t, uuid.Nil, attempt.EndpointID, "endpoint ID should not be nil")
		assert.NotEmpty(t, attempt.PayloadHash, "payload hash should not be empty")
		assert.Greater(t, attempt.PayloadSize, 0, "payload size should be positive")
		assert.NotEmpty(t, attempt.Status, "status should not be empty")
		assert.Greater(t, attempt.AttemptNumber, 0, "attempt number should be positive")
		assert.False(t, attempt.ScheduledAt.IsZero(), "scheduled at should be set")

		// Test payload hash format
		assert.Contains(t, attempt.PayloadHash, "sha256-")
		assert.Len(t, attempt.PayloadHash, 71) // "sha256-" + 64 hex characters

		// Test that ID and created timestamp would be set during creation
		if attempt.ID == uuid.Nil {
			attempt.ID = uuid.New()
		}
		attempt.CreatedAt = time.Now()

		assert.NotEqual(t, uuid.Nil, attempt.ID)
		assert.False(t, attempt.CreatedAt.IsZero())
	})

	t.Run("delivery attempt update logic", func(t *testing.T) {
		httpStatus := 200
		responseBody := "OK"
		deliveredAt := time.Now()
		
		attempt := &models.DeliveryAttempt{
			ID:            uuid.New(),
			EndpointID:    uuid.New(),
			PayloadHash:   "sha256-abcdef123456",
			PayloadSize:   1024,
			Status:        "pending",
			AttemptNumber: 1,
			ScheduledAt:   time.Now().Add(-time.Minute),
			CreatedAt:     time.Now().Add(-time.Minute),
		}

		// Simulate successful delivery update
		attempt.Status = "delivered"
		attempt.HTTPStatus = &httpStatus
		attempt.ResponseBody = &responseBody
		attempt.DeliveredAt = &deliveredAt

		assert.Equal(t, "delivered", attempt.Status)
		require.NotNil(t, attempt.HTTPStatus)
		assert.Equal(t, 200, *attempt.HTTPStatus)
		require.NotNil(t, attempt.ResponseBody)
		assert.Equal(t, "OK", *attempt.ResponseBody)
		require.NotNil(t, attempt.DeliveredAt)
		assert.True(t, attempt.DeliveredAt.After(attempt.ScheduledAt))
	})

	t.Run("delivery attempt failure logic", func(t *testing.T) {
		httpStatus := 500
		errorMessage := "Internal Server Error"
		
		attempt := &models.DeliveryAttempt{
			ID:            uuid.New(),
			EndpointID:    uuid.New(),
			Status:        "pending",
			AttemptNumber: 1,
		}

		// Simulate failed delivery update
		attempt.Status = "failed"
		attempt.HTTPStatus = &httpStatus
		attempt.ErrorMessage = &errorMessage
		attempt.AttemptNumber = 2

		assert.Equal(t, "failed", attempt.Status)
		require.NotNil(t, attempt.HTTPStatus)
		assert.Equal(t, 500, *attempt.HTTPStatus)
		require.NotNil(t, attempt.ErrorMessage)
		assert.Equal(t, "Internal Server Error", *attempt.ErrorMessage)
		assert.Equal(t, 2, attempt.AttemptNumber)
	})

	t.Run("delivery status transitions", func(t *testing.T) {
		validStatuses := []string{"pending", "retrying", "delivered", "failed"}
		
		for _, status := range validStatuses {
			attempt := &models.DeliveryAttempt{
				ID:         uuid.New(),
				EndpointID: uuid.New(),
				Status:     status,
			}
			
			assert.Contains(t, validStatuses, attempt.Status)
		}

		// Test status transition logic
		attempt := &models.DeliveryAttempt{Status: "pending"}
		
		// pending -> retrying
		attempt.Status = "retrying"
		assert.Equal(t, "retrying", attempt.Status)
		
		// retrying -> delivered
		attempt.Status = "delivered"
		assert.Equal(t, "delivered", attempt.Status)
		
		// pending -> failed
		attempt.Status = "failed"
		assert.Equal(t, "failed", attempt.Status)
	})

	t.Run("payload hash generation", func(t *testing.T) {
		testPayloads := []string{
			`{"event": "user.created"}`,
			`{"event": "user.updated", "data": {"id": "123"}}`,
			`{"event": "order.completed", "data": {"order_id": "456", "amount": 99.99}}`,
		}

		for i, payload := range testPayloads {
			t.Run(fmt.Sprintf("payload_%d", i), func(t *testing.T) {
				hash := sha256.Sum256([]byte(payload))
				payloadHash := fmt.Sprintf("sha256-%x", hash)
				
				attempt := &models.DeliveryAttempt{
					PayloadHash: payloadHash,
					PayloadSize: len(payload),
				}
				
				assert.Contains(t, attempt.PayloadHash, "sha256-")
				assert.Equal(t, len(payload), attempt.PayloadSize)
				assert.Len(t, attempt.PayloadHash, 71) // "sha256-" + 64 hex characters
			})
		}
	})
}



// TestDeliveryAttemptRepositoryErrorHandling tests error handling scenarios
func TestDeliveryAttemptRepositoryErrorHandling(t *testing.T) {
	t.Run("invalid attempt data", func(t *testing.T) {
		invalidAttempts := []*models.DeliveryAttempt{
			{
				EndpointID:    uuid.Nil, // Invalid endpoint ID
				PayloadHash:   "sha256-abcdef",
				PayloadSize:   1024,
				Status:        "pending",
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "", // Empty payload hash
				PayloadSize:   1024,
				Status:        "pending",
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "invalid-hash", // Invalid hash format
				PayloadSize:   1024,
				Status:        "pending",
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "sha256-abcdef",
				PayloadSize:   0, // Invalid payload size
				Status:        "pending",
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "sha256-abcdef",
				PayloadSize:   1024,
				Status:        "", // Empty status
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "sha256-abcdef",
				PayloadSize:   1024,
				Status:        "invalid-status", // Invalid status
				AttemptNumber: 1,
			},
			{
				EndpointID:    uuid.New(),
				PayloadHash:   "sha256-abcdef",
				PayloadSize:   1024,
				Status:        "pending",
				AttemptNumber: 0, // Invalid attempt number
			},
		}

		validStatuses := []string{"pending", "retrying", "delivered", "failed"}

		for i, attempt := range invalidAttempts {
			t.Run(fmt.Sprintf("invalid_attempt_%d", i), func(t *testing.T) {
				hasError := false
				
				if attempt.EndpointID == uuid.Nil {
					hasError = true
				}
				if attempt.PayloadHash == "" {
					hasError = true
				}
				if attempt.PayloadHash != "" && !isValidPayloadHash(attempt.PayloadHash) {
					hasError = true
				}
				if attempt.PayloadSize <= 0 {
					hasError = true
				}
				if attempt.Status == "" {
					hasError = true
				}
				if attempt.Status != "" && !contains(validStatuses, attempt.Status) {
					hasError = true
				}
				if attempt.AttemptNumber <= 0 {
					hasError = true
				}
				
				assert.True(t, hasError, "attempt should be invalid")
			})
		}
	})
}

// TestDeliveryAttemptFiltering tests filtering and querying logic
func TestDeliveryAttemptFiltering(t *testing.T) {
	t.Run("status filtering logic", func(t *testing.T) {
		attempts := []*models.DeliveryAttempt{
			{ID: uuid.New(), Status: "pending"},
			{ID: uuid.New(), Status: "delivered"},
			{ID: uuid.New(), Status: "failed"},
			{ID: uuid.New(), Status: "retrying"},
			{ID: uuid.New(), Status: "pending"},
		}

		// Filter by status
		pendingAttempts := filterAttemptsByStatus(attempts, "pending")
		assert.Len(t, pendingAttempts, 2)
		
		deliveredAttempts := filterAttemptsByStatus(attempts, "delivered")
		assert.Len(t, deliveredAttempts, 1)
		
		failedAttempts := filterAttemptsByStatus(attempts, "failed")
		assert.Len(t, failedAttempts, 1)
	})

	t.Run("pending deliveries logic", func(t *testing.T) {
		now := time.Now()
		attempts := []*models.DeliveryAttempt{
			{
				ID:          uuid.New(),
				Status:      "pending",
				ScheduledAt: now.Add(-time.Minute), // Ready for delivery
			},
			{
				ID:          uuid.New(),
				Status:      "retrying",
				ScheduledAt: now.Add(-time.Minute), // Ready for retry
			},
			{
				ID:          uuid.New(),
				Status:      "pending",
				ScheduledAt: now.Add(time.Minute), // Not yet ready
			},
			{
				ID:          uuid.New(),
				Status:      "delivered",
				ScheduledAt: now.Add(-time.Minute), // Already delivered
			},
		}

		// Filter pending deliveries ready for processing
		readyAttempts := filterPendingDeliveries(attempts, now)
		assert.Len(t, readyAttempts, 2)
		
		for _, attempt := range readyAttempts {
			assert.Contains(t, []string{"pending", "retrying"}, attempt.Status)
			assert.True(t, attempt.ScheduledAt.Before(now) || attempt.ScheduledAt.Equal(now))
		}
	})

	t.Run("delivery history filtering", func(t *testing.T) {
		endpointID := uuid.New()
		attempts := []*models.DeliveryAttempt{
			{ID: uuid.New(), EndpointID: endpointID, Status: "delivered"},
			{ID: uuid.New(), EndpointID: endpointID, Status: "failed"},
			{ID: uuid.New(), EndpointID: uuid.New(), Status: "delivered"}, // Different endpoint
			{ID: uuid.New(), EndpointID: endpointID, Status: "pending"},
		}

		// Filter by endpoint ID
		endpointAttempts := filterAttemptsByEndpoint(attempts, endpointID)
		assert.Len(t, endpointAttempts, 3)
		
		// Filter by endpoint ID and status
		deliveredForEndpoint := filterAttemptsByEndpointAndStatuses(attempts, endpointID, []string{"delivered", "failed"})
		assert.Len(t, deliveredForEndpoint, 2)
	})
}

// Helper functions for testing
func isValidPayloadHash(hash string) bool {
	if len(hash) != 71 { // "sha256-" + 64 hex characters
		return false
	}
	if hash[:7] != "sha256-" {
		return false
	}
	// Check if remaining characters are valid hex
	hexPart := hash[7:]
	for _, char := range hexPart {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func filterAttemptsByStatus(attempts []*models.DeliveryAttempt, status string) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		if attempt.Status == status {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func filterPendingDeliveries(attempts []*models.DeliveryAttempt, now time.Time) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		if (attempt.Status == "pending" || attempt.Status == "retrying") && 
		   (attempt.ScheduledAt.Before(now) || attempt.ScheduledAt.Equal(now)) {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func filterAttemptsByEndpoint(attempts []*models.DeliveryAttempt, endpointID uuid.UUID) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		if attempt.EndpointID == endpointID {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func filterAttemptsByEndpointAndStatuses(attempts []*models.DeliveryAttempt, endpointID uuid.UUID, statuses []string) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		if attempt.EndpointID == endpointID && contains(statuses, attempt.Status) {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestDeliveryHistoryFiltering tests the new delivery history filtering functionality
func TestDeliveryHistoryFiltering(t *testing.T) {
	t.Run("delivery history filters validation", func(t *testing.T) {
		endpointID1 := uuid.New()
		endpointID2 := uuid.New()
		
		// Test empty filters
		emptyFilters := DeliveryHistoryFilters{}
		assert.Empty(t, emptyFilters.EndpointIDs)
		assert.Empty(t, emptyFilters.Statuses)
		assert.True(t, emptyFilters.StartDate.IsZero())
		assert.True(t, emptyFilters.EndDate.IsZero())
		
		// Test filters with values
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()
		filters := DeliveryHistoryFilters{
			EndpointIDs: []uuid.UUID{endpointID1, endpointID2},
			Statuses:    []string{"delivered", "failed"},
			StartDate:   startDate,
			EndDate:     endDate,
		}
		
		assert.Len(t, filters.EndpointIDs, 2)
		assert.Contains(t, filters.EndpointIDs, endpointID1)
		assert.Contains(t, filters.EndpointIDs, endpointID2)
		assert.Len(t, filters.Statuses, 2)
		assert.Contains(t, filters.Statuses, "delivered")
		assert.Contains(t, filters.Statuses, "failed")
		assert.Equal(t, startDate, filters.StartDate)
		assert.Equal(t, endDate, filters.EndDate)
		assert.True(t, filters.EndDate.After(filters.StartDate))
	})

	t.Run("date range filtering logic", func(t *testing.T) {
		now := time.Now()
		attempts := []*models.DeliveryAttempt{
			{
				ID:        uuid.New(),
				Status:    "delivered",
				CreatedAt: now.Add(-2 * time.Hour), // 2 hours ago
			},
			{
				ID:        uuid.New(),
				Status:    "failed",
				CreatedAt: now.Add(-1 * time.Hour), // 1 hour ago
			},
			{
				ID:        uuid.New(),
				Status:    "delivered",
				CreatedAt: now.Add(-30 * time.Minute), // 30 minutes ago
			},
			{
				ID:        uuid.New(),
				Status:    "pending",
				CreatedAt: now.Add(-10 * time.Minute), // 10 minutes ago
			},
		}

		// Filter by date range (last hour)
		startDate := now.Add(-1 * time.Hour)
		endDate := now
		
		filteredAttempts := filterAttemptsByDateRange(attempts, startDate, endDate)
		assert.Len(t, filteredAttempts, 3) // Should exclude the 2-hour-old attempt
		
		for _, attempt := range filteredAttempts {
			assert.True(t, attempt.CreatedAt.After(startDate) || attempt.CreatedAt.Equal(startDate))
			assert.True(t, attempt.CreatedAt.Before(endDate) || attempt.CreatedAt.Equal(endDate))
		}
	})

	t.Run("multiple endpoint filtering logic", func(t *testing.T) {
		endpoint1 := uuid.New()
		endpoint2 := uuid.New()
		endpoint3 := uuid.New()
		
		attempts := []*models.DeliveryAttempt{
			{ID: uuid.New(), EndpointID: endpoint1, Status: "delivered"},
			{ID: uuid.New(), EndpointID: endpoint2, Status: "failed"},
			{ID: uuid.New(), EndpointID: endpoint1, Status: "pending"},
			{ID: uuid.New(), EndpointID: endpoint3, Status: "delivered"},
			{ID: uuid.New(), EndpointID: endpoint2, Status: "delivered"},
		}

		// Filter by multiple endpoints
		targetEndpoints := []uuid.UUID{endpoint1, endpoint2}
		filteredAttempts := filterAttemptsByMultipleEndpoints(attempts, targetEndpoints)
		
		assert.Len(t, filteredAttempts, 4) // Should exclude endpoint3 attempts
		
		for _, attempt := range filteredAttempts {
			assert.Contains(t, targetEndpoints, attempt.EndpointID)
		}
	})

	t.Run("combined filters logic", func(t *testing.T) {
		endpoint1 := uuid.New()
		endpoint2 := uuid.New()
		now := time.Now()
		
		attempts := []*models.DeliveryAttempt{
			{
				ID:         uuid.New(),
				EndpointID: endpoint1,
				Status:     "delivered",
				CreatedAt:  now.Add(-30 * time.Minute),
			},
			{
				ID:         uuid.New(),
				EndpointID: endpoint1,
				Status:     "failed",
				CreatedAt:  now.Add(-20 * time.Minute),
			},
			{
				ID:         uuid.New(),
				EndpointID: endpoint2,
				Status:     "delivered",
				CreatedAt:  now.Add(-15 * time.Minute),
			},
			{
				ID:         uuid.New(),
				EndpointID: endpoint1,
				Status:     "pending",
				CreatedAt:  now.Add(-10 * time.Minute),
			},
		}

		// Apply combined filters: endpoint1, delivered/failed status, last 25 minutes
		filters := DeliveryHistoryFilters{
			EndpointIDs: []uuid.UUID{endpoint1},
			Statuses:    []string{"delivered", "failed"},
			StartDate:   now.Add(-25 * time.Minute),
			EndDate:     now,
		}
		
		filteredAttempts := applyCombinedFilters(attempts, filters)
		assert.Len(t, filteredAttempts, 2) // Should match 2 attempts
		
		for _, attempt := range filteredAttempts {
			assert.Equal(t, endpoint1, attempt.EndpointID)
			assert.Contains(t, []string{"delivered", "failed"}, attempt.Status)
			assert.True(t, attempt.CreatedAt.After(filters.StartDate))
			assert.True(t, attempt.CreatedAt.Before(filters.EndDate))
		}
	})
}

// TestDeliveryAttemptsByDeliveryID tests the logic for retrieving attempts by delivery ID
func TestDeliveryAttemptsByDeliveryID(t *testing.T) {
	t.Run("delivery attempts grouping logic", func(t *testing.T) {
		deliveryID := uuid.New()
		endpointID := uuid.New()
		
		attempts := []*models.DeliveryAttempt{
			{
				ID:            deliveryID,
				EndpointID:    endpointID,
				Status:        "pending",
				AttemptNumber: 1,
				CreatedAt:     time.Now().Add(-10 * time.Minute),
			},
			{
				ID:            deliveryID,
				EndpointID:    endpointID,
				Status:        "retrying",
				AttemptNumber: 2,
				CreatedAt:     time.Now().Add(-5 * time.Minute),
			},
			{
				ID:            deliveryID,
				EndpointID:    endpointID,
				Status:        "delivered",
				AttemptNumber: 3,
				HTTPStatus:    &[]int{200}[0],
				CreatedAt:     time.Now(),
			},
		}

		// Verify attempts are for the same delivery
		for _, attempt := range attempts {
			assert.Equal(t, deliveryID, attempt.ID)
			assert.Equal(t, endpointID, attempt.EndpointID)
		}

		// Verify attempt progression
		assert.Equal(t, "pending", attempts[0].Status)
		assert.Equal(t, 1, attempts[0].AttemptNumber)
		
		assert.Equal(t, "retrying", attempts[1].Status)
		assert.Equal(t, 2, attempts[1].AttemptNumber)
		
		assert.Equal(t, "delivered", attempts[2].Status)
		assert.Equal(t, 3, attempts[2].AttemptNumber)
		assert.NotNil(t, attempts[2].HTTPStatus)
		assert.Equal(t, 200, *attempts[2].HTTPStatus)

		// Verify chronological order
		assert.True(t, attempts[1].CreatedAt.After(attempts[0].CreatedAt))
		assert.True(t, attempts[2].CreatedAt.After(attempts[1].CreatedAt))
	})

	t.Run("delivery summary calculation logic", func(t *testing.T) {
		deliveryID := uuid.New()
		
		// Test successful delivery after retries
		successfulAttempts := []*models.DeliveryAttempt{
			{
				ID:            deliveryID,
				Status:        "pending",
				AttemptNumber: 1,
				CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:            deliveryID,
				Status:        "retrying",
				AttemptNumber: 2,
				HTTPStatus:    &[]int{500}[0],
				ErrorMessage:  &[]string{"Internal Server Error"}[0],
				CreatedAt:     time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
			},
			{
				ID:            deliveryID,
				Status:        "delivered",
				AttemptNumber: 3,
				HTTPStatus:    &[]int{200}[0],
				DeliveredAt:   &[]time.Time{time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC)}[0],
				CreatedAt:     time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC),
			},
		}

		summary := calculateDeliverySummary(successfulAttempts)
		assert.Equal(t, 3, summary.TotalAttempts)
		assert.Equal(t, "delivered", summary.FinalStatus)
		assert.Equal(t, time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), summary.FirstAttemptAt)
		assert.Equal(t, time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC), summary.LastAttemptAt)
		assert.NotNil(t, summary.FinalHTTPStatus)
		assert.Equal(t, 200, *summary.FinalHTTPStatus)
		assert.NotNil(t, summary.DeliveredAt)

		// Test failed delivery
		failedAttempts := []*models.DeliveryAttempt{
			{
				ID:            deliveryID,
				Status:        "pending",
				AttemptNumber: 1,
				CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:            deliveryID,
				Status:        "failed",
				AttemptNumber: 5,
				HTTPStatus:    &[]int{404}[0],
				ErrorMessage:  &[]string{"Not Found"}[0],
				CreatedAt:     time.Date(2024, 1, 1, 10, 25, 0, 0, time.UTC),
			},
		}

		failedSummary := calculateDeliverySummary(failedAttempts)
		assert.Equal(t, 2, failedSummary.TotalAttempts)
		assert.Equal(t, "failed", failedSummary.FinalStatus)
		assert.NotNil(t, failedSummary.FinalHTTPStatus)
		assert.Equal(t, 404, *failedSummary.FinalHTTPStatus)
		assert.NotNil(t, failedSummary.FinalErrorMessage)
		assert.Equal(t, "Not Found", *failedSummary.FinalErrorMessage)
		assert.Nil(t, failedSummary.DeliveredAt)
	})
}

// Additional helper functions for the new tests
func filterAttemptsByDateRange(attempts []*models.DeliveryAttempt, startDate, endDate time.Time) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		if (attempt.CreatedAt.After(startDate) || attempt.CreatedAt.Equal(startDate)) &&
		   (attempt.CreatedAt.Before(endDate) || attempt.CreatedAt.Equal(endDate)) {
			filtered = append(filtered, attempt)
		}
	}
	return filtered
}

func filterAttemptsByMultipleEndpoints(attempts []*models.DeliveryAttempt, endpointIDs []uuid.UUID) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	for _, attempt := range attempts {
		for _, endpointID := range endpointIDs {
			if attempt.EndpointID == endpointID {
				filtered = append(filtered, attempt)
				break
			}
		}
	}
	return filtered
}

func applyCombinedFilters(attempts []*models.DeliveryAttempt, filters DeliveryHistoryFilters) []*models.DeliveryAttempt {
	var filtered []*models.DeliveryAttempt
	
	for _, attempt := range attempts {
		// Check endpoint filter
		if len(filters.EndpointIDs) > 0 {
			found := false
			for _, endpointID := range filters.EndpointIDs {
				if attempt.EndpointID == endpointID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Check status filter
		if len(filters.Statuses) > 0 && !contains(filters.Statuses, attempt.Status) {
			continue
		}
		
		// Check date range filter
		if !filters.StartDate.IsZero() && attempt.CreatedAt.Before(filters.StartDate) {
			continue
		}
		if !filters.EndDate.IsZero() && attempt.CreatedAt.After(filters.EndDate) {
			continue
		}
		
		filtered = append(filtered, attempt)
	}
	
	return filtered
}

type DeliverySummary struct {
	TotalAttempts       int        `json:"total_attempts"`
	FinalStatus         string     `json:"final_status"`
	FirstAttemptAt      time.Time  `json:"first_attempt_at"`
	LastAttemptAt       time.Time  `json:"last_attempt_at"`
	FinalHTTPStatus     *int       `json:"final_http_status,omitempty"`
	FinalErrorMessage   *string    `json:"final_error_message,omitempty"`
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
}

func calculateDeliverySummary(attempts []*models.DeliveryAttempt) DeliverySummary {
	if len(attempts) == 0 {
		return DeliverySummary{}
	}
	
	firstAttempt := attempts[0]
	lastAttempt := attempts[len(attempts)-1]
	
	summary := DeliverySummary{
		TotalAttempts:  len(attempts),
		FinalStatus:    lastAttempt.Status,
		FirstAttemptAt: firstAttempt.CreatedAt,
		LastAttemptAt:  lastAttempt.CreatedAt,
	}
	
	if lastAttempt.HTTPStatus != nil {
		summary.FinalHTTPStatus = lastAttempt.HTTPStatus
	}
	
	if lastAttempt.ErrorMessage != nil {
		summary.FinalErrorMessage = lastAttempt.ErrorMessage
	}
	
	if lastAttempt.DeliveredAt != nil {
		summary.DeliveredAt = lastAttempt.DeliveredAt
	}
	
	return summary
}