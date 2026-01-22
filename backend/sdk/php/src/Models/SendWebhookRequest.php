<?php

declare(strict_types=1);

namespace WAAS\Models;

class SendWebhookRequest
{
    /**
     * @param mixed $payload
     * @param array<string, string>|null $headers
     */
    public function __construct(
        public readonly ?string $endpointId = null,
        public readonly mixed $payload = null,
        public readonly ?array $headers = null,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        $data = ['payload' => $this->payload];

        if ($this->endpointId !== null) {
            $data['endpoint_id'] = $this->endpointId;
        }

        if ($this->headers !== null) {
            $data['headers'] = $this->headers;
        }

        return $data;
    }
}
