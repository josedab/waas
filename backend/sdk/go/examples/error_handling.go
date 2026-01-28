package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/webhook-platform/go-sdk/client"
)

func main() {
	c := client.New("your-api-key-here")
	ctx := context.Background()

	// Example 1: Handle specific API errors
	fmt.Println("=== API Error Handling ===")
	handleAPIErrors(ctx, c)

	// Example 2: Handle rate limiting with retry
	fmt.Println("\n=== Rate Limit Handling ===")
	handleRateLimiting(ctx, c)

	// Example 3: Handle network and timeout errors
	fmt.Println("\n=== Timeout Handling ===")
	handleTimeouts(c)

	fmt.Println("\nError handling examples completed!")
}

// handleAPIErrors demonstrates checking for typed API errors and
// branching on the error code returned by the server.
func handleAPIErrors(ctx context.Context, c *client.Client) {
	_, err := c.Webhooks.GetEndpoint(ctx, "non-existent-id")
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case http.StatusNotFound:
				fmt.Printf("Endpoint not found (code=%s): %s\n", apiErr.Code, apiErr.Message)
			case http.StatusUnauthorized:
				fmt.Printf("Authentication failed: %s\n", apiErr.Message)
			case http.StatusForbidden:
				fmt.Printf("Access denied: %s\n", apiErr.Message)
			case http.StatusBadRequest:
				fmt.Printf("Bad request (code=%s): %s\n", apiErr.Code, apiErr.Message)
				if apiErr.Details != nil {
					fmt.Printf("  Details: %v\n", apiErr.Details)
				}
			default:
				fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
			}
		} else {
			// Network error, DNS failure, TLS error, etc.
			fmt.Printf("Non-API error: %v\n", err)
		}
	}
}

// handleRateLimiting demonstrates a simple back-off loop when the
// server responds with 429 Too Many Requests.
func handleRateLimiting(ctx context.Context, c *client.Client) {
	const maxRetries = 3

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		_, lastErr = c.Webhooks.ListEndpoints(ctx, &client.ListEndpointsOptions{Limit: 10})
		if lastErr == nil {
			fmt.Println("Request succeeded")
			return
		}

		var apiErr *client.APIError
		if errors.As(lastErr, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			fmt.Printf("Rate limited, retrying in %v (attempt %d/%d)\n", backoff, attempt+1, maxRetries)
			time.Sleep(backoff)
			continue
		}

		// Not a rate-limit error — stop retrying.
		break
	}

	log.Printf("Request failed after retries: %v", lastErr)
}

// handleTimeouts demonstrates using a context deadline so that slow
// requests are cancelled automatically.
func handleTimeouts(c *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Webhooks.ListEndpoints(ctx, &client.ListEndpointsOptions{Limit: 10})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Request timed out — consider increasing the timeout or checking connectivity")
		} else {
			fmt.Printf("Request failed: %v\n", err)
		}
		return
	}

	fmt.Println("Request completed within timeout")
}
