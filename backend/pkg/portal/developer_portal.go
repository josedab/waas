package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// StartOnboarding initiates the guided onboarding wizard for a tenant
func (s *Service) StartOnboarding(ctx context.Context, tenantID string, req *StartOnboardingRequest) (*OnboardingWizard, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	steps := DefaultOnboardingSteps()
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal steps: %w", err)
	}

	wizard := &OnboardingWizard{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		CurrentStep: StepCreateTenant,
		Steps:       steps,
		StepsJSON:   string(stepsJSON),
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Mark first step as in_progress
	wizard.Steps[0].Status = StepStatusInProgress

	return wizard, nil
}

// CompleteOnboardingStep marks a step as completed and advances to next
func (s *Service) CompleteOnboardingStep(ctx context.Context, tenantID string, wizard *OnboardingWizard, req *CompleteStepRequest) (*OnboardingWizard, error) {
	if wizard == nil {
		return nil, fmt.Errorf("onboarding wizard not found")
	}

	found := false
	nextStepIdx := -1
	for i, step := range wizard.Steps {
		if step.ID == req.StepID {
			now := time.Now()
			wizard.Steps[i].Status = StepStatusCompleted
			wizard.Steps[i].CompletedAt = &now
			found = true
			if i+1 < len(wizard.Steps) {
				nextStepIdx = i + 1
			}
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("step %q not found in onboarding wizard", req.StepID)
	}

	if nextStepIdx >= 0 {
		wizard.Steps[nextStepIdx].Status = StepStatusInProgress
		wizard.CurrentStep = wizard.Steps[nextStepIdx].ID
	} else {
		now := time.Now()
		wizard.CompletedAt = &now
		wizard.CurrentStep = ""
	}

	wizard.UpdatedAt = time.Now()
	return wizard, nil
}

// GetOnboardingProgress calculates the completion percentage
func (s *Service) GetOnboardingProgress(wizard *OnboardingWizard) map[string]interface{} {
	if wizard == nil {
		return map[string]interface{}{
			"completion_pct": 0,
			"total_steps":    0,
			"completed":      0,
		}
	}

	total := len(wizard.Steps)
	completed := 0
	for _, step := range wizard.Steps {
		if step.Status == StepStatusCompleted {
			completed++
		}
	}

	pct := 0.0
	if total > 0 {
		pct = float64(completed) / float64(total) * 100
	}

	return map[string]interface{}{
		"completion_pct": pct,
		"total_steps":    total,
		"completed":      completed,
		"current_step":   wizard.CurrentStep,
		"is_complete":    wizard.CompletedAt != nil,
	}
}

// GetAPIExplorer builds the interactive API explorer configuration
func (s *Service) GetAPIExplorer(ctx context.Context, baseURL string) *APIExplorerConfig {
	return &APIExplorerConfig{
		BaseURL: baseURL,
		Title:   "WaaS API Explorer",
		Version: "1.0.0",
		Categories: []string{
			"Webhooks", "Endpoints", "Deliveries", "Events",
			"Tenants", "Analytics", "Transforms", "Testing",
		},
		Endpoints: buildCoreAPIEndpoints(),
	}
}

// buildCoreAPIEndpoints returns the core WaaS API endpoints for the explorer
func buildCoreAPIEndpoints() []APIExplorerEndpoint {
	return []APIExplorerEndpoint{
		{
			Method:      "POST",
			Path:        "/api/v1/webhooks",
			Summary:     "Send Webhook",
			Description: "Send a webhook event to all subscribed endpoints",
			Tags:        []string{"Webhooks"},
			RequestBody: &ExplorerBody{
				ContentType: "application/json",
				Example:     json.RawMessage(`{"event_type":"order.created","payload":{"order_id":"ord_123","amount":99.99}}`),
			},
			Responses: []ExplorerResponse{
				{StatusCode: 202, Description: "Webhook accepted for delivery", Example: json.RawMessage(`{"id":"wh_abc123","status":"queued"}`)},
				{StatusCode: 400, Description: "Invalid request"},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/v1/endpoints",
			Summary:     "List Endpoints",
			Description: "Retrieve all webhook endpoints for the tenant",
			Tags:        []string{"Endpoints"},
			Parameters: []ExplorerParameter{
				{Name: "limit", In: "query", Type: "integer", Description: "Maximum results to return", Example: "50"},
				{Name: "offset", In: "query", Type: "integer", Description: "Offset for pagination", Example: "0"},
			},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "List of endpoints", Example: json.RawMessage(`{"endpoints":[],"total":0}`)},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/v1/endpoints",
			Summary:     "Create Endpoint",
			Description: "Register a new webhook endpoint",
			Tags:        []string{"Endpoints"},
			RequestBody: &ExplorerBody{
				ContentType: "application/json",
				Example:     json.RawMessage(`{"url":"https://example.com/webhook","event_types":["order.created","order.updated"],"description":"My webhook endpoint"}`),
			},
			Responses: []ExplorerResponse{
				{StatusCode: 201, Description: "Endpoint created"},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/v1/deliveries",
			Summary:     "List Deliveries",
			Description: "Retrieve webhook delivery history",
			Tags:        []string{"Deliveries"},
			Parameters: []ExplorerParameter{
				{Name: "endpoint_id", In: "query", Type: "string", Description: "Filter by endpoint"},
				{Name: "status", In: "query", Type: "string", Description: "Filter by status (success, failed, pending)"},
				{Name: "limit", In: "query", Type: "integer", Description: "Maximum results", Example: "50"},
			},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "Delivery list"},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/v1/deliveries/{id}/retry",
			Summary:     "Retry Delivery",
			Description: "Retry a failed webhook delivery",
			Tags:        []string{"Deliveries"},
			Parameters: []ExplorerParameter{
				{Name: "id", In: "path", Type: "string", Required: true, Description: "Delivery ID"},
			},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "Retry initiated"},
				{StatusCode: 404, Description: "Delivery not found"},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/v1/events",
			Summary:     "List Event Types",
			Description: "Retrieve available webhook event types",
			Tags:        []string{"Events"},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "Event type catalog"},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/v1/analytics/overview",
			Summary:     "Analytics Overview",
			Description: "Get webhook delivery analytics and metrics",
			Tags:        []string{"Analytics"},
			Parameters: []ExplorerParameter{
				{Name: "period", In: "query", Type: "string", Description: "Time period (1h, 24h, 7d, 30d)", Example: "24h"},
			},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "Analytics data"},
			},
		},
		{
			Method:      "POST",
			Path:        "/api/v1/test/send",
			Summary:     "Send Test Event",
			Description: "Send a test webhook event for verification",
			Tags:        []string{"Testing"},
			RequestBody: &ExplorerBody{
				ContentType: "application/json",
				Example:     json.RawMessage(`{"endpoint_id":"ep_123","event_type":"test.ping","payload":{"message":"Hello from WaaS!"}}`),
			},
			Responses: []ExplorerResponse{
				{StatusCode: 200, Description: "Test event sent"},
			},
		},
	}
}

