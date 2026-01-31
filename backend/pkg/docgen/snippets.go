package docgen

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SnippetGenerator generates code snippets for webhook event handling in 6 languages
type SnippetGenerator struct{}

// NewSnippetGenerator creates a new snippet generator
func NewSnippetGenerator() *SnippetGenerator {
	return &SnippetGenerator{}
}

// GenerateAll generates code snippets for all supported languages
func (g *SnippetGenerator) GenerateAll(eventName string, schema json.RawMessage, example json.RawMessage) map[string]string {
	snippets := make(map[string]string)
	languages := []string{LangGo, LangPython, LangNodeJS, LangJava, LangRuby, LangPHP, LangCurl}
	for _, lang := range languages {
		snippets[lang] = g.Generate(lang, eventName, schema, example)
	}
	return snippets
}

// Generate creates a code snippet for a specific language
func (g *SnippetGenerator) Generate(language, eventName string, schema json.RawMessage, example json.RawMessage) string {
	exampleStr := "{}"
	if len(example) > 0 {
		var pretty json.RawMessage
		if json.Unmarshal(example, &pretty) == nil {
			prettyBytes, _ := json.MarshalIndent(pretty, "", "  ")
			exampleStr = string(prettyBytes)
		}
	}

	fields := extractFields(schema)

	switch language {
	case LangGo:
		return g.generateGo(eventName, exampleStr, fields)
	case LangPython:
		return g.generatePython(eventName, exampleStr, fields)
	case LangNodeJS:
		return g.generateNodeJS(eventName, exampleStr, fields)
	case LangJava:
		return g.generateJava(eventName, exampleStr, fields)
	case LangRuby:
		return g.generateRuby(eventName, exampleStr, fields)
	case LangPHP:
		return g.generatePHP(eventName, exampleStr, fields)
	case LangCurl:
		return g.generateCurl(eventName, exampleStr)
	default:
		return ""
	}
}

type schemaField struct {
	Name string
	Type string
	Desc string
}

