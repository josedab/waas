<?php

declare(strict_types=1);

namespace WAAS\Exceptions;

class NotFoundException extends WAASException
{
    public function __construct(string $message = 'Resource not found')
    {
        parent::__construct($message, 404);
    }
}
