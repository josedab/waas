package com.waas.sdk.client;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.PropertyNamingStrategies;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.waas.sdk.exceptions.*;
import com.waas.sdk.models.*;
import okhttp3.*;

import java.io.IOException;
import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

/**
 * Main WAAS API Client.
 */
public class WAASClient {
    private static final String DEFAULT_BASE_URL = "https://api.waas-platform.com/api/v1";
    private static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(30);
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final String baseUrl;
    private final String apiKey;

    private final EndpointsService endpoints;
    private final DeliveriesService deliveries;
    private final WebhooksService webhooks;

    private WAASClient(Builder builder) {
        this.apiKey = builder.apiKey;
        this.baseUrl = builder.baseUrl != null ? builder.baseUrl : DEFAULT_BASE_URL;
        
        Duration timeout = builder.timeout != null ? builder.timeout : DEFAULT_TIMEOUT;
        
        this.httpClient = new OkHttpClient.Builder()
            .connectTimeout(timeout.toMillis(), TimeUnit.MILLISECONDS)
            .readTimeout(timeout.toMillis(), TimeUnit.MILLISECONDS)
            .writeTimeout(timeout.toMillis(), TimeUnit.MILLISECONDS)
            .addInterceptor(chain -> {
                Request original = chain.request();
                Request request = original.newBuilder()
                    .header("X-API-Key", apiKey)
                    .header("Content-Type", "application/json")
                    .header("User-Agent", "waas-sdk-java/1.0.0")
                    .build();
                return chain.proceed(request);
            })
            .build();

        this.objectMapper = new ObjectMapper()
            .registerModule(new JavaTimeModule())
            .setPropertyNamingStrategy(PropertyNamingStrategies.SNAKE_CASE)
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

        this.endpoints = new EndpointsService(this);
        this.deliveries = new DeliveriesService(this);
        this.webhooks = new WebhooksService(this);
    }

    public static Builder builder() {
        return new Builder();
    }

    public EndpointsService endpoints() { return endpoints; }
    public DeliveriesService deliveries() { return deliveries; }
    public WebhooksService webhooks() { return webhooks; }

    // Internal HTTP methods
    <T> T get(String path, Class<T> responseType) throws WAASException {
        Request request = new Request.Builder()
            .url(baseUrl + path)
            .get()
            .build();
        return execute(request, responseType);
    }

    <T> T post(String path, Object body, Class<T> responseType) throws WAASException {
        try {
            String json = objectMapper.writeValueAsString(body);
            RequestBody requestBody = RequestBody.create(json, JSON);
            Request request = new Request.Builder()
                .url(baseUrl + path)
                .post(requestBody)
                .build();
            return execute(request, responseType);
        } catch (IOException e) {
            throw new WAASException("Failed to serialize request body", e);
        }
    }

    <T> T patch(String path, Object body, Class<T> responseType) throws WAASException {
        try {
            String json = objectMapper.writeValueAsString(body);
            RequestBody requestBody = RequestBody.create(json, JSON);
            Request request = new Request.Builder()
                .url(baseUrl + path)
                .patch(requestBody)
                .build();
            return execute(request, responseType);
        } catch (IOException e) {
            throw new WAASException("Failed to serialize request body", e);
        }
    }

    void delete(String path) throws WAASException {
        Request request = new Request.Builder()
            .url(baseUrl + path)
            .delete()
            .build();
        execute(request, Void.class);
    }

    private <T> T execute(Request request, Class<T> responseType) throws WAASException {
        try (Response response = httpClient.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                handleErrorResponse(response);
            }
            
            if (responseType == Void.class || response.code() == 204) {
                return null;
            }

            ResponseBody body = response.body();
            if (body == null) {
                throw new WAASException("Empty response body");
            }

            return objectMapper.readValue(body.string(), responseType);
        } catch (IOException e) {
            throw new WAASException("Request failed: " + e.getMessage(), e);
        }
    }

    private void handleErrorResponse(Response response) throws WAASException {
        int statusCode = response.code();
        String message = "API error";
        String code = "UNKNOWN_ERROR";

        try {
            ResponseBody body = response.body();
            if (body != null) {
                Map<String, Object> errorBody = objectMapper.readValue(body.string(), Map.class);
                message = (String) errorBody.getOrDefault("message", message);
                code = (String) errorBody.getOrDefault("code", code);
            }
        } catch (IOException ignored) {}

        switch (statusCode) {
            case 401:
                throw new WAASAuthenticationException(message);
            case 404:
                throw new WAASNotFoundException(message);
            case 429:
                String retryAfterHeader = response.header("Retry-After");
                Integer retryAfter = retryAfterHeader != null ? Integer.parseInt(retryAfterHeader) : null;
                throw new WAASRateLimitException(message, retryAfter);
            default:
                throw new WAASApiException(message, statusCode, code);
        }
    }

    public static class Builder {
        private String apiKey;
        private String baseUrl;
        private Duration timeout;

        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }

        public Builder baseUrl(String baseUrl) {
            this.baseUrl = baseUrl;
            return this;
        }

        public Builder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        public WAASClient build() {
            if (apiKey == null || apiKey.isEmpty()) {
                throw new IllegalArgumentException("API key is required");
            }
            return new WAASClient(this);
        }
    }

    /**
     * Endpoints Service
     */
    public static class EndpointsService {
        private final WAASClient client;

        EndpointsService(WAASClient client) {
            this.client = client;
        }

        public WebhookEndpoint create(CreateEndpointRequest request) {
            return client.post("/webhooks/endpoints", request, WebhookEndpoint.class);
        }

        public WebhookEndpoint get(UUID endpointId) {
            return client.get("/webhooks/endpoints/" + endpointId, WebhookEndpoint.class);
        }

        public void delete(UUID endpointId) {
            client.delete("/webhooks/endpoints/" + endpointId);
        }
    }

    /**
     * Deliveries Service
     */
    public static class DeliveriesService {
        private final WAASClient client;

        DeliveriesService(WAASClient client) {
            this.client = client;
        }

        public DeliveryAttempt get(UUID deliveryId) {
            return client.get("/webhooks/deliveries/" + deliveryId, DeliveryAttempt.class);
        }

        public SendWebhookResponse retry(UUID deliveryId) {
            return client.post("/webhooks/deliveries/" + deliveryId + "/retry", Map.of(), SendWebhookResponse.class);
        }
    }

    /**
     * Webhooks Service
     */
    public static class WebhooksService {
        private final WAASClient client;

        WebhooksService(WAASClient client) {
            this.client = client;
        }

        public SendWebhookResponse send(SendWebhookRequest request) {
            return client.post("/webhooks/send", request, SendWebhookResponse.class);
        }
    }
}
