<?php

declare(strict_types=1);

namespace WaaS\Laravel\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;
use WaaS\Laravel\WebhookVerifier;

/**
 * Laravel middleware that verifies incoming WaaS webhook signatures.
 *
 * Register in your route or middleware group:
 *   Route::post('/webhooks', [WebhookController::class, 'handle'])
 *       ->middleware(VerifyWebhookSignature::class);
 *
 * Or in app/Http/Kernel.php:
 *   'waas.verify' => \WaaS\Laravel\Middleware\VerifyWebhookSignature::class,
 */
class VerifyWebhookSignature
{
    private WebhookVerifier $verifier;

    public function __construct(WebhookVerifier $verifier)
    {
        $this->verifier = $verifier;
    }

    public function handle(Request $request, Closure $next): Response
    {
        $signature = $request->header(WebhookVerifier::SIGNATURE_HEADER, '');
        $timestamp = $request->header(WebhookVerifier::TIMESTAMP_HEADER, '');
        $secret = config('waas.signing_secret', '');
        $body = $request->getContent();

        try {
            $this->verifier->verifySignature($body, $signature, $timestamp, $secret);
        } catch (\Throwable $e) {
            return response()->json([
                'error' => 'Webhook signature verification failed',
                'message' => $e->getMessage(),
            ], 403);
        }

        return $next($request);
    }
}