func extractFields(schema json.RawMessage) []schemaField {
	if len(schema) == 0 {
		return nil
	}
	var s struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
	}
	if json.Unmarshal(schema, &s) != nil || s.Properties == nil {
		return nil
	}

	var fields []schemaField
	for name, prop := range s.Properties {
		fields = append(fields, schemaField{Name: name, Type: prop.Type, Desc: prop.Description})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

func (g *SnippetGenerator) generateGo(eventName, example string, fields []schemaField) string {
	structName := toPascalCase(eventName)
	var fieldLines []string
	for _, f := range fields {
		goType := jsonTypeToGo(f.Type)
		tag := fmt.Sprintf("`json:\"%s\"`", f.Name)
		fieldLines = append(fieldLines, fmt.Sprintf("\t%s %s %s", toPascalCase(f.Name), goType, tag))
	}

	return fmt.Sprintf(`// Handle %s webhook event
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type %sPayload struct {
%s
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload %sPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received %s event: %%+v\n", payload)

	// Process the event...

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`+"`"+`{"received": true}`+"`"+`))
}

/*
Example payload:
%s
*/`, eventName, structName, strings.Join(fieldLines, "\n"), structName, eventName, example)
}

func (g *SnippetGenerator) generatePython(eventName, example string, fields []schemaField) string {
	return fmt.Sprintf(`# Handle %s webhook event
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/webhooks', methods=['POST'])
def handle_webhook():
    payload = request.get_json()

    event_type = payload.get('event', '')
    if event_type == '%s':
        process_%s(payload)

    return jsonify({"received": True}), 200

def process_%s(payload):
    """Process %s event."""
    print(f"Received %s: {payload}")
    # Add your business logic here

"""
Example payload:
%s
"""`, eventName, eventName,
		strings.ReplaceAll(eventName, ".", "_"),
		strings.ReplaceAll(eventName, ".", "_"),
		eventName, eventName, example)
}

func (g *SnippetGenerator) generateNodeJS(eventName, example string, fields []schemaField) string {
	var typeFields []string
	for _, f := range fields {
		tsType := jsonTypeToTS(f.Type)
		typeFields = append(typeFields, fmt.Sprintf("  %s: %s;", f.Name, tsType))
	}
	typeBlock := ""
	if len(typeFields) > 0 {
		typeName := toPascalCase(eventName) + "Payload"
		typeBlock = fmt.Sprintf("\ninterface %s {\n%s\n}\n", typeName, strings.Join(typeFields, "\n"))
	}

	return fmt.Sprintf(`// Handle %s webhook event
import express from 'express';
%s
const app = express();
app.use(express.json());

app.post('/webhooks', (req, res) => {
  const payload = req.body;

  if (payload.event === '%s') {
    console.log('Received %s:', JSON.stringify(payload, null, 2));
    // Add your business logic here
  }

  res.json({ received: true });
});

app.listen(3000, () => console.log('Webhook server on :3000'));

/*
Example payload:
%s
*/`, eventName, typeBlock, eventName, eventName, example)
}

func (g *SnippetGenerator) generateJava(eventName, example string, fields []schemaField) string {
	return fmt.Sprintf(`// Handle %s webhook event
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import java.util.Map;

@RestController
public class WebhookController {

    @PostMapping("/webhooks")
    public ResponseEntity<Map<String, Boolean>> handleWebhook(
            @RequestBody Map<String, Object> payload) {

        String eventType = (String) payload.get("event");
        if ("%s".equals(eventType)) {
            System.out.println("Received %s: " + payload);
            // Add your business logic here
        }

        return ResponseEntity.ok(Map.of("received", true));
    }
}

/*
Example payload:
%s
*/`, eventName, eventName, eventName, example)
}

func (g *SnippetGenerator) generateRuby(eventName, example string, fields []schemaField) string {
	return fmt.Sprintf(`# Handle %s webhook event
require 'sinatra'
require 'json'

post '/webhooks' do
  payload = JSON.parse(request.body.read)

  case payload['event']
  when '%s'
    puts "Received %s: #{payload.inspect}"
    # Add your business logic here
  end

  content_type :json
  { received: true }.to_json
end

# Example payload:
# %s`, eventName, eventName, eventName, strings.ReplaceAll(example, "\n", "\n# "))
}

func (g *SnippetGenerator) generatePHP(eventName, example string, fields []schemaField) string {
	return fmt.Sprintf(`<?php
// Handle %s webhook event

$payload = json_decode(file_get_contents('php://input'), true);

if ($payload['event'] === '%s') {
    error_log('Received %s: ' . json_encode($payload));
    // Add your business logic here
}

http_response_code(200);
header('Content-Type: application/json');
echo json_encode(['received' => true]);

/*
Example payload:
%s
*/`, eventName, eventName, eventName, example)
}

func (g *SnippetGenerator) generateCurl(eventName, example string) string {
	return fmt.Sprintf(`# Send a test %s webhook
curl -X POST https://your-app.com/webhooks \
  -H "Content-Type: application/json" \
  -H "X-WaaS-Signature: v1=<signature>" \
  -H "X-WaaS-Timestamp: $(date +%%s)" \
  -d '%s'`, eventName, strings.ReplaceAll(example, "'", "'\\''"))
}

// GenerateStaticSite generates a complete static documentation site as HTML
func GenerateStaticSite(doc *WebhookDoc, events []EventTypeDoc) string {
	var b strings.Builder
	g := NewSnippetGenerator()

	b.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - Webhook Documentation</title>
  <style>
    :root { --primary: #6366f1; --bg: #fff; --surface: #f9fafb; --text: #111827; --muted: #6b7280; --border: #e5e7eb; }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; color: var(--text); line-height: 1.6; }
    .container { max-width: 960px; margin: 0 auto; padding: 40px 20px; }
    h1 { font-size: 2em; margin-bottom: 8px; }
    h2 { font-size: 1.4em; margin: 32px 0 12px; padding-bottom: 8px; border-bottom: 1px solid var(--border); }
    h3 { font-size: 1.1em; margin: 20px 0 8px; }
    .subtitle { color: var(--muted); margin-bottom: 32px; }
    .badge { display: inline-block; padding: 2px 10px; border-radius: 12px; font-size: 12px; font-weight: 600; }
    .badge--event { background: #dbeafe; color: #1e40af; }
    .badge--deprecated { background: #fee2e2; color: #991b1b; }
    .field-table { width: 100%%; border-collapse: collapse; margin: 12px 0; }
    .field-table th { text-align: left; padding: 8px; border-bottom: 2px solid var(--border); font-size: 12px; text-transform: uppercase; color: var(--muted); }
    .field-table td { padding: 8px; border-bottom: 1px solid var(--border); font-size: 14px; }
    .code-tabs { display: flex; gap: 0; border-bottom: 1px solid var(--border); margin-top: 16px; }
    .code-tab { padding: 6px 14px; border: none; background: transparent; cursor: pointer; font-size: 13px; color: var(--muted); border-bottom: 2px solid transparent; }
    .code-tab:hover, .code-tab.active { color: var(--primary); border-bottom-color: var(--primary); }
    pre { background: #1e293b; color: #e2e8f0; padding: 16px; border-radius: 8px; overflow-x: auto; font-size: 13px; line-height: 1.5; margin: 8px 0; }
    code { font-family: "SF Mono", Monaco, Consolas, monospace; }
    .event-card { border: 1px solid var(--border); border-radius: 8px; padding: 20px; margin: 16px 0; }
    .nav { position: sticky; top: 0; background: var(--bg); border-bottom: 1px solid var(--border); padding: 12px 0; margin-bottom: 32px; }
    .nav a { color: var(--primary); text-decoration: none; margin-right: 16px; font-size: 14px; }
    .nav a:hover { text-decoration: underline; }
  </style>
</head>
<body>
<div class="container">
  <h1>%s</h1>
  <p class="subtitle">%s</p>
  <p><strong>Version:</strong> %s | <strong>Auth:</strong> %s</p>

  <nav class="nav">
`, doc.Name, doc.Name, doc.Description, doc.Version, doc.AuthMethod))

	// Navigation
	for _, evt := range events {
		b.WriteString(fmt.Sprintf("    <a href=\"#%s\">%s</a>\n", evt.Name, evt.Name))
	}
	b.WriteString("  </nav>\n\n  <h2>Events</h2>\n")

	// Event documentation
	for _, evt := range events {
		b.WriteString(fmt.Sprintf("  <div class=\"event-card\" id=\"%s\">\n", evt.Name))
		b.WriteString(fmt.Sprintf("    <h3><span class=\"badge badge--event\">%s</span>", evt.Name))
		if evt.Deprecated {
			b.WriteString(" <span class=\"badge badge--deprecated\">Deprecated</span>")
		}
		b.WriteString("</h3>\n")
		b.WriteString(fmt.Sprintf("    <p>%s</p>\n", evt.Description))

		// Field descriptions from schema
		fields := extractFields(evt.PayloadSchema)
		if len(fields) > 0 {
			b.WriteString("    <h4>Payload Fields</h4>\n")
			b.WriteString("    <table class=\"field-table\">\n")
			b.WriteString("      <thead><tr><th>Field</th><th>Type</th><th>Description</th></tr></thead>\n")
			b.WriteString("      <tbody>\n")
			for _, f := range fields {
				b.WriteString(fmt.Sprintf("        <tr><td><code>%s</code></td><td>%s</td><td>%s</td></tr>\n",
					f.Name, f.Type, f.Desc))
			}
			b.WriteString("      </tbody>\n    </table>\n")
		}

		// Example payload
		if len(evt.ExamplePayload) > 0 {
			b.WriteString("    <h4>Example Payload</h4>\n")
			prettyExample, _ := json.MarshalIndent(json.RawMessage(evt.ExamplePayload), "", "  ")
			b.WriteString(fmt.Sprintf("    <pre><code>%s</code></pre>\n", string(prettyExample)))
		}

		// Code snippets
		snippets := g.GenerateAll(evt.Name, evt.PayloadSchema, evt.ExamplePayload)
		b.WriteString("    <h4>Code Examples</h4>\n")
		for lang, code := range snippets {
			b.WriteString(fmt.Sprintf("    <details><summary>%s</summary>\n", langDisplayName(lang)))
			b.WriteString(fmt.Sprintf("    <pre><code>%s</code></pre>\n", escapeHTML(code)))
			b.WriteString("    </details>\n")
		}

		b.WriteString("  </div>\n\n")
	}

	b.WriteString("</div>\n</body>\n</html>")
	return b.String()
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func jsonTypeToGo(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}

func jsonTypeToTS(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "any[]"
	case "object":
		return "Record<string, any>"
	default:
		return "any"
	}
}

func langDisplayName(lang string) string {
	names := map[string]string{
		LangGo:     "Go",
		LangPython: "Python",
		LangNodeJS: "Node.js / TypeScript",
		LangJava:   "Java (Spring)",
		LangRuby:   "Ruby (Sinatra)",
		LangPHP:    "PHP",
		LangCurl:   "cURL",
	}
	if name, ok := names[lang]; ok {
		return name
	}
	return lang
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
