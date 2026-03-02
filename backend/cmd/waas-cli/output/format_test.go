package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout captures stdout output during fn execution.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// captureStderr captures stderr output during fn execution.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// =====================
// PrintTable
// =====================

func TestPrintTable_Basic(t *testing.T) {
	headers := []string{"ID", "Name", "Status"}
	rows := [][]string{
		{"1", "webhook-1", "active"},
		{"2", "webhook-2", "failed"},
	}

	out := captureStdout(t, func() {
		PrintTable(headers, rows)
	})

	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "Name")
	assert.Contains(t, out, "Status")
	assert.Contains(t, out, "webhook-1")
	assert.Contains(t, out, "webhook-2")
}

func TestPrintTable_EmptyData(t *testing.T) {
	headers := []string{"ID", "Name"}

	out := captureStdout(t, func() {
		PrintTable(headers, nil)
	})

	// Should still print headers
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "Name")
}

func TestPrintTable_SpecialCharacters(t *testing.T) {
	headers := []string{"URL", "Status"}
	rows := [][]string{
		{"https://example.com/webhook?key=val&foo=bar", "active"},
		{"http://localhost:8080/path/to/resource", "pending"},
	}

	out := captureStdout(t, func() {
		PrintTable(headers, rows)
	})

	assert.Contains(t, out, "https://example.com/webhook?key=val&foo=bar")
}

// =====================
// PrintJSON
// =====================

func TestPrintJSON_ValidOutput(t *testing.T) {
	data := map[string]interface{}{
		"id":     "test-1",
		"name":   "webhook",
		"active": true,
	}

	out := captureStdout(t, func() {
		err := PrintJSON(data)
		require.NoError(t, err)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "test-1", parsed["id"])
	assert.Equal(t, true, parsed["active"])
}

func TestPrintJSON_NestedStructures(t *testing.T) {
	data := map[string]interface{}{
		"endpoint": map[string]interface{}{
			"url":     "https://example.com",
			"headers": map[string]string{"X-Custom": "value"},
		},
		"retries": []int{1, 2, 3},
	}

	out := captureStdout(t, func() {
		err := PrintJSON(data)
		require.NoError(t, err)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err)

	endpoint, ok := parsed["endpoint"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com", endpoint["url"])
}

// =====================
// PrintSuccess / PrintError / PrintWarning / PrintInfo
// =====================

func TestPrintSuccess(t *testing.T) {
	out := captureStdout(t, func() {
		PrintSuccess("Webhook created")
	})
	assert.Contains(t, out, "✓")
	assert.Contains(t, out, "Webhook created")
}

func TestPrintError(t *testing.T) {
	out := captureStderr(t, func() {
		PrintError("connection failed")
	})
	assert.Contains(t, out, "✗")
	assert.Contains(t, out, "connection failed")
}

func TestPrintWarning(t *testing.T) {
	out := captureStderr(t, func() {
		PrintWarning("rate limit approaching")
	})
	assert.Contains(t, out, "⚠")
	assert.Contains(t, out, "rate limit approaching")
}

func TestPrintInfo(t *testing.T) {
	out := captureStdout(t, func() {
		PrintInfo("Processing...")
	})
	assert.Contains(t, out, "ℹ")
	assert.Contains(t, out, "Processing...")
}

// =====================
// PrintKeyValue
// =====================

func TestPrintKeyValue(t *testing.T) {
	out := captureStdout(t, func() {
		PrintKeyValue("Status", "active")
	})
	assert.Contains(t, out, "Status:")
	assert.Contains(t, out, "active")
}

// =====================
// PrintHeader
// =====================

func TestPrintHeader(t *testing.T) {
	out := captureStdout(t, func() {
		PrintHeader("Endpoints")
	})
	assert.Contains(t, out, "Endpoints")
	assert.Contains(t, out, "─")
}

// =====================
// ColorStatus
// =====================

func TestColorStatus(t *testing.T) {
	tests := []struct {
		status string
		color  string
	}{
		{"delivered", colorGreen},
		{"active", colorGreen},
		{"success", colorGreen},
		{"failed", colorRed},
		{"inactive", colorRed},
		{"error", colorRed},
		{"retrying", colorYellow},
		{"pending", colorYellow},
		{"warning", colorYellow},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := ColorStatus(tt.status)
			if tt.color != "" {
				assert.Contains(t, result, tt.color)
				assert.Contains(t, result, colorReset)
			} else {
				assert.Equal(t, tt.status, result)
			}
		})
	}
}

// =====================
// FormatCSV
// =====================

func TestFormatCSV(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{
		{"1", "webhook-1"},
		{"2", "webhook-2"},
	}

	out := captureStdout(t, func() {
		err := FormatCSV(headers, rows)
		require.NoError(t, err)
	})

	r := csv.NewReader(strings.NewReader(out))
	records, err := r.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 3) // header + 2 rows
	assert.Equal(t, []string{"ID", "Name"}, records[0])
}

// =====================
// Truncate
// =====================

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"very short max", "abcdefgh", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Truncate(tt.s, tt.max))
		})
	}
}

// =====================
// PrintYAML
// =====================

func TestPrintYAML(t *testing.T) {
	data := map[string]interface{}{
		"name":   "test",
		"active": true,
	}

	out := captureStdout(t, func() {
		err := PrintYAML(data)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "name: test")
	assert.Contains(t, out, "active: true")
}

// =====================
// PrintOutput
// =====================

func TestPrintOutput_Formats(t *testing.T) {
	headers := []string{"ID"}
	rows := [][]string{{"1"}}
	data := map[string]string{"ID": "1"}

	tests := []struct {
		format string
		check  func(string)
	}{
		{"json", func(out string) {
			assert.Contains(t, out, "ID")
		}},
		{"csv", func(out string) {
			assert.Contains(t, out, "ID")
		}},
		{"yaml", func(out string) {
			assert.Contains(t, out, "ID")
		}},
		{"table", func(out string) {
			assert.Contains(t, out, "ID")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			out := captureStdout(t, func() {
				PrintOutput(tt.format, headers, rows, data)
			})
			tt.check(out)
		})
	}
}
