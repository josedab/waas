package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [delivery-id]",
	Short: "View webhook delivery logs",
	Long: `View webhook delivery history and attempt logs.

Without arguments, shows recent deliveries. With a delivery ID, shows detailed attempt logs.

Examples:
  waas logs                           # List recent deliveries
  waas logs --endpoint ep_123         # Filter by endpoint
  waas logs del_abc123                # Show attempts for specific delivery
  waas logs --tail                    # Stream logs in real-time
  waas logs --status failed           # Show only failed deliveries`,
	RunE: runLogs,
}

var (
	logsEndpoint string
	logsLimit    int
	logsTail     bool
	logsStatus   string
)

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringVar(&logsEndpoint, "endpoint", "", "Filter by endpoint ID")
	logsCmd.Flags().IntVar(&logsLimit, "limit", 20, "Number of deliveries to show")
	logsCmd.Flags().BoolVar(&logsTail, "tail", false, "Stream logs in real-time")
	logsCmd.Flags().StringVar(&logsStatus, "status", "", "Filter by status (pending, delivered, failed, retrying)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	// If delivery ID is provided, show detailed logs
	if len(args) > 0 {
		return showDeliveryLogs(client, args[0])
	}

	// If tail mode, stream logs
	if logsTail {
		return streamLogs(client)
	}

	// Otherwise, list recent deliveries
	return listDeliveries(client)
}

