<?php

declare(strict_types=1);

namespace WAAS\Exceptions;

class AuthenticationException extends WAASException
{
    public function __construct(string $message = 'Authentication failed')
    {
        parent::__construct($message, 401);
    }
}
