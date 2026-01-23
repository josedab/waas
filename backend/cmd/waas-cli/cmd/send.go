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
	Long: `Send a webhook payload to a specified endpoint or publish to a topic.

The payload can be provided via stdin, a file, or inline JSON.

Examples:
  waas send --endpoint ep_123 --data '{"event": "test"}'
  waas send --endpoint ep_123 --file payload.json
  echo '{"event": "test"}' | waas send --endpoint ep_123
  waas send --endpoint ep_123 --data '{"event": "test"}' --header "X-Custom: value"
  waas send --topic orders --event-type "order.created" --data '{"id": "123"}'`,
	RunE: runSend,
}

var (
	sendEndpointID  string
	sendData        string
	sendFile        string
	sendHeaders     []string
	sendTopic       string
	sendEventType   string
)

func init() {
	rootCmd.AddCommand(sendCmd)

	sendCmd.Flags().StringVar(&sendEndpointID, "endpoint", "", "Target endpoint ID")
	sendCmd.Flags().StringVar(&sendData, "data", "", "JSON payload to send")
	sendCmd.Flags().StringVar(&sendFile, "file", "", "File containing JSON payload")
	sendCmd.Flags().StringArrayVar(&sendHeaders, "header", nil, "Additional headers (format: Key: Value)")
	sendCmd.Flags().StringVar(&sendTopic, "topic", "", "Publish to a topic instead of a specific endpoint")
	sendCmd.Flags().StringVar(&sendEventType, "event-type", "", "Event type for topic publishing")
}

func runSend(cmd *cobra.Command, args []string) error {
	if sendEndpointID == "" && sendTopic == "" {
		return fmt.Errorf("either --endpoint or --topic is required")
	}

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

	// Topic-based publishing
	if sendTopic != "" {
		body := map[string]interface{}{
			"topic":   sendTopic,
			"payload": json.RawMessage(payload),
		}
		if sendEventType != "" {
			body["event_type"] = sendEventType
		}
		if len(headers) > 0 {
			body["headers"] = headers
		}

		resp, err := client.doRequest("POST", "/api/v1/webhooks/send", body)
		if err != nil {
			return fmt.Errorf("failed to publish to topic: %w", err)
		}

		var result SendWebhookResponse
		if err := parseResponse(resp, &result); err != nil {
			return fmt.Errorf("failed to publish to topic: %w", err)
		}

		if output == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		fmt.Printf("✓ Published to topic %q\n", sendTopic)
		if sendEventType != "" {
			fmt.Printf("  Event Type:  %s\n", sendEventType)
		}
		fmt.Printf("  Delivery ID: %s\n", result.DeliveryID)
		fmt.Printf("  Status:      %s\n", result.Status)
		return nil
	}

	// Direct endpoint send
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
