/**
 * Standalone, zero-dependency WaaS webhook signature verification for Next.js.
 *
 * Algorithm:
 *   1. Parse X-WaaS-Signature header (format: v1=<hex_encoded_hmac>)
 *   2. Parse X-WaaS-Timestamp header (Unix seconds string)
 *   3. Validate timestamp is within tolerance (default 300s)
 *   4. Construct signed content: timestamp + "." + raw_payload_body
 *   5. Compute HMAC-SHA256 with secret key
 *   6. Compare using crypto.timingSafeEqual()
 *
 * App Router (Next.js 13+):
 *   import { verifySignature, withWebhookVerification } from '@waas/nextjs/verify';
 *   export const POST = withWebhookVerification(handler, { secret: process.env.WAAS_SIGNING_SECRET! });
 *
 * Pages Router:
 *   import { verifySignatureForPages } from '@waas/nextjs/verify';
 *   export default verifySignatureForPages(handler, { secret: process.env.WAAS_SIGNING_SECRET! });
 */

import crypto from 'crypto';

export const SIGNATURE_HEADER = 'x-waas-signature';
export const TIMESTAMP_HEADER = 'x-waas-timestamp';
const DEFAULT_TOLERANCE_SECONDS = 300;

export class WebhookVerificationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'WebhookVerificationError';
  }
}

/**
 * Verify a WaaS webhook signature.
 *
 * @param payload          Raw request body string
 * @param signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
 * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
 * @param secret           Signing secret
 * @param toleranceSeconds Max allowed timestamp age (default 300, 0 to disable)
 * @returns true if valid
 * @throws WebhookVerificationError
 */
export function verifySignature(
  payload: string,
  signatureHeader: string,
  timestampHeader: string,
  secret: string,
  toleranceSeconds: number = DEFAULT_TOLERANCE_SECONDS,
): boolean {
  if (!secret) throw new WebhookVerificationError('No signing secret configured');
  if (!signatureHeader) throw new WebhookVerificationError('Missing signature header');

  const sig = parseSignature(signatureHeader);
  validateTimestamp(timestampHeader, toleranceSeconds);

  const signedContent = `${timestampHeader}.${payload}`;
  const expected = crypto.createHmac('sha256', secret).update(signedContent).digest('hex');

  if (!timingSafeEqual(sig, expected)) {
    throw new WebhookVerificationError('Signature does not match');
  }

  return true;
}

/**
 * Verify a webhook signature against multiple secrets (for key rotation).
 *
 * Tries each secret in order and returns true on the first match.
 *
 * @param payload          Raw request body string
 * @param signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
 * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
 * @param secrets          Array of signing secrets to try
 * @param toleranceSeconds Max allowed timestamp age (default 300, 0 to disable)
 * @returns true if valid against any secret
 * @throws WebhookVerificationError
 */
export function verifyWithMultipleSecrets(
  payload: string,
  signatureHeader: string,
  timestampHeader: string,
  secrets: string[],
  toleranceSeconds: number = DEFAULT_TOLERANCE_SECONDS,
): boolean {
  if (!secrets || secrets.length === 0) {
    throw new WebhookVerificationError('No signing secrets configured');
  }
  if (!signatureHeader) throw new WebhookVerificationError('Missing signature header');

  const sig = parseSignature(signatureHeader);
  validateTimestamp(timestampHeader, toleranceSeconds);

  const signedContent = `${timestampHeader}.${payload}`;

  for (const secret of secrets) {
    const expected = crypto.createHmac('sha256', secret).update(signedContent).digest('hex');
    if (timingSafeEqual(sig, expected)) {
      return true;
    }
  }

  throw new WebhookVerificationError('Signature does not match');
}

/**
 * Generate a signature for testing purposes.
 */
export function sign(payload: string, timestamp: string, secret: string): string {
  const signedContent = `${timestamp}.${payload}`;
  const mac = crypto.createHmac('sha256', secret).update(signedContent).digest('hex');
  return `v1=${mac}`;
}

// ----- Next.js App Router Helper (13+) -----

