<?php

declare(strict_types=1);

namespace WAAS;

class Config
{
    public function __construct(
        public readonly string $apiKey,
        public readonly string $baseUrl = 'https://api.waas-platform.com/api/v1',
        public readonly float $timeout = 30.0,
        public readonly float $connectTimeout = 10.0,
    ) {}
}
