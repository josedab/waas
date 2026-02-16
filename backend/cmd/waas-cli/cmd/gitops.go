package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	out "github.com/josedab/waas/cmd/waas-cli/output"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply declarative configuration from YAML files",
	Long: `Apply webhook configuration defined in YAML manifest files.

Supports environment overlays, dry-run mode, and drift detection.

Examples:
  waas apply -f config.yaml                    # Apply a manifest
  waas apply -f config.yaml --dry-run          # Preview changes
  waas apply -f config/ --recursive            # Apply all files in directory
  waas apply -f config.yaml --env production   # Apply with env overlay`,
	RunE: runApply,
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between local config and live state",
	Long: `Compare a local YAML manifest against the current live configuration.

Examples:
  waas diff -f config.yaml                     # Show diff
  waas diff -f config.yaml --env staging       # Diff with env overlay`,
	RunE: runDiff,
}

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect configuration drift from declared state",
	Long: `Check if live configuration has drifted from the declared manifests.

Examples:
  waas drift                                   # Check for drift
  waas drift --fix                             # Auto-fix detected drift`,
	RunE: runDrift,
}

var (
	applyFile      string
	applyDryRun    bool
	applyForce     bool
	applyRecursive bool
	applyEnv       string
	diffFile       string
	diffEnv        string
	driftFix       bool
)

func init() {
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(driftCmd)

	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "Path to YAML manifest file or directory (required)")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes without applying")
	applyCmd.Flags().BoolVar(&applyForce, "force", false, "Force apply even with destructive changes")
	applyCmd.Flags().BoolVar(&applyRecursive, "recursive", false, "Recursively apply files in directory")
	applyCmd.Flags().StringVar(&applyEnv, "env", "", "Environment overlay (e.g., staging, production)")
	applyCmd.MarkFlagRequired("file")

	diffCmd.Flags().StringVarP(&diffFile, "file", "f", "", "Path to YAML manifest file (required)")
	diffCmd.Flags().StringVar(&diffEnv, "env", "", "Environment overlay")
	diffCmd.MarkFlagRequired("file")

	driftCmd.Flags().BoolVar(&driftFix, "fix", false, "Automatically re-apply manifests to fix drift")
}

