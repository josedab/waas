package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data to a file",
	Long: `Export webhook data (endpoints, deliveries, events) to a file.

Supported formats: json, csv

Examples:
  waas export --type endpoints --format json --output endpoints.json
  waas export --type deliveries --format csv --output deliveries.csv
  waas export --type events --format json --output events.json`,
	RunE: runExport,
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import data from a file",
	Long: `Import webhook data (endpoints, events) from a file.

Examples:
  waas import --type endpoints --file endpoints.json
  waas import --type events --file events.json --dry-run`,
	RunE: runImport,
}

var (
	exportFormat string
	exportOutput string
	exportType   string
	importFile   string
	importType   string
	importDryRun bool
)

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)

	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format: json, csv")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (required)")
	exportCmd.Flags().StringVar(&exportType, "type", "", "Data type to export: endpoints, deliveries, events (required)")
	exportCmd.MarkFlagRequired("output")
	exportCmd.MarkFlagRequired("type")

	importCmd.Flags().StringVar(&importFile, "file", "", "Path to the import file (required)")
	importCmd.Flags().StringVar(&importType, "type", "", "Data type to import: endpoints, events (required)")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Simulate import without making changes")
	importCmd.MarkFlagRequired("file")
	importCmd.MarkFlagRequired("type")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate type
	validTypes := map[string]bool{"endpoints": true, "deliveries": true, "events": true}
	if !validTypes[exportType] {
		return fmt.Errorf("invalid type %q: must be one of endpoints, deliveries, events", exportType)
	}

	// Validate format
	validFormats := map[string]bool{"json": true, "csv": true}
	if !validFormats[exportFormat] {
		return fmt.Errorf("invalid format %q: must be one of json, csv", exportFormat)
	}

	client := NewClient(getAPIURL(), getAPIKey())

	var data interface{}
	var err error

	switch exportType {
	case "endpoints":
		data, err = client.ExportEndpoints()
	case "deliveries":
		data, err = client.ExportDeliveries()
	case "events":
		// Events use the same deliveries endpoint
		data, err = client.ExportDeliveries()
	}
	if err != nil {
		return fmt.Errorf("failed to export %s: %w", exportType, err)
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(exportOutput, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if output == "json" {
		return jsonOutput(map[string]interface{}{
			"type":   exportType,
			"format": exportFormat,
			"file":   exportOutput,
			"bytes":  len(jsonBytes),
		})
	}

	out.PrintSuccess(fmt.Sprintf("Exported %s to %s", exportType, exportOutput))
	out.PrintKeyValue("Type", exportType)
	out.PrintKeyValue("Format", exportFormat)
	out.PrintKeyValue("File", exportOutput)
	out.PrintKeyValue("Size", fmt.Sprintf("%d bytes", len(jsonBytes)))

	return nil
}

func runImport(cmd *cobra.Command, args []string) error {
	// Validate type
	validTypes := map[string]bool{"endpoints": true, "events": true}
	if !validTypes[importType] {
		return fmt.Errorf("invalid type %q: must be one of endpoints, events", importType)
	}

	// Validate file exists
	if _, err := os.Stat(importFile); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", importFile)
	}

	fileContent, err := os.ReadFile(importFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var entries []map[string]interface{}
	if err := json.Unmarshal(fileContent, &entries); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	client := NewClient(getAPIURL(), getAPIKey())

	result, err := client.ImportEndpoints(entries, importDryRun)
	if err != nil {
		return fmt.Errorf("failed to import %s: %w", importType, err)
	}

	if output == "json" {
		return jsonOutput(result)
	}

	if importDryRun {
		out.PrintWarning("Dry run mode — no changes were applied")
	}

	out.PrintSuccess(fmt.Sprintf("Imported %s from %s", importType, importFile))
	if v, ok := result["type"]; ok {
		out.PrintKeyValue("Type", fmt.Sprintf("%v", v))
	}
	if v, ok := result["imported"]; ok {
		out.PrintKeyValue("Imported", fmt.Sprintf("%v", v))
	}
	if v, ok := result["skipped"]; ok {
		out.PrintKeyValue("Skipped", fmt.Sprintf("%v", v))
	}
	if v, ok := result["errors"]; ok {
		out.PrintKeyValue("Errors", fmt.Sprintf("%v", v))
	}

	return nil
}
