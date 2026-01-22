package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Delivery status enum.
 */
public enum DeliveryStatus {
    PENDING("pending"),
    PROCESSING("processing"),
    SUCCESS("success"),
    FAILED("failed"),
    RETRYING("retrying");

    private final String value;

    DeliveryStatus(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }

    public static DeliveryStatus fromString(String value) {
        for (DeliveryStatus status : values()) {
            if (status.value.equalsIgnoreCase(value)) {
                return status;
            }
        }
        throw new IllegalArgumentException("Unknown delivery status: " + value);
    }
}
