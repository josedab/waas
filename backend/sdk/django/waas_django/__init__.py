"""
WaaS Django Integration

Drop-in webhook sending and receiving for Django (4.2+ / 5.x).

Quick setup (3 lines):
    # settings.py
    INSTALLED_APPS = [..., 'waas_django']
    WAAS_API_KEY = os.environ['WAAS_API_KEY']
    WAAS_SIGNING_SECRET = os.environ.get('WAAS_SIGNING_SECRET', '')
"""

from waas_django.client import WaaSClient
from waas_django.verify import verify_webhook_signature, WebhookVerificationError
from waas_django.middleware import WaaSWebhookMiddleware
from waas_django.decorators import waas_webhook, verify_waas_signature

__all__ = [
    'WaaSClient',
    'verify_webhook_signature',
    'WebhookVerificationError',
    'WaaSWebhookMiddleware',
    'waas_webhook',
    'verify_waas_signature',
]

__version__ = '1.0.0'
