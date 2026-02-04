"""Django view decorators for WaaS webhooks"""

import functools
import json

from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from django.views.decorators.http import require_POST
from django.conf import settings

from waas_django.verify import verify_webhook_signature, WebhookVerificationError


def verify_waas_signature(view_func):
    """Decorator that verifies WaaS webhook signature before calling the view."""

    @functools.wraps(view_func)
    def wrapper(request, *args, **kwargs):
        secret = getattr(settings, 'WAAS_SIGNING_SECRET', '')
        if secret:
            signature = request.META.get('HTTP_X_WEBHOOK_SIGNATURE', '')
            try:
                verify_webhook_signature(secret, signature, request.body)
            except WebhookVerificationError as e:
                return JsonResponse({'error': str(e)}, status=401)

        return view_func(request, *args, **kwargs)

    return wrapper


def waas_webhook(view_func):
    """Combined decorator: CSRF exempt + POST only + signature verification.

    Usage:
        @waas_webhook
        def handle_webhook(request):
            payload = json.loads(request.body)
            ...
    """

    @csrf_exempt
    @require_POST
    @verify_waas_signature
    @functools.wraps(view_func)
    def wrapper(request, *args, **kwargs):
        return view_func(request, *args, **kwargs)

    return wrapper
