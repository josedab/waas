package com.waas.spring.autoconfigure;

import com.waas.spring.WaaSClient;
import com.waas.spring.WaaSProperties;
import com.waas.spring.WebhookVerifier;

/**
 * Spring Boot auto-configuration for WaaS.
 *
 * Automatically creates {@link WaaSClient} and {@link WebhookVerifier} beans
 * when {@code waas.api-key} is present in application properties.
 *
 * Example application.properties:
 * <pre>
 * waas.api-key=wh_your_key
 * waas.base-url=http://localhost:8080/api/v1
 * waas.signing-secret=your_secret
 * waas.timeout-seconds=30
 * waas.tolerance-seconds=300
 * </pre>
 *
 * Note: This auto-configuration requires Spring Boot 3.x. Add the following
 * to your {@code META-INF/spring/org.springframework.boot.autoconfigure.AutoConfiguration.imports}:
 * <pre>
 * com.waas.spring.autoconfigure.WaaSAutoConfiguration
 * </pre>
 *
 * Or register beans manually:
 * <pre>
 * &#64;Configuration
 * public class WaaSConfig {
 *     &#64;Bean
 *     public WaaSClient waaSClient() {
 *         return WaaSAutoConfiguration.createClient(properties());
 *     }
 * }
 * </pre>
 */
public class WaaSAutoConfiguration {

    /**
     * Creates a WaaSClient from the given properties.
     */
    public static WaaSClient createClient(WaaSProperties properties) {
        if (properties.getApiKey() == null || properties.getApiKey().isEmpty()) {
            throw new IllegalStateException("waas.api-key is required");
        }
        return new WaaSClient(
                properties.getApiKey(),
                properties.getBaseUrl(),
                properties.getTimeoutSeconds()
        );
    }

    /**
     * Creates a WebhookVerifier from the given properties.
     * Returns null if no signing secret is configured.
     */
    public static WebhookVerifier createVerifier(WaaSProperties properties) {
        if (properties.getSigningSecret() == null || properties.getSigningSecret().isEmpty()) {
            return null;
        }
        return new WebhookVerifier(
                properties.getSigningSecret(),
                properties.getToleranceSeconds()
        );
    }
}
