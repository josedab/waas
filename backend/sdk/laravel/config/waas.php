<?php

return [
    /*
    |--------------------------------------------------------------------------
    | WaaS API Key
    |--------------------------------------------------------------------------
    |
    | Your WaaS API key, used to authenticate requests. Create a tenant at
    | your WaaS instance to receive one (format: wh_...).
    |
    */
    'api_key' => env('WAAS_API_KEY', ''),

    /*
    |--------------------------------------------------------------------------
    | WaaS API URL
    |--------------------------------------------------------------------------
    |
    | Base URL for the WaaS API. Defaults to a local development instance.
    |
    */
    'api_url' => env('WAAS_API_URL', 'http://localhost:8080'),

    /*
    |--------------------------------------------------------------------------
    | Signing Secret
    |--------------------------------------------------------------------------
    |
    | Used to verify incoming webhook signatures. Each endpoint has its own
    | secret, set via the WaaS dashboard or API.
    |
    */
    'signing_secret' => env('WAAS_SIGNING_SECRET', ''),

    /*
    |--------------------------------------------------------------------------
    | Request Timeout
    |--------------------------------------------------------------------------
    |
    | Maximum number of seconds to wait for an API response.
    |
    */
    'timeout' => env('WAAS_TIMEOUT', 30),
];
