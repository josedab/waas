package com.waas.sdk.exceptions;

/**
 * Exception for authentication failures.
 */
public class WAASAuthenticationException extends WAASApiException {
    public WAASAuthenticationException() {
        this("Authentication failed");
    }

    public WAASAuthenticationException(String message) {
        super(message, 401, "AUTHENTICATION_FAILED");
    }
}
