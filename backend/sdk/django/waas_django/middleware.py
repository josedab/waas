"""Django middleware for automatic webhook signature verification"""

import json
import logging

from django.conf import settings
from django.http import JsonResponse

from waas_django.verify import verify_webhook_signature, WebhookVerificationError

logger = logging.getLogger('waas')


class WaaSWebhookMiddleware:
    """Middleware that verifies WaaS webhook signatures on incoming requests.

    Add to settings.py MIDDLEWARE list and configure WAAS_WEBHOOK_PATH.
    """

    def __init__(self, get_response):
        self.get_response = get_response
        self.webhook_path = getattr(settings, 'WAAS_WEBHOOK_PATH', '/webhooks/waas/')
        self.signing_secret = getattr(settings, 'WAAS_SIGNING_SECRET', '')

    def __call__(self, request):
        if request.path == self.webhook_path and request.method == 'POST':
            if self.signing_secret:
                signature = request.META.get('HTTP_X_WEBHOOK_SIGNATURE', '')
                try:
                    verify_webhook_signature(
                        self.signing_secret,
                        signature,
                        request.body,
                    )
                    request.waas_verified = True
                except WebhookVerificationError as e:
                    logger.warning('WaaS webhook verification failed: %s', e)
                    return JsonResponse({'error': str(e)}, status=401)
            else:
                request.waas_verified = False

            try:
                request.waas_payload = json.loads(request.body)
            except (json.JSONDecodeError, ValueError):
                request.waas_payload = None

        return self.get_response(request)
