package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Validator validates JSON payloads against schemas
type Validator struct {
	repo Repository
}

// NewValidator creates a new schema validator
func NewValidator(repo Repository) *Validator {
	return &Validator{repo: repo}
}

// Validate validates a payload against the schema assigned to an endpoint
func (v *Validator) Validate(ctx context.Context, tenantID, endpointID string, payload []byte) (*ValidationResult, error) {
	// Get schema assignment for endpoint
	assignment, err := v.repo.GetEndpointSchema(ctx, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint schema: %w", err)
	}

	if assignment == nil || assignment.ValidationMode == ValidationModeNone {
		return &ValidationResult{Valid: true}, nil
	}

	// Get the schema
	schema, err := v.repo.GetSchema(ctx, tenantID, assignment.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	if schema == nil {
		return &ValidationResult{Valid: true}, nil
	}

	// Get specific version if specified
	var jsonSchema json.RawMessage
	if assignment.SchemaVersion != "" {
		version, err := v.repo.GetVersion(ctx, assignment.SchemaID, assignment.SchemaVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema version: %w", err)
		}
		if version != nil {
			jsonSchema = version.JSONSchema
		}
	}

	if jsonSchema == nil {
		jsonSchema = schema.JSONSchema
	}

	// Validate the payload
	errors := v.validatePayload(payload, jsonSchema)

	return &ValidationResult{
		Valid:      len(errors) == 0,
		Errors:     errors,
		SchemaID:   schema.ID,
		SchemaName: schema.Name,
		Version:    schema.Version,
	}, nil
}

// ValidatePayloadDirect validates a payload directly against a JSON schema
func (v *Validator) ValidatePayloadDirect(payload []byte, jsonSchema json.RawMessage) *ValidationResult {
	errors := v.validatePayload(payload, jsonSchema)
	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func (v *Validator) validatePayload(payload []byte, jsonSchema json.RawMessage) []ValidationError {
	var errors []ValidationError

	// Parse payload
	var payloadData interface{}
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		errors = append(errors, ValidationError{
			Path:    "$",
			Message: "Invalid JSON: " + err.Error(),
			Type:    "parse_error",
		})
		return errors
	}

	// Parse schema
	var schemaData map[string]interface{}
	if err := json.Unmarshal(jsonSchema, &schemaData); err != nil {
		errors = append(errors, ValidationError{
			Path:    "$",
			Message: "Invalid schema: " + err.Error(),
			Type:    "schema_error",
		})
		return errors
	}

	// Validate
	v.validateValue(payloadData, schemaData, "$", &errors)

	return errors
}

func (v *Validator) validateValue(value interface{}, schema map[string]interface{}, path string, errors *[]ValidationError) {
	// Check type
	if expectedType, ok := schema["type"].(string); ok {
		actualType := getJSONType(value)
		if !typeMatches(actualType, expectedType) {
			*errors = append(*errors, ValidationError{
				Path:    path,
				Message: fmt.Sprintf("Expected type %s but got %s", expectedType, actualType),
				Type:    "type_mismatch",
			})
			return
		}
	}

	// Check required fields for objects
	if obj, ok := value.(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok {
			for _, req := range required {
				field := req.(string)
				if _, exists := obj[field]; !exists {
					*errors = append(*errors, ValidationError{
						Path:    path + "." + field,
						Message: fmt.Sprintf("Required field '%s' is missing", field),
						Type:    "required",
					})
				}
			}
		}

		// Validate properties
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for key, val := range obj {
				if propSchema, ok := properties[key].(map[string]interface{}); ok {
					v.validateValue(val, propSchema, path+"."+key, errors)
				}
			}
		}
	}

	// Check array items
	if arr, ok := value.([]interface{}); ok {
		if items, ok := schema["items"].(map[string]interface{}); ok {
			for i, item := range arr {
				v.validateValue(item, items, fmt.Sprintf("%s[%d]", path, i), errors)
			}
		}

		// Check minItems/maxItems
		if minItems, ok := schema["minItems"].(float64); ok {
			if float64(len(arr)) < minItems {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Array has %d items, minimum is %d", len(arr), int(minItems)),
					Type:    "min_items",
				})
			}
		}
		if maxItems, ok := schema["maxItems"].(float64); ok {
			if float64(len(arr)) > maxItems {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Array has %d items, maximum is %d", len(arr), int(maxItems)),
					Type:    "max_items",
				})
			}
		}
	}

	// Check string constraints
	if str, ok := value.(string); ok {
		if minLength, ok := schema["minLength"].(float64); ok {
			if float64(len(str)) < minLength {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("String length %d is less than minimum %d", len(str), int(minLength)),
					Type:    "min_length",
				})
			}
		}
		if maxLength, ok := schema["maxLength"].(float64); ok {
			if float64(len(str)) > maxLength {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("String length %d exceeds maximum %d", len(str), int(maxLength)),
					Type:    "max_length",
				})
			}
		}

		// Check enum
		if enum, ok := schema["enum"].([]interface{}); ok {
			found := false
			for _, e := range enum {
				if e == str {
					found = true
					break
				}
			}
			if !found {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Value '%s' is not one of the allowed values", str),
					Type:    "enum",
				})
			}
		}
	}

	// Check number constraints
	if num, ok := value.(float64); ok {
		if minimum, ok := schema["minimum"].(float64); ok {
			if num < minimum {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Value %v is less than minimum %v", num, minimum),
					Type:    "minimum",
				})
			}
		}
		if maximum, ok := schema["maximum"].(float64); ok {
			if num > maximum {
				*errors = append(*errors, ValidationError{
					Path:    path,
					Message: fmt.Sprintf("Value %v exceeds maximum %v", num, maximum),
					Type:    "maximum",
				})
			}
		}
	}
}

