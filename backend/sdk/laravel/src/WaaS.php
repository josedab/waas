<?php

declare(strict_types=1);

namespace WaaS\Laravel;

use Illuminate\Support\Facades\Facade;

/**
 * WaaS Facade for Laravel.
 *
 * Usage:
 *   WaaS::sendWebhook($endpointId, 'user.created', $payload);
 *   WaaS::listEndpoints();
 */
class WaaS extends Facade
{
    protected static function getFacadeAccessor(): string
    {
        return WaaSClient::class;
    }
}
