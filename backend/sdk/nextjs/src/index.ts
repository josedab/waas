/**
 * WaaS Next.js Integration
 *
 * Drop-in webhook handling for Next.js App Router and Pages Router.
 *
 * Quick setup:
 *   // app/api/webhooks/route.ts
 *   import { waasWebhookHandler } from '@waas/nextjs';
 *   export const POST = waasWebhookHandler({
 *     secret: process.env.WAAS_SIGNING_SECRET!,
 *     onEvent: async (event) => { console.log(event); },
 *   });
 */

import crypto from 'crypto';

export class WebhookVerificationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'WebhookVerificationError';
  }
}

export function verifySignature(secret: string, signature: string, body: string, toleranceMs = 300_000): boolean {
  const parts: Record<string, string> = {};
  for (const p of signature.split(',')) {
    const [k, v] = p.split('=', 2);
    if (k && v) parts[k] = v;
  }
  const ts = parts['t'];
  const sig = parts['v1'];
  if (!ts || !sig) throw new WebhookVerificationError('Invalid signature format');
  if (toleranceMs > 0 && Date.now() - parseInt(ts) * 1000 > toleranceMs) {
    throw new WebhookVerificationError('Timestamp expired');
  }
  const expected = crypto.createHmac('sha256', secret).update(`${ts}.${body}`).digest('hex');
  if (!crypto.timingSafeEqual(Buffer.from(sig), Buffer.from(expected))) {
    throw new WebhookVerificationError('Signature mismatch');
  }
  return true;
}

// ----- App Router Handler (Next.js 13+) -----

export interface WaaSHandlerOptions {
  secret: string;
  toleranceMs?: number;
  onEvent: (event: { type: string; payload: Record<string, unknown>; headers: Record<string, string> }) => Promise<void>;
  onError?: (error: Error) => Promise<Response | void>;
}

/**
 * Creates a Next.js App Router webhook handler.
 *
 * Usage in app/api/webhooks/route.ts:
 *   export const POST = waasWebhookHandler({ secret, onEvent });
 */
export function waasWebhookHandler(options: WaaSHandlerOptions) {
  return async (request: Request): Promise<Response> => {
    try {
      const body = await request.text();
      const signature = request.headers.get('x-webhook-signature') || '';

      verifySignature(options.secret, signature, body, options.toleranceMs);

      const event = JSON.parse(body);
      const headers: Record<string, string> = {};
      request.headers.forEach((v, k) => { headers[k] = v; });

      await options.onEvent({
        type: event.event_type || event.type || 'unknown',
        payload: event.payload || event,
        headers,
      });

      return new Response(JSON.stringify({ received: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    } catch (error) {
      if (options.onError) {
        const result = await options.onError(error as Error);
        if (result) return result;
      }
      if (error instanceof WebhookVerificationError) {
        return new Response(JSON.stringify({ error: error.message }), { status: 401 });
      }
      return new Response(JSON.stringify({ error: 'Internal error' }), { status: 500 });
    }
  };
}

// ----- API Client -----

export class WaaSClient {
  private apiKey: string;
  private apiUrl: string;

  constructor(apiKey: string, apiUrl?: string) {
    this.apiKey = apiKey;
    this.apiUrl = apiUrl || process.env.WAAS_API_URL || 'http://localhost:8080';
  }

  async sendWebhook(eventType: string, payload: Record<string, unknown>): Promise<any> {
    const res = await fetch(`${this.apiUrl}/api/v1/webhooks/send`, {
      method: 'POST',
      headers: { 'X-API-Key': this.apiKey, 'Content-Type': 'application/json' },
      body: JSON.stringify({ event_type: eventType, payload }),
    });
    if (!res.ok) throw new Error(`WaaS error: ${res.status}`);
    return res.json();
  }
}
