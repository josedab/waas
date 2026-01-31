"""
WaaS Webhook Signature Verification - Standalone, zero-dependency module.

Usage:
    from waas_sdk.verify import WebhookVerifier

    verifier = WebhookVerifier("whsec_your_secret")
    verifier.verify(payload_bytes, signature_header, timestamp_header)

Framework middleware:
    # Django
    from waas_sdk.verify import django_middleware
    MIDDLEWARE = ['waas_sdk.verify.django_middleware']

    # Flask/Express-like
    from waas_sdk.verify import verify_request
"""

import hashlib
import hmac
import math
import time as _time


class VerificationError(Exception):
    """Base class for signature verification errors."""
    pass


class MissingSignatureError(VerificationError):
    """Raised when the signature header is missing."""
    pass


class InvalidSignatureError(VerificationError):
    """Raised when the signature does not match."""
    pass


class TimestampExpiredError(VerificationError):
    """Raised when the timestamp is outside the tolerance window."""
    pass


class WebhookVerifier:
    """Verifies WaaS webhook signatures using HMAC-SHA256.

    Args:
        secret: Primary signing secret.
        secrets: Additional secrets for key rotation.
        timestamp_tolerance: Maximum allowed age in seconds (default: 300).
    """

    SIGNATURE_HEADER = "X-WaaS-Signature"
    TIMESTAMP_HEADER = "X-WaaS-Timestamp"

    def __init__(self, secret: str, secrets: list = None, timestamp_tolerance: int = 300):
        self._secrets = [secret] + (secrets or [])
        self._timestamp_tolerance = timestamp_tolerance

    def verify(self, payload: bytes, signature: str, timestamp: str = "") -> bool:
        """Verify a webhook signature.

        Args:
            payload: Raw request body bytes.
            signature: Value of X-WaaS-Signature header (format: v1=<hex>).
            timestamp: Value of X-WaaS-Timestamp header (Unix seconds string).

        Returns:
            True if signature is valid.

        Raises:
            MissingSignatureError: If signature is empty.
            InvalidSignatureError: If signature doesn't match any secret.
            TimestampExpiredError: If timestamp is outside tolerance.
        """
        if not signature:
            raise MissingSignatureError("Missing signature header")

        self._verify_timestamp(timestamp)

        sig_bytes = self._parse_signature(signature)
        signed_payload = f"{timestamp}.{payload.decode('utf-8') if isinstance(payload, bytes) else payload}"

        for secret in self._secrets:
            expected = hmac.new(
                secret.encode("utf-8"),
                signed_payload.encode("utf-8"),
                hashlib.sha256,
            ).digest()
            if hmac.compare_digest(sig_bytes, expected):
                return True

        raise InvalidSignatureError("Signature does not match")

    def sign(self, payload: bytes, timestamp: str) -> str:
        """Generate a signature for testing purposes."""
        signed_payload = f"{timestamp}.{payload.decode('utf-8') if isinstance(payload, bytes) else payload}"
        mac = hmac.new(
            self._secrets[0].encode("utf-8"),
            signed_payload.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        return f"v1={mac}"

    def _verify_timestamp(self, timestamp: str):
        if not timestamp or self._timestamp_tolerance <= 0:
            return
        try:
            ts = int(timestamp)
        except ValueError as e:
            raise VerificationError(f"Malformed timestamp: {e}")

        diff = abs(_time.time() - ts)
        if diff > self._timestamp_tolerance:
            raise TimestampExpiredError(
                f"Timestamp expired: {diff:.0f}s > {self._timestamp_tolerance}s tolerance"
            )

    @staticmethod
    def _parse_signature(header: str) -> bytes:
        parts = header.split("=", 1)
        if len(parts) != 2:
            raise VerificationError("Malformed signature header")
        return bytes.fromhex(parts[1])


def verify_request(secret: str, payload: bytes, headers: dict) -> bool:
    """Convenience function to verify a webhook request.

    Args:
        secret: Signing secret.
        payload: Raw request body.
        headers: Request headers dict.
    """
    v = WebhookVerifier(secret)
    sig = headers.get(WebhookVerifier.SIGNATURE_HEADER, "")
    ts = headers.get(WebhookVerifier.TIMESTAMP_HEADER, "")
    return v.verify(payload, sig, ts)


def django_middleware(get_response):
    """Django middleware for WaaS webhook signature verification.

    Configure WAAS_WEBHOOK_SECRET in Django settings.

    Usage in settings.py:
        MIDDLEWARE = [..., 'waas_sdk.verify.django_middleware']
        WAAS_WEBHOOK_SECRET = 'whsec_...'
    """
    from django.conf import settings
    from django.http import HttpResponseForbidden

    secret = getattr(settings, "WAAS_WEBHOOK_SECRET", "")

    def middleware(request):
        if request.method == "POST" and request.content_type == "application/json":
            sig = request.META.get("HTTP_X_WAAS_SIGNATURE", "")
            ts = request.META.get("HTTP_X_WAAS_TIMESTAMP", "")
            if sig:
                try:
                    v = WebhookVerifier(secret)
                    v.verify(request.body, sig, ts)
                except VerificationError as e:
                    return HttpResponseForbidden(f"Webhook verification failed: {e}")
        return get_response(request)

    return middleware
