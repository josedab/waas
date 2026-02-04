"""Standalone, zero-dependency WaaS webhook signature verification for Django.

Algorithm:
    1. Parse X-WaaS-Signature header (format: v1=<hex_encoded_hmac>)
    2. Parse X-WaaS-Timestamp header (Unix seconds string)
    3. Validate timestamp is within tolerance (default 300s)
    4. Construct signed content: timestamp + "." + raw_payload_body
    5. Compute HMAC-SHA256 with secret key
    6. Compare using hmac.compare_digest() (timing-safe)

Usage:
    from waas_django.verify import verify_signature, verify_with_multiple_secrets

    verify_signature(body, sig_header, ts_header, secret)
    verify_with_multiple_secrets(body, sig_header, ts_header, [secret1, secret2])

Django middleware:
    MIDDLEWARE = [..., 'waas_django.verify.WaaSWebhookMiddleware']
    WAAS_SIGNING_SECRET = 'whsec_...'
"""

import hashlib
import hmac
import time


SIGNATURE_HEADER = "X-WaaS-Signature"
TIMESTAMP_HEADER = "X-WaaS-Timestamp"
DEFAULT_TOLERANCE = 300


class WebhookVerificationError(Exception):
    """Raised when webhook signature verification fails."""
    pass


def verify_signature(
    payload: str,
    signature_header: str,
    timestamp_header: str,
    secret: str,
    tolerance_seconds: int = DEFAULT_TOLERANCE,
) -> bool:
    """Verify a WaaS webhook signature.

    Args:
        payload: Raw request body string
        signature_header: Value of X-WaaS-Signature header (v1=<hex>)
        timestamp_header: Value of X-WaaS-Timestamp header (Unix seconds)
        secret: Signing secret
        tolerance_seconds: Max age in seconds (default 300, 0 to disable)

    Returns:
        True if signature is valid

    Raises:
        WebhookVerificationError: If verification fails
    """
    if not secret:
        raise WebhookVerificationError("No signing secret configured")
    if not signature_header:
        raise WebhookVerificationError("Missing signature header")

    sig_hex = _parse_signature(signature_header)
    _validate_timestamp(timestamp_header, tolerance_seconds)

    if isinstance(payload, bytes):
        payload = payload.decode("utf-8")

    signed_content = f"{timestamp_header}.{payload}"
    expected = hmac.new(
        secret.encode("utf-8"),
        signed_content.encode("utf-8"),
        hashlib.sha256,
    ).hexdigest()

    if not hmac.compare_digest(sig_hex, expected):
        raise WebhookVerificationError("Signature does not match")

    return True


