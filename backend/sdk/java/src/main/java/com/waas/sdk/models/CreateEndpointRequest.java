package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;
import java.util.UUID;

/**
 * Request to create a webhook endpoint.
 */
public class CreateEndpointRequest {
    private String url;
    
    @JsonProperty("custom_headers")
    private Map<String, String> customHeaders;
    
    @JsonProperty("retry_config")
    private RetryConfiguration retryConfig;

    private CreateEndpointRequest(Builder builder) {
        this.url = builder.url;
        this.customHeaders = builder.customHeaders;
        this.retryConfig = builder.retryConfig;
    }

    public static Builder builder() {
        return new Builder();
    }

    public String getUrl() { return url; }
    public Map<String, String> getCustomHeaders() { return customHeaders; }
    public RetryConfiguration getRetryConfig() { return retryConfig; }

    public static class Builder {
        private String url;
        private Map<String, String> customHeaders;
        private RetryConfiguration retryConfig;

        public Builder url(String url) {
            this.url = url;
            return this;
        }

        public Builder customHeaders(Map<String, String> customHeaders) {
            this.customHeaders = customHeaders;
            return this;
        }

        public Builder retryConfig(RetryConfiguration retryConfig) {
            this.retryConfig = retryConfig;
            return this;
        }

        public CreateEndpointRequest build() {
            if (url == null || url.isEmpty()) {
                throw new IllegalArgumentException("URL is required");
            }
            return new CreateEndpointRequest(this);
        }
    }
}
