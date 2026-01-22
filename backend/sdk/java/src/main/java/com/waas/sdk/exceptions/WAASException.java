package com.waas.sdk.exceptions;

/**
 * Base exception for WAAS SDK errors.
 */
public class WAASException extends RuntimeException {
    public WAASException(String message) {
        super(message);
    }

    public WAASException(String message, Throwable cause) {
        super(message, cause);
    }
}
