<?php

declare(strict_types=1);

namespace WaaS\Laravel;

use Illuminate\Support\Facades\Http;

/**
 * WaaS API Client for Laravel.
 *
 * Wraps Laravel's HTTP client to provide a simple interface for the WaaS API.
 *
 * Usage:
 *   $client = app(WaaSClient::class);
 *   $client->sendWebhook($endpointId, 'user.created', ['user_id' => 1]);
 */
class WaaSClient
{
    private string $apiKey;
    private string $baseUrl;
    private int $timeout;

    public function __construct(string $apiKey, string $baseUrl = 'http://localhost:8080', int $timeout = 30)
    {
        $this->apiKey = $apiKey;
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->timeout = $timeout;
    }

    // --- Endpoint Management ---

    /**
     * Create a new webhook endpoint.
     *
     * @param string $url Target URL for webhooks
     * @param array<string, mixed> $options Optional: custom_headers, retry_config
     * @return array<string, mixed>
     */
    public function createEndpoint(string $url, array $options = []): array
    {
        return $this->post('/api/v1/webhooks/endpoints', array_merge(['url' => $url], $options));
    }

    /**
     * List all webhook endpoints for the current tenant.
     *
     * @param int $limit Max results (default 50, max 100)
     * @param int $offset Offset for pagination
     * @return array<string, mixed>
     */
    public function listEndpoints(int $limit = 50, int $offset = 0): array
    {
        return $this->get("/api/v1/webhooks/endpoints?limit={$limit}&offset={$offset}");
    }

    /**
     * Get a single webhook endpoint.
     */
    public function getEndpoint(string $endpointId): array
    {
        return $this->get("/api/v1/webhooks/endpoints/{$endpointId}");
    }

    /**
     * Delete a webhook endpoint.
     */
    public function deleteEndpoint(string $endpointId): void
    {
        $this->delete("/api/v1/webhooks/endpoints/{$endpointId}");
    }

    // --- Webhook Sending ---

    /**
     * Send a webhook to a specific endpoint.
     *
     * @param string $endpointId Target endpoint UUID
     * @param string $eventType  Event type (e.g. 'user.created')
     * @param array<string, mixed> $payload Event data
     * @return array<string, mixed>
     */
    public function sendWebhook(string $endpointId, string $eventType, array $payload): array
    {
        return $this->post('/api/v1/webhooks/send', [
            'endpoint_id' => $endpointId,
            'event_type'  => $eventType,
            'payload'     => $payload,
        ]);
    }

    /**
     * Send a webhook to all active endpoints for the tenant.
     *
     * @param string $eventType Event type
     * @param array<string, mixed> $payload Event data
     * @return array<string, mixed>
     */
    public function broadcastWebhook(string $eventType, array $payload): array
    {
        return $this->post('/api/v1/webhooks/send/batch', [
            'event_type' => $eventType,
            'payload'    => $payload,
        ]);
    }

    // --- Delivery ---

    /**
     * Get delivery details.
     */
    public function getDelivery(string $deliveryId): array
    {
        return $this->get("/api/v1/webhooks/deliveries/{$deliveryId}");
    }

    /**
     * List delivery history with optional filters.
     *
     * @param array<string, string> $filters Optional: status, endpoint_id, start_date, end_date
     */
    public function listDeliveries(array $filters = [], int $limit = 50, int $offset = 0): array
    {
        $query = http_build_query(array_merge($filters, [
            'limit'  => $limit,
            'offset' => $offset,
        ]));
        return $this->get("/api/v1/webhooks/deliveries?{$query}");
    }

    // --- Tenant ---

    /**
     * Get the current tenant info.
     */
    public function getTenant(): array
    {
        return $this->get('/api/v1/tenant');
    }

    // --- HTTP Internals ---

    private function get(string $path): array
    {
        $response = Http::withHeaders($this->headers())
            ->timeout($this->timeout)
            ->get($this->baseUrl . $path);

        $this->handleErrors($response);
        return $response->json() ?? [];
    }

    private function post(string $path, array $body): array
    {
        $response = Http::withHeaders($this->headers())
            ->timeout($this->timeout)
            ->post($this->baseUrl . $path, $body);

        $this->handleErrors($response);
        return $response->json() ?? [];
    }

    private function delete(string $path): void
    {
        $response = Http::withHeaders($this->headers())
            ->timeout($this->timeout)
            ->delete($this->baseUrl . $path);

        $this->handleErrors($response);
    }

    /**
     * @return array<string, string>
     */
    private function headers(): array
    {
        return [
            'Authorization' => 'Bearer ' . $this->apiKey,
            'Content-Type'  => 'application/json',
            'Accept'        => 'application/json',
            'User-Agent'    => 'waas-laravel/1.0.0',
        ];
    }

    private function handleErrors($response): void
    {
        if ($response->successful()) {
            return;
        }

        $body = $response->json();
        $message = $body['message'] ?? $body['error'] ?? 'API request failed';

        match ($response->status()) {
            401 => throw new \RuntimeException("WaaS authentication failed: {$message}"),
            404 => throw new \RuntimeException("WaaS not found: {$message}"),
            429 => throw new \RuntimeException("WaaS rate limited: {$message}"),
            default => throw new \RuntimeException("WaaS API error ({$response->status()}): {$message}"),
        };
    }
}
