package docgen

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Service provides documentation generation business logic
type Service struct {
	repo Repository
}

// NewService creates a new docgen service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateDoc creates a new webhook doc
func (s *Service) CreateDoc(ctx context.Context, tenantID uuid.UUID, req *CreateDocRequest) (*WebhookDoc, error) {
	doc := &WebhookDoc{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		EventTypes:  req.EventTypes,
		BaseURL:     req.BaseURL,
		AuthMethod:  req.AuthMethod,
	}
	if doc.Version == "" {
		doc.Version = "1.0.0"
	}

	if err := s.repo.CreateDoc(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// GetDoc retrieves a webhook doc by ID
func (s *Service) GetDoc(ctx context.Context, id uuid.UUID) (*WebhookDoc, error) {
	return s.repo.GetDoc(ctx, id)
}

// UpdateDoc updates a webhook doc
func (s *Service) UpdateDoc(ctx context.Context, id uuid.UUID, req *CreateDocRequest) (*WebhookDoc, error) {
	doc, err := s.repo.GetDoc(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		doc.Name = req.Name
	}
	if req.Description != "" {
		doc.Description = req.Description
	}
	if req.Version != "" {
		doc.Version = req.Version
	}
	if req.EventTypes != nil {
		doc.EventTypes = req.EventTypes
	}
	if req.BaseURL != "" {
		doc.BaseURL = req.BaseURL
	}
	if req.AuthMethod != "" {
		doc.AuthMethod = req.AuthMethod
	}

	if err := s.repo.UpdateDoc(ctx, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// ListDocs lists webhook docs for a tenant
func (s *Service) ListDocs(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*WebhookDoc, int, error) {
	return s.repo.ListDocs(ctx, tenantID, limit, offset)
}

// DeleteDoc deletes a webhook doc
func (s *Service) DeleteDoc(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDoc(ctx, id)
}

// AddEventType adds an event type to a webhook doc
func (s *Service) AddEventType(ctx context.Context, docID uuid.UUID, req *AddEventTypeRequest) (*EventTypeDoc, error) {
	et := &EventTypeDoc{
		DocID:             docID,
		Name:              req.Name,
		Description:       req.Description,
		Category:          req.Category,
		PayloadSchema:     req.PayloadSchema,
		ExamplePayload:    req.ExamplePayload,
		Deprecated:        req.Deprecated,
		DeprecationNotice: req.DeprecationNotice,
		Version:           req.Version,
	}

	if err := s.repo.CreateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// UpdateEventType updates an event type doc
func (s *Service) UpdateEventType(ctx context.Context, id uuid.UUID, req *AddEventTypeRequest) (*EventTypeDoc, error) {
	et, err := s.repo.GetEventType(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		et.Name = req.Name
	}
	if req.Description != "" {
		et.Description = req.Description
	}
	if req.Category != "" {
		et.Category = req.Category
	}
	if req.PayloadSchema != nil {
		et.PayloadSchema = req.PayloadSchema
	}
	if req.ExamplePayload != nil {
		et.ExamplePayload = req.ExamplePayload
	}
	et.Deprecated = req.Deprecated
	if req.DeprecationNotice != "" {
		et.DeprecationNotice = req.DeprecationNotice
	}
	if req.Version != "" {
		et.Version = req.Version
	}

	if err := s.repo.UpdateEventType(ctx, et); err != nil {
		return nil, err
	}

	return et, nil
}

// ListEventTypes lists event types for a webhook doc
func (s *Service) ListEventTypes(ctx context.Context, docID uuid.UUID) ([]*EventTypeDoc, error) {
	return s.repo.ListEventTypes(ctx, docID)
}

// DeleteEventType deletes an event type doc
func (s *Service) DeleteEventType(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEventType(ctx, id)
}

// GenerateCodeSample generates a webhook handler code sample for a given event type and language
func (s *Service) GenerateCodeSample(ctx context.Context, req *GenerateCodeRequest) (*CodeSample, error) {
	et, err := s.repo.GetEventType(ctx, req.EventTypeID)
	if err != nil {
		return nil, fmt.Errorf("event type not found: %w", err)
	}

	code, framework, err := generateCode(et, req.Language)
	if err != nil {
		return nil, err
	}

	sample := &CodeSample{
		EventTypeID: req.EventTypeID,
		Language:    req.Language,
		Code:        code,
		Framework:   framework,
		Description: fmt.Sprintf("Webhook handler for %s event", et.Name),
	}

	if err := s.repo.CreateCodeSample(ctx, sample); err != nil {
		return nil, err
	}

	return sample, nil
}

// GetEventCatalog builds a browsable catalog of all event types for a doc
func (s *Service) GetEventCatalog(ctx context.Context, docID uuid.UUID, tenantID uuid.UUID) (*EventCatalog, error) {
	eventTypes, err := s.repo.ListEventTypes(ctx, docID)
	if err != nil {
		return nil, err
	}

	categorySet := make(map[string]bool)
	var entries []EventCatalogEntry

	for _, et := range eventTypes {
		entry := EventCatalogEntry{
			EventTypeID: et.ID,
			Name:        et.Name,
			Category:    et.Category,
			Description: et.Description,
			Version:     et.Version,
			Deprecated:  et.Deprecated,
		}
		entries = append(entries, entry)

		if et.Category != "" {
			categorySet[et.Category] = true
		}
	}

	var categories []string
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	return &EventCatalog{
		TenantID:    tenantID,
		Entries:     entries,
		Categories:  categories,
		TotalEvents: len(entries),
	}, nil
}

// CreateWidget creates a new embeddable doc widget
func (s *Service) CreateWidget(ctx context.Context, tenantID uuid.UUID, req *CreateWidgetRequest) (*DocWidget, error) {
	embedKey, err := generateEmbedKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate embed key: %w", err)
	}

	theme := req.Theme
	if theme == "" {
		theme = ThemeLight
	}

	widget := &DocWidget{
		TenantID:       tenantID,
		DocID:          req.DocID,
		Theme:          theme,
		CustomCSS:      req.CustomCSS,
		AllowedDomains: req.AllowedDomains,
		EmbedKey:       "dw_" + embedKey,
		ViewCount:      0,
	}

	if err := s.repo.CreateWidget(ctx, widget); err != nil {
		return nil, err
	}

	return widget, nil
}

// GetWidget retrieves a doc widget by ID
func (s *Service) GetWidget(ctx context.Context, id uuid.UUID) (*DocWidget, error) {
	return s.repo.GetWidget(ctx, id)
}

// ValidateWidgetDomain checks if a domain is allowed for a widget
func (s *Service) ValidateWidgetDomain(ctx context.Context, widgetID uuid.UUID, domain string) (bool, error) {
	widget, err := s.repo.GetWidget(ctx, widgetID)
	if err != nil {
		return false, err
	}

	if len(widget.AllowedDomains) == 0 {
		return true, nil
	}

	for _, d := range widget.AllowedDomains {
		if d == "*" || d == domain {
			return true, nil
		}
		if strings.HasPrefix(d, "*.") {
			suffix := d[1:]
			if strings.HasSuffix(domain, suffix) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetDocAnalytics retrieves analytics for a doc
func (s *Service) GetDocAnalytics(ctx context.Context, docID uuid.UUID) (*DocAnalytics, error) {
	return s.repo.GetDocAnalytics(ctx, docID)
}

// RecordDocView records a view for a doc
func (s *Service) RecordDocView(ctx context.Context, docID uuid.UUID) error {
	return s.repo.RecordDocView(ctx, docID)
}

func generateCode(et *EventTypeDoc, language string) (string, string, error) {
	eventName := et.Name
	var exampleJSON string
	if et.ExamplePayload != nil {
		var pretty json.RawMessage
		if err := json.Unmarshal(et.ExamplePayload, &pretty); err == nil {
			indented, _ := json.MarshalIndent(pretty, "    ", "  ")
			exampleJSON = string(indented)
		}
	}
	if exampleJSON == "" {
		exampleJSON = `{"event": "` + eventName + `", "data": {}}`
	}

	switch language {
	case LangGo:
		return generateGoCode(eventName, exampleJSON), "net/http", nil
	case LangPython:
		return generatePythonCode(eventName, exampleJSON), "Flask", nil
	case LangNodeJS:
		return generateNodeCode(eventName, exampleJSON), "Express", nil
	case LangJava:
		return generateJavaCode(eventName, exampleJSON), "Spring Boot", nil
	case LangRuby:
		return generateRubyCode(eventName, exampleJSON), "Sinatra", nil
	case LangPHP:
		return generatePHPCode(eventName, exampleJSON), "PHP", nil
	case LangCSharp:
		return generateCSharpCode(eventName, exampleJSON), "ASP.NET Core", nil
	case LangCurl:
		return generateCurlCode(eventName, exampleJSON), "cURL", nil
	default:
		return "", "", fmt.Errorf("unsupported language: %s", language)
	}
}

func generateGoCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle %s event
	fmt.Printf("Received %s event: %%v\n", payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
// Example payload:
// %s`, eventName, eventName, exampleJSON)
}

func generatePythonCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook_handler():
    payload = request.get_json()

    # Handle %s event
    print(f"Received %s event: {payload}")

    return jsonify({"status": "ok"}), 200

if __name__ == '__main__':
    app.run(port=8080)
# Example payload:
# %s`, eventName, eventName, exampleJSON)
}

func generateNodeCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`const express = require('express');
const app = express();

app.use(express.json());

app.post('/webhook', (req, res) => {
  const payload = req.body;

  // Handle %s event
  console.log('Received %s event:', payload);

  res.status(200).json({ status: 'ok' });
});

app.listen(8080, () => {
  console.log('Listening on port 8080');
});
// Example payload:
// %s`, eventName, eventName, exampleJSON)
}

func generateJavaCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@SpringBootApplication
@RestController
public class WebhookHandler {

    @PostMapping("/webhook")
    public Map<String, String> handleWebhook(@RequestBody Map<String, Object> payload) {
        // Handle %s event
        System.out.println("Received %s event: " + payload);

        return Map.of("status", "ok");
    }

    public static void main(String[] args) {
        SpringApplication.run(WebhookHandler.class, args);
    }
}
// Example payload:
// %s`, eventName, eventName, exampleJSON)
}

func generateRubyCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`require 'sinatra'
require 'json'

post '/webhook' do
  payload = JSON.parse(request.body.read)

  # Handle %s event
  puts "Received %s event: #{payload}"

  content_type :json
  { status: 'ok' }.to_json
end
# Example payload:
# %s`, eventName, eventName, exampleJSON)
}

func generatePHPCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`<?php

$payload = json_decode(file_get_contents('php://input'), true);

if ($payload === null) {
    http_response_code(400);
    echo json_encode(['error' => 'Invalid JSON']);
    exit;
}

// Handle %s event
error_log("Received %s event: " . json_encode($payload));

http_response_code(200);
echo json_encode(['status' => 'ok']);
// Example payload:
// %s`, eventName, eventName, exampleJSON)
}

func generateCSharpCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`using Microsoft.AspNetCore.Mvc;

[ApiController]
[Route("[controller]")]
public class WebhookController : ControllerBase
{
    [HttpPost]
    public IActionResult HandleWebhook([FromBody] dynamic payload)
    {
        // Handle %s event
        Console.WriteLine($"Received %s event: {payload}");

        return Ok(new { status = "ok" });
    }
}
// Example payload:
// %s`, eventName, eventName, exampleJSON)
}

func generateCurlCode(eventName, exampleJSON string) string {
	return fmt.Sprintf(`# Send a test %s webhook event
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '%s'`, eventName, exampleJSON)
}

func generateEmbedKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
