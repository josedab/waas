package openapigen

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SDKLanguage represents a supported SDK language
type SDKLanguage string

const (
	SDKLanguageGo     SDKLanguage = "go"
	SDKLanguagePython SDKLanguage = "python"
	SDKLanguageNode   SDKLanguage = "nodejs"
	SDKLanguageJava   SDKLanguage = "java"
	SDKLanguageRuby   SDKLanguage = "ruby"
	SDKLanguagePHP    SDKLanguage = "php"
	SDKLanguageRust   SDKLanguage = "rust"
	SDKLanguageCSharp SDKLanguage = "csharp"
)

// AllSDKLanguages returns all supported SDK languages
func AllSDKLanguages() []SDKLanguage {
	return []SDKLanguage{
		SDKLanguageGo, SDKLanguagePython, SDKLanguageNode,
		SDKLanguageJava, SDKLanguageRuby, SDKLanguagePHP,
		SDKLanguageRust, SDKLanguageCSharp,
	}
}

// SDKGeneratorConfig configures SDK generation for a language
type SDKGeneratorConfig struct {
	Language        SDKLanguage       `json:"language"`
	PackageName     string            `json:"package_name"`
	PackageVersion  string            `json:"package_version"`
	OutputDir       string            `json:"output_dir"`
	SpecPath        string            `json:"spec_path"`
	GitUserID       string            `json:"git_user_id"`
	GitRepoID       string            `json:"git_repo_id"`
	AdditionalProps map[string]string `json:"additional_properties,omitempty"`
}

// SDKGenerationResult contains the result of SDK generation
type SDKGenerationResult struct {
	Language    SDKLanguage `json:"language"`
	Success     bool        `json:"success"`
	OutputDir   string      `json:"output_dir"`
	FilesCount  int         `json:"files_count"`
	Error       string      `json:"error,omitempty"`
	Duration    string      `json:"duration"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// EventClassDefinition represents a strongly-typed event class to generate
type EventClassDefinition struct {
	Name        string                 `json:"name"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Schema      map[string]interface{} `json:"schema"`
	Fields      []EventField           `json:"fields"`
}

// EventField represents a field in an event class
type EventField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Example     string `json:"example,omitempty"`
}

// WebhookVerificationHelper generates webhook verification code for each language
type WebhookVerificationHelper struct {
	Language SDKLanguage `json:"language"`
	Code     string      `json:"code"`
	TestCode string      `json:"test_code"`
}

// GenerateVerificationHelper creates webhook signature verification code
func GenerateVerificationHelper(lang SDKLanguage) *WebhookVerificationHelper {
	helper := &WebhookVerificationHelper{Language: lang}

	switch lang {
	case SDKLanguageGo:
		helper.Code = goVerificationCode
		helper.TestCode = goVerificationTestCode
	case SDKLanguagePython:
		helper.Code = pythonVerificationCode
		helper.TestCode = pythonVerificationTestCode
	case SDKLanguageNode:
		helper.Code = nodeVerificationCode
		helper.TestCode = nodeVerificationTestCode
	case SDKLanguageJava:
		helper.Code = javaVerificationCode
	case SDKLanguageRuby:
		helper.Code = rubyVerificationCode
	case SDKLanguagePHP:
		helper.Code = phpVerificationCode
	case SDKLanguageRust:
		helper.Code = rustVerificationCode
	case SDKLanguageCSharp:
		helper.Code = csharpVerificationCode
	}

	return helper
}

// GenerateEventClasses creates strongly-typed event classes from a spec
func GenerateEventClasses(spec *OpenAPISpec) []EventClassDefinition {
	var classes []EventClassDefinition

	for name, webhook := range spec.Webhooks {
		if webhook.Post == nil || webhook.Post.RequestBody == nil {
			continue
		}

		cls := EventClassDefinition{
			Name:        toPascalCase(name),
			EventType:   name,
			Description: webhook.Post.Description,
			Version:     spec.Info.Version,
		}

		// Extract fields from schema if available
		for _, mediaType := range webhook.Post.RequestBody.Content {
			if mediaType.Schema != nil {
				var schemaMap map[string]interface{}
				if err := json.Unmarshal(mediaType.Schema, &schemaMap); err == nil {
					cls.Schema = schemaMap
					cls.Fields = extractFieldsFromSchema(schemaMap)
				}
			}
		}

		classes = append(classes, cls)
	}

	return classes
}

func extractFieldsFromSchema(schema map[string]interface{}) []EventField {
	var fields []EventField

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return fields
	}

	requiredSet := make(map[string]bool)
	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	for name, propRaw := range props {
		prop, ok := propRaw.(map[string]interface{})
		if !ok {
			continue
		}

		field := EventField{
			Name:     name,
			Type:     fmt.Sprintf("%v", prop["type"]),
			Required: requiredSet[name],
		}

		if desc, ok := prop["description"].(string); ok {
			field.Description = desc
		}

		fields = append(fields, field)
	}

	return fields
}

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// --- Verification helper code templates ---