// TryAPIEndpoint executes a live API call from the explorer
func (s *Service) TryAPIEndpoint(ctx context.Context, baseURL, tenantID, apiKey string, req *TryEndpointRequest) (*TryEndpointResponse, error) {
	fullURL := baseURL + req.Path
	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Tenant-ID", tenantID)
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	return &TryEndpointResponse{
		StatusCode: resp.StatusCode,
		Headers:    respHeaders,
		Body:       body,
		LatencyMs:  time.Since(start).Milliseconds(),
	}, nil
}

// GenerateSDKCode generates webhook handler code for a specific language/SDK
func (s *Service) GenerateSDKCode(ctx context.Context, req *SDKCodeGenRequest) (*SDKCodeGenResponse, error) {
	generators := map[string]struct {
		framework  string
		installCmd string
		genFunc    func(string) string
	}{
		"go":         {"net/http", "go get github.com/josedab/waas/sdk/go", generateGoSDKCode},
		"python":     {"Flask", "pip install waas-sdk", generatePythonSDKCode},
		"nodejs":     {"Express", "npm install @waas/sdk", generateNodeSDKCode},
		"typescript": {"Express", "npm install @waas/sdk @types/express", generateTypeScriptSDKCode},
		"java":       {"Spring Boot", "<!-- Add waas-sdk to pom.xml -->", generateJavaSDKCode},
		"ruby":       {"Sinatra", "gem install waas-sdk", generateRubySDKCode},
		"php":        {"PHP", "composer require waas/sdk", generatePHPSDKCode},
		"csharp":     {"ASP.NET Core", "dotnet add package WaaS.SDK", generateCSharpSDKCode},
		"rust":       {"Actix-web", "cargo add waas-sdk", generateRustSDKCode},
		"kotlin":     {"Ktor", "implementation(\"com.waas:sdk:1.0.0\")", generateKotlinSDKCode},
		"swift":      {"Vapor", "// Add WaaS SDK to Package.swift", generateSwiftSDKCode},
		"elixir":     {"Phoenix", "# Add {:waas_sdk, \"~> 1.0\"} to mix.exs", generateElixirSDKCode},
	}

	gen, ok := generators[req.Language]
	if !ok {
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}

	eventType := req.EventType
	if eventType == "" {
		eventType = "order.created"
	}

	return &SDKCodeGenResponse{
		Language:    req.Language,
		Framework:   gen.framework,
		Code:        gen.genFunc(eventType),
		InstallCmd:  gen.installCmd,
		Description: fmt.Sprintf("Webhook handler for %s using %s", req.Language, gen.framework),
	}, nil
}

