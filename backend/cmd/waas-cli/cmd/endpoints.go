package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var endpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "Manage webhook endpoints",
	Long:  `Create, list, update, and delete webhook endpoints.`,
}

var endpointsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all webhook endpoints",
	Long: `List all webhook endpoints for your tenant.

Examples:
  waas endpoints list
  waas endpoints list -o json`,
	RunE: runEndpointsList,
}

var endpointsGetCmd = &cobra.Command{
	Use:   "get <endpoint-id>",
	Short: "Get details of a webhook endpoint",
	Long: `Get detailed information about a specific webhook endpoint.

Examples:
  waas endpoints get ep_123abc`,
	Args: cobra.ExactArgs(1),
	RunE: runEndpointsGet,
}

var endpointsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new webhook endpoint",
	Long: `Create a new webhook endpoint.

Examples:
  waas endpoints create --url https://example.com/webhook
  waas endpoints create --url https://example.com/webhook --header "X-Custom: value"`,
	RunE: runEndpointsCreate,
}

var endpointsDeleteCmd = &cobra.Command{
	Use:   "delete <endpoint-id>",
	Short: "Delete a webhook endpoint",
	Long: `Delete a webhook endpoint.

Examples:
  waas endpoints delete ep_123abc`,
	Args: cobra.ExactArgs(1),
	RunE: runEndpointsDelete,
}

var endpointsUpdateCmd = &cobra.Command{
	Use:   "update <endpoint-id>",
	Short: "Update a webhook endpoint",
	Long: `Update properties of an existing webhook endpoint.

Examples:
  waas endpoints update ep_123 --url https://new.example.com/webhook
  waas endpoints update ep_123 --active false
  waas endpoints update ep_123 --url https://new.example.com --active true`,
	Args: cobra.ExactArgs(1),
	RunE: runEndpointsUpdate,
}

var (
	endpointURL      string
	endpointHeaders  []string
	maxAttempts      int
	initialDelay     int
	maxDelay         int
	endpointActive   string
	endpointUpdateURL string
)

func init() {
	rootCmd.AddCommand(endpointsCmd)

	endpointsCmd.AddCommand(endpointsListCmd)
	endpointsCmd.AddCommand(endpointsGetCmd)
	endpointsCmd.AddCommand(endpointsCreateCmd)
	endpointsCmd.AddCommand(endpointsDeleteCmd)
	endpointsCmd.AddCommand(endpointsUpdateCmd)

	endpointsCreateCmd.Flags().StringVar(&endpointURL, "url", "", "Webhook endpoint URL (required)")
	endpointsCreateCmd.Flags().StringArrayVar(&endpointHeaders, "header", nil, "Custom headers (format: Key: Value)")
	endpointsCreateCmd.Flags().IntVar(&maxAttempts, "max-attempts", 5, "Maximum retry attempts")
	endpointsCreateCmd.Flags().IntVar(&initialDelay, "initial-delay", 1000, "Initial retry delay in milliseconds")
	endpointsCreateCmd.Flags().IntVar(&maxDelay, "max-delay", 300000, "Maximum retry delay in milliseconds")
	endpointsCreateCmd.MarkFlagRequired("url")

	endpointsUpdateCmd.Flags().StringVar(&endpointUpdateURL, "url", "", "New webhook endpoint URL")
	endpointsUpdateCmd.Flags().StringVar(&endpointActive, "active", "", "Set endpoint active status (true/false)")
	endpointsUpdateCmd.Flags().StringArrayVar(&endpointHeaders, "header", nil, "Custom headers (format: Key: Value)")
}

