<?php

declare(strict_types=1);

namespace WAAS\Services;

use WAAS\Client;
use WAAS\Models\{SendWebhookRequest, SendWebhookResponse};

class Webhooks
{
    public function __construct(private readonly Client $client) {}

    public function send(SendWebhookRequest $request): SendWebhookResponse
    {
        $response = $this->client->post('/webhooks/send', $request->toArray());
        return SendWebhookResponse::fromArray($response);
    }
}
