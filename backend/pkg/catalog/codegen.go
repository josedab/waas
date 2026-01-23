package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GenerateGoTypes generates Go struct definitions from an event type's schema
func GenerateGoTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	structName := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString("package events\n\n")
	b.WriteString(fmt.Sprintf("// %s represents the %s event payload\n", structName, eventType.Name))
	b.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	props, _ := schema["properties"].(map[string]interface{})
	required := extractRequiredFields(schema)

	for name, def := range props {
		goType := jsonTypeToGo(def)
		fieldName := toGoName(name)
		tag := fmt.Sprintf("`json:\"%s", name)
		if !stringSliceContains(required, name) {
			tag += ",omitempty"
		}
		tag += "\"`"
		b.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, goType, tag))
	}

	b.WriteString("}\n")
	return b.String(), nil
}

// GeneratePythonTypes generates Python dataclass definitions from an event type's schema
func GeneratePythonTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	className := toPythonClassName(eventType.Name)
	var b strings.Builder
	b.WriteString("from __future__ import annotations\n")
	b.WriteString("from dataclasses import dataclass\n")
	b.WriteString("from typing import Optional\n\n\n")
	b.WriteString("@dataclass\n")
	b.WriteString(fmt.Sprintf("class %s:\n", className))
	b.WriteString(fmt.Sprintf("    \"\"\"%s event payload.\"\"\"\n\n", eventType.Name))

	props, _ := schema["properties"].(map[string]interface{})
	required := extractRequiredFields(schema)

	if len(props) == 0 {
		b.WriteString("    pass\n")
		return b.String(), nil
	}

	// Required fields first
	for name, def := range props {
		if stringSliceContains(required, name) {
			pyType := jsonTypeToPython(def)
			b.WriteString(fmt.Sprintf("    %s: %s\n", toSnakeCase(name), pyType))
		}
	}
	// Optional fields
	for name, def := range props {
		if !stringSliceContains(required, name) {
			pyType := jsonTypeToPython(def)
			b.WriteString(fmt.Sprintf("    %s: Optional[%s] = None\n", toSnakeCase(name), pyType))
		}
	}

	return b.String(), nil
}

// GenerateTypeScriptTypes generates TypeScript interface definitions from an event type's schema
func GenerateTypeScriptTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	interfaceName := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("/** %s event payload */\n", eventType.Name))
	b.WriteString(fmt.Sprintf("export interface %s {\n", interfaceName))

	props, _ := schema["properties"].(map[string]interface{})
	required := extractRequiredFields(schema)

	for name, def := range props {
		tsType := jsonTypeToTypeScript(def)
		optional := ""
		if !stringSliceContains(required, name) {
			optional = "?"
		}
		b.WriteString(fmt.Sprintf("  %s%s: %s;\n", name, optional, tsType))
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func parseSchema(eventType *EventType) (map[string]interface{}, error) {
	// Try Schema field from joined data first
	var schemaBytes []byte
	if eventType.Schema != nil && eventType.Schema.Schema != nil {
		schemaBytes = eventType.Schema.Schema
	} else if eventType.ExamplePayload != nil {
		// Fall back to generating from example payload structure
		return inferSchemaFromPayload(eventType.ExamplePayload)
	} else {
		return nil, fmt.Errorf("event type has no schema defined")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, fmt.Errorf("invalid schema JSON: %w", err)
	}
	return schema, nil
}

func inferSchemaFromPayload(payload json.RawMessage) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	properties := make(map[string]interface{})
	var required []interface{}
	for key, val := range data {
		properties[key] = map[string]interface{}{
			"type": inferJSONType(val),
		}
		required = append(required, key)
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}, nil
}

func inferJSONType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}

func jsonTypeToGo(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "interface{}"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		items := jsonTypeToGo(m["items"])
		return "[]" + items
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func jsonTypeToPython(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "Any"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		items := jsonTypeToPython(m["items"])
		return "list[" + items + "]"
	case "object":
		return "dict"
	default:
		return "Any"
	}
}

func jsonTypeToTypeScript(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "unknown"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		items := jsonTypeToTypeScript(m["items"])
		return items + "[]"
	case "object":
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}

func toGoName(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, ".", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func toPythonClassName(s string) string {
	return toGoName(s)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + 32)
		} else if r == '-' || r == '.' {
			result.WriteByte('_')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
