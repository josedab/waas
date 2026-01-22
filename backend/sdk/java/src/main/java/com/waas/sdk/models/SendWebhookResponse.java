package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Response from sending a webhook.
 */
public class SendWebhookResponse {
    @JsonProperty("delivery_id")
    private UUID deliveryId;
    
    @JsonProperty("endpoint_id")
    private UUID endpointId;
    
    private String status;
    
    @JsonProperty("scheduled_at")
    private OffsetDateTime scheduledAt;

    public UUID getDeliveryId() { return deliveryId; }
    public void setDeliveryId(UUID deliveryId) { this.deliveryId = deliveryId; }
    
    public UUID getEndpointId() { return endpointId; }
    public void setEndpointId(UUID endpointId) { this.endpointId = endpointId; }
    
    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }
    
    public OffsetDateTime getScheduledAt() { return scheduledAt; }
    public void setScheduledAt(OffsetDateTime scheduledAt) { this.scheduledAt = scheduledAt; }
}
