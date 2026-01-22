package com.waas.sdk.exceptions;

/**
 * Exception for rate limit errors.
 */
public class WAASRateLimitException extends WAASApiException {
    private final Integer retryAfter;

    public WAASRateLimitException() {
        this("Rate limit exceeded", null);
    }

    public WAASRateLimitException(String message, Integer retryAfter) {
        super(message, 429, "RATE_LIMIT_EXCEEDED");
        this.retryAfter = retryAfter;
    }

    public Integer getRetryAfter() { return retryAfter; }
}