// GetUnifiedPortalView returns the unified portal dashboard
func (s *Service) GetUnifiedPortalView(ctx context.Context, tenantID, apiURL string) (*UnifiedPortalView, error) {
	view := &UnifiedPortalView{
		QuickLinks: []QuickLink{
			{Title: "API Explorer", Description: "Interactive API documentation", URL: apiURL + "/portal/explorer", Icon: "code"},
			{Title: "Playground", Description: "Test webhooks in a sandbox", URL: apiURL + "/portal/playground", Icon: "play"},
			{Title: "Documentation", Description: "Webhook integration guides", URL: apiURL + "/portal/docs", Icon: "book"},
			{Title: "SDK Downloads", Description: "Client libraries for 12 languages", URL: apiURL + "/portal/sdks", Icon: "download"},
			{Title: "Analytics", Description: "Delivery metrics and insights", URL: apiURL + "/portal/analytics", Icon: "chart"},
			{Title: "Settings", Description: "Configure endpoints and alerts", URL: apiURL + "/portal/settings", Icon: "gear"},
		},
	}

	if s.repo != nil {
		portal, err := s.repo.GetPortalByTenantID(ctx, tenantID)
		if err == nil {
			view.Portal = portal
		}
		stats, err := s.repo.GetPortalStats(ctx, tenantID)
		if err == nil {
			view.Stats = stats
		}
	}

	return view, nil
}

// SDK code generation functions for all 12 languages

func generateGoSDKCode(eventType string) string {
	return fmt.Sprintf(`package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func verifySignature(payload []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-WaaS-Signature")
	if !verifySignature(body, signature, "your-webhook-secret") {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	var event map[string]interface{}
	json.Unmarshal(body, &event)
	fmt.Printf("Received %s event: %%v\n", event)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	http.ListenAndServe(":8080", nil)
}`, eventType)
}

func generatePythonSDKCode(eventType string) string {
	return fmt.Sprintf(`import hmac
import hashlib
from flask import Flask, request, jsonify

app = Flask(__name__)
WEBHOOK_SECRET = "your-webhook-secret"

def verify_signature(payload: bytes, signature: str) -> bool:
    expected = hmac.new(
        WEBHOOK_SECRET.encode(), payload, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

@app.route("/webhook", methods=["POST"])
def webhook_handler():
    payload = request.get_data()
    signature = request.headers.get("X-WaaS-Signature", "")
    if not verify_signature(payload, signature):
        return jsonify({"error": "Invalid signature"}), 401

    event = request.get_json()
    print(f"Received %s event: {event}")
    return jsonify({"status": "ok"}), 200

if __name__ == "__main__":
    app.run(port=8080)`, eventType)
}

func generateNodeSDKCode(eventType string) string {
	return fmt.Sprintf(`const express = require('express');
const crypto = require('crypto');
const app = express();
app.use(express.raw({ type: 'application/json' }));

const WEBHOOK_SECRET = 'your-webhook-secret';

function verifySignature(payload, signature) {
  const expected = crypto.createHmac('sha256', WEBHOOK_SECRET)
    .update(payload).digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(expected), Buffer.from(signature)
  );
}

app.post('/webhook', (req, res) => {
  const signature = req.headers['x-waas-signature'] || '';
  if (!verifySignature(req.body, signature)) {
    return res.status(401).json({ error: 'Invalid signature' });
  }
  const event = JSON.parse(req.body);
  console.log('Received %s event:', event);
  res.json({ status: 'ok' });
});

app.listen(8080, () => console.log('Listening on :8080'));`, eventType)
}

