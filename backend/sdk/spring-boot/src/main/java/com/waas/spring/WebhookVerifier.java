package com.waas.spring;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.time.Instant;

/**
 * WaaS Spring Boot Starter
 * 
 * Drop-in webhook integration for Spring Boot (3.x).
 * 
 * Quick setup:
 *   waas.api-key=${WAAS_API_KEY}
 *   waas.signing-secret=${WAAS_SIGNING_SECRET}
 *   @Autowired WaaSClient waasClient;
 */
public class WebhookVerifier {

    private final String secret;
    private final long toleranceSeconds;

    public WebhookVerifier(String secret) {
        this(secret, 300);
    }

    public WebhookVerifier(String secret, long toleranceSeconds) {
        this.secret = secret;
        this.toleranceSeconds = toleranceSeconds;
    }

    /**
     * Verify a WaaS webhook signature.
     *
     * @param signature The X-Webhook-Signature header value
     * @param body      The raw request body
     * @return true if valid
     * @throws WebhookVerificationException if verification fails
     */
    public boolean verify(String signature, byte[] body) throws WebhookVerificationException {
        if (secret == null || secret.isEmpty()) {
            throw new WebhookVerificationException("No signing secret configured");
        }
        if (signature == null || signature.isEmpty()) {
            throw new WebhookVerificationException("Missing signature header");
        }

        String ts = null;
        String sig = null;
        for (String part : signature.split(",")) {
            String[] kv = part.split("=", 2);
            if (kv.length == 2) {
                if ("t".equals(kv[0])) ts = kv[1];
                if ("v1".equals(kv[0])) sig = kv[1];
            }
        }

        if (ts == null || sig == null) {
            throw new WebhookVerificationException("Invalid signature format");
        }

        // Check timestamp tolerance
        long age = Instant.now().getEpochSecond() - Long.parseLong(ts);
        if (toleranceSeconds > 0 && age > toleranceSeconds) {
            throw new WebhookVerificationException("Timestamp expired");
        }

        // Compute expected signature
        String payload = ts + "." + new String(body, StandardCharsets.UTF_8);
        String expected = hmacSHA256(payload);

        if (!sig.equals(expected)) {
            throw new WebhookVerificationException("Signature mismatch");
        }

        return true;
    }

    private String hmacSHA256(String data) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            byte[] hash = mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
            StringBuilder sb = new StringBuilder();
            for (byte b : hash) {
                sb.append(String.format("%02x", b));
            }
            return sb.toString();
        } catch (Exception e) {
            throw new RuntimeException("Failed to compute HMAC", e);
        }
    }

    public static class WebhookVerificationException extends Exception {
        public WebhookVerificationException(String message) {
            super(message);
        }
    }
}
