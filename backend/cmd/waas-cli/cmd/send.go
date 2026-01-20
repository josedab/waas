package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a webhook to an endpoint",
	Long: `Send a webhook payload to a specified endpoint.

The payload can be provided via stdin, a file, or inline JSON.

Examples:
  waas send --endpoint ep_123 --data '{"event": "test"}'
  waas send --endpoint ep_123 --file payload.json
  echo '{"event": "test"}' | waas send --endpoint ep_123
  waas send --endpoint ep_123 --data '{"event": "test"}' --header "X-Custom: value"`,
	RunE: runSend,
}

var (
	sendEndpointID  string
	sendData        string
	sendFile        string
	sendHeaders     []string
)

func init() {
	rootCmd.AddCommand(sendCmd)

	sendCmd.Flags().StringVar(&sendEndpointID, "endpoint", "", "Target endpoint ID (required)")
	sendCmd.Flags().StringVar(&sendData, "data", "", "JSON payload to send")
	sendCmd.Flags().StringVar(&sendFile, "file", "", "File containing JSON payload")
	sendCmd.Flags().StringArrayVar(&sendHeaders, "header", nil, "Additional headers (format: Key: Value)")
	sendCmd.MarkFlagRequired("endpoint")
}

func runSend(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	// Determine payload source
	var payload []byte
	var err error

	if sendData != "" {
		payload = []byte(sendData)
	} else if sendFile != "" {
		payload, err = os.ReadFile(sendFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Check if there's data on stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			payload, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
		}
	}

	if len(payload) == 0 {
		return fmt.Errorf("no payload provided. Use --data, --file, or pipe via stdin")
	}

	// Validate JSON
	var jsonCheck interface{}
	if err := json.Unmarshal(payload, &jsonCheck); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	// Parse headers
	headers := make(map[string]string)
	for _, h := range sendHeaders {
		parts := splitHeader(h)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	req := &SendWebhookRequest{
		EndpointID: sendEndpointID,
		Payload:    payload,
		Headers:    headers,
	}

	resp, err := client.SendWebhook(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	fmt.Printf("✓ Webhook queued for delivery\n")
	fmt.Printf("  Delivery ID: %s\n", resp.DeliveryID)
	fmt.Printf("  Status:      %s\n", resp.Status)
	fmt.Printf("\nTrack delivery with: waas logs %s\n", resp.DeliveryID)

	return nil
}

func splitHeader(h string) []string {
	for i, c := range h {
		if c == ':' {
			key := h[:i]
			value := h[i+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
			return []string{key, value}
		}
	}
	return nil
}
