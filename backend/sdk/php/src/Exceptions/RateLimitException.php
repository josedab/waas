<?php

declare(strict_types=1);

namespace WAAS\Exceptions;

class RateLimitException extends WAASException
{
    public function __construct(
        string $message = 'Rate limit exceeded',
        public readonly ?int $retryAfter = null,
    ) {
        parent::__construct($message, 429);
    }
}