func generateTypeScriptSDKCode(eventType string) string {
	return fmt.Sprintf(`import express, { Request, Response } from 'express';
import crypto from 'crypto';

const app = express();
app.use(express.raw({ type: 'application/json' }));

const WEBHOOK_SECRET = 'your-webhook-secret';

function verifySignature(payload: Buffer, signature: string): boolean {
  const expected = crypto.createHmac('sha256', WEBHOOK_SECRET)
    .update(payload).digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(expected), Buffer.from(signature)
  );
}

interface WebhookEvent {
  event_type: string;
  payload: Record<string, unknown>;
  timestamp: string;
}

app.post('/webhook', (req: Request, res: Response) => {
  const signature = (req.headers['x-waas-signature'] as string) || '';
  if (!verifySignature(req.body, signature)) {
    return res.status(401).json({ error: 'Invalid signature' });
  }
  const event: WebhookEvent = JSON.parse(req.body.toString());
  console.log('Received %s event:', event);
  res.json({ status: 'ok' });
});

app.listen(8080, () => console.log('Listening on :8080'));`, eventType)
}

func generateJavaSDKCode(eventType string) string {
	return fmt.Sprintf(`import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.*;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.util.Map;

@SpringBootApplication
@RestController
public class WebhookHandler {
    private static final String SECRET = "your-webhook-secret";

    private boolean verifySignature(String payload, String signature) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(SECRET.getBytes(), "HmacSHA256"));
            String expected = bytesToHex(mac.doFinal(payload.getBytes()));
            return expected.equals(signature);
        } catch (Exception e) { return false; }
    }

    @PostMapping("/webhook")
    public Map<String, String> handle(
            @RequestBody String body,
            @RequestHeader("X-WaaS-Signature") String sig) {
        if (!verifySignature(body, sig)) {
            throw new RuntimeException("Invalid signature");
        }
        System.out.println("Received %s event: " + body);
        return Map.of("status", "ok");
    }

    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) sb.append(String.format("%%02x", b));
        return sb.toString();
    }

    public static void main(String[] args) {
        SpringApplication.run(WebhookHandler.class, args);
    }
}`, eventType)
}

func generateRubySDKCode(eventType string) string {
	return fmt.Sprintf(`require 'sinatra'
require 'json'
require 'openssl'

WEBHOOK_SECRET = 'your-webhook-secret'

def verify_signature(payload, signature)
  expected = OpenSSL::HMAC.hexdigest('SHA256', WEBHOOK_SECRET, payload)
  Rack::Utils.secure_compare(expected, signature)
end

post '/webhook' do
  payload = request.body.read
  signature = request.env['HTTP_X_WAAS_SIGNATURE'] || ''
  halt 401, { error: 'Invalid signature' }.to_json unless verify_signature(payload, signature)

  event = JSON.parse(payload)
  puts "Received %s event: #{event}"
  content_type :json
  { status: 'ok' }.to_json
end`, eventType)
}

func generatePHPSDKCode(eventType string) string {
	return fmt.Sprintf(`<?php
$secret = 'your-webhook-secret';
$payload = file_get_contents('php://input');
$signature = $_SERVER['HTTP_X_WAAS_SIGNATURE'] ?? '';

$expected = hash_hmac('sha256', $payload, $secret);
if (!hash_equals($expected, $signature)) {
    http_response_code(401);
    echo json_encode(['error' => 'Invalid signature']);
    exit;
}

$event = json_decode($payload, true);
error_log("Received %s event: " . json_encode($event));

http_response_code(200);
echo json_encode(['status' => 'ok']);`, eventType)
}

func generateCSharpSDKCode(eventType string) string {
	return fmt.Sprintf(`using Microsoft.AspNetCore.Mvc;
using System.Security.Cryptography;
using System.Text;

[ApiController]
[Route("[controller]")]
public class WebhookController : ControllerBase
{
    private const string Secret = "your-webhook-secret";

    private static bool VerifySignature(string payload, string signature)
    {
        using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(Secret));
        var hash = hmac.ComputeHash(Encoding.UTF8.GetBytes(payload));
        var expected = BitConverter.ToString(hash).Replace("-", "").ToLower();
        return expected == signature;
    }

    [HttpPost]
    public IActionResult Handle([FromBody] dynamic payload,
        [FromHeader(Name = "X-WaaS-Signature")] string signature)
    {
        string body = payload.ToString();
        if (!VerifySignature(body, signature))
            return Unauthorized(new { error = "Invalid signature" });

        Console.WriteLine($"Received %s event: {body}");
        return Ok(new { status = "ok" });
    }
}`, eventType)
}

