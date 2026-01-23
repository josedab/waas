package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Run a local mock webhook endpoint",
	Long: `Start a local HTTP server that receives and logs webhooks.

Useful for testing webhook delivery without deploying a real endpoint.
The server logs all incoming requests and can respond with configurable status codes.

Examples:
  waas mock                         # Start mock server on :9090
  waas mock --port 8888             # Custom port
  waas mock --status 500            # Respond with 500 to simulate failures
  waas mock --delay 2s              # Add artificial latency
  waas mock --record requests.json  # Record all requests to file`,
	RunE: runMock,
}

var (
	mockPort       int
	mockStatus     int
	mockDelay      string
	mockRecordFile string
)

func init() {
	rootCmd.AddCommand(mockCmd)

	mockCmd.Flags().IntVar(&mockPort, "port", 9090, "Port to listen on")
	mockCmd.Flags().IntVar(&mockStatus, "status", 200, "HTTP status code to respond with")
	mockCmd.Flags().StringVar(&mockDelay, "delay", "", "Response delay (e.g., 500ms, 2s)")
	mockCmd.Flags().StringVar(&mockRecordFile, "record", "", "File to record received requests")
}

type receivedRequest struct {
	Timestamp  time.Time         `json:"timestamp"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	RemoteAddr string            `json:"remote_addr"`
}

func runMock(cmd *cobra.Command, args []string) error {
	var delay time.Duration
	if mockDelay != "" {
		var err error
		delay, err = time.ParseDuration(mockDelay)
		if err != nil {
			return fmt.Errorf("invalid delay format: %w", err)
		}
	}

	var mu sync.Mutex
	var received []receivedRequest
	count := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		mu.Lock()
		count++
		num := count

		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		req := receivedRequest{
			Timestamp:  time.Now(),
			Method:     r.Method,
			Path:       r.URL.Path,
			Headers:    headers,
			Body:       string(body),
			RemoteAddr: r.RemoteAddr,
		}
		received = append(received, req)
		mu.Unlock()

		// Log the request
		fmt.Printf("[%s] #%d %s %s",
			req.Timestamp.Format("15:04:05"),
			num,
			r.Method,
			r.URL.Path,
		)
		if len(body) > 0 {
			preview := string(body)
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			fmt.Printf(" → %s", preview)
		}
		fmt.Println()

		if delay > 0 {
			time.Sleep(delay)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(mockStatus)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "received",
			"request":  num,
			"mock":     true,
		})
	})

	addr := fmt.Sprintf(":%d", mockPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start mock server: %w", err)
	}

	fmt.Printf("🎯 Mock webhook endpoint running on http://localhost:%d\n", mockPort)
	fmt.Printf("   Response status: %d\n", mockStatus)
	if delay > 0 {
		fmt.Printf("   Response delay:  %s\n", delay)
	}
	fmt.Println("   Ctrl+C to stop")
	fmt.Println()
	fmt.Println("Register this endpoint with:")
	fmt.Printf("   waas endpoints create --url http://localhost:%d/webhook\n\n", mockPort)

	server := &http.Server{Handler: handler}

	// Handle graceful shutdown
	go func() {
		<-cmd.Context().Done()
		if mockRecordFile != "" {
			mu.Lock()
			data, _ := json.MarshalIndent(received, "", "  ")
			mu.Unlock()
			if err := os.WriteFile(mockRecordFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save recorded requests: %v\n", err)
			} else {
				fmt.Printf("\n✓ Saved %d requests to %s\n", len(received), mockRecordFile)
			}
		}
		server.Close()
	}()

	return server.Serve(listener)
}

var mockListCmd = &cobra.Command{
	Use:   "mock-endpoints",
	Short: "List remote mock endpoints",
	Long: `List mock webhook endpoints configured in your WAAS account.

Examples:
  waas mock-endpoints`,
	RunE: runMockList,
}

func init() {
	rootCmd.AddCommand(mockListCmd)
}

func runMockList(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	resp, err := client.doRequest("GET", "/api/v1/mocking/endpoints", nil)
	if err != nil {
		return fmt.Errorf("failed to list mock endpoints: %w", err)
	}

	var result struct {
		Endpoints []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Status    int    `json:"status_code"`
			CreatedAt string `json:"created_at"`
		} `json:"endpoints"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result.Endpoints)
	}

	if len(result.Endpoints) == 0 {
		fmt.Println("No mock endpoints found. Start one with: waas mock")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCREATED")
	for _, ep := range result.Endpoints {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", ep.ID, ep.Name, ep.Status, ep.CreatedAt)
	}
	return w.Flush()
}
