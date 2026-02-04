/**
 * WaaS Express.js Integration
 *
 * Drop-in webhook sending and receiving for Express (4.x / 5.x).
 *
 * Quick setup (3 lines):
 *   import { waasMiddleware, WaaSClient } from '@waas/express';
 *   app.use('/webhooks', waasMiddleware({ secret: process.env.WAAS_SIGNING_SECRET }));
 *   const waas = new WaaSClient({ apiKey: process.env.WAAS_API_KEY });
 */

import { Request, Response, NextFunction, RequestHandler } from 'express';
import crypto from 'crypto';

// ----- Webhook Verification -----

export class WebhookVerificationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'WebhookVerificationError';
  }
}

export interface VerifyOptions {
  secret: string;
  toleranceMs?: number;
  signatureHeader?: string;
}

export function verifyWebhookSignature(
  secret: string,
  signature: string,
  body: string | Buffer,
  toleranceMs: number = 300_000,
): boolean {
  if (!secret) throw new WebhookVerificationError('No signing secret configured');
  if (!signature) throw new WebhookVerificationError('Missing signature header');

  const parts: Record<string, string> = {};
  for (const p of signature.split(',')) {
    const [k, v] = p.split('=', 2);
    if (k && v) parts[k] = v;
  }

  const ts = parts['t'];
  const sig = parts['v1'];
  if (!ts || !sig) throw new WebhookVerificationError('Invalid signature format');

  if (toleranceMs > 0) {
    const age = Date.now() - parseInt(ts, 10) * 1000;
    if (age > toleranceMs) throw new WebhookVerificationError('Timestamp expired');
  }

  const payload = `${ts}.${typeof body === 'string' ? body : body.toString('utf-8')}`;
  const expected = crypto.createHmac('sha256', secret).update(payload).digest('hex');

  if (!crypto.timingSafeEqual(Buffer.from(sig, 'utf-8'), Buffer.from(expected, 'utf-8'))) {
    throw new WebhookVerificationError('Signature mismatch');
  }

  return true;
}

// ----- Express Middleware -----

export interface WaaSMiddlewareOptions {
  secret: string;
  toleranceMs?: number;
  signatureHeader?: string;
  onError?: (err: WebhookVerificationError, req: Request, res: Response) => void;
}

/**
 * Express middleware that verifies WaaS webhook signatures.
 * Attaches `req.waasVerified` and `req.waasPayload` on success.
 */
export function waasMiddleware(options: WaaSMiddlewareOptions): RequestHandler {
  const { secret, toleranceMs = 300_000, signatureHeader = 'x-webhook-signature' } = options;

  return (req: Request, res: Response, next: NextFunction) => {
    if (req.method !== 'POST') return next();

    const signature = req.headers[signatureHeader] as string;
    const body = typeof req.body === 'string' ? req.body : JSON.stringify(req.body);

    try {
      verifyWebhookSignature(secret, signature, body, toleranceMs);
      (req as any).waasVerified = true;
      (req as any).waasPayload = typeof req.body === 'string' ? JSON.parse(req.body) : req.body;
      next();
    } catch (err) {
      if (options.onError && err instanceof WebhookVerificationError) {
        options.onError(err, req, res);
      } else {
        res.status(401).json({ error: 'Invalid webhook signature' });
      }
    }
  };
}

// ----- WaaS API Client -----

export interface WaaSClientOptions {
  apiKey: string;
  apiUrl?: string;
  timeout?: number;
}

export class WaaSClient {
  private apiKey: string;
  private apiUrl: string;
  private timeout: number;

  constructor(options: WaaSClientOptions) {
    this.apiKey = options.apiKey;
    this.apiUrl = options.apiUrl || process.env.WAAS_API_URL || 'http://localhost:8080';
    this.timeout = options.timeout || 30_000;
  }

  async sendWebhook(eventType: string, payload: Record<string, unknown>, endpointIds?: string[]): Promise<any> {
    const body: Record<string, unknown> = { event_type: eventType, payload };
    if (endpointIds) body.endpoint_ids = endpointIds;
    return this.request('POST', '/api/v1/webhooks/send', body);
  }

  async createEndpoint(url: string, eventTypes?: string[]): Promise<any> {
    const body: Record<string, unknown> = { url };
    if (eventTypes) body.event_types = eventTypes;
    return this.request('POST', '/api/v1/endpoints', body);
  }

  async listDeliveries(limit = 20, offset = 0): Promise<any> {
    return this.request('GET', `/api/v1/webhooks/deliveries?limit=${limit}&offset=${offset}`);
  }

  async health(): Promise<any> {
    return this.request('GET', '/health');
  }

  private async request(method: string, path: string, body?: Record<string, unknown>): Promise<any> {
    const url = `${this.apiUrl}${path}`;
    const options: RequestInit = {
      method,
      headers: {
        'X-API-Key': this.apiKey,
        'Content-Type': 'application/json',
      },
      signal: AbortSignal.timeout(this.timeout),
    };
    if (body) options.body = JSON.stringify(body);

    const response = await fetch(url, options);
    if (!response.ok) {
      throw new Error(`WaaS API error (${response.status}): ${await response.text()}`);
    }
    return response.json();
  }
}