func generateRustSDKCode(eventType string) string {
	return fmt.Sprintf(`use actix_web::{web, App, HttpServer, HttpRequest, HttpResponse};
use hmac::{Hmac, Mac};
use sha2::Sha256;

type HmacSha256 = Hmac<Sha256>;
const SECRET: &str = "your-webhook-secret";

fn verify_signature(payload: &[u8], signature: &str) -> bool {
    let mut mac = HmacSha256::new_from_slice(SECRET.as_bytes()).unwrap();
    mac.update(payload);
    let result = hex::encode(mac.finalize().into_bytes());
    result == signature
}

async fn webhook_handler(req: HttpRequest, body: web::Bytes) -> HttpResponse {
    let sig = req.headers().get("x-waas-signature")
        .and_then(|v| v.to_str().ok()).unwrap_or("");
    if !verify_signature(&body, sig) {
        return HttpResponse::Unauthorized().json(serde_json::json!({"error": "Invalid signature"}));
    }
    println!("Received %s event: {}", String::from_utf8_lossy(&body));
    HttpResponse::Ok().json(serde_json::json!({"status": "ok"}))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| App::new().route("/webhook", web::post().to(webhook_handler)))
        .bind("0.0.0.0:8080")?.run().await
}`, eventType)
}

func generateKotlinSDKCode(eventType string) string {
	return fmt.Sprintf(`import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.routing.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

val SECRET = "your-webhook-secret"

fun verifySignature(payload: String, signature: String): Boolean {
    val mac = Mac.getInstance("HmacSHA256")
    mac.init(SecretKeySpec(SECRET.toByteArray(), "HmacSHA256"))
    val expected = mac.doFinal(payload.toByteArray()).joinToString("") { "%%02x".format(it) }
    return expected == signature
}

fun main() {
    embeddedServer(Netty, port = 8080) {
        routing {
            post("/webhook") {
                val body = call.receiveText()
                val signature = call.request.headers["X-WaaS-Signature"] ?: ""
                if (!verifySignature(body, signature)) {
                    call.respondText("""{"error":"Invalid signature"}""", status = io.ktor.http.HttpStatusCode.Unauthorized)
                    return@post
                }
                println("Received %s event: $body")
                call.respondText("""{"status":"ok"}""")
            }
        }
    }.start(wait = true)
}`, eventType)
}

func generateSwiftSDKCode(eventType string) string {
	return fmt.Sprintf(`import Vapor
import Crypto

let SECRET = "your-webhook-secret"

func routes(_ app: Application) throws {
    app.post("webhook") { req -> Response in
        let body = try req.content.decode([String: AnyCodable].self)
        let signature = req.headers.first(name: "X-WaaS-Signature") ?? ""
        let key = SymmetricKey(data: SECRET.utf8)
        let bodyData = try JSONEncoder().encode(body)
        let mac = HMAC<SHA256>.authenticationCode(for: bodyData, using: key)
        let expected = mac.map { String(format: "%%02x", $0) }.joined()
        guard expected == signature else {
            return Response(status: .unauthorized)
        }
        print("Received %s event: \(body)")
        return Response(status: .ok)
    }
}`, eventType)
}

func generateElixirSDKCode(eventType string) string {
	return fmt.Sprintf(`defmodule WebhookController do
  use Plug.Router
  plug :match
  plug Plug.Parsers, parsers: [:json], json_decoder: Jason
  plug :dispatch

  @secret "your-webhook-secret"

  defp verify_signature(payload, signature) do
    expected = :crypto.mac(:hmac, :sha256, @secret, payload) |> Base.encode16(case: :lower)
    Plug.Crypto.secure_compare(expected, signature)
  end

  post "/webhook" do
    {:ok, body, conn} = Plug.Conn.read_body(conn)
    signature = Plug.Conn.get_req_header(conn, "x-waas-signature") |> List.first() || ""
    unless verify_signature(body, signature) do
      send_resp(conn, 401, Jason.encode!(%%{error: "Invalid signature"}))
    end
    IO.puts("Received %s event: #{body}")
    send_resp(conn, 200, Jason.encode!(%%{status: "ok"}))
  end
end`, eventType)
}
