package com.waas.spring;

/**
 * Configuration properties for the WaaS Spring Boot Starter.
 *
 * Add to application.properties:
 *   waas.api-key=wh_your_key
 *   waas.base-url=http://localhost:8080/api/v1
 *   waas.signing-secret=your_secret
 */
public class WaaSProperties {

    private String apiKey;
    private String baseUrl = "http://localhost:8080/api/v1";
    private String signingSecret;
    private int timeoutSeconds = 30;
    private int toleranceSeconds = 300;

    public String getApiKey() { return apiKey; }
    public void setApiKey(String apiKey) { this.apiKey = apiKey; }

    public String getBaseUrl() { return baseUrl; }
    public void setBaseUrl(String baseUrl) { this.baseUrl = baseUrl; }

    public String getSigningSecret() { return signingSecret; }
    public void setSigningSecret(String signingSecret) { this.signingSecret = signingSecret; }

    public int getTimeoutSeconds() { return timeoutSeconds; }
    public void setTimeoutSeconds(int timeoutSeconds) { this.timeoutSeconds = timeoutSeconds; }

    public int getToleranceSeconds() { return toleranceSeconds; }
    public void setToleranceSeconds(int toleranceSeconds) { this.toleranceSeconds = toleranceSeconds; }
}