func runEndpointsList(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	endpoints, err := client.ListEndpoints()
	if err != nil {
		return fmt.Errorf("failed to list endpoints: %w", err)
	}

	switch output {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(endpoints)
	case "csv":
		headers := []string{"ID", "URL", "Status", "Created"}
		var rows [][]string
		for _, ep := range endpoints {
			status := "active"
			if !ep.IsActive {
				status = "inactive"
			}
			rows = append(rows, []string{ep.ID, ep.URL, status, ep.CreatedAt.Format("2006-01-02 15:04")})
		}
		w := csvWriter(os.Stdout)
		w.Write(headers)
		for _, r := range rows {
			w.Write(r)
		}
		w.Flush()
		return w.Error()
	}

	if len(endpoints) == 0 {
		fmt.Println("No endpoints found. Create one with: waas endpoints create --url <url>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tURL\tSTATUS\tCREATED")
	for _, ep := range endpoints {
		status := "active"
		if !ep.IsActive {
			status = "inactive"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			ep.ID,
			truncate(ep.URL, 50),
			status,
			ep.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	return w.Flush()
}

func runEndpointsGet(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	endpoint, err := client.GetEndpoint(args[0])
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(endpoint)
	}

	fmt.Printf("ID:         %s\n", endpoint.ID)
	fmt.Printf("URL:        %s\n", endpoint.URL)
	fmt.Printf("Status:     %s\n", boolToStatus(endpoint.IsActive))
	fmt.Printf("Created:    %s\n", endpoint.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:    %s\n", endpoint.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(endpoint.CustomHeaders) > 0 {
		fmt.Println("\nCustom Headers:")
		for k, v := range endpoint.CustomHeaders {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	if endpoint.RetryConfig != nil {
		fmt.Println("\nRetry Configuration:")
		fmt.Printf("  Max Attempts:  %d\n", endpoint.RetryConfig.MaxAttempts)
		fmt.Printf("  Initial Delay: %dms\n", endpoint.RetryConfig.InitialDelay)
		fmt.Printf("  Max Delay:     %dms\n", endpoint.RetryConfig.MaxDelay)
	}

	return nil
}

func runEndpointsCreate(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	headers := make(map[string]string)
	for _, h := range endpointHeaders {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	req := &CreateEndpointRequest{
		URL:           endpointURL,
		CustomHeaders: headers,
		RetryConfig: &RetryConfig{
			MaxAttempts:       maxAttempts,
			InitialDelay:      initialDelay,
			MaxDelay:          maxDelay,
			BackoffMultiplier: 2,
		},
	}

	endpoint, err := client.CreateEndpoint(req)
	if err != nil {
		return fmt.Errorf("failed to create endpoint: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(endpoint)
	}

	fmt.Printf("✓ Endpoint created successfully\n")
	fmt.Printf("  ID:  %s\n", endpoint.ID)
	fmt.Printf("  URL: %s\n", endpoint.URL)

	return nil
}

func runEndpointsDelete(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	if err := client.DeleteEndpoint(args[0]); err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}

	fmt.Printf("✓ Endpoint %s deleted successfully\n", args[0])
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func boolToStatus(b bool) string {
	if b {
		return "active"
	}
	return "inactive"
}

func runEndpointsUpdate(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())
	id := args[0]

	req := &UpdateEndpointRequest{}
	hasChanges := false

	if endpointUpdateURL != "" {
		req.URL = endpointUpdateURL
		hasChanges = true
	}

	if endpointActive != "" {
		active := endpointActive == "true"
		req.IsActive = &active
		hasChanges = true
	}

	if len(endpointHeaders) > 0 {
		headers := make(map[string]string)
		for _, h := range endpointHeaders {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		req.CustomHeaders = headers
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified. Use --url, --active, or --header flags")
	}

	endpoint, err := client.UpdateEndpoint(id, req)
	if err != nil {
		return fmt.Errorf("failed to update endpoint: %w", err)
	}

	if output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(endpoint)
	}

	fmt.Printf("✓ Endpoint updated successfully\n")
	fmt.Printf("  ID:     %s\n", endpoint.ID)
	fmt.Printf("  URL:    %s\n", endpoint.URL)
	fmt.Printf("  Status: %s\n", boolToStatus(endpoint.IsActive))

	return nil
}

func csvWriter(w io.Writer) *csv.Writer {
	return csv.NewWriter(w)
}
