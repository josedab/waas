package models

import "fmt"

// Well-known webhook event types. Applications may use any string as an event
// type; these constants are provided for discoverability and consistency.
const (
	// Commerce events
	EventOrderCreated   = "order.created"
	EventOrderUpdated   = "order.updated"
	EventOrderCancelled = "order.cancelled"
	EventOrderCompleted = "order.completed"

	// Payment events
	EventPaymentCompleted = "payment.completed"
	EventPaymentFailed    = "payment.failed"
	EventPaymentRefunded  = "payment.refunded"

	// User events
	EventUserCreated = "user.created"
	EventUserUpdated = "user.updated"
	EventUserDeleted = "user.deleted"
	EventUserSignup  = "user.signup"

	// Subscription events
	EventSubscriptionCreated   = "subscription.created"
	EventSubscriptionCancelled = "subscription.cancelled"
	EventSubscriptionRenewed   = "subscription.renewed"

	// Webhook infrastructure events
	EventDeliverySucceeded = "delivery.succeeded"
	EventDeliveryFailed    = "delivery.failed"
	EventEndpointDisabled  = "endpoint.disabled"

	// Generic events
	EventPing = "ping"
	EventTest = "test"
)

// WellKnownEventTypes returns the list of well-known event types for
// documentation and auto-complete purposes. Custom event types are always
// allowed — this list is advisory, not restrictive.
func WellKnownEventTypes() []string {
	return []string{
		EventOrderCreated, EventOrderUpdated, EventOrderCancelled, EventOrderCompleted,
		EventPaymentCompleted, EventPaymentFailed, EventPaymentRefunded,
		EventUserCreated, EventUserUpdated, EventUserDeleted, EventUserSignup,
		EventSubscriptionCreated, EventSubscriptionCancelled, EventSubscriptionRenewed,
		EventDeliverySucceeded, EventDeliveryFailed, EventEndpointDisabled,
		EventPing, EventTest,
	}
}

// IsWellKnownEventType returns true if the event type is in the well-known list.
func IsWellKnownEventType(eventType string) bool {
	for _, t := range WellKnownEventTypes() {
		if t == eventType {
			return true
		}
	}
	return false
}

// ValidateEventType checks that an event type string is non-empty and
// reasonably formatted (1–255 chars, no whitespace).
func ValidateEventType(eventType string) error {
	if eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}
	if len(eventType) > 255 {
		return fmt.Errorf("event type must be 255 characters or fewer")
	}
	for _, r := range eventType {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return fmt.Errorf("event type must not contain whitespace")
		}
	}
	return nil
}
