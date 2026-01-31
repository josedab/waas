<?php

declare(strict_types=1);

namespace WaaS\SDK;

/**
 * Standalone, zero-dependency WaaS webhook signature verification.
 *
 * Usage:
 *   $verifier = new WebhookVerifier('whsec_your_secret');
 *   $verifier->verify($payload, $signatureHeader, $timestampHeader);
 *
 * Laravel middleware:
 *   use WaaS\SDK\WebhookVerifier;
 *   // In app/Http/Kernel.php route middleware:
 *   'waas.verify' => \WaaS\SDK\VerifyWebhookMiddleware::class,
 */
class WebhookVerifier
{
    public const SIGNATURE_HEADER = 'X-WaaS-Signature';
    public const TIMESTAMP_HEADER = 'X-WaaS-Timestamp';
    private const DEFAULT_TOLERANCE = 300;

    /** @var string[] */
    private array $secrets;
    private int $timestampTolerance;

    /**
     * @param string   $secret              Primary signing secret
     * @param string[] $additionalSecrets   Additional secrets for key rotation
     * @param int      $timestampTolerance  Max allowed timestamp age in seconds
     */
    public function __construct(
        string $secret,
        array $additionalSecrets = [],
        int $timestampTolerance = self::DEFAULT_TOLERANCE
    ) {
        $this->secrets = array_merge([$secret], $additionalSecrets);
        $this->timestampTolerance = $timestampTolerance;
    }

    /**
     * Verify a webhook signature.
     *
     * @param string $payload   Raw request body
     * @param string $signature Value of X-WaaS-Signature header (v1=<hex>)
     * @param string $timestamp Value of X-WaaS-Timestamp header (Unix seconds)
     * @return bool True if valid
     * @throws MissingSignatureException
     * @throws InvalidSignatureException
     * @throws TimestampExpiredException
     * @throws VerificationException
     */
    public function verify(string $payload, string $signature, string $timestamp = ''): bool
    {
        if (empty($signature)) {
            throw new MissingSignatureException('Missing signature header');
        }

        $this->verifyTimestamp($timestamp);

        $sigBytes = $this->parseSignature($signature);
        $signedPayload = "{$timestamp}.{$payload}";

        foreach ($this->secrets as $secret) {
            $expected = hash_hmac('sha256', $signedPayload, $secret, true);
            if (hash_equals($expected, $sigBytes)) {
                return true;
            }
        }

        throw new InvalidSignatureException('Signature does not match');
    }

    /** Generate a signature for testing purposes. */
    public function sign(string $payload, string $timestamp): string
    {
        $signedPayload = "{$timestamp}.{$payload}";
        $mac = hash_hmac('sha256', $signedPayload, $this->secrets[0]);
        return "v1={$mac}";
    }

    private function verifyTimestamp(string $timestamp): void
    {
        if (empty($timestamp) || $this->timestampTolerance <= 0) {
            return;
        }

        if (!is_numeric($timestamp)) {
            throw new VerificationException("Malformed timestamp: {$timestamp}");
        }

        $diff = abs(time() - (int) $timestamp);
        if ($diff > $this->timestampTolerance) {
            throw new TimestampExpiredException(
                "Timestamp expired: {$diff}s > {$this->timestampTolerance}s tolerance"
            );
        }
    }

    private function parseSignature(string $header): string
    {
        $parts = explode('=', $header, 2);
        if (count($parts) !== 2) {
            throw new VerificationException('Malformed signature header');
        }

        $decoded = hex2bin($parts[1]);
        if ($decoded === false) {
            throw new VerificationException('Invalid hex in signature');
        }

        return $decoded;
    }
}

class VerificationException extends \RuntimeException {}
class MissingSignatureException extends VerificationException {}
class InvalidSignatureException extends VerificationException {}
class TimestampExpiredException extends VerificationException {}

/**
 * Laravel middleware for WaaS webhook verification.
 *
 * Register in app/Http/Kernel.php:
 *   'waas.verify' => \WaaS\SDK\VerifyWebhookMiddleware::class
 *
 * Usage in routes:
 *   Route::post('/webhooks', ...)->middleware('waas.verify');
 *
 * Set WAAS_WEBHOOK_SECRET in your .env file.
 */
class VerifyWebhookMiddleware
{
    public function handle($request, \Closure $next)
    {
        $secret = config('services.waas.webhook_secret', env('WAAS_WEBHOOK_SECRET', ''));
        if (empty($secret)) {
            return $next($request);
        }

        $signature = $request->header(WebhookVerifier::SIGNATURE_HEADER, '');
        if (empty($signature)) {
            return $next($request);
        }

        $timestamp = $request->header(WebhookVerifier::TIMESTAMP_HEADER, '');
        $payload = $request->getContent();

        try {
            $verifier = new WebhookVerifier($secret);
            $verifier->verify($payload, $signature, $timestamp);
        } catch (VerificationException $e) {
            return response()->json(['error' => $e->getMessage()], 401);
        }

        return $next($request);
    }
}
