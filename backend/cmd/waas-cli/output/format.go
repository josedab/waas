package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// PrintTable renders data in a formatted table with headers.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, colorBold+strings.Join(headers, "\t")+colorReset)
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// PrintJSON outputs data as pretty-printed JSON.
func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintSuccess prints a success message with a green checkmark.
func PrintSuccess(message string) {
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, message)
}

// PrintError prints an error message in red.
func PrintError(message string) {
	fmt.Fprintf(os.Stderr, "%s✗ Error:%s %s\n", colorRed, colorReset, message)
}

// PrintWarning prints a warning message in yellow.
func PrintWarning(message string) {
	fmt.Fprintf(os.Stderr, "%s⚠ Warning:%s %s\n", colorYellow, colorReset, message)
}

// PrintInfo prints an informational message in cyan.
func PrintInfo(message string) {
	fmt.Printf("%sℹ%s %s\n", colorCyan, colorReset, message)
}

// PrintKeyValue prints a labeled value with aligned formatting.
func PrintKeyValue(key, value string) {
	fmt.Printf("  %-16s %s\n", key+":", value)
}

// PrintHeader prints a section header with a separator line.
func PrintHeader(title string) {
	fmt.Printf("\n%s%s%s\n", colorBold, title, colorReset)
	fmt.Println(strings.Repeat("─", len(title)+4))
}

// ColorStatus returns a colorized status string.
func ColorStatus(status string) string {
	switch status {
	case "delivered", "active", "success":
		return colorGreen + status + colorReset
	case "failed", "inactive", "error":
		return colorRed + status + colorReset
	case "retrying", "pending", "warning":
		return colorYellow + status + colorReset
	default:
		return status
	}
}

// FormatCSV writes headers and rows as CSV to stdout.
func FormatCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	w.Flush()
	return w.Error()
}

// Truncate shortens a string to max length, appending "..." if truncated.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// PrintOutput selects the output format based on the format string and renders accordingly.
func PrintOutput(format string, headers []string, rows [][]string, data interface{}) {
	switch format {
	case "json":
		PrintJSON(data)
	case "csv":
		FormatCSV(headers, rows)
	default:
		PrintTable(headers, rows)
	}
}
