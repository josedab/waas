package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Retry configuration for webhook endpoints.
 */
public class RetryConfiguration {
    @JsonProperty("max_attempts")
    private int maxAttempts = 5;
    
    @JsonProperty("initial_delay_ms")
    private int initialDelayMs = 1000;
    
    @JsonProperty("max_delay_ms")
    private int maxDelayMs = 300000;
    
    @JsonProperty("backoff_multiplier")
    private int backoffMultiplier = 2;

    public RetryConfiguration() {}

    private RetryConfiguration(Builder builder) {
        this.maxAttempts = builder.maxAttempts;
        this.initialDelayMs = builder.initialDelayMs;
        this.maxDelayMs = builder.maxDelayMs;
        this.backoffMultiplier = builder.backoffMultiplier;
    }

    public static Builder builder() {
        return new Builder();
    }

    public int getMaxAttempts() { return maxAttempts; }
    public void setMaxAttempts(int maxAttempts) { this.maxAttempts = maxAttempts; }
    
    public int getInitialDelayMs() { return initialDelayMs; }
    public void setInitialDelayMs(int initialDelayMs) { this.initialDelayMs = initialDelayMs; }
    
    public int getMaxDelayMs() { return maxDelayMs; }
    public void setMaxDelayMs(int maxDelayMs) { this.maxDelayMs = maxDelayMs; }
    
    public int getBackoffMultiplier() { return backoffMultiplier; }
    public void setBackoffMultiplier(int backoffMultiplier) { this.backoffMultiplier = backoffMultiplier; }

    public static class Builder {
        private int maxAttempts = 5;
        private int initialDelayMs = 1000;
        private int maxDelayMs = 300000;
        private int backoffMultiplier = 2;

        public Builder maxAttempts(int maxAttempts) {
            this.maxAttempts = maxAttempts;
            return this;
        }

        public Builder initialDelayMs(int initialDelayMs) {
            this.initialDelayMs = initialDelayMs;
            return this;
        }

        public Builder maxDelayMs(int maxDelayMs) {
            this.maxDelayMs = maxDelayMs;
            return this;
        }

        public Builder backoffMultiplier(int backoffMultiplier) {
            this.backoffMultiplier = backoffMultiplier;
            return this;
        }

        public RetryConfiguration build() {
            return new RetryConfiguration(this);
        }
    }
}