export interface WebhookHandlerOptions {
  /** Single secret or array of secrets for key rotation. */
  secret: string | string[];
  /** Max allowed timestamp age in seconds (default 300). */
  toleranceSeconds?: number;
}

/**
 * Wraps a Next.js App Router handler with webhook signature verification.
 *
 * Usage in app/api/webhooks/route.ts:
 *   export const POST = withWebhookVerification(
 *     async (request, body) => {
 *       const event = JSON.parse(body);
 *       return new Response(JSON.stringify({ received: true }), { status: 200 });
 *     },
 *     { secret: process.env.WAAS_SIGNING_SECRET! },
 *   );
 */
export function withWebhookVerification(
  handler: (request: Request, body: string) => Promise<Response>,
  options: WebhookHandlerOptions,
) {
  const secrets = Array.isArray(options.secret) ? options.secret : [options.secret];
  const toleranceSeconds = options.toleranceSeconds ?? DEFAULT_TOLERANCE_SECONDS;

  return async (request: Request): Promise<Response> => {
    try {
      const body = await request.text();
      const signatureHeader = request.headers.get(SIGNATURE_HEADER) || '';
      const timestampHeader = request.headers.get(TIMESTAMP_HEADER) || '';

      verifyWithMultipleSecrets(body, signatureHeader, timestampHeader, secrets, toleranceSeconds);

      return handler(request, body);
    } catch (error) {
      if (error instanceof WebhookVerificationError) {
        return new Response(JSON.stringify({ error: error.message }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      return new Response(JSON.stringify({ error: 'Internal error' }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      });
    }
  };
}

// ----- Next.js Pages Router Helper -----

/**
 * Wraps a Next.js Pages Router API handler with webhook signature verification.
 *
 * Requires body parsing disabled for the route:
 *   export const config = { api: { bodyParser: false } };
 *
 * Usage in pages/api/webhooks.ts:
 *   export default verifySignatureForPages(handler, { secret: process.env.WAAS_SIGNING_SECRET! });
 */
export function verifySignatureForPages(
  handler: (req: any, res: any) => Promise<void> | void,
  options: WebhookHandlerOptions,
) {
  const secrets = Array.isArray(options.secret) ? options.secret : [options.secret];
  const toleranceSeconds = options.toleranceSeconds ?? DEFAULT_TOLERANCE_SECONDS;

  return async (req: any, res: any) => {
    if (req.method !== 'POST') return handler(req, res);

    try {
      const body = await readBody(req);
      const signatureHeader: string = (req.headers[SIGNATURE_HEADER] as string) || '';
      const timestampHeader: string = (req.headers[TIMESTAMP_HEADER] as string) || '';

      verifyWithMultipleSecrets(body, signatureHeader, timestampHeader, secrets, toleranceSeconds);

      req.waasVerified = true;
      req.waasRawBody = body;
      return handler(req, res);
    } catch (error) {
      if (error instanceof WebhookVerificationError) {
        return res.status(401).json({ error: (error as Error).message });
      }
      return res.status(500).json({ error: 'Internal error' });
    }
  };
}

// ----- Internal helpers -----

function parseSignature(header: string): string {
  const parts = header.split('=', 2);
  if (parts.length !== 2 || parts[0] !== 'v1') {
    throw new WebhookVerificationError('Malformed signature header: expected v1=<hex>');
  }
  return parts[1];
}

function validateTimestamp(timestamp: string, toleranceSeconds: number): void {
  if (!timestamp || toleranceSeconds <= 0) return;
  const ts = parseInt(timestamp, 10);
  if (isNaN(ts)) {
    throw new WebhookVerificationError(`Malformed timestamp: ${timestamp}`);
  }
  const diff = Math.abs(Math.floor(Date.now() / 1000) - ts);
  if (diff > toleranceSeconds) {
    throw new WebhookVerificationError(
      `Timestamp expired: ${diff}s > ${toleranceSeconds}s tolerance`,
    );
  }
}

function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  return crypto.timingSafeEqual(Buffer.from(a, 'utf-8'), Buffer.from(b, 'utf-8'));
}

function readBody(req: any): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on('data', (chunk: Buffer) => chunks.push(chunk));
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf-8')));
    req.on('error', reject);
  });
}
