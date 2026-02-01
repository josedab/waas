package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var deliveryCmd = &cobra.Command{
	Use:   "delivery",
	Short: "Manage webhook deliveries",
	Long:  `List, inspect, and retry webhook deliveries.`,
}

var deliveryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent deliveries",
	Long: `List recent webhook deliveries with optional filters.

Examples:
  waas delivery list
  waas delivery list --status failed
  waas delivery list --endpoint ep_123 --limit 50
  waas delivery list -o json`,
	RunE: runDeliveryList,
}

var deliveryInspectCmd = &cobra.Command{
	Use:   "inspect <delivery-id>",
	Short: "Detailed delivery inspection",
	Long: `Inspect a webhook delivery in detail, including request/response data and attempt history.

Examples:
  waas delivery inspect del_abc123
  waas delivery inspect del_abc123 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runDeliveryInspect,
}

var deliveryRetryCmd = &cobra.Command{
	Use:   "retry <delivery-id>",
	Short: "Retry a failed delivery",
	Long: `Retry a previously failed webhook delivery.

Examples:
  waas delivery retry del_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runDeliveryRetry,
}

var (
	deliveryEndpoint string
	deliveryStatus   string
	deliveryLimit    int
)

func init() {
	rootCmd.AddCommand(deliveryCmd)

	deliveryCmd.AddCommand(deliveryListCmd)
	deliveryCmd.AddCommand(deliveryInspectCmd)
	deliveryCmd.AddCommand(deliveryRetryCmd)

	deliveryListCmd.Flags().StringVar(&deliveryEndpoint, "endpoint", "", "Filter by endpoint ID")
	deliveryListCmd.Flags().StringVar(&deliveryStatus, "status", "", "Filter by status (pending, delivered, failed, retrying)")
	deliveryListCmd.Flags().IntVar(&deliveryLimit, "limit", 20, "Maximum number of deliveries to show")
}

func runDeliveryList(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	deliveries, err := client.ListDeliveries(deliveryEndpoint, deliveryLimit)
	if err != nil {
		return fmt.Errorf("failed to list deliveries: %w", err)
	}

	// Filter by status if specified
	if deliveryStatus != "" {
		var filtered []Delivery
		for _, d := range deliveries {
			if d.Status == deliveryStatus {
				filtered = append(filtered, d)
			}
		}
		deliveries = filtered
	}

	if output == "json" {
		return jsonOutput(deliveries)
	}

	if len(deliveries) == 0 {
		fmt.Println("No deliveries found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tENDPOINT\tSTATUS\tATTEMPTS\tHTTP\tLAST ATTEMPT")
	for _, d := range deliveries {
		lastAttempt := "-"
		if !d.LastAttemptAt.IsZero() {
			lastAttempt = d.LastAttemptAt.Format("2006-01-02 15:04:05")
		}
		httpStatus := "-"
		if d.LastHTTPStatus > 0 {
			httpStatus = fmt.Sprintf("%d", d.LastHTTPStatus)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			truncate(d.ID, 24),
			truncate(d.EndpointID, 18),
			out.ColorStatus(d.Status),
			d.AttemptCount,
			httpStatus,
			lastAttempt,
		)
	}
	return w.Flush()
}

func runDeliveryInspect(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())
	deliveryID := args[0]

	detail, err := client.InspectDelivery(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to inspect delivery: %w", err)
	}

	if output == "json" {
		return jsonOutput(detail)
	}

	out.PrintHeader("Delivery Inspection")
	out.PrintKeyValue("ID", detail.Delivery.ID)
	out.PrintKeyValue("Endpoint", detail.Delivery.EndpointID)
	out.PrintKeyValue("Status", out.ColorStatus(detail.Delivery.Status))
	out.PrintKeyValue("Attempts", fmt.Sprintf("%d", detail.Delivery.AttemptCount))
	out.PrintKeyValue("Created", detail.Delivery.CreatedAt.Format(time.RFC3339))

	if detail.Delivery.LastError != "" {
		out.PrintKeyValue("Last Error", detail.Delivery.LastError)
	}

	if detail.Request != nil {
		out.PrintHeader("Request")
		out.PrintKeyValue("URL", detail.Request.URL)
		out.PrintKeyValue("Method", detail.Request.Method)
		if len(detail.Request.Headers) > 0 {
			fmt.Println("  Headers:")
			for k, v := range detail.Request.Headers {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
		if len(detail.Request.Payload) > 0 {
			fmt.Println("  Payload:")
			var pretty json.RawMessage
			if err := json.Unmarshal(detail.Request.Payload, &pretty); err == nil {
				formatted, _ := json.MarshalIndent(pretty, "    ", "  ")
				fmt.Printf("    %s\n", string(formatted))
			}
		}
	}

	if len(detail.Attempts) > 0 {
		out.PrintHeader("Attempt History")
		for _, a := range detail.Attempts {
			icon := "\033[32m✓\033[0m"
			if a.HTTPStatus >= 400 || a.ErrorMessage != "" {
				icon = "\033[31m✗\033[0m"
			}

			fmt.Printf("\n  %s Attempt #%d - %s\n", icon, a.AttemptNumber, a.CreatedAt.Format("15:04:05"))
			if a.HTTPStatus > 0 {
				fmt.Printf("    HTTP Status: %d\n", a.HTTPStatus)
			}
			if a.ErrorMessage != "" {
				fmt.Printf("    Error: %s\n", a.ErrorMessage)
			}
			if a.ResponseBody != "" && len(a.ResponseBody) < 500 {
				fmt.Printf("    Response: %s\n", a.ResponseBody)
			}
		}
	}

	return nil
}

func runDeliveryRetry(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())
	deliveryID := args[0]

	resp, err := client.RetryDelivery(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to retry delivery: %w", err)
	}

	if output == "json" {
		return jsonOutput(map[string]interface{}{
			"original_delivery_id": deliveryID,
			"new_delivery_id":      resp.DeliveryID,
			"status":               resp.Status,
		})
	}

	out.PrintSuccess("Delivery retried successfully")
	out.PrintKeyValue("Original ID", deliveryID)
	out.PrintKeyValue("New Delivery", resp.DeliveryID)
	out.PrintKeyValue("Status", resp.Status)
	fmt.Printf("\n  Track with: waas delivery inspect %s\n", resp.DeliveryID)

	return nil
}
