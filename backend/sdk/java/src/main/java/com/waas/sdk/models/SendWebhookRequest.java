package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.Map;
import java.util.UUID;

/**
 * Request to send a webhook.
 */
public class SendWebhookRequest {
    @JsonProperty("endpoint_id")
    private UUID endpointId;
    
    private Object payload;
    
    private Map<String, String> headers;

    private SendWebhookRequest(Builder builder) {
        this.endpointId = builder.endpointId;
        this.payload = builder.payload;
        this.headers = builder.headers;
    }

    public static Builder builder() {
        return new Builder();
    }

    public UUID getEndpointId() { return endpointId; }
    public Object getPayload() { return payload; }
    public Map<String, String> getHeaders() { return headers; }

    public static class Builder {
        private UUID endpointId;
        private Object payload;
        private Map<String, String> headers;

        public Builder endpointId(UUID endpointId) {
            this.endpointId = endpointId;
            return this;
        }

        public Builder payload(Object payload) {
            this.payload = payload;
            return this;
        }

        public Builder headers(Map<String, String> headers) {
            this.headers = headers;
            return this;
        }

        public SendWebhookRequest build() {
            if (payload == null) {
                throw new IllegalArgumentException("Payload is required");
            }
            return new SendWebhookRequest(this);
        }
    }
}
