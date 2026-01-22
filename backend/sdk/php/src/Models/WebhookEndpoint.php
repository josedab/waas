<?php

declare(strict_types=1);

namespace WAAS\Models;

use DateTimeImmutable;

class WebhookEndpoint
{
    public function __construct(
        public readonly string $id,
        public readonly string $tenantId,
        public readonly string $url,
        public readonly bool $isActive,
        public readonly ?RetryConfiguration $retryConfig,
        /** @var array<string, string>|null */
        public readonly ?array $customHeaders,
        public readonly DateTimeImmutable $createdAt,
        public readonly DateTimeImmutable $updatedAt,
        public readonly ?string $secret = null,
    ) {}

    /**
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: $data['id'],
            tenantId: $data['tenant_id'],
            url: $data['url'],
            isActive: $data['is_active'] ?? true,
            retryConfig: isset($data['retry_config']) 
                ? RetryConfiguration::fromArray($data['retry_config']) 
                : null,
            customHeaders: $data['custom_headers'] ?? null,
            createdAt: new DateTimeImmutable($data['created_at']),
            updatedAt: new DateTimeImmutable($data['updated_at']),
            secret: $data['secret'] ?? null,
        );
    }
}
