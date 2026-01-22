<?php

declare(strict_types=1);

namespace WAAS\Models;

use DateTimeImmutable;

class SendWebhookResponse
{
    public function __construct(
        public readonly string $deliveryId,
        public readonly string $endpointId,
        public readonly string $status,
        public readonly DateTimeImmutable $scheduledAt,
    ) {}

    /**
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            deliveryId: $data['delivery_id'],
            endpointId: $data['endpoint_id'],
            status: $data['status'],
            scheduledAt: new DateTimeImmutable($data['scheduled_at']),
        );
    }
}
