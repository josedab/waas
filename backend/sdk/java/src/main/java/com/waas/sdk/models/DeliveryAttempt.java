package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.OffsetDateTime;
import java.util.UUID;

/**
 * Delivery attempt model.
 */
public class DeliveryAttempt {
    private UUID id;
    
    @JsonProperty("endpoint_id")
    private UUID endpointId;
    
    @JsonProperty("payload_hash")
    private String payloadHash;
    
    @JsonProperty("payload_size")
    private int payloadSize;
    
    private DeliveryStatus status;
    
    @JsonProperty("http_status")
    private Integer httpStatus;
    
    @JsonProperty("response_body")
    private String responseBody;
    
    @JsonProperty("error_message")
    private String errorMessage;
    
    @JsonProperty("attempt_number")
    private int attemptNumber;
    
    @JsonProperty("scheduled_at")
    private OffsetDateTime scheduledAt;
    
    @JsonProperty("delivered_at")
    private OffsetDateTime deliveredAt;
    
    @JsonProperty("created_at")
    private OffsetDateTime createdAt;

    public UUID getId() { return id; }
    public void setId(UUID id) { this.id = id; }
    
    public UUID getEndpointId() { return endpointId; }
    public void setEndpointId(UUID endpointId) { this.endpointId = endpointId; }
    
    public String getPayloadHash() { return payloadHash; }
    public void setPayloadHash(String payloadHash) { this.payloadHash = payloadHash; }
    
    public int getPayloadSize() { return payloadSize; }
    public void setPayloadSize(int payloadSize) { this.payloadSize = payloadSize; }
    
    public DeliveryStatus getStatus() { return status; }
    public void setStatus(DeliveryStatus status) { this.status = status; }
    
    public Integer getHttpStatus() { return httpStatus; }
    public void setHttpStatus(Integer httpStatus) { this.httpStatus = httpStatus; }
    
    public String getResponseBody() { return responseBody; }
    public void setResponseBody(String responseBody) { this.responseBody = responseBody; }
    
    public String getErrorMessage() { return errorMessage; }
    public void setErrorMessage(String errorMessage) { this.errorMessage = errorMessage; }
    
    public int getAttemptNumber() { return attemptNumber; }
    public void setAttemptNumber(int attemptNumber) { this.attemptNumber = attemptNumber; }
    
    public OffsetDateTime getScheduledAt() { return scheduledAt; }
    public void setScheduledAt(OffsetDateTime scheduledAt) { this.scheduledAt = scheduledAt; }
    
    public OffsetDateTime getDeliveredAt() { return deliveredAt; }
    public void setDeliveredAt(OffsetDateTime deliveredAt) { this.deliveredAt = deliveredAt; }
    
    public OffsetDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(OffsetDateTime createdAt) { this.createdAt = createdAt; }
}
