<?php

namespace WaaS\Laravel;

class WebhookVerifier
{
    private string $secret;
    private int $toleranceSeconds;

    public function __construct(string $secret, int $toleranceSeconds = 300)
    {
        $this->secret = $secret;
        $this->toleranceSeconds = $toleranceSeconds;
    }

    /**
     * Verify a WaaS webhook signature.
     *
     * @throws WebhookVerificationException
     */
    public function verify(string $signature, string $body): bool
    {
        if (empty($this->secret)) {
            throw new WebhookVerificationException('No signing secret configured');
        }

        $parts = [];
        foreach (explode(',', $signature) as $p) {
            $kv = explode('=', $p, 2);
            if (count($kv) === 2) {
                $parts[$kv[0]] = $kv[1];
            }
        }

        $ts = $parts['t'] ?? null;
        $sig = $parts['v1'] ?? null;

        if (!$ts || !$sig) {
            throw new WebhookVerificationException('Invalid signature format');
        }

        if ($this->toleranceSeconds > 0 && (time() - intval($ts)) > $this->toleranceSeconds) {
            throw new WebhookVerificationException('Timestamp expired');
        }

        $expected = hash_hmac('sha256', $ts . '.' . $body, $this->secret);

        if (!hash_equals($sig, $expected)) {
            throw new WebhookVerificationException('Signature mismatch');
        }

        return true;
    }
}

class WebhookVerificationException extends \Exception {}
