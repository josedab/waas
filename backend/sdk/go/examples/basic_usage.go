package main

import (
	"context"
	"fmt"
	"log"

	"github.com/webhook-platform/go-sdk/client"
)

func main() {
	// Initialize the client with your API key
	c := client.New("your-api-key-here")
	
	ctx := context.Background()

	// Example 1: Create a webhook endpoint
	fmt.Println("Creating webhook endpoint...")
	endpoint, err := c.Webhooks.CreateEndpoint(ctx, &client.CreateEndpointRequest{
		URL: "https://your-app.com/webhooks",
		CustomHeaders: map[string]string{
			"Authorization": "Bearer your-token",
		},
	})
	if err != nil {
		log.Fatal("Failed to create endpoint:", err)
	}
	fmt.Printf("Created endpoint: %s (URL: %s)\n", endpoint.ID, endpoint.URL)

	// Example 2: List all endpoints
	fmt.Println("\nListing webhook endpoints...")
	endpoints, err := c.Webhooks.ListEndpoints(ctx, &client.ListEndpointsOptions{
		Limit: 10,
	})
	if err != nil {
		log.Fatal("Failed to list endpoints:", err)
	}
	fmt.Printf("Found %d endpoints\n", len(endpoints.Endpoints))

	// Example 3: Send a webhook
	fmt.Println("\nSending webhook...")
	delivery, err := c.Webhooks.Send(ctx, &client.SendWebhookRequest{
		EndpointID: &endpoint.ID,
		Payload: map[string]interface{}{
			"event": "user.created",
			"data": map[string]interface{}{
				"user_id": "12345",
				"email":   "user@example.com",
			},
		},
		Headers: map[string]string{
			"X-Event-Type": "user.created",
		},
	})
	if err != nil {
		log.Fatal("Failed to send webhook:", err)
	}
	fmt.Printf("Webhook sent with delivery ID: %s\n", delivery.DeliveryID)

	// Example 4: Get delivery details
	fmt.Println("\nGetting delivery details...")
	details, err := c.Webhooks.GetDeliveryDetails(ctx, delivery.DeliveryID)
	if err != nil {
		log.Fatal("Failed to get delivery details:", err)
	}
	fmt.Printf("Delivery status: %s, attempts: %d\n", details.Summary.Status, details.Summary.TotalAttempts)

	// Example 5: Get tenant information
	fmt.Println("\nGetting tenant information...")
	tenant, err := c.Tenants.GetTenant(ctx)
	if err != nil {
		log.Fatal("Failed to get tenant:", err)
	}
	fmt.Printf("Tenant: %s (Tier: %s)\n", tenant.Name, tenant.SubscriptionTier)

	fmt.Println("\nExample completed successfully!")
}