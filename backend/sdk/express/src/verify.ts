/**
 * Standalone, zero-dependency WaaS webhook signature verification for Express.
 *
 * Algorithm:
 *   1. Parse X-WaaS-Signature header (format: v1=<hex_encoded_hmac>)
 *   2. Parse X-WaaS-Timestamp header (Unix seconds string)
 *   3. Validate timestamp is within tolerance (default 300s)
 *   4. Construct signed content: timestamp + "." + raw_payload_body
 *   5. Compute HMAC-SHA256 with secret key
 *   6. Compare using crypto.timingSafeEqual()
 *
 * Usage:
 *   import { verifySignature, verifyWebhookMiddleware } from '@waas/express/verify';
 *
 *   // Standalone:
 *   verifySignature(body, sigHeader, tsHeader, secret);
 *
 *   // Express middleware:
 *   app.post('/webhooks', verifyWebhookMiddleware({ secret: 'whsec_...' }), handler);
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
 * @param payload          Raw request body string or Buffer
 * @param signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
 * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
 * @param secret           Signing secret
 * @param toleranceSeconds Max allowed timestamp age (default 300, 0 to disable)
 * @returns true if valid
 * @throws WebhookVerificationError
 */
export function verifySignature(
  payload: string | Buffer,
  signatureHeader: string,
  timestampHeader: string,
  secret: string,
  toleranceSeconds: number = DEFAULT_TOLERANCE_SECONDS,
): boolean {
  if (!secret) throw new WebhookVerificationError('No signing secret configured');
  if (!signatureHeader) throw new WebhookVerificationError('Missing signature header');

  const sig = parseSignature(signatureHeader);
  validateTimestamp(timestampHeader, toleranceSeconds);

  const body = typeof payload === 'string' ? payload : payload.toString('utf-8');
  const signedContent = `${timestampHeader}.${body}`;
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
 * @param payload          Raw request body string or Buffer
 * @param signatureHeader  Value of X-WaaS-Signature header (v1=<hex>)
 * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
 * @param secrets          Array of signing secrets to try
 * @param toleranceSeconds Max allowed timestamp age (default 300, 0 to disable)
 * @returns true if valid against any secret
 * @throws WebhookVerificationError
 */
export function verifyWithMultipleSecrets(
  payload: string | Buffer,
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

  const body = typeof payload === 'string' ? payload : payload.toString('utf-8');
  const signedContent = `${timestampHeader}.${body}`;

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

// ----- Express Middleware -----

export interface VerifyWebhookMiddlewareOptions {
  /** Single secret or array of secrets for key rotation. */
  secret: string | string[];
  /** Max allowed timestamp age in seconds (default 300). */
  toleranceSeconds?: number;
  /** Custom error handler. If not provided, responds with 401 JSON. */
  onError?: (err: WebhookVerificationError, req: any, res: any) => void;
}

/**
 * Express middleware that verifies WaaS webhook signatures.
 *
 * Requires raw body access — use express.raw() or express.text() on the route.
 */
export function verifyWebhookMiddleware(options: VerifyWebhookMiddlewareOptions) {
  const { toleranceSeconds = DEFAULT_TOLERANCE_SECONDS } = options;
  const secrets = Array.isArray(options.secret) ? options.secret : [options.secret];

  return (req: any, res: any, next: any) => {
    if (req.method !== 'POST') return next();

    const signatureHeader: string = req.headers[SIGNATURE_HEADER] || '';
    const timestampHeader: string = req.headers[TIMESTAMP_HEADER] || '';
    const body: string =
      typeof req.body === 'string'
        ? req.body
        : Buffer.isBuffer(req.body)
          ? req.body.toString('utf-8')
          : JSON.stringify(req.body);

    try {
      verifyWithMultipleSecrets(body, signatureHeader, timestampHeader, secrets, toleranceSeconds);
      req.waasVerified = true;
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
