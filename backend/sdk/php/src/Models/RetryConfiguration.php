<?php

declare(strict_types=1);

namespace WAAS\Models;

class RetryConfiguration
{
    public function __construct(
        public readonly int $maxAttempts = 5,
        public readonly int $initialDelayMs = 1000,
        public readonly int $maxDelayMs = 300000,
        public readonly int $backoffMultiplier = 2,
    ) {}

    /**
     * @param array<string, mixed> $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            maxAttempts: $data['max_attempts'] ?? 5,
            initialDelayMs: $data['initial_delay_ms'] ?? 1000,
            maxDelayMs: $data['max_delay_ms'] ?? 300000,
            backoffMultiplier: $data['backoff_multiplier'] ?? 2,
        );
    }

    /**
     * @return array<string, int>
     */
    public function toArray(): array
    {
        return [
            'max_attempts' => $this->maxAttempts,
            'initial_delay_ms' => $this->initialDelayMs,
            'max_delay_ms' => $this->maxDelayMs,
            'backoff_multiplier' => $this->backoffMultiplier,
        ];
    }
}
