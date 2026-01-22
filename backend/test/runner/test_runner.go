package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TestSuite represents a test suite configuration
type TestSuite struct {
	Name        string
	Path        string
	EnvVars     map[string]string
	Timeout     time.Duration
	Description string
}

// TestRunner manages and executes different test suites
type TestRunner struct {
	suites []TestSuite
	verbose bool
}

func main() {
	var (
		suiteFlag = flag.String("suite", "all", "Test suite to run (unit|integration|e2e|performance|chaos|all)")
		verboseFlag = flag.Bool("verbose", false, "Enable verbose output")
		listFlag = flag.Bool("list", false, "List available test suites")
	)
	flag.Parse()

	runner := NewTestRunner(*verboseFlag)
	
	if *listFlag {
		runner.listSuites()
		return
	}
	
	if err := runner.runSuite(*suiteFlag); err != nil {
		fmt.Printf("Test execution failed: %v\n", err)
		os.Exit(1)
	}
}

func NewTestRunner(verbose bool) *TestRunner {
	return &TestRunner{
		verbose: verbose,
		suites: []TestSuite{
			{
				Name:        "unit",
				Path:        "./...",
				EnvVars:     map[string]string{},
				Timeout:     5 * time.Minute,
				Description: "Unit tests for all packages",
			},
			{
				Name: "integration",
				Path: "./test/integration",
				EnvVars: map[string]string{
					"TEST_DATABASE_URL": "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable",
					"TEST_REDIS_URL":    "redis://localhost:6379/12",
				},
				Timeout:     10 * time.Minute,
				Description: "Integration tests with real database and Redis",
			},
			{
				Name: "e2e",
				Path: "./test/e2e",
				EnvVars: map[string]string{
					"TEST_DATABASE_URL": "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable",
					"TEST_REDIS_URL":    "redis://localhost:6379/15",
				},
				Timeout:     15 * time.Minute,
				Description: "End-to-end tests with complete system simulation",
			},
			{
				Name: "performance",
				Path: "./test/performance",
				EnvVars: map[string]string{
					"RUN_LOAD_TESTS":    "true",
					"TEST_DATABASE_URL": "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable",
					"TEST_REDIS_URL":    "redis://localhost:6379/14",
				},
				Timeout:     30 * time.Minute,
				Description: "Performance and load tests",
			},
			{
				Name: "chaos",
				Path: "./test/chaos",
				EnvVars: map[string]string{
					"RUN_CHAOS_TESTS":   "true",
					"TEST_DATABASE_URL": "postgres://postgres:password@localhost:5432/webhook_platform_test?sslmode=disable",
					"TEST_REDIS_URL":    "redis://localhost:6379/13",
				},
				Timeout:     20 * time.Minute,
				Description: "Chaos engineering tests for resilience validation",
			},
		},
	}
}

func (tr *TestRunner) listSuites() {
	fmt.Println("Available test suites:")
	fmt.Println()
	
	for _, suite := range tr.suites {
		fmt.Printf("  %-12s %s\n", suite.Name, suite.Description)
	}
	
	fmt.Println()
	fmt.Println("  all          Run all test suites in sequence")
	fmt.Println()
	fmt.Println("Usage: go run test/runner/test_runner.go -suite=<suite_name>")
}

func (tr *TestRunner) runSuite(suiteName string) error {
	if suiteName == "all" {
		return tr.runAllSuites()
	}
	
	suite := tr.findSuite(suiteName)
	if suite == nil {
		return fmt.Errorf("unknown test suite: %s", suiteName)
	}
	
	return tr.executeSuite(*suite)
}

