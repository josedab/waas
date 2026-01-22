<?php

declare(strict_types=1);

namespace WAAS\Exceptions;

class ValidationException extends WAASException
{
    /**
     * @param array<string, mixed>|null $details
     */
    public function __construct(
        string $message = 'Validation failed',
        public readonly ?array $details = null,
    ) {
        parent::__construct($message, 422);
    }
}
