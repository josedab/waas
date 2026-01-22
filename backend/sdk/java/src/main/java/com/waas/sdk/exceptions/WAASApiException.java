package com.waas.sdk.exceptions;

import java.util.Map;

/**
 * Exception for API errors from the WAAS service.
 */
public class WAASApiException extends WAASException {
    private final int statusCode;
    private final String code;
    private final Map<String, Object> details;

    public WAASApiException(String message, int statusCode, String code) {
        this(message, statusCode, code, null);
    }

    public WAASApiException(String message, int statusCode, String code, Map<String, Object> details) {
        super(message);
        this.statusCode = statusCode;
        this.code = code;
        this.details = details;
    }

    public int getStatusCode() { return statusCode; }
    public String getCode() { return code; }
    public Map<String, Object> getDetails() { return details; }

    @Override
    public String toString() {
        return String.format("WAASApiException(%d): %s - %s", statusCode, code, getMessage());
    }
}