func (tr *TestRunner) runAllSuites() error {
	fmt.Println("Running all test suites...")
	fmt.Println()
	
	results := make(map[string]error)
	totalStart := time.Now()
	
	for _, suite := range tr.suites {
		// Skip performance and chaos tests in "all" mode unless explicitly enabled
		if (suite.Name == "performance" || suite.Name == "chaos") && 
		   os.Getenv("RUN_ALL_TESTS") != "true" {
			fmt.Printf("Skipping %s tests (set RUN_ALL_TESTS=true to include)\n", suite.Name)
			continue
		}
		
		err := tr.executeSuite(suite)
		results[suite.Name] = err
		
		if err != nil {
			fmt.Printf("❌ %s tests failed: %v\n", suite.Name, err)
		} else {
			fmt.Printf("✅ %s tests passed\n", suite.Name)
		}
		fmt.Println()
	}
	
	// Print summary
	fmt.Println("=== Test Summary ===")
	fmt.Printf("Total execution time: %v\n", time.Since(totalStart))
	fmt.Println()
	
	passed := 0
	failed := 0
	
	for suiteName, err := range results {
		if err != nil {
			fmt.Printf("❌ %s: FAILED - %v\n", suiteName, err)
			failed++
		} else {
			fmt.Printf("✅ %s: PASSED\n", suiteName)
			passed++
		}
	}
	
	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	
	if failed > 0 {
		return fmt.Errorf("%d test suite(s) failed", failed)
	}
	
	return nil
}

func (tr *TestRunner) findSuite(name string) *TestSuite {
	for _, suite := range tr.suites {
		if suite.Name == name {
			return &suite
		}
	}
	return nil
}

func (tr *TestRunner) executeSuite(suite TestSuite) error {
	fmt.Printf("Running %s tests...\n", suite.Name)
	fmt.Printf("Description: %s\n", suite.Description)
	fmt.Printf("Path: %s\n", suite.Path)
	fmt.Printf("Timeout: %v\n", suite.Timeout)
	
	if len(suite.EnvVars) > 0 {
		fmt.Println("Environment variables:")
		for key, value := range suite.EnvVars {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}
	
	fmt.Println()
	
	start := time.Now()
	
	// Prepare command
	args := []string{"test"}
	
	if tr.verbose {
		args = append(args, "-v")
	}
	
	// Add timeout
	args = append(args, "-timeout", suite.Timeout.String())
	
	// Add coverage for unit tests
	if suite.Name == "unit" {
		args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
	}
	
	// Add race detection for integration tests
	if suite.Name == "integration" || suite.Name == "e2e" {
		args = append(args, "-race")
	}
	
	// Add path
	args = append(args, suite.Path)
	
	cmd := exec.Command("go", args...)
	
	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range suite.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Set up output handling
	if tr.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Capture output for summary
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Test output:\n%s\n", string(output))
			return fmt.Errorf("test execution failed: %v", err)
		}
		
		// Print summary line
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "PASS") || strings.Contains(line, "FAIL") {
				if strings.Contains(line, "coverage:") || strings.Contains(line, "ok") {
					fmt.Println(line)
				}
			}
		}
	}
	
	if !tr.verbose {
		// Run the command if not already run in verbose mode
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("test execution failed: %v", err)
		}
	}
	
	duration := time.Since(start)
	fmt.Printf("Completed in %v\n", duration)
	
	return nil
}

// Additional helper functions for test environment setup
func (tr *TestRunner) checkPrerequisites() error {
	// Check if Go is installed
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go is not installed or not in PATH")
	}
	
	// Check if required environment variables are set for database tests
	requiredEnvVars := []string{
		"TEST_DATABASE_URL",
		"TEST_REDIS_URL",
	}
	
	missing := []string{}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}
	
	if len(missing) > 0 {
		fmt.Printf("Warning: Missing environment variables: %s\n", strings.Join(missing, ", "))
		fmt.Println("Some tests may be skipped or use default values.")
	}
	
	return nil
}

func (tr *TestRunner) setupTestDatabase() error {
	// This could include database migration, test data setup, etc.
	fmt.Println("Setting up test database...")
	
	// Run migrations
	cmd := exec.Command("go", "run", "cmd/migrate/main.go", "-up")
	cmd.Env = append(os.Environ(), "DATABASE_URL="+os.Getenv("TEST_DATABASE_URL"))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run database migrations: %v", err)
	}
	
	return nil
}

func (tr *TestRunner) cleanupTestEnvironment() error {
	fmt.Println("Cleaning up test environment...")
	
	// Clean up test databases, temporary files, etc.
	// This is a placeholder for cleanup logic
	
	return nil
}