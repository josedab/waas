<?php

namespace WaaS\Laravel;

/**
 * Standalone, zero-dependency WaaS webhook signature verification for Laravel.
 *
 * Algorithm:
 *   1. Parse X-WaaS-Signature header (format: v1=<hex_encoded_hmac>)
 *   2. Parse X-WaaS-Timestamp header (Unix seconds string)
 *   3. Validate timestamp is within tolerance (default 300s)
 *   4. Construct signed content: timestamp + "." + raw_payload_body
 *   5. Compute HMAC-SHA256 with secret key
 *   6. Compare using timing-safe hash_equals()
 *
 * Usage:
 *   $verifier = new WebhookVerifier();
 *   $verifier->verifySignature($body, $sigHeader, $tsHeader, $secret);
 *   $verifier->verifyWithMultipleSecrets($body, $sigHeader, $tsHeader, [$s1, $s2]);
 */
class WebhookVerifier
{
    public const SIGNATURE_HEADER = 'X-WaaS-Signature';
    public const TIMESTAMP_HEADER = 'X-WaaS-Timestamp';
    private const DEFAULT_TOLERANCE = 300;

    /**
     * Verify a WaaS webhook signature.
     *
     * @param string $payload          Raw request body
     * @param string $signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
     * @param string $timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
     * @param string $secret           Signing secret
     * @param int    $toleranceSeconds Max allowed timestamp age in seconds (0 to disable)
     * @return bool True if valid
     * @throws WebhookVerificationException
     */
    public function verifySignature(
        string $payload,
        string $signatureHeader,
        string $timestampHeader,
        string $secret,
        int $toleranceSeconds = self::DEFAULT_TOLERANCE
    ): bool {
        if (empty($secret)) {
            throw new WebhookVerificationException('No signing secret configured');
        }
        if (empty($signatureHeader)) {
            throw new WebhookVerificationException('Missing signature header');
        }

        $sig = $this->parseSignature($signatureHeader);
        $this->validateTimestamp($timestampHeader, $toleranceSeconds);

        $signedContent = $timestampHeader . '.' . $payload;
        $expected = hash_hmac('sha256', $signedContent, $secret);

        if (!hash_equals($expected, $sig)) {
            throw new WebhookVerificationException('Signature does not match');
        }

        return true;
    }

    /**
     * Verify a webhook signature against multiple secrets (for key rotation).
     *
     * Tries each secret in order and returns true on the first match.
     *
     * @param string   $payload          Raw request body
     * @param string   $signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
     * @param string   $timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
     * @param string[] $secrets          Array of signing secrets to try
     * @param int      $toleranceSeconds Max allowed timestamp age in seconds (0 to disable)
     * @return bool True if valid against any secret
     * @throws WebhookVerificationException
     */
    public function verifyWithMultipleSecrets(
        string $payload,
        string $signatureHeader,
        string $timestampHeader,
        array $secrets,
        int $toleranceSeconds = self::DEFAULT_TOLERANCE
    ): bool {
        if (empty($secrets)) {
            throw new WebhookVerificationException('No signing secrets configured');
        }
        if (empty($signatureHeader)) {
            throw new WebhookVerificationException('Missing signature header');
        }

        $sig = $this->parseSignature($signatureHeader);
        $this->validateTimestamp($timestampHeader, $toleranceSeconds);

        $signedContent = $timestampHeader . '.' . $payload;

        foreach ($secrets as $secret) {
            $expected = hash_hmac('sha256', $signedContent, $secret);
            if (hash_equals($expected, $sig)) {
                return true;
            }
        }

        throw new WebhookVerificationException('Signature does not match');
    }

    /**
     * Generate a signature for testing purposes.
     */
    public function sign(string $payload, string $timestamp, string $secret): string
    {
        $signedContent = $timestamp . '.' . $payload;
        return 'v1=' . hash_hmac('sha256', $signedContent, $secret);
    }

    private function parseSignature(string $header): string
    {
        $parts = explode('=', $header, 2);
        if (count($parts) !== 2 || $parts[0] !== 'v1') {
            throw new WebhookVerificationException('Malformed signature header: expected v1=<hex>');
        }
        return $parts[1];
    }

    private function validateTimestamp(string $timestamp, int $toleranceSeconds): void
    {
        if (empty($timestamp) || $toleranceSeconds <= 0) {
            return;
        }

        if (!is_numeric($timestamp)) {
            throw new WebhookVerificationException("Malformed timestamp: {$timestamp}");
        }

        $diff = abs(time() - (int) $timestamp);
        if ($diff > $toleranceSeconds) {
            throw new WebhookVerificationException(
                "Timestamp expired: {$diff}s > {$toleranceSeconds}s tolerance"
            );
        }
    }
}

class WebhookVerificationException extends \RuntimeException {}
