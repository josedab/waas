<?php

declare(strict_types=1);

namespace WAAS\Models;

use DateTimeImmutable;

class DeliveryAttempt
{
    public function __construct(
        public readonly string $id,
        public readonly string $endpointId,
        public readonly string $payloadHash,
        public readonly int $payloadSize,
        public readonly string $status,
        public readonly ?int $httpStatus,
        public readonly ?string $responseBody,
        public readonly ?string $errorMessage,
        public readonly int $attemptNumber,
        public readonly DateTimeImmutable $scheduledAt,
        public readonly ?DateTimeImmutable $deliveredAt,
        public readonly DateTimeImmutable $createdAt,
    ) {}

    /**
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: $data['id'],
            endpointId: $data['endpoint_id'],
            payloadHash: $data['payload_hash'],
            payloadSize: $data['payload_size'],
            status: $data['status'],
            httpStatus: $data['http_status'] ?? null,
            responseBody: $data['response_body'] ?? null,
            errorMessage: $data['error_message'] ?? null,
            attemptNumber: $data['attempt_number'],
            scheduledAt: new DateTimeImmutable($data['scheduled_at']),
            deliveredAt: isset($data['delivered_at']) ? new DateTimeImmutable($data['delivered_at']) : null,
            createdAt: new DateTimeImmutable($data['created_at']),
        );
    }
}
