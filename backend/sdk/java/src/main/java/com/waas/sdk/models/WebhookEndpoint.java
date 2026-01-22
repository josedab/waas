package com.waas.sdk.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.OffsetDateTime;
import java.util.Map;
import java.util.UUID;

/**
 * Webhook endpoint model.
 */
public class WebhookEndpoint {
    private UUID id;
    
    @JsonProperty("tenant_id")
    private UUID tenantId;
    
    private String url;
    
    @JsonProperty("is_active")
    private boolean isActive;
    
    @JsonProperty("retry_config")
    private RetryConfiguration retryConfig;
    
    @JsonProperty("custom_headers")
    private Map<String, String> customHeaders;
    
    @JsonProperty("created_at")
    private OffsetDateTime createdAt;
    
    @JsonProperty("updated_at")
    private OffsetDateTime updatedAt;

    // Secret is only returned on creation
    private String secret;

    public UUID getId() { return id; }
    public void setId(UUID id) { this.id = id; }
    
    public UUID getTenantId() { return tenantId; }
    public void setTenantId(UUID tenantId) { this.tenantId = tenantId; }
    
    public String getUrl() { return url; }
    public void setUrl(String url) { this.url = url; }
    
    public boolean isActive() { return isActive; }
    public void setActive(boolean active) { isActive = active; }
    
    public RetryConfiguration getRetryConfig() { return retryConfig; }
    public void setRetryConfig(RetryConfiguration retryConfig) { this.retryConfig = retryConfig; }
    
    public Map<String, String> getCustomHeaders() { return customHeaders; }
    public void setCustomHeaders(Map<String, String> customHeaders) { this.customHeaders = customHeaders; }
    
    public OffsetDateTime getCreatedAt() { return createdAt; }
    public void setCreatedAt(OffsetDateTime createdAt) { this.createdAt = createdAt; }
    
    public OffsetDateTime getUpdatedAt() { return updatedAt; }
    public void setUpdatedAt(OffsetDateTime updatedAt) { this.updatedAt = updatedAt; }
    
    public String getSecret() { return secret; }
    public void setSecret(String secret) { this.secret = secret; }
}
