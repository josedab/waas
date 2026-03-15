package com.waas.spring;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.PropertyNamingStrategies;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.List;
import java.util.Map;

/**
 * HTTP client for the WaaS API.
 *
 * Usage with Spring Boot:
 * <pre>
 * &#64;Autowired WaaSClient waasClient;
 * waasClient.sendWebhook(endpointId, "user.created", payload);
 * </pre>
 */
public class WaaSClient {

    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final String baseUrl;
    private final String apiKey;

    public WaaSClient(String apiKey, String baseUrl, int timeoutSeconds) {
        this.apiKey = apiKey;
        this.baseUrl = baseUrl;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(timeoutSeconds))
                .build();
        this.objectMapper = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .setPropertyNamingStrategy(PropertyNamingStrategies.SNAKE_CASE)
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    // --- Endpoint Management ---

    public Map<String, Object> createEndpoint(String url, Map<String, Object> options) throws WaaSException {
        Map<String, Object> body = new java.util.HashMap<>(options);
        body.put("url", url);
        return post("/webhooks/endpoints", body);
    }

    public Map<String, Object> getEndpoint(String endpointId) throws WaaSException {
        return get("/webhooks/endpoints/" + endpointId);
    }

    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> listEndpoints() throws WaaSException {
        Map<String, Object> response = get("/webhooks/endpoints");
        return (List<Map<String, Object>>) response.get("endpoints");
    }

    public void deleteEndpoint(String endpointId) throws WaaSException {
        delete("/webhooks/endpoints/" + endpointId);
    }

    // --- Webhook Sending ---

    public Map<String, Object> sendWebhook(String endpointId, String eventType, Object payload) throws WaaSException {
        return post("/webhooks/send", Map.of(
                "endpoint_id", endpointId,
                "event_type", eventType,
                "payload", payload
        ));
    }

    // --- Delivery ---

    public Map<String, Object> getDelivery(String deliveryId) throws WaaSException {
        return get("/webhooks/deliveries/" + deliveryId);
    }

    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> listDeliveries(Map<String, String> filters) throws WaaSException {
        StringBuilder qs = new StringBuilder();
        if (filters != null) {
            for (Map.Entry<String, String> entry : filters.entrySet()) {
                qs.append(qs.length() == 0 ? "?" : "&")
                        .append(entry.getKey()).append("=").append(entry.getValue());
            }
        }
        Map<String, Object> response = get("/webhooks/deliveries" + qs);
        return (List<Map<String, Object>>) response.get("deliveries");
    }

    // --- Tenant ---

    public Map<String, Object> getTenant() throws WaaSException {
        return get("/tenant");
    }

    // --- Internal HTTP Methods ---

    @SuppressWarnings("unchecked")
    private Map<String, Object> get(String path) throws WaaSException {
        HttpRequest request = newRequestBuilder(path).GET().build();
        return execute(request, Map.class);
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> post(String path, Object body) throws WaaSException {
        try {
            String json = objectMapper.writeValueAsString(body);
            HttpRequest request = newRequestBuilder(path)
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();
            return execute(request, Map.class);
        } catch (IOException e) {
            throw new WaaSException("Failed to serialize request", e);
        }
    }

    private void delete(String path) throws WaaSException {
        HttpRequest request = newRequestBuilder(path).DELETE().build();
        execute(request, Map.class);
    }

    private HttpRequest.Builder newRequestBuilder(String path) {
        return HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Authorization", "Bearer " + apiKey)
                .header("Content-Type", "application/json")
                .header("User-Agent", "waas-spring-boot/1.0.0");
    }

    private <T> T execute(HttpRequest request, Class<T> responseType) throws WaaSException {
        try {
            HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
            int status = response.statusCode();

            if (status == 204) return null;

            if (status >= 400) {
                handleError(status, response.body());
            }

            return objectMapper.readValue(response.body(), responseType);
        } catch (WaaSException e) {
            throw e;
        } catch (Exception e) {
            throw new WaaSException("Request failed: " + e.getMessage(), e);
        }
    }

    private void handleError(int status, String body) throws WaaSException {
        String message = "API error (HTTP " + status + ")";
        try {
            @SuppressWarnings("unchecked")
            Map<String, Object> error = objectMapper.readValue(body, Map.class);
            if (error.containsKey("message")) {
                message = error.get("message").toString();
            }
        } catch (Exception ignored) {}

        switch (status) {
            case 401: throw new WaaSException("Authentication failed: " + message);
            case 404: throw new WaaSException("Not found: " + message);
            case 429: throw new WaaSException("Rate limited: " + message);
            default: throw new WaaSException(message);
        }
    }

    public static class WaaSException extends Exception {
        public WaaSException(String message) { super(message); }
        public WaaSException(String message, Throwable cause) { super(message, cause); }
    }
}
