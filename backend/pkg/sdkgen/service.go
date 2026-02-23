package sdkgen

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides SDK generation business logic.
type Service struct{}

// NewService creates a new SDK generation service.
func NewService() *Service {
	return &Service{}
}

// GenerateSDK generates typed event classes from schemas.
func (s *Service) GenerateSDK(tenantID string, req *GenerateSDKRequest) (*GeneratedSDK, error) {
	if err := validateLanguage(req.Language); err != nil {
		return nil, err
	}
	if len(req.Schemas) == 0 {
		return nil, fmt.Errorf("at least one schema is required")
	}

	pkgName := req.PackageName
	if pkgName == "" {
		pkgName = "waas_events"
	}

	files := make(map[string]string)

	switch req.Language {
	case LangTypeScript:
		files = s.generateTypeScript(req.Schemas, pkgName)
	case LangPython:
		files = s.generatePython(req.Schemas, pkgName)
	case LangGo:
		files = s.generateGo(req.Schemas, pkgName)
	case LangJava:
		files = s.generateJava(req.Schemas, pkgName)
	}

	return &GeneratedSDK{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Language:  req.Language,
		Version:   "1.0.0",
		Files:     files,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Service) generateTypeScript(schemas []SchemaDefinition, pkgName string) map[string]string {
	files := make(map[string]string)
	var indexExports []string

	for _, schema := range schemas {
		typeName := toTypeName(schema.EventType)
		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("// Auto-generated from %s schema\n", schema.EventType))
		sb.WriteString(fmt.Sprintf("export interface %s {\n", typeName))

		for name, prop := range schema.Properties {
			tsType := toTSType(prop.Type, prop.Format)
			optional := ""
			if !prop.Required {
				optional = "?"
			}
			if prop.Description != "" {
				sb.WriteString(fmt.Sprintf("  /** %s */\n", prop.Description))
			}
			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", name, optional, tsType))
		}
		sb.WriteString("}\n")

		filename := toFileName(schema.EventType) + ".ts"
		files[filename] = sb.String()
		indexExports = append(indexExports, fmt.Sprintf("export * from './%s';", toFileName(schema.EventType)))
	}

	files["index.ts"] = strings.Join(indexExports, "\n") + "\n"
	return files
}

func (s *Service) generatePython(schemas []SchemaDefinition, pkgName string) map[string]string {
	files := make(map[string]string)
	var imports []string

	for _, schema := range schemas {
		className := toTypeName(schema.EventType)
		var sb strings.Builder

		sb.WriteString("from dataclasses import dataclass\nfrom typing import Optional\n\n")
		sb.WriteString(fmt.Sprintf("@dataclass\nclass %s:\n", className))
		sb.WriteString(fmt.Sprintf("    \"\"\"Auto-generated from %s schema.\"\"\"\n\n", schema.EventType))

		for name, prop := range schema.Properties {
			pyType := toPythonType(prop.Type, prop.Format)
			if !prop.Required {
				pyType = fmt.Sprintf("Optional[%s]", pyType)
			}
			sb.WriteString(fmt.Sprintf("    %s: %s\n", name, pyType))
		}

		filename := toFileName(schema.EventType) + ".py"
		files[filename] = sb.String()
		imports = append(imports, fmt.Sprintf("from .%s import %s", toFileName(schema.EventType), className))
	}

	files["__init__.py"] = strings.Join(imports, "\n") + "\n"
	return files
}

func (s *Service) generateGo(schemas []SchemaDefinition, pkgName string) map[string]string {
	files := make(map[string]string)

	for _, schema := range schemas {
		structName := toTypeName(schema.EventType)
		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
		sb.WriteString(fmt.Sprintf("// %s is auto-generated from %s schema.\n", structName, schema.EventType))
		sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

		for name, prop := range schema.Properties {
			goType := toGoType(prop.Type, prop.Format)
			jsonTag := name
			if !prop.Required {
				jsonTag += ",omitempty"
			}
			fieldName := toExportedName(name)
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, goType, jsonTag))
		}
		sb.WriteString("}\n")

		filename := toFileName(schema.EventType) + ".go"
		files[filename] = sb.String()
	}

	return files
}

func (s *Service) generateJava(schemas []SchemaDefinition, pkgName string) map[string]string {
	files := make(map[string]string)

	for _, schema := range schemas {
		className := toTypeName(schema.EventType)
		var sb strings.Builder

		sb.WriteString(fmt.Sprintf("package %s;\n\n", pkgName))
		sb.WriteString(fmt.Sprintf("/** Auto-generated from %s schema. */\n", schema.EventType))
		sb.WriteString(fmt.Sprintf("public class %s {\n", className))

		for name, prop := range schema.Properties {
			javaType := toJavaType(prop.Type, prop.Format)
			sb.WriteString(fmt.Sprintf("    private %s %s;\n", javaType, name))
		}

		sb.WriteString("\n")
		for name, prop := range schema.Properties {
			javaType := toJavaType(prop.Type, prop.Format)
			capName := toExportedName(name)
			sb.WriteString(fmt.Sprintf("    public %s get%s() { return this.%s; }\n", javaType, capName, name))
			sb.WriteString(fmt.Sprintf("    public void set%s(%s %s) { this.%s = %s; }\n", capName, javaType, name, name, name))
		}

		sb.WriteString("}\n")

		filename := className + ".java"
		files[filename] = sb.String()
	}

	return files
}

func toTypeName(eventType string) string {
	parts := strings.Split(eventType, ".")
	var result string
	for _, p := range parts {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

func toFileName(eventType string) string {
	return strings.ReplaceAll(eventType, ".", "_")
}

func toExportedName(name string) string {
	parts := strings.Split(name, "_")
	var result string
	for _, p := range parts {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

func toTSType(t, format string) string {
	switch t {
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "any[]"
	default:
		return "string"
	}
}

func toPythonType(t, format string) string {
	switch t {
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "list"
	default:
		return "str"
	}
}

func toGoType(t, format string) string {
	switch t {
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		return "[]interface{}"
	default:
		return "string"
	}
}

func toJavaType(t, format string) string {
	switch t {
	case "integer":
		return "Long"
	case "number":
		return "Double"
	case "boolean":
		return "Boolean"
	case "array":
		return "java.util.List<Object>"
	default:
		return "String"
	}
}

func validateLanguage(lang string) error {
	switch lang {
	case LangTypeScript, LangPython, LangGo, LangJava:
		return nil
	}
	return fmt.Errorf("unsupported language %q: must be typescript, python, go, or java", lang)
}
