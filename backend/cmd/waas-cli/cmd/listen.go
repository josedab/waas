package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Stream webhook events in real-time",
	Long: `Listen for webhook events in real-time using WebSocket connection.

Similar to 'stripe listen', this streams events as they happen with colorized output.

Examples:
  waas listen                              # Stream all events
  waas listen --endpoint ep_123            # Stream for specific endpoint
  waas listen --event-type "order.*"       # Filter by event type`,
	RunE: runListen,
}

var (
	listenEndpoint  string
	listenEventType string
)

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().StringVar(&listenEndpoint, "endpoint", "", "Filter events by endpoint ID")
	listenCmd.Flags().StringVar(&listenEventType, "event-type", "", "Filter by event type pattern (supports wildcards)")
}

// wsEvent represents a real-time webhook event from the WebSocket
type wsEvent struct {
	Type       string          `json:"type"`
	DeliveryID string          `json:"delivery_id"`
	EndpointID string          `json:"endpoint_id"`
	Status     string          `json:"status"`
	EventType  string          `json:"event_type"`
	HTTPStatus int             `json:"http_status,omitempty"`
	Error      string          `json:"error,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

func runListen(cmd *cobra.Command, args []string) error {
	apiURL := getAPIURL()
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	// Convert HTTP URL to WebSocket URL
	wsURL, err := buildWSURL(apiURL, apiKey)
	if err != nil {
		return fmt.Errorf("failed to build WebSocket URL: %w", err)
	}

	out.PrintInfo(fmt.Sprintf("Connecting to %s...", apiURL))

	header := map[string][]string{
		"Authorization": {"Bearer " + apiKey},
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	out.PrintSuccess("Connected! Listening for webhook events...")
	if listenEndpoint != "" {
		fmt.Printf("  Filtering: endpoint=%s\n", listenEndpoint)
	}
	if listenEventType != "" {
		fmt.Printf("  Filtering: event-type=%s\n", listenEventType)
	}
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println()

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					out.PrintError(fmt.Sprintf("Connection error: %v", err))
				}
				return
			}

			var event wsEvent
			if err := json.Unmarshal(message, &event); err != nil {
				continue
			}

			// Apply filters
			if listenEndpoint != "" && event.EndpointID != listenEndpoint {
				continue
			}
			if listenEventType != "" && !matchEventType(event.EventType, listenEventType) {
				continue
			}

			if output == "json" {
				json.NewEncoder(os.Stdout).Encode(event)
			} else {
				printEvent(event)
			}
		}
	}()

	select {
	case <-done:
		return nil
	case <-interrupt:
		fmt.Println("\n\nDisconnecting...")
		conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return nil
	}
}

func buildWSURL(apiURL, apiKey string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}

	u.Path = "/api/v1/webhooks/realtime"
	q := u.Query()
	if listenEndpoint != "" {
		q.Set("endpoint_id", listenEndpoint)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func printEvent(event wsEvent) {
	ts := event.Timestamp.Format("15:04:05")
	if event.Timestamp.IsZero() {
		ts = time.Now().Format("15:04:05")
	}

	statusIcon := "→"
	switch event.Status {
	case "delivered":
		statusIcon = "\033[32m✓\033[0m"
	case "failed":
		statusIcon = "\033[31m✗\033[0m"
	case "retrying":
		statusIcon = "\033[33m↻\033[0m"
	case "pending":
		statusIcon = "\033[36m●\033[0m"
	}

	eventType := event.EventType
	if eventType == "" {
		eventType = event.Type
	}

	fmt.Printf("  %s %s [%s] %s → %s",
		ts,
		statusIcon,
		out.ColorStatus(event.Status),
		eventType,
		out.Truncate(event.EndpointID, 20),
	)

	if event.HTTPStatus > 0 {
		fmt.Printf(" (HTTP %d)", event.HTTPStatus)
	}
	if event.Error != "" {
		fmt.Printf(" \033[31m%s\033[0m", event.Error)
	}
	fmt.Println()
}

// matchEventType matches an event type against a pattern with wildcard support
func matchEventType(eventType, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix+".")
	}
	return eventType == pattern
}