def verify_with_multiple_secrets(
    payload: str,
    signature_header: str,
    timestamp_header: str,
    secrets: list,
    tolerance_seconds: int = DEFAULT_TOLERANCE,
) -> bool:
    """Verify a webhook signature against multiple secrets (for key rotation).

    Tries each secret in order and returns True on the first match.

    Args:
        payload: Raw request body string
        signature_header: Value of X-WaaS-Signature header (v1=<hex>)
        timestamp_header: Value of X-WaaS-Timestamp header (Unix seconds)
        secrets: List of signing secrets to try
        tolerance_seconds: Max age in seconds (default 300, 0 to disable)

    Returns:
        True if signature matches any secret

    Raises:
        WebhookVerificationError: If verification fails
    """
    if not secrets:
        raise WebhookVerificationError("No signing secrets configured")
    if not signature_header:
        raise WebhookVerificationError("Missing signature header")

    sig_hex = _parse_signature(signature_header)
    _validate_timestamp(timestamp_header, tolerance_seconds)

    if isinstance(payload, bytes):
        payload = payload.decode("utf-8")

    signed_content = f"{timestamp_header}.{payload}"

    for secret in secrets:
        expected = hmac.new(
            secret.encode("utf-8"),
            signed_content.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        if hmac.compare_digest(sig_hex, expected):
            return True

    raise WebhookVerificationError("Signature does not match")


def sign(payload: str, timestamp: str, secret: str) -> str:
    """Generate a signature for testing purposes.

    Args:
        payload: Request body
        timestamp: Unix seconds string
        secret: Signing secret

    Returns:
        Signature in v1=<hex> format
    """
    signed_content = f"{timestamp}.{payload}"
    mac = hmac.new(
        secret.encode("utf-8"),
        signed_content.encode("utf-8"),
        hashlib.sha256,
    ).hexdigest()
    return f"v1={mac}"


# ----- Django Middleware -----


class WaaSWebhookMiddleware:
    """Django middleware that verifies WaaS webhook signatures on POST requests.

    Add to MIDDLEWARE in settings.py:
        MIDDLEWARE = [
            ...,
            'waas_django.verify.WaaSWebhookMiddleware',
        ]

    Configure in settings.py:
        WAAS_SIGNING_SECRET = 'whsec_...'
        # Optional: multiple secrets for key rotation
        WAAS_SIGNING_SECRETS = ['whsec_new', 'whsec_old']
        # Optional: only verify specific paths (default: all POST with signature header)
        WAAS_WEBHOOK_PATH = '/webhooks/waas'
        # Optional: timestamp tolerance in seconds (default: 300)
        WAAS_TIMESTAMP_TOLERANCE = 300
    """

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        if request.method == "POST" and self._should_verify(request):
            try:
                self._verify_request(request)
                request.waas_verified = True
            except WebhookVerificationError as e:
                from django.http import JsonResponse
                return JsonResponse({"error": str(e)}, status=401)

        return self.get_response(request)

    def _should_verify(self, request):
        """Only verify requests that have the signature header."""
        signature = request.META.get("HTTP_X_WAAS_SIGNATURE", "")
        if not signature:
            return False

        from django.conf import settings
        webhook_path = getattr(settings, "WAAS_WEBHOOK_PATH", None)
        if webhook_path and request.path != webhook_path:
            return False

        return True

    def _verify_request(self, request):
        from django.conf import settings

        secrets = getattr(settings, "WAAS_SIGNING_SECRETS", None)
        if not secrets:
            secret = getattr(settings, "WAAS_SIGNING_SECRET", "")
            secrets = [secret] if secret else []

        if not secrets:
            return

        signature = request.META.get("HTTP_X_WAAS_SIGNATURE", "")
        timestamp = request.META.get("HTTP_X_WAAS_TIMESTAMP", "")
        body = request.body.decode("utf-8") if isinstance(request.body, bytes) else request.body
        tolerance = getattr(settings, "WAAS_TIMESTAMP_TOLERANCE", DEFAULT_TOLERANCE)

        verify_with_multiple_secrets(body, signature, timestamp, secrets, tolerance)


# ----- Internal helpers -----


def _parse_signature(header: str) -> str:
    parts = header.split("=", 1)
    if len(parts) != 2 or parts[0] != "v1":
        raise WebhookVerificationError("Malformed signature header: expected v1=<hex>")
    return parts[1]


def _validate_timestamp(timestamp: str, tolerance_seconds: int) -> None:
    if not timestamp or tolerance_seconds <= 0:
        return

    try:
        ts = int(timestamp)
    except ValueError:
        raise WebhookVerificationError(f"Malformed timestamp: {timestamp}")

    diff = abs(time.time() - ts)
    if diff > tolerance_seconds:
        raise WebhookVerificationError(
            f"Timestamp expired: {int(diff)}s > {tolerance_seconds}s tolerance"
        )


# ----- Backward compatibility -----


def verify_webhook_signature(
    secret: str,
    signature: str,
    body: bytes,
    tolerance: int = DEFAULT_TOLERANCE,
) -> bool:
    """Backward-compatible wrapper for legacy callers using the combined header format.

    Supports both the legacy combined format (t=<ts>,v1=<sig>) and the new
    separate-header format. When called with the combined format, parses out the
    timestamp and signature components automatically.

    Args:
        secret: Signing secret
        signature: Combined signature header (t=<ts>,v1=<sig>) or v1=<hex>
        body: Raw request body bytes
        tolerance: Max age in seconds (default 300)

    Returns:
        True if valid
    """
    if isinstance(body, bytes):
        body = body.decode("utf-8")

    # Detect legacy combined format: t=<ts>,v1=<sig>
    if "," in signature:
        parts = {}
        for part in signature.split(","):
            if "=" in part:
                k, v = part.split("=", 1)
                parts[k] = v
        ts = parts.get("t", "")
        sig_header = f"v1={parts.get('v1', '')}"
    else:
        ts = ""
        sig_header = signature

    return verify_signature(body, sig_header, ts, secret, tolerance)
