package fanout

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FilterEngine evaluates JSONPath-like filter expressions against JSON payloads
type FilterEngine struct{}

// NewFilterEngine creates a new FilterEngine
func NewFilterEngine() *FilterEngine {
	return &FilterEngine{}
}

// Evaluate checks if the payload matches the given filter expression.
// An empty expression always matches.
func (f *FilterEngine) Evaluate(expression string, payload json.RawMessage) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return true, nil
	}

	var data interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return false, fmt.Errorf("invalid JSON payload: %w", err)
	}

	return f.evaluateExpression(expression, data)
}

// Validate checks if a filter expression is syntactically valid
func (f *FilterEngine) Validate(expression string) error {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil
	}

	_, _, _, err := f.parseExpression(expression)
	return err
}

// evaluateExpression parses and evaluates a single filter expression
func (f *FilterEngine) evaluateExpression(expression string, data interface{}) (bool, error) {
	path, op, value, err := f.parseExpression(expression)
	if err != nil {
		return false, err
	}

	fieldValue, err := f.resolvePath(path, data)
	if err != nil {
		return false, nil // field not found means no match
	}

	return f.compare(fieldValue, op, value)
}

// parseExpression parses "$.field.path op value" into components
func (f *FilterEngine) parseExpression(expression string) (path string, op string, value string, err error) {
	operators := []string{" in ", " == ", " != ", " >= ", " <= ", " > ", " < "}
	for _, operator := range operators {
		idx := strings.Index(expression, operator)
		if idx >= 0 {
			path = strings.TrimSpace(expression[:idx])
			op = strings.TrimSpace(operator)
			value = strings.TrimSpace(expression[idx+len(operator):])
			return path, op, value, nil
		}
	}
	return "", "", "", fmt.Errorf("invalid filter expression: no operator found in %q", expression)
}

// resolvePath resolves a JSONPath-like path (e.g., $.metadata.region) against data
func (f *FilterEngine) resolvePath(path string, data interface{}) (interface{}, error) {
	// Strip leading "$."
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot traverse into non-object at %q", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("field %q not found", part)
		}
		current = val
	}

	return current, nil
}

// compare compares a field value with the expected value using the given operator
func (f *FilterEngine) compare(fieldValue interface{}, op string, rawExpected string) (bool, error) {
	if op == "in" {
		return f.compareIn(fieldValue, rawExpected)
	}

	// Parse the expected value
	expected := f.parseValue(rawExpected)

	switch op {
	case "==":
		return f.equals(fieldValue, expected), nil
	case "!=":
		return !f.equals(fieldValue, expected), nil
	case ">":
		return f.numericCompare(fieldValue, expected, func(a, b float64) bool { return a > b })
	case ">=":
		return f.numericCompare(fieldValue, expected, func(a, b float64) bool { return a >= b })
	case "<":
		return f.numericCompare(fieldValue, expected, func(a, b float64) bool { return a < b })
	case "<=":
		return f.numericCompare(fieldValue, expected, func(a, b float64) bool { return a <= b })
	default:
		return false, fmt.Errorf("unsupported operator: %q", op)
	}
}

// compareIn checks if a value is in a set like ["active", "pending"]
func (f *FilterEngine) compareIn(fieldValue interface{}, rawSet string) (bool, error) {
	rawSet = strings.TrimSpace(rawSet)
	if !strings.HasPrefix(rawSet, "[") || !strings.HasSuffix(rawSet, "]") {
		return false, fmt.Errorf("in operator requires an array value, got %q", rawSet)
	}

	var items []interface{}
	if err := json.Unmarshal([]byte(rawSet), &items); err != nil {
		return false, fmt.Errorf("invalid array for in operator: %w", err)
	}

	for _, item := range items {
		if f.equals(fieldValue, item) {
			return true, nil
		}
	}

	return false, nil
}

// parseValue parses a string value into its native type
func (f *FilterEngine) parseValue(raw string) interface{} {
	// Try unquoting string
	if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) {
		s, err := strconv.Unquote(raw)
		if err == nil {
			return s
		}
	}

	// Try number
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		return n
	}

	// Try boolean
	if raw == "true" {
		return true
	}
	if raw == "false" {
		return false
	}

	if raw == "null" {
		return nil
	}

	return raw
}

// equals compares two values for equality
func (f *FilterEngine) equals(a, b interface{}) bool {
	// Normalize numbers for comparison
	aNum, aIsNum := f.toFloat64(a)
	bNum, bIsNum := f.toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// numericCompare compares two values numerically
func (f *FilterEngine) numericCompare(a, b interface{}, cmp func(float64, float64) bool) (bool, error) {
	aNum, aOk := f.toFloat64(a)
	bNum, bOk := f.toFloat64(b)
	if !aOk || !bOk {
		return false, fmt.Errorf("numeric comparison requires numeric operands")
	}
	return cmp(aNum, bNum), nil
}

// toFloat64 converts an interface{} to float64 if possible
func (f *FilterEngine) toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f64, err := n.Float64()
		return f64, err == nil
	case string:
		f64, err := strconv.ParseFloat(n, 64)
		return f64, err == nil
	}
	return 0, false
}