func getJSONType(value interface{}) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func typeMatches(actual, expected string) bool {
	if actual == expected {
		return true
	}
	// integer is a subset of number
	if expected == "integer" && actual == "number" {
		return true
	}
	return false
}

// CheckCompatibility checks if a new schema version is compatible with the previous
func (v *Validator) CheckCompatibility(oldSchema, newSchema json.RawMessage) (*CompatibilityResult, error) {
	var oldMap, newMap map[string]interface{}
	
	if err := json.Unmarshal(oldSchema, &oldMap); err != nil {
		return nil, fmt.Errorf("invalid old schema: %w", err)
	}
	if err := json.Unmarshal(newSchema, &newMap); err != nil {
		return nil, fmt.Errorf("invalid new schema: %w", err)
	}

	result := &CompatibilityResult{Compatible: true}
	
	v.checkBreakingChanges(oldMap, newMap, "$", result)

	return result, nil
}

func (v *Validator) checkBreakingChanges(old, new map[string]interface{}, path string, result *CompatibilityResult) {
	// Check for removed required fields
	if oldRequired, ok := old["required"].([]interface{}); ok {
		newRequired, _ := new["required"].([]interface{})
		newRequiredSet := make(map[string]bool)
		for _, r := range newRequired {
			newRequiredSet[r.(string)] = true
		}
		
		for _, r := range oldRequired {
			field := r.(string)
			if !newRequiredSet[field] {
				// Field is no longer required - this is OK (backward compatible)
			}
		}
	}

	// Check for new required fields (breaking!)
	if newRequired, ok := new["required"].([]interface{}); ok {
		oldRequired, _ := old["required"].([]interface{})
		oldRequiredSet := make(map[string]bool)
		for _, r := range oldRequired {
			oldRequiredSet[r.(string)] = true
		}
		
		for _, r := range newRequired {
			field := r.(string)
			if !oldRequiredSet[field] {
				result.Compatible = false
				result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
					Type:        "required_added",
					Path:        path + ".required",
					Description: fmt.Sprintf("New required field '%s' added", field),
					NewValue:    field,
				})
			}
		}
	}

	// Check for type changes
	if oldType, ok := old["type"].(string); ok {
		if newType, ok := new["type"].(string); ok && oldType != newType {
			result.Compatible = false
			result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
				Type:        "type_change",
				Path:        path + ".type",
				Description: fmt.Sprintf("Type changed from '%s' to '%s'", oldType, newType),
				OldValue:    oldType,
				NewValue:    newType,
			})
		}
	}

	// Check for removed properties
	if oldProps, ok := old["properties"].(map[string]interface{}); ok {
		newProps, _ := new["properties"].(map[string]interface{})
		if newProps == nil {
			newProps = make(map[string]interface{})
		}
		
		for field := range oldProps {
			if _, exists := newProps[field]; !exists {
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("Field '%s' was removed from schema", strings.TrimPrefix(path+"."+field, "$.")))
			}
		}

		// Recursively check nested properties
		for field, oldProp := range oldProps {
			if newProp, exists := newProps[field]; exists {
				if oldPropMap, ok := oldProp.(map[string]interface{}); ok {
					if newPropMap, ok := newProp.(map[string]interface{}); ok {
						v.checkBreakingChanges(oldPropMap, newPropMap, path+"."+field, result)
					}
				}
			}
		}
	}
}
