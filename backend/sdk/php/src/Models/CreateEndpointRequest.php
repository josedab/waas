<?php

declare(strict_types=1);

namespace WAAS\Models;

class CreateEndpointRequest
{
    /**
     * @param array<string, int>|RetryConfiguration|null $retryConfig
     * @param array<string, string>|null $customHeaders
     */
    public function __construct(
        public readonly string $url,
        public readonly array|RetryConfiguration|null $retryConfig = null,
        public readonly ?array $customHeaders = null,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        $data = ['url' => $this->url];

        if ($this->retryConfig !== null) {
            $data['retry_config'] = $this->retryConfig instanceof RetryConfiguration 
                ? $this->retryConfig->toArray() 
                : $this->retryConfig;
        }

        if ($this->customHeaders !== null) {
            $data['custom_headers'] = $this->customHeaders;
        }

        return $data;
    }
}
