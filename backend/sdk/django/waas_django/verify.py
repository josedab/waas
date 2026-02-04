"""Webhook signature verification for WaaS"""

import hashlib
import hmac
import time


class WebhookVerificationError(Exception):
    """Raised when webhook signature verification fails."""
    pass


def verify_webhook_signature(
    secret: str,
    signature: str,
    body: bytes,
    tolerance: int = 300,
) -> bool:
    """Verify a WaaS webhook signature.

    Args:
        secret: Your endpoint signing secret
        signature: The X-Webhook-Signature header value
        body: Raw request body bytes
        tolerance: Max age in seconds (default 300 = 5 minutes)

    Returns:
        True if signature is valid

    Raises:
        WebhookVerificationError: If verification fails
    """
    if not secret:
        raise WebhookVerificationError('No signing secret configured')
    if not signature:
        raise WebhookVerificationError('Missing signature header')

    parts = {}
    for part in signature.split(','):
        if '=' in part:
            k, v = part.split('=', 1)
            parts[k] = v

    ts = parts.get('t')
    sig = parts.get('v1')
    if not ts or not sig:
        raise WebhookVerificationError('Invalid signature format')

    # Check timestamp tolerance
    if tolerance > 0:
        age = time.time() - int(ts)
        if age > tolerance:
            raise WebhookVerificationError('Timestamp expired')

    # Compute expected signature
    if isinstance(body, bytes):
        body_str = body.decode('utf-8')
    else:
        body_str = body

    payload = f'{ts}.{body_str}'
    expected = hmac.new(
        secret.encode('utf-8'),
        payload.encode('utf-8'),
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(sig, expected):
        raise WebhookVerificationError('Signature mismatch')

    return True
