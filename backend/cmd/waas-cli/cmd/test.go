package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run webhook integration tests",
	Long: `Send a test webhook and verify delivery succeeds.

This command sends a test payload to a specified endpoint (or all endpoints)
and waits for the delivery to complete, reporting success or failure.

Examples:
  waas test --endpoint ep_123                       # Test specific endpoint
  waas test --all                                   # Test all endpoints
  waas test --endpoint ep_123 --file payload.json   # Custom payload
  waas test --endpoint ep_123 --timeout 30s         # Custom timeout`,
	RunE: runTest,
}

var (
	testEndpoint string
	testAll      bool
	testPayload  string
	testFile     string
	testTimeout  string
)

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVar(&testEndpoint, "endpoint", "", "Endpoint ID to test")
	testCmd.Flags().BoolVar(&testAll, "all", false, "Test all endpoints")
	testCmd.Flags().StringVar(&testPayload, "data", "", "Custom test payload JSON")
	testCmd.Flags().StringVar(&testFile, "file", "", "File containing test payload")
	testCmd.Flags().StringVar(&testTimeout, "timeout", "15s", "Timeout waiting for delivery")
}

func runTest(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	timeout, err := time.ParseDuration(testTimeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	// Determine payload
	payload := []byte(`{"event":"waas.test","timestamp":"` + time.Now().Format(time.RFC3339) + `","data":{"test":true}}`)
	if testPayload != "" {
		payload = []byte(testPayload)
	} else if testFile != "" {
		payload, err = os.ReadFile(testFile)
		if err != nil {
			return fmt.Errorf("failed to read payload file: %w", err)
		}
	}

	// Determine endpoints to test
	var endpointIDs []string
	if testAll {
		endpoints, err := client.ListEndpoints()
		if err != nil {
			return fmt.Errorf("failed to list endpoints: %w", err)
		}
		for _, ep := range endpoints {
			if ep.IsActive {
				endpointIDs = append(endpointIDs, ep.ID)
			}
		}
		if len(endpointIDs) == 0 {
			fmt.Println("No active endpoints found.")
			return nil
		}
	} else if testEndpoint != "" {
		endpointIDs = []string{testEndpoint}
	} else {
		return fmt.Errorf("specify --endpoint or --all")
	}

	fmt.Printf("Testing %d endpoint(s)...\n\n", len(endpointIDs))

	passed := 0
	failed := 0

	for _, epID := range endpointIDs {
		result := testSingleEndpoint(client, epID, payload, timeout)
		if result {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n%s\n", repeatString("─", 50))
	fmt.Printf("Results: %d passed, %d failed, %d total\n", passed, failed, passed+failed)

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

func testSingleEndpoint(client *Client, endpointID string, payload []byte, timeout time.Duration) bool {
	fmt.Printf("● Testing endpoint %s... ", truncate(endpointID, 20))

	req := &SendWebhookRequest{
		EndpointID: endpointID,
		Payload:    payload,
	}

	resp, err := client.SendWebhook(req)
	if err != nil {
		fmt.Printf("\033[31mFAIL\033[0m (send error: %v)\n", err)
		return false
	}

	// Poll for delivery status
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		delivery, err := client.GetDelivery(resp.DeliveryID)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		switch delivery.Status {
		case "delivered":
			fmt.Printf("\033[32mPASS\033[0m (HTTP %d)\n", delivery.LastHTTPStatus)
			return true
		case "failed":
			fmt.Printf("\033[31mFAIL\033[0m (%s)\n", delivery.LastError)
			return false
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\033[33mTIMEOUT\033[0m (no response within %s)\n", timeout)
	return false
}

var testResultsCmd = &cobra.Command{
	Use:   "test-results",
	Short: "Show recent test results",
	Long: `Display results of recent webhook integration tests.

Examples:
  waas test-results
  waas test-results -o json`,
	RunE: runTestResults,
}

func init() {
	rootCmd.AddCommand(testResultsCmd)
}

func runTestResults(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	deliveries, err := client.ListDeliveries("", 20)
	if err != nil {
		return fmt.Errorf("failed to fetch deliveries: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(deliveries)
	}

	if len(deliveries) == 0 {
		fmt.Println("No recent test results. Run: waas test --endpoint <id>")
		return nil
	}

	for _, d := range deliveries {
		icon := "●"
		switch d.Status {
		case "delivered":
			icon = "\033[32m✓\033[0m"
		case "failed":
			icon = "\033[31m✗\033[0m"
		case "retrying":
			icon = "\033[33m↻\033[0m"
		}

		fmt.Printf("%s %s → %s (%s, %d attempts)\n",
			icon,
			truncate(d.EndpointID, 15),
			d.Status,
			d.CreatedAt.Format("15:04:05"),
			d.AttemptCount,
		)
	}

	return nil
}
