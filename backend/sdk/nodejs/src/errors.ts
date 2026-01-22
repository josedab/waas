/**
 * WAAS SDK Error classes
 */

export class WAASError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'WAASError';
  }
}

export class WAASApiError extends WAASError {
  public readonly statusCode: number;
  public readonly code: string;
  public readonly details?: Record<string, unknown>;

  constructor(
    message: string,
    statusCode: number,
    code: string,
    details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'WAASApiError';
    this.statusCode = statusCode;
    this.code = code;
    this.details = details;
  }

  toString(): string {
    return `WAASApiError(${this.statusCode}): ${this.code} - ${this.message}`;
  }
}

export class WAASAuthenticationError extends WAASApiError {
  constructor(message = 'Authentication failed') {
    super(message, 401, 'AUTHENTICATION_FAILED');
    this.name = 'WAASAuthenticationError';
  }
}

export class WAASRateLimitError extends WAASApiError {
  public readonly retryAfter?: number;

  constructor(message = 'Rate limit exceeded', retryAfter?: number) {
    super(message, 429, 'RATE_LIMIT_EXCEEDED');
    this.name = 'WAASRateLimitError';
    this.retryAfter = retryAfter;
  }
}

export class WAASValidationError extends WAASApiError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 400, 'VALIDATION_ERROR', details);
    this.name = 'WAASValidationError';
  }
}

export class WAASNotFoundError extends WAASApiError {
  constructor(message = 'Resource not found') {
    super(message, 404, 'NOT_FOUND');
    this.name = 'WAASNotFoundError';
  }
}

export class WAASConnectionError extends WAASError {
  constructor(message = 'Connection failed') {
    super(message);
    this.name = 'WAASConnectionError';
  }
}

export class WAASTimeoutError extends WAASError {
  constructor(message = 'Request timeout') {
    super(message);
    this.name = 'WAASTimeoutError';
  }
}
