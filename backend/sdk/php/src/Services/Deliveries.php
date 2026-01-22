<?php

declare(strict_types=1);

namespace WAAS\Services;

use WAAS\Client;
use WAAS\Models\{DeliveryAttempt, SendWebhookResponse};

class Deliveries
{
    public function __construct(private readonly Client $client) {}

    public function get(string $deliveryId): DeliveryAttempt
    {
        $response = $this->client->get("/webhooks/deliveries/{$deliveryId}");
        return DeliveryAttempt::fromArray($response);
    }

    /**
     * @return array{deliveries: DeliveryAttempt[], total: int, page: int, per_page: int}
     */
    public function list(
        ?string $endpointId = null,
        ?string $status = null,
        int $page = 1,
        int $perPage = 20
    ): array {
        $params = ["page={$page}", "per_page={$perPage}"];
        if ($endpointId !== null) {
            $params[] = "endpoint_id={$endpointId}";
        }
        if ($status !== null) {
            $params[] = "status={$status}";
        }

        $response = $this->client->get('/webhooks/deliveries?' . implode('&', $params));
        return [
            'deliveries' => array_map(
                fn($d) => DeliveryAttempt::fromArray($d),
                $response['deliveries'] ?? []
            ),
            'total' => $response['total'] ?? 0,
            'page' => $response['page'] ?? $page,
            'per_page' => $response['per_page'] ?? $perPage,
        ];
    }

    public function retry(string $deliveryId): SendWebhookResponse
    {
        $response = $this->client->post("/webhooks/deliveries/{$deliveryId}/retry");
        return SendWebhookResponse::fromArray($response);
    }
}
