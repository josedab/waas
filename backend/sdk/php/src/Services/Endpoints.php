<?php

declare(strict_types=1);

namespace WAAS\Services;

use WAAS\Client;
use WAAS\Models\{CreateEndpointRequest, WebhookEndpoint};

class Endpoints
{
    public function __construct(private readonly Client $client) {}

    public function create(CreateEndpointRequest $request): WebhookEndpoint
    {
        $response = $this->client->post('/webhooks/endpoints', $request->toArray());
        return WebhookEndpoint::fromArray($response);
    }

    public function get(string $endpointId): WebhookEndpoint
    {
        $response = $this->client->get("/webhooks/endpoints/{$endpointId}");
        return WebhookEndpoint::fromArray($response);
    }

    /**
     * @return array{endpoints: WebhookEndpoint[], total: int, page: int, per_page: int}
     */
    public function list(int $page = 1, int $perPage = 20): array
    {
        $response = $this->client->get("/webhooks/endpoints?page={$page}&per_page={$perPage}");
        return [
            'endpoints' => array_map(
                fn($e) => WebhookEndpoint::fromArray($e),
                $response['endpoints'] ?? []
            ),
            'total' => $response['total'] ?? 0,
            'page' => $response['page'] ?? $page,
            'per_page' => $response['per_page'] ?? $perPage,
        ];
    }

    /**
     * @param array<string, mixed> $attributes
     */
    public function update(string $endpointId, array $attributes): WebhookEndpoint
    {
        $response = $this->client->patch("/webhooks/endpoints/{$endpointId}", $attributes);
        return WebhookEndpoint::fromArray($response);
    }

    public function delete(string $endpointId): void
    {
        $this->client->delete("/webhooks/endpoints/{$endpointId}");
    }

    public function rotateSecret(string $endpointId): string
    {
        $response = $this->client->post("/webhooks/endpoints/{$endpointId}/rotate-secret");
        return $response['secret'];
    }
}
