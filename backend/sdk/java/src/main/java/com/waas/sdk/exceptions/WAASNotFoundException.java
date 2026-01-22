package com.waas.sdk.exceptions;

/**
 * Exception for resource not found errors.
 */
public class WAASNotFoundException extends WAASApiException {
    public WAASNotFoundException() {
        this("Resource not found");
    }

    public WAASNotFoundException(String message) {
        super(message, 404, "NOT_FOUND");
    }
}