func runApply(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	files, err := resolveFiles(applyFile, applyRecursive)
	if err != nil {
		return fmt.Errorf("failed to resolve files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no YAML files found at %s", applyFile)
	}

	totalApplied := 0
	totalFailed := 0

	for _, file := range files {
		content, err := readManifestWithOverlay(file, applyEnv)
		if err != nil {
			out.PrintError(fmt.Sprintf("Failed to read %s: %v", file, err))
			totalFailed++
			continue
		}

		if applyDryRun {
			fmt.Printf("📋 Dry-run: %s\n", file)
		} else {
			fmt.Printf("🔄 Applying: %s\n", file)
		}

		// Validate manifest
		resp, err := client.doRequest("POST", "/api/v1/gitops/manifests/validate", map[string]interface{}{
			"content": content,
		})
		if err != nil {
			out.PrintError(fmt.Sprintf("Validation failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		var validateResult map[string]interface{}
		if err := parseResponse(resp, &validateResult); err != nil {
			out.PrintError(fmt.Sprintf("Validation failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		// Upload manifest
		uploadResp, err := client.doRequest("POST", "/api/v1/gitops/manifests", map[string]interface{}{
			"name":    filepath.Base(file),
			"content": content,
		})
		if err != nil {
			out.PrintError(fmt.Sprintf("Upload failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		var manifest struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		}
		if err := parseResponse(uploadResp, &manifest); err != nil {
			out.PrintError(fmt.Sprintf("Upload failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		// Apply manifest
		applyReq := map[string]interface{}{
			"manifest_id": manifest.ID,
			"dry_run":     applyDryRun,
			"force":       applyForce,
		}

		applyResp, err := client.doRequest("POST", "/api/v1/gitops/manifests/apply", applyReq)
		if err != nil {
			out.PrintError(fmt.Sprintf("Apply failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		var result map[string]interface{}
		if err := parseResponse(applyResp, &result); err != nil {
			out.PrintError(fmt.Sprintf("Apply failed for %s: %v", file, err))
			totalFailed++
			continue
		}

		if output == "json" {
			out.PrintJSON(result)
		} else {
			status := "applied"
			if applyDryRun {
				status = "dry-run"
			}
			out.PrintSuccess(fmt.Sprintf("%s (%s, v%d)", file, status, manifest.Version))

			if resources, ok := result["resources"].([]interface{}); ok {
				for _, r := range resources {
					if rm, ok := r.(map[string]interface{}); ok {
						action := rm["action"]
						rType := rm["resource_type"]
						rID := rm["resource_id"]
						icon := "  "
						switch action {
						case "create":
							icon = "  + "
						case "update":
							icon = "  ~ "
						case "delete":
							icon = "  - "
						default:
							icon = "    "
						}
						fmt.Printf("%s%s/%s (%s)\n", icon, rType, rID, action)
					}
				}
			}
		}

		totalApplied++
	}

	fmt.Printf("\n%d applied, %d failed (out of %d files)\n", totalApplied, totalFailed, len(files))

	if totalFailed > 0 {
		os.Exit(1)
	}
	return nil
}

func runDiff(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	content, err := readManifestWithOverlay(diffFile, diffEnv)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", diffFile, err)
	}

	resp, err := client.doRequest("POST", "/api/v1/gitops/manifests/diff", map[string]interface{}{
		"content": content,
	})
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	if output == "json" {
		return out.PrintJSON(result)
	}

	out.PrintHeader("Configuration Diff: " + filepath.Base(diffFile))

	if diffs, ok := result["diffs"].([]interface{}); ok && len(diffs) > 0 {
		for _, d := range diffs {
			if dm, ok := d.(map[string]interface{}); ok {
				field := dm["field"]
				expected := dm["expected"]
				actual := dm["actual"]
				fmt.Printf("  \033[33m~\033[0m %s\n", field)
				fmt.Printf("    \033[31m- %v\033[0m\n", actual)
				fmt.Printf("    \033[32m+ %v\033[0m\n", expected)
			}
		}
	} else {
		out.PrintSuccess("No differences found — configuration is in sync")
	}

	return nil
}

func runDrift(cmd *cobra.Command, args []string) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}
	client := NewClient(getAPIURL(), apiKey)

	resp, err := client.doRequest("GET", "/api/v1/gitops/drift", nil)
	if err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	var result map[string]interface{}
	if err := parseResponse(resp, &result); err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	if output == "json" {
		return out.PrintJSON(result)
	}

	drifted, _ := result["drifted_count"].(float64)
	total, _ := result["resource_count"].(float64)

	if drifted == 0 {
		out.PrintSuccess(fmt.Sprintf("No drift detected (%d resources in sync)", int(total)))
		return nil
	}

	out.PrintWarning(fmt.Sprintf("Drift detected: %d/%d resources", int(drifted), int(total)))

	if details, ok := result["details"].([]interface{}); ok {
		for _, d := range details {
			if dm, ok := d.(map[string]interface{}); ok {
				rType := dm["resource_type"]
				rID := dm["resource_id"]
				field := dm["field"]
				expected := dm["expected_value"]
				actual := dm["actual_value"]
				fmt.Printf("  ⚠ %s/%s.%s: expected %v, got %v\n", rType, rID, field, expected, actual)
			}
		}
	}

	if driftFix {
		fmt.Println("\nRe-applying manifests to fix drift...")
		fixResp, err := client.doRequest("POST", "/api/v1/gitops/drift/fix", nil)
		if err != nil {
			return fmt.Errorf("failed to fix drift: %w", err)
		}
		var fixResult map[string]interface{}
		if err := parseResponse(fixResp, &fixResult); err != nil {
			return fmt.Errorf("failed to fix drift: %w", err)
		}
		out.PrintSuccess("Drift resolved")
	}

	return nil
}

// resolveFiles finds YAML files at the given path
func resolveFiles(path string, recursive bool) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	var files []string
	walkFn := func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && !recursive && p != path {
			return filepath.SkipDir
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, p)
		}
		return nil
	}

	if err := filepath.WalkDir(path, walkFn); err != nil {
		return nil, err
	}

	return files, nil
}

// validEnvName matches only alphanumeric characters and hyphens.
var validEnvName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// readManifestWithOverlay reads a YAML manifest and merges environment overlay
func readManifestWithOverlay(file, env string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	if env == "" {
		return string(content), nil
	}

	// Validate env to prevent path traversal
	if !validEnvName.MatchString(env) {
		return "", fmt.Errorf("invalid environment name %q: must contain only alphanumeric characters and hyphens", env)
	}

	// Look for environment overlay file: config.yaml -> config.staging.yaml
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	overlayFile := filepath.Clean(base + "." + env + ext)

	// Ensure the overlay stays within the same directory as the base file
	baseDir := filepath.Dir(file)
	if !strings.HasPrefix(overlayFile, baseDir+string(filepath.Separator)) && overlayFile != baseDir {
		return "", fmt.Errorf("overlay path %q escapes base directory %q", overlayFile, baseDir)
	}

	overlayContent, err := os.ReadFile(overlayFile)
	if os.IsNotExist(err) {
		// No overlay found, return base content
		return string(content), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read overlay %s: %w", overlayFile, err)
	}

	// Merge: base content with overlay appended (server-side merges)
	merged := string(content) + "\n---\n# Environment overlay: " + env + "\n" + string(overlayContent)
	return merged, nil
}
