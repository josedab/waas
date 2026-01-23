package catalog

import (
	"encoding/json"
	"fmt"
)

// CompatibilityChecker validates schema compatibility between versions
type CompatibilityChecker struct{}

// NewCompatibilityChecker creates a new CompatibilityChecker
func NewCompatibilityChecker() *CompatibilityChecker {
	return &CompatibilityChecker{}
}

// CheckBackwardCompatibility checks if a new schema can read data written with the old schema.
// New fields must be optional; required fields must not be removed.
func (c *CompatibilityChecker) CheckBackwardCompatibility(oldSchema, newSchema json.RawMessage) (bool, []string) {
	var oldMap, newMap map[string]interface{}
	if err := json.Unmarshal(oldSchema, &oldMap); err != nil {
		return false, []string{fmt.Sprintf("failed to parse old schema: %v", err)}
	}
	if err := json.Unmarshal(newSchema, &newMap); err != nil {
		return false, []string{fmt.Sprintf("failed to parse new schema: %v", err)}
	}

	var issues []string

	oldProps := extractProperties(oldMap)
	newProps := extractProperties(newMap)

	// Removed properties break backward compatibility
	for field := range oldProps {
		if _, exists := newProps[field]; !exists {
			issues = append(issues, fmt.Sprintf("field '%s' was removed", field))
		}
	}

	// Type changes break backward compatibility
	for field, oldDef := range oldProps {
		newDef, exists := newProps[field]
		if !exists {
			continue
		}
		oldType := extractFieldType(oldDef)
		newType := extractFieldType(newDef)
		if oldType != "" && newType != "" && oldType != newType {
			issues = append(issues, fmt.Sprintf("field '%s' type changed from '%s' to '%s'", field, oldType, newType))
		}
	}

	// New required fields that didn't exist before break backward compatibility
	oldRequired := extractRequiredFields(oldMap)
	newRequired := extractRequiredFields(newMap)
	for _, field := range newRequired {
		if !stringSliceContains(oldRequired, field) {
			if _, existedBefore := oldProps[field]; !existedBefore {
				issues = append(issues, fmt.Sprintf("new required field '%s' added", field))
			}
		}
	}

	return len(issues) == 0, issues
}

// CheckForwardCompatibility checks if the old schema can read data written with the new schema.
// Type changes are breaking; removed fields generate warnings.
func (c *CompatibilityChecker) CheckForwardCompatibility(oldSchema, newSchema json.RawMessage) (bool, []string) {
	var oldMap, newMap map[string]interface{}
	if err := json.Unmarshal(oldSchema, &oldMap); err != nil {
		return false, []string{fmt.Sprintf("failed to parse old schema: %v", err)}
	}
	if err := json.Unmarshal(newSchema, &newMap); err != nil {
		return false, []string{fmt.Sprintf("failed to parse new schema: %v", err)}
	}

	var issues []string

	oldProps := extractProperties(oldMap)
	newProps := extractProperties(newMap)

	// Type changes break forward compatibility
	for field, oldDef := range oldProps {
		newDef, exists := newProps[field]
		if !exists {
			continue
		}
		oldType := extractFieldType(oldDef)
		newType := extractFieldType(newDef)
		if oldType != "" && newType != "" && oldType != newType {
			issues = append(issues, fmt.Sprintf("field '%s' type changed from '%s' to '%s'", field, oldType, newType))
		}
	}

	// New required fields in the new schema that the old schema doesn't know about
	oldRequired := extractRequiredFields(oldMap)
	newRequired := extractRequiredFields(newMap)
	for _, field := range newRequired {
		if !stringSliceContains(oldRequired, field) {
			if _, existedBefore := oldProps[field]; !existedBefore {
				issues = append(issues, fmt.Sprintf("new required field '%s' unknown to old schema", field))
			}
		}
	}

	// Removed required fields break forward compat (old schema expects them)
	for _, field := range oldRequired {
		if _, exists := newProps[field]; !exists {
			issues = append(issues, fmt.Sprintf("required field '%s' removed from new schema", field))
		}
	}

	return len(issues) == 0, issues
}

// CheckFullCompatibility checks both backward and forward compatibility
func (c *CompatibilityChecker) CheckFullCompatibility(oldSchema, newSchema json.RawMessage) (bool, []string) {
	backCompat, backIssues := c.CheckBackwardCompatibility(oldSchema, newSchema)
	fwdCompat, fwdIssues := c.CheckForwardCompatibility(oldSchema, newSchema)

	var allIssues []string
	allIssues = append(allIssues, backIssues...)
	allIssues = append(allIssues, fwdIssues...)

	// Deduplicate issues
	seen := make(map[string]bool)
	var unique []string
	for _, issue := range allIssues {
		if !seen[issue] {
			seen[issue] = true
			unique = append(unique, issue)
		}
	}

	return backCompat && fwdCompat, unique
}

func extractProperties(schema map[string]interface{}) map[string]interface{} {
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return make(map[string]interface{})
	}
	return props
}

func extractRequiredFields(schema map[string]interface{}) []string {
	arr, ok := schema["required"].([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func extractFieldType(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return ""
	}
	t, ok := m["type"].(string)
	if !ok {
		return ""
	}
	return t
}

func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