func listDeliveries(client *Client) error {
	deliveries, err := client.ListDeliveries(logsEndpoint, logsLimit)
	if err != nil {
		return fmt.Errorf("failed to list deliveries: %w", err)
	}

	// Filter by status if specified
	if logsStatus != "" {
		var filtered []Delivery
		for _, d := range deliveries {
			if d.Status == logsStatus {
				filtered = append(filtered, d)
			}
		}
		deliveries = filtered
	}

	switch output {
	case "json":
		return out.PrintJSON(deliveries)
	case "yaml":
		return out.PrintYAML(deliveries)
	case "csv":
		headers := []string{"ID", "Endpoint", "Status", "Attempts", "Last Attempt"}
		var rows [][]string
		for _, d := range deliveries {
			lastAttempt := ""
			if !d.LastAttemptAt.IsZero() {
				lastAttempt = d.LastAttemptAt.Format(time.RFC3339)
			}
			rows = append(rows, []string{d.ID, d.EndpointID, d.Status, fmt.Sprintf("%d", d.AttemptCount), lastAttempt})
		}
		return out.FormatCSV(headers, rows)
	}

	if len(deliveries) == 0 {
		fmt.Println("No deliveries found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tENDPOINT\tSTATUS\tATTEMPTS\tLAST ATTEMPT")
	for _, d := range deliveries {
		lastAttempt := "-"
		if !d.LastAttemptAt.IsZero() {
			lastAttempt = d.LastAttemptAt.Format("15:04:05")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			truncate(d.ID, 20),
			truncate(d.EndpointID, 15),
			colorStatus(d.Status),
			d.AttemptCount,
			lastAttempt,
		)
	}
	return w.Flush()
}

func showDeliveryLogs(client *Client, deliveryID string) error {
	// Get delivery details
	delivery, err := client.GetDelivery(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to get delivery: %w", err)
	}

	// Get attempt logs
	attempts, err := client.GetDeliveryLogs(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to get delivery logs: %w", err)
	}

	if output == "json" {
		result := map[string]interface{}{
			"delivery": delivery,
			"attempts": attempts,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Display delivery summary
	fmt.Printf("Delivery: %s\n", delivery.ID)
	fmt.Printf("Status:   %s\n", colorStatus(delivery.Status))
	fmt.Printf("Endpoint: %s\n", delivery.EndpointID)
	fmt.Printf("Attempts: %d\n", delivery.AttemptCount)
	fmt.Printf("Created:  %s\n", delivery.CreatedAt.Format(time.RFC3339))

	if delivery.LastError != "" {
		fmt.Printf("Error:    %s\n", delivery.LastError)
	}

	if len(attempts) > 0 {
		fmt.Println("\nAttempt History:")
		fmt.Println(repeatString("-", 60))

		for _, a := range attempts {
			status := "✓"
			if a.HTTPStatus >= 400 || a.ErrorMessage != "" {
				status = "✗"
			}

			fmt.Printf("\n%s Attempt #%d - %s\n", status, a.AttemptNumber, a.CreatedAt.Format("15:04:05"))

			if a.HTTPStatus > 0 {
				fmt.Printf("  HTTP Status: %d\n", a.HTTPStatus)
			}

			if a.ErrorMessage != "" {
				fmt.Printf("  Error: %s\n", a.ErrorMessage)
			}

			if a.ResponseBody != "" && len(a.ResponseBody) < 200 {
				fmt.Printf("  Response: %s\n", a.ResponseBody)
			}
		}
	}

	return nil
}

func streamLogs(client *Client) error {
	fmt.Println("Streaming delivery logs... (Ctrl+C to stop)")
	fmt.Println()

	apiURL := getAPIURL()
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("failed to parse API URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/api/v1/webhooks/realtime"
	if logsEndpoint != "" {
		q := u.Query()
		q.Set("endpoint_id", logsEndpoint)
		u.RawQuery = q.Encode()
	}

	header := map[string][]string{
		"Authorization": {"Bearer " + apiKey},
	}

	conn, _, wsErr := websocket.DefaultDialer.Dial(u.String(), header)
	if wsErr != nil {
		// Fallback to polling if WebSocket is unavailable
		return streamLogsPoll(client)
	}
	defer conn.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var event struct {
				DeliveryID string `json:"delivery_id"`
				EndpointID string `json:"endpoint_id"`
				Status     string `json:"status"`
				EventType  string `json:"event_type"`
				HTTPStatus int    `json:"http_status"`
				Error      string `json:"error"`
			}
			if err := json.Unmarshal(message, &event); err != nil {
				continue
			}
			if logsStatus != "" && event.Status != logsStatus {
				continue
			}
			ts := time.Now().Format("15:04:05")
			fmt.Printf("[%s] %s %s → %s",
				ts,
				out.ColorStatus(event.Status),
				out.Truncate(event.EndpointID, 15),
				out.Truncate(event.DeliveryID, 20),
			)
			if event.HTTPStatus > 0 {
				fmt.Printf(" (HTTP %d)", event.HTTPStatus)
			}
			if event.Error != "" {
				fmt.Printf(" %s", event.Error)
			}
			fmt.Println()
		}
	}()

	select {
	case <-done:
		return nil
	case <-interrupt:
		fmt.Println("\n🛑 Stream stopped")
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return nil
	}
}

// streamLogsPoll falls back to HTTP polling when WebSocket is unavailable
func streamLogsPoll(client *Client) error {
	seen := make(map[string]bool)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-interrupt:
			fmt.Println("\n🛑 Stream stopped")
			return nil
		case <-ticker.C:
			deliveries, err := client.ListDeliveries(logsEndpoint, 10)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching logs: %v\n", err)
				continue
			}
			for _, d := range deliveries {
				if logsStatus != "" && d.Status != logsStatus {
					continue
				}
				key := fmt.Sprintf("%s-%d", d.ID, d.AttemptCount)
				if !seen[key] {
					seen[key] = true
					fmt.Printf("[%s] %s %s → %s (attempts: %d)\n",
						time.Now().Format("15:04:05"),
						colorStatus(d.Status),
						truncate(d.EndpointID, 15),
						truncate(d.ID, 20),
						d.AttemptCount,
					)
				}
			}
		}
	}
}

func colorStatus(status string) string {
	switch status {
	case "delivered":
		return "\033[32m" + status + "\033[0m" // green
	case "failed":
		return "\033[31m" + status + "\033[0m" // red
	case "retrying":
		return "\033[33m" + status + "\033[0m" // yellow
	default:
		return status
	}
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
