package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	out "webhook-platform/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data from other webhook providers",
	Long: `Migrate webhook configurations and data from other providers such as Svix, Convoy, or flat files.

Examples:
  waas migrate start --source svix --file export.json
  waas migrate start --source csv --file endpoints.csv --dry-run
  waas migrate status
  waas migrate rollback`,
}

var migrateStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new migration",
	Long: `Start migrating data from another webhook provider or file.

Supported sources: svix, convoy, csv, json

Examples:
  waas migrate start --source svix --file svix-export.json
  waas migrate start --source convoy --file convoy-export.json
  waas migrate start --source csv --file endpoints.csv
  waas migrate start --source json --file data.json --dry-run`,
	RunE: runMigrateStart,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration progress",
	Long: `Display the status and progress of current or recent migrations.

Examples:
  waas migrate status`,
	RunE: runMigrateStatus,
}

var migrateRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Roll back the last migration",
	Long: `Roll back the most recent migration, removing imported data.

Examples:
  waas migrate rollback`,
	RunE: runMigrateRollback,
}

var (
	migrateSource string
	migrateFile   string
	migrateDryRun bool
	migrateJobID  string
)

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.AddCommand(migrateStartCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateRollbackCmd)

	migrateStartCmd.Flags().StringVar(&migrateSource, "source", "", "Migration source: svix, convoy, csv, json (required)")
	migrateStartCmd.Flags().StringVar(&migrateFile, "file", "", "Path to the import file (required)")
	migrateStartCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Simulate migration without making changes")
	migrateStartCmd.MarkFlagRequired("source")
	migrateStartCmd.MarkFlagRequired("file")

	migrateRollbackCmd.Flags().StringVar(&migrateJobID, "job-id", "", "Migration job ID to rollback (required)")
	migrateRollbackCmd.MarkFlagRequired("job-id")
}

func runMigrateStart(cmd *cobra.Command, args []string) error {
	// Validate source
	validSources := map[string]bool{"svix": true, "convoy": true, "csv": true, "json": true}
	if !validSources[migrateSource] {
		return fmt.Errorf("invalid source %q: must be one of svix, convoy, csv, json", migrateSource)
	}

	// Validate file exists
	if _, err := os.Stat(migrateFile); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", migrateFile)
	}

	client := NewClient(getAPIURL(), getAPIKey())

	result, err := client.StartMigration(migrateSource, migrateFile, migrateDryRun)
	if err != nil {
		return fmt.Errorf("failed to start migration: %w", err)
	}

	if output == "json" {
		return jsonOutput(result)
	}

	if migrateDryRun {
		out.PrintWarning("Dry run mode — no changes will be applied")
	}

	out.PrintSuccess("Migration started")
	if v, ok := result["migration_id"]; ok {
		out.PrintKeyValue("Migration ID", fmt.Sprintf("%v", v))
	}
	out.PrintKeyValue("Source", migrateSource)
	if v, ok := result["status"]; ok {
		out.PrintKeyValue("Status", out.ColorStatus(fmt.Sprintf("%v", v)))
	}
	if v, ok := result["items_found"]; ok {
		out.PrintKeyValue("Items Found", fmt.Sprintf("%v", v))
	}
	fmt.Printf("\n  Track progress: waas migrate status\n")

	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	jobs, err := client.GetMigrationStatus()
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	if output == "json" {
		return jsonOutput(map[string]interface{}{"jobs": jobs})
	}

	if len(jobs) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE\tSTATUS\tPROGRESS\tFAILED\tCREATED")
	for _, m := range jobs {
		id := fmt.Sprintf("%v", m["id"])
		source := fmt.Sprintf("%v", m["source"])
		status := fmt.Sprintf("%v", m["status"])
		total := m["total"]
		completed := m["completed"]
		failed := fmt.Sprintf("%v", m["failed"])
		createdAt := fmt.Sprintf("%v", m["created_at"])
		progress := fmt.Sprintf("%v/%v", completed, total)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(id, 24),
			source,
			out.ColorStatus(status),
			progress,
			failed,
			createdAt,
		)
	}
	return w.Flush()
}

func runMigrateRollback(cmd *cobra.Command, args []string) error {
	client := NewClient(getAPIURL(), getAPIKey())

	result, err := client.RollbackMigration(migrateJobID)
	if err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	if output == "json" {
		return jsonOutput(result)
	}

	out.PrintSuccess("Migration rolled back")
	if v, ok := result["migration_id"]; ok {
		out.PrintKeyValue("Migration ID", fmt.Sprintf("%v", v))
	}
	if v, ok := result["status"]; ok {
		out.PrintKeyValue("Status", out.ColorStatus(fmt.Sprintf("%v", v)))
	}
	if v, ok := result["items_rolled_back"]; ok {
		out.PrintKeyValue("Items Removed", fmt.Sprintf("%v", v))
	}

	return nil
}