const goVerificationCode = `package waas

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidSignature = errors.New("invalid webhook signature")
var ErrExpiredTimestamp = errors.New("webhook timestamp expired")

// VerifyWebhookSignature verifies a WaaS webhook signature.
// secret: your endpoint's signing secret
// signature: the X-Webhook-Signature header value
// body: the raw request body bytes
// tolerance: max age of the timestamp (e.g. 5*time.Minute)
func VerifyWebhookSignature(secret, signature string, body []byte, tolerance time.Duration) error {
	parts := strings.Split(signature, ",")
	if len(parts) < 2 {
		return ErrInvalidSignature
	}

	var ts string
	var sig string
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 { continue }
		switch kv[0] {
		case "t": ts = kv[1]
		case "v1": sig = kv[1]
		}
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil { return ErrInvalidSignature }

	if tolerance > 0 {
		age := time.Since(time.Unix(tsInt, 0))
		if age > tolerance { return ErrExpiredTimestamp }
	}

	payload := fmt.Sprintf("%s.%s", ts, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return ErrInvalidSignature
	}
	return nil
}
`

const goVerificationTestCode = `package waas

import (
	"testing"
	"time"
)

func TestVerifyWebhookSignature_Invalid(t *testing.T) {
	err := VerifyWebhookSignature("secret", "invalid", []byte("body"), 5*time.Minute)
	if !errors.Is(err, ErrInvalidSignature) { t.Fatal("expected ErrInvalidSignature") }
}
`

const pythonVerificationCode = `import hashlib
import hmac
import time

class WebhookVerificationError(Exception):
    pass

def verify_webhook_signature(secret: str, signature: str, body: bytes, tolerance: int = 300) -> bool:
    """Verify a WaaS webhook signature.
    
    Args:
        secret: Your endpoint signing secret
        signature: The X-Webhook-Signature header value
        body: Raw request body bytes
        tolerance: Max age in seconds (default 300 = 5 minutes)
    """
    parts = dict(p.split("=", 1) for p in signature.split(",") if "=" in p)
    
    ts = parts.get("t")
    sig = parts.get("v1")
    if not ts or not sig:
        raise WebhookVerificationError("Invalid signature format")
    
    if tolerance > 0:
        age = time.time() - int(ts)
        if age > tolerance:
            raise WebhookVerificationError("Timestamp expired")
    
    payload = f"{ts}.{body.decode('utf-8')}"
    expected = hmac.new(secret.encode(), payload.encode(), hashlib.sha256).hexdigest()
    
    if not hmac.compare_digest(sig, expected):
        raise WebhookVerificationError("Invalid signature")
    
    return True
`

const pythonVerificationTestCode = `import pytest
from waas.verify import verify_webhook_signature, WebhookVerificationError

def test_invalid_signature():
    with pytest.raises(WebhookVerificationError):
        verify_webhook_signature("secret", "invalid", b"body")
`

const nodeVerificationCode = `const crypto = require('crypto');

class WebhookVerificationError extends Error {
  constructor(message) { super(message); this.name = 'WebhookVerificationError'; }
}

function verifyWebhookSignature(secret, signature, body, toleranceMs = 300000) {
  const parts = Object.fromEntries(
    signature.split(',').map(p => p.split('=', 2)).filter(p => p.length === 2)
  );

  const ts = parts.t;
  const sig = parts.v1;
  if (!ts || !sig) throw new WebhookVerificationError('Invalid signature format');

  if (toleranceMs > 0) {
    const age = Date.now() - parseInt(ts) * 1000;
    if (age > toleranceMs) throw new WebhookVerificationError('Timestamp expired');
  }

  const payload = ` + "`${ts}.${body}`" + `;
  const expected = crypto.createHmac('sha256', secret).update(payload).digest('hex');

  if (!crypto.timingSafeEqual(Buffer.from(sig), Buffer.from(expected))) {
    throw new WebhookVerificationError('Invalid signature');
  }
  return true;
}

module.exports = { verifyWebhookSignature, WebhookVerificationError };
`

const nodeVerificationTestCode = `const { verifyWebhookSignature, WebhookVerificationError } = require('./verify');
const assert = require('assert');

assert.throws(() => verifyWebhookSignature('secret', 'invalid', 'body'), WebhookVerificationError);
`

