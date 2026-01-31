/**
 * WaaS Webhook Signature Verification - Standalone, zero-dependency module.
 *
 * Usage:
 *   import { WebhookVerifier } from '@waas/sdk/verify';
 *   const verifier = new WebhookVerifier('whsec_your_secret');
 *   verifier.verify(payload, signatureHeader, timestampHeader);
 *
 * Express middleware:
 *   import { expressMiddleware } from '@waas/sdk/verify';
 *   app.use('/webhooks', expressMiddleware('whsec_...'));
 */

import { createHmac, timingSafeEqual } from 'crypto';

export const SIGNATURE_HEADER = 'x-waas-signature';
export const TIMESTAMP_HEADER = 'x-waas-timestamp';
const DEFAULT_TOLERANCE = 300; // 5 minutes

export class VerificationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'VerificationError';
  }
}

export class MissingSignatureError extends VerificationError {
  constructor() {
    super('Missing signature header');
    this.name = 'MissingSignatureError';
  }
}

export class InvalidSignatureError extends VerificationError {
  constructor() {
    super('Invalid signature');
    this.name = 'InvalidSignatureError';
  }
}

export class TimestampExpiredError extends VerificationError {
  constructor(diff: number, tolerance: number) {
    super(`Timestamp expired: ${Math.round(diff)}s > ${tolerance}s tolerance`);
    this.name = 'TimestampExpiredError';
  }
}

export interface VerifierOptions {
  /** Additional secrets for key rotation */
  secrets?: string[];
  /** Maximum allowed timestamp age in seconds (default: 300) */
  timestampTolerance?: number;
}

export class WebhookVerifier {
  private secrets: string[];
  private timestampTolerance: number;

  constructor(secret: string, options: VerifierOptions = {}) {
    this.secrets = [secret, ...(options.secrets || [])];
    this.timestampTolerance = options.timestampTolerance ?? DEFAULT_TOLERANCE;
  }

  /**
   * Verify a webhook signature.
   * @param payload - Raw request body (string or Buffer).
   * @param signature - Value of X-WaaS-Signature header.
   * @param timestamp - Value of X-WaaS-Timestamp header.
   * @throws {VerificationError} If verification fails.
   */
  verify(payload: string | Buffer, signature: string, timestamp: string = ''): boolean {
    if (!signature) {
      throw new MissingSignatureError();
    }

    this.verifyTimestamp(timestamp);

    const sigBytes = this.parseSignature(signature);
    const payloadStr = typeof payload === 'string' ? payload : payload.toString('utf-8');
    const signedPayload = `${timestamp}.${payloadStr}`;

    for (const secret of this.secrets) {
      const expected = createHmac('sha256', secret).update(signedPayload).digest();
      if (sigBytes.length === expected.length && timingSafeEqual(sigBytes, expected)) {
        return true;
      }
    }

    throw new InvalidSignatureError();
  }

  /** Generate a signature for testing purposes. */
  sign(payload: string | Buffer, timestamp: string): string {
    const payloadStr = typeof payload === 'string' ? payload : payload.toString('utf-8');
    const signedPayload = `${timestamp}.${payloadStr}`;
    const mac = createHmac('sha256', this.secrets[0]).update(signedPayload).digest('hex');
    return `v1=${mac}`;
  }

  private verifyTimestamp(timestamp: string): void {
    if (!timestamp || this.timestampTolerance <= 0) return;

    const ts = parseInt(timestamp, 10);
    if (isNaN(ts)) {
      throw new VerificationError('Malformed timestamp');
    }

    const diff = Math.abs(Date.now() / 1000 - ts);
    if (diff > this.timestampTolerance) {
      throw new TimestampExpiredError(diff, this.timestampTolerance);
    }
  }

  private parseSignature(header: string): Buffer {
    const parts = header.split('=');
    if (parts.length < 2) {
      throw new VerificationError('Malformed signature header');
    }
    return Buffer.from(parts.slice(1).join('='), 'hex');
  }
}

/**
 * Express middleware for WaaS webhook signature verification.
 *
 * Usage:
 *   app.use('/webhooks', expressMiddleware('whsec_...'));
 */
export function expressMiddleware(secret: string, options?: VerifierOptions) {
  const verifier = new WebhookVerifier(secret, options);

  return (req: any, res: any, next: any) => {
    // Collect raw body
    const chunks: Buffer[] = [];
    req.on('data', (chunk: Buffer) => chunks.push(chunk));
    req.on('end', () => {
      const body = Buffer.concat(chunks);
      const sig = (req.headers[SIGNATURE_HEADER] || '') as string;
      const ts = (req.headers[TIMESTAMP_HEADER] || '') as string;

      try {
        verifier.verify(body, sig, ts);
        // Re-attach body for downstream handlers
        req.body = JSON.parse(body.toString());
        next();
      } catch (err) {
        res.status(401).json({ error: (err as Error).message });
      }
    });
  };
}

/**
 * Convenience function to verify a request's headers and body.
 */
export function verifyRequest(
  secret: string,
  payload: string | Buffer,
  headers: Record<string, string>
): boolean {
  const v = new WebhookVerifier(secret);
  return v.verify(
    payload,
    headers[SIGNATURE_HEADER] || headers['X-WaaS-Signature'] || '',
    headers[TIMESTAMP_HEADER] || headers['X-WaaS-Timestamp'] || ''
  );
}
