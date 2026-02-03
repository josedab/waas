package main

import (
	"context"
	"fmt"
	"log"

	"github.com/josedab/waas/sdk/go/client"
)

func main() {
	// Initialize the client
	c := client.New("your-api-key-here")
	ctx := context.Background()

	// Example 1: Test a webhook endpoint
	fmt.Println("Testing webhook endpoint...")
	testResult, err := c.Testing.TestWebhook(ctx, &client.TestWebhookRequest{
		URL: "https://httpbin.org/post",
		Payload: map[string]interface{}{
			"test":    true,
			"message": "Hello from webhook test!",
		},
		Headers: map[string]string{
			"X-Test-Header": "test-value",
		},
		Method:  "POST",
		Timeout: 30,
	})
	if err != nil {
		log.Fatal("Failed to test webhook:", err)
	}

	fmt.Printf("Test Result:\n")
	fmt.Printf("  Status: %s\n", testResult.Status)
	fmt.Printf("  HTTP Status: %d\n", *testResult.HTTPStatus)
	fmt.Printf("  Latency: %dms\n", *testResult.Latency)
	if testResult.ErrorMessage != nil {
		fmt.Printf("  Error: %s\n", *testResult.ErrorMessage)
	}

	// Example 2: Create a temporary test endpoint
	fmt.Println("\nCreating temporary test endpoint...")
	testEndpoint, err := c.Testing.CreateTestEndpoint(ctx, &client.CreateTestEndpointRequest{
		Name:        "My Test Endpoint",
		Description: "Temporary endpoint for testing webhooks",
		TTL:         3600, // 1 hour
	})
	if err != nil {
		log.Fatal("Failed to create test endpoint:", err)
	}

	fmt.Printf("Test Endpoint Created:\n")
	fmt.Printf("  ID: %s\n", testEndpoint.ID)
	fmt.Printf("  URL: %s\n", testEndpoint.URL)
	fmt.Printf("  Expires At: %s\n", testEndpoint.ExpiresAt)

	// Example 3: Create a real webhook endpoint and send a test
	fmt.Println("\nCreating real webhook endpoint...")
	endpoint, err := c.Webhooks.CreateEndpoint(ctx, &client.CreateEndpointRequest{
		URL: testEndpoint.URL, // Use our test endpoint as the target
	})
	if err != nil {
		log.Fatal("Failed to create webhook endpoint:", err)
	}

	// Send a webhook to the test endpoint
	fmt.Println("Sending webhook to test endpoint...")
	delivery, err := c.Webhooks.Send(ctx, &client.SendWebhookRequest{
		EndpointID: &endpoint.ID,
		Payload: map[string]interface{}{
			"event":     "test.webhook",
			"timestamp": "2024-01-01T00:00:00Z",
			"data": map[string]interface{}{
				"message": "This is a test webhook delivery",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to send webhook:", err)
	}

	fmt.Printf("Webhook sent with delivery ID: %s\n", delivery.DeliveryID)

	// Example 4: Inspect the delivery for debugging
	fmt.Println("\nInspecting delivery...")
	inspection, err := c.Testing.InspectDelivery(ctx, delivery.DeliveryID)
	if err != nil {
		log.Fatal("Failed to inspect delivery:", err)
	}

	fmt.Printf("Delivery Inspection:\n")
	fmt.Printf("  Status: %s\n", inspection.Status)
	fmt.Printf("  Attempt Number: %d\n", inspection.AttemptNumber)
	fmt.Printf("  Request URL: %s\n", inspection.Request.URL)
	fmt.Printf("  Payload Size: %d bytes\n", inspection.Request.PayloadSize)

	if inspection.Response != nil {
		fmt.Printf("  Response Status: %d\n", inspection.Response.HTTPStatus)
		fmt.Printf("  Response Latency: %dms\n", inspection.Response.Latency)
	}

	if inspection.ErrorDetails != nil {
		fmt.Printf("  Error Type: %s\n", inspection.ErrorDetails.ErrorType)
		fmt.Printf("  Error Message: %s\n", inspection.ErrorDetails.ErrorMessage)
		if len(inspection.ErrorDetails.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range inspection.ErrorDetails.Suggestions {
				fmt.Printf("    - %s\n", suggestion)
			}
		}
	}

	fmt.Printf("\nTimeline (%d events):\n", len(inspection.Timeline))
	for _, event := range inspection.Timeline {
		fmt.Printf("  %s: %s - %s\n",
			event.Timestamp.Format("15:04:05"),
			event.Event,
			event.Description)
	}

	fmt.Println("\nTesting example completed!")
}
