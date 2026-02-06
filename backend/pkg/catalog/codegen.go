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

// GenerateJavaTypes generates Java class definitions from an event type's schema
func GenerateJavaTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	className := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString("import com.fasterxml.jackson.annotation.JsonProperty;\n\n")
	b.WriteString(fmt.Sprintf("/** %s event payload. */\n", eventType.Name))
	b.WriteString(fmt.Sprintf("public class %s {\n\n", className))

	props, _ := schema["properties"].(map[string]interface{})
	for name, def := range props {
		jType := jsonTypeToJava(def)
		fieldName := toCamelCase(name)
		b.WriteString(fmt.Sprintf("    @JsonProperty(\"%s\")\n", name))
		b.WriteString(fmt.Sprintf("    private %s %s;\n\n", jType, fieldName))
	}

	// Getters/setters
	for name, def := range props {
		jType := jsonTypeToJava(def)
		fieldName := toCamelCase(name)
		getterName := "get" + toGoName(name)
		setterName := "set" + toGoName(name)
		b.WriteString(fmt.Sprintf("    public %s %s() { return %s; }\n", jType, getterName, fieldName))
		b.WriteString(fmt.Sprintf("    public void %s(%s %s) { this.%s = %s; }\n\n", setterName, jType, fieldName, fieldName, fieldName))
	}

	b.WriteString("}\n")
	return b.String(), nil
}

// GenerateRubyTypes generates Ruby class definitions from an event type's schema
func GenerateRubyTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	className := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString("# frozen_string_literal: true\n\n")
	b.WriteString(fmt.Sprintf("# %s event payload\n", eventType.Name))
	b.WriteString(fmt.Sprintf("class %s\n", className))

	props, _ := schema["properties"].(map[string]interface{})
	var attrNames []string
	for name := range props {
		attrNames = append(attrNames, ":"+toSnakeCase(name))
	}
	if len(attrNames) > 0 {
		b.WriteString(fmt.Sprintf("  attr_accessor %s\n\n", strings.Join(attrNames, ", ")))
	}

	b.WriteString("  def initialize(attrs = {})\n")
	for name := range props {
		snakeName := toSnakeCase(name)
		b.WriteString(fmt.Sprintf("    @%s = attrs['%s'] || attrs[:%s]\n", snakeName, name, snakeName))
	}
	b.WriteString("  end\n")
	b.WriteString("end\n")
	return b.String(), nil
}

// GeneratePHPTypes generates PHP class definitions from an event type's schema
func GeneratePHPTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	className := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString("<?php\n\n")
	b.WriteString(fmt.Sprintf("/** %s event payload. */\n", eventType.Name))
	b.WriteString(fmt.Sprintf("class %s\n{\n", className))

	props, _ := schema["properties"].(map[string]interface{})
	for name, def := range props {
		phpType := jsonTypeToPHP(def)
		b.WriteString(fmt.Sprintf("    public %s $%s;\n", phpType, toCamelCase(name)))
	}
	b.WriteString("\n    public function __construct(array $data = [])\n    {\n")
	for name := range props {
		camelName := toCamelCase(name)
		b.WriteString(fmt.Sprintf("        $this->%s = $data['%s'] ?? null;\n", camelName, name))
	}
	b.WriteString("    }\n}\n")
	return b.String(), nil
}

// GenerateCSharpTypes generates C# class definitions from an event type's schema
func GenerateCSharpTypes(eventType *EventType) (string, error) {
	schema, err := parseSchema(eventType)
	if err != nil {
		return "", err
	}

	className := toGoName(eventType.Name)
	var b strings.Builder
	b.WriteString("using System.Text.Json.Serialization;\n\n")
	b.WriteString(fmt.Sprintf("/// <summary>%s event payload.</summary>\n", eventType.Name))
	b.WriteString(fmt.Sprintf("public class %s\n{\n", className))

	props, _ := schema["properties"].(map[string]interface{})
	for name, def := range props {
		csType := jsonTypeToCSharp(def)
		propName := toGoName(name)
		b.WriteString(fmt.Sprintf("    [JsonPropertyName(\"%s\")]\n", name))
		b.WriteString(fmt.Sprintf("    public %s %s { get; set; }\n\n", csType, propName))
	}

	b.WriteString("}\n")
	return b.String(), nil
}

func jsonTypeToJava(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "Object"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "String"
	case "integer":
		return "Long"
	case "number":
		return "Double"
	case "boolean":
		return "Boolean"
	case "array":
		items := jsonTypeToJava(m["items"])
		return "java.util.List<" + items + ">"
	case "object":
		return "java.util.Map<String, Object>"
	default:
		return "Object"
	}
}

func jsonTypeToPHP(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "mixed"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "?string"
	case "integer":
		return "?int"
	case "number":
		return "?float"
	case "boolean":
		return "?bool"
	case "array":
		return "?array"
	case "object":
		return "?array"
	default:
		return "mixed"
	}
}

func jsonTypeToCSharp(fieldDef interface{}) string {
	m, ok := fieldDef.(map[string]interface{})
	if !ok {
		return "object"
	}
	t, _ := m["type"].(string)
	switch t {
	case "string":
		return "string?"
	case "integer":
		return "long?"
	case "number":
		return "double?"
	case "boolean":
		return "bool?"
	case "array":
		items := jsonTypeToCSharp(m["items"])
		return "List<" + items + ">?"
	case "object":
		return "Dictionary<string, object>?"
	default:
		return "object?"
	}
}

func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	if len(words) == 0 {
		return s
	}
	result := strings.ToLower(words[0])
	for _, w := range words[1:] {
		if len(w) > 0 {
			result += strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return result
}
