// Webhook Service Platform API
//
// A comprehensive webhook-as-a-service platform that enables companies to reliably send, receive, and manage webhooks without building their own infrastructure.
//
// The platform provides APIs for webhook management, reliable delivery with retry mechanisms, monitoring and analytics, and developer-friendly tools for testing and debugging webhooks.
//
//	Title: Webhook Service Platform API
//	Description: A comprehensive webhook-as-a-service platform that enables companies to reliably send, receive, and manage webhooks without building their own infrastructure.
//	Version: 1.0.0
//	Host: localhost:8080
//	BasePath: /api/v1
//	Schemes: http, https
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	SecurityDefinitions:
//	ApiKeyAuth:
//	  type: apiKey
//	  in: header
//	  name: X-API-Key
//	  description: API key for authentication. Get your API key by creating a tenant account.
//
// swagger:meta
package main

import (
	_ "webhook-platform/docs"
)