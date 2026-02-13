package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var replayCmd = &cobra.Command{
	Use:   "replay <delivery-id>",
	Short: "Replay a webhook delivery",
	Long: `Replay a previously sent webhook delivery.

This will create a new delivery attempt using the same payload and endpoint.

Examples:
  waas replay del_abc123
  waas replay del_abc123 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runReplay,
}

func init() {
	rootCmd.AddCommand(replayCmd)
}

func runReplay(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)
	deliveryID := args[0]

	// Get original delivery to show what we're replaying
	original, err := client.GetDelivery(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to get delivery: %w", err)
	}

	// Replay the delivery
	resp, err := client.ReplayDelivery(deliveryID)
	if err != nil {
		return fmt.Errorf("failed to replay delivery: %w", err)
	}

	if output == "json" {
		result := map[string]interface{}{
			"original_delivery_id": deliveryID,
			"new_delivery_id":      resp.DeliveryID,
			"status":               resp.Status,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("✓ Delivery replayed successfully\n")
	fmt.Printf("  Original Delivery: %s (%s)\n", deliveryID, original.Status)
	fmt.Printf("  New Delivery ID:   %s\n", resp.DeliveryID)
	fmt.Printf("  Status:            %s\n", resp.Status)
	fmt.Printf("\nTrack delivery with: waas logs %s\n", resp.DeliveryID)

	return nil
}

var bulkReplayCmd = &cobra.Command{
	Use:   "bulk-replay",
	Short: "Replay multiple failed deliveries",
	Long: `Replay multiple failed webhook deliveries matching the specified criteria.

Examples:
  waas bulk-replay --status failed --endpoint ep_123
  waas bulk-replay --status failed --since 2024-01-01`,
	RunE: runBulkReplay,
}

var (
	bulkReplayStatus   string
	bulkReplayEndpoint string
	bulkReplayLimit    int
	bulkReplayDryRun   bool
)

func init() {
	rootCmd.AddCommand(bulkReplayCmd)

	bulkReplayCmd.Flags().StringVar(&bulkReplayStatus, "status", "failed", "Filter by status")
	bulkReplayCmd.Flags().StringVar(&bulkReplayEndpoint, "endpoint", "", "Filter by endpoint ID")
	bulkReplayCmd.Flags().IntVar(&bulkReplayLimit, "limit", 100, "Maximum deliveries to replay")
	bulkReplayCmd.Flags().BoolVar(&bulkReplayDryRun, "dry-run", false, "Show what would be replayed without actually replaying")
}

func runBulkReplay(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	// Get deliveries matching criteria
	deliveries, err := client.ListDeliveries(bulkReplayEndpoint, bulkReplayLimit)
	if err != nil {
		return fmt.Errorf("failed to list deliveries: %w", err)
	}

	// Filter by status
	var toReplay []Delivery
	for _, d := range deliveries {
		if d.Status == bulkReplayStatus {
			toReplay = append(toReplay, d)
		}
	}

	if len(toReplay) == 0 {
		fmt.Println("No deliveries found matching criteria.")
		return nil
	}

	fmt.Printf("Found %d deliveries to replay\n", len(toReplay))

	if bulkReplayDryRun {
		fmt.Println("\n[DRY RUN] Would replay the following deliveries:")
		for _, d := range toReplay {
			fmt.Printf("  - %s (endpoint: %s)\n", d.ID, d.EndpointID)
		}
		return nil
	}

	// Replay each delivery
	var succeeded, failed int
	for _, d := range toReplay {
		_, err := client.ReplayDelivery(d.ID)
		if err != nil {
			fmt.Printf("✗ Failed to replay %s: %v\n", d.ID, err)
			failed++
		} else {
			fmt.Printf("✓ Replayed %s\n", d.ID)
			succeeded++
		}
	}

	fmt.Printf("\nSummary: %d succeeded, %d failed\n", succeeded, failed)

	return nil
}