const javaVerificationCode = `package com.waas.sdk;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.time.Instant;

public class WebhookVerifier {
    public static boolean verify(String secret, String signature, byte[] body, long toleranceSecs) throws Exception {
        String[] parts = signature.split(",");
        String ts = null, sig = null;
        for (String p : parts) {
            String[] kv = p.split("=", 2);
            if (kv.length == 2) {
                if ("t".equals(kv[0])) ts = kv[1];
                if ("v1".equals(kv[0])) sig = kv[1];
            }
        }
        if (ts == null || sig == null) throw new SecurityException("Invalid signature format");
        if (toleranceSecs > 0) {
            long age = Instant.now().getEpochSecond() - Long.parseLong(ts);
            if (age > toleranceSecs) throw new SecurityException("Timestamp expired");
        }
        String payload = ts + "." + new String(body);
        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(), "HmacSHA256"));
        String expected = bytesToHex(mac.doFinal(payload.getBytes()));
        if (!sig.equals(expected)) throw new SecurityException("Invalid signature");
        return true;
    }
    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) sb.append(String.format("%02x", b));
        return sb.toString();
    }
}
`

const rubyVerificationCode = `require 'openssl'
require 'time'

module Waas
  class WebhookVerificationError < StandardError; end

  def self.verify_webhook_signature(secret:, signature:, body:, tolerance: 300)
    parts = signature.split(',').map { |p| p.split('=', 2) }.to_h
    ts = parts['t']
    sig = parts['v1']
    raise WebhookVerificationError, 'Invalid signature' unless ts && sig
    raise WebhookVerificationError, 'Timestamp expired' if tolerance > 0 && (Time.now.to_i - ts.to_i) > tolerance
    expected = OpenSSL::HMAC.hexdigest('SHA256', secret, "#{ts}.#{body}")
    raise WebhookVerificationError, 'Invalid signature' unless Rack::Utils.secure_compare(sig, expected)
    true
  end
end
`

const phpVerificationCode = `<?php
namespace Waas;

class WebhookVerifier {
    public static function verify(string $secret, string $signature, string $body, int $tolerance = 300): bool {
        $parts = [];
        foreach (explode(',', $signature) as $p) {
            [$k, $v] = explode('=', $p, 2);
            $parts[$k] = $v;
        }
        if (!isset($parts['t'], $parts['v1'])) throw new \RuntimeException('Invalid signature');
        if ($tolerance > 0 && (time() - intval($parts['t'])) > $tolerance) throw new \RuntimeException('Expired');
        $expected = hash_hmac('sha256', $parts['t'] . '.' . $body, $secret);
        if (!hash_equals($parts['v1'], $expected)) throw new \RuntimeException('Invalid signature');
        return true;
    }
}
`

const rustVerificationCode = `use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::time::{SystemTime, UNIX_EPOCH};

type HmacSha256 = Hmac<Sha256>;

pub fn verify_webhook_signature(secret: &str, signature: &str, body: &[u8], tolerance_secs: u64) -> Result<(), String> {
    let parts: std::collections::HashMap<&str, &str> = signature
        .split(',')
        .filter_map(|p| p.split_once('='))
        .collect();
    let ts = parts.get("t").ok_or("Missing timestamp")?;
    let sig = parts.get("v1").ok_or("Missing signature")?;
    if tolerance_secs > 0 {
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        let ts_int: u64 = ts.parse().map_err(|_| "Invalid timestamp")?;
        if now - ts_int > tolerance_secs { return Err("Timestamp expired".into()); }
    }
    let payload = format!("{}.{}", ts, String::from_utf8_lossy(body));
    let mut mac = HmacSha256::new_from_slice(secret.as_bytes()).unwrap();
    mac.update(payload.as_bytes());
    let expected = hex::encode(mac.finalize().into_bytes());
    if *sig != expected { return Err("Invalid signature".into()); }
    Ok(())
}
`

const csharpVerificationCode = `using System;
using System.Security.Cryptography;
using System.Text;

namespace Waas {
    public static class WebhookVerifier {
        public static bool Verify(string secret, string signature, byte[] body, int toleranceSecs = 300) {
            var parts = new System.Collections.Generic.Dictionary<string, string>();
            foreach (var p in signature.Split(',')) {
                var kv = p.Split('=', 2);
                if (kv.Length == 2) parts[kv[0]] = kv[1];
            }
            if (!parts.TryGetValue("t", out var ts) || !parts.TryGetValue("v1", out var sig))
                throw new SecurityException("Invalid signature format");
            if (toleranceSecs > 0) {
                var age = DateTimeOffset.UtcNow.ToUnixTimeSeconds() - long.Parse(ts);
                if (age > toleranceSecs) throw new SecurityException("Timestamp expired");
            }
            var payload = $"{ts}.{Encoding.UTF8.GetString(body)}";
            using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(secret));
            var expected = BitConverter.ToString(hmac.ComputeHash(Encoding.UTF8.GetBytes(payload))).Replace("-", "").ToLower();
            if (sig != expected) throw new SecurityException("Invalid signature");
            return true;
        }
    }
}
`
