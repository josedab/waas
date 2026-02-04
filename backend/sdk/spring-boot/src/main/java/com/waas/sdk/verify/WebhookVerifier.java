package com.waas.sdk.verify;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.List;

/**
 * Standalone, zero-dependency WaaS webhook signature verification.
 *
 * <p>Algorithm:
 * <ol>
 *   <li>Parse X-WaaS-Signature header (format: {@code v1=<hex_encoded_hmac>})</li>
 *   <li>Parse X-WaaS-Timestamp header (Unix seconds string)</li>
 *   <li>Validate timestamp is within tolerance (default 300s)</li>
 *   <li>Construct signed content: {@code timestamp + "." + raw_payload_body}</li>
 *   <li>Compute HMAC-SHA256 with secret key</li>
 *   <li>Compare using {@link MessageDigest#isEqual} (timing-safe)</li>
 * </ol>
 *
 * <p>Usage:
 * <pre>{@code
 *   WebhookVerifier verifier = new WebhookVerifier();
 *   verifier.verifySignature(body, sigHeader, tsHeader, secret);
 *   verifier.verifyWithMultipleSecrets(body, sigHeader, tsHeader, List.of(s1, s2));
 * }</pre>
 */
public class WebhookVerifier {

    public static final String SIGNATURE_HEADER = "X-WaaS-Signature";
    public static final String TIMESTAMP_HEADER = "X-WaaS-Timestamp";
    private static final long DEFAULT_TOLERANCE_SECONDS = 300;

    /**
     * Verify a WaaS webhook signature.
     *
     * @param payload          Raw request body
     * @param signatureHeader  Value of X-WaaS-Signature header ({@code v1=<hex>})
     * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
     * @param secret           Signing secret
     * @param toleranceSeconds Max allowed timestamp age in seconds (0 to disable)
     * @return true if the signature is valid
     * @throws WebhookVerificationException if verification fails
     */
    public boolean verifySignature(
            String payload,
            String signatureHeader,
            String timestampHeader,
            String secret,
            long toleranceSeconds
    ) throws WebhookVerificationException {
        if (secret == null || secret.isEmpty()) {
            throw new WebhookVerificationException("No signing secret configured");
        }
        if (signatureHeader == null || signatureHeader.isEmpty()) {
            throw new WebhookVerificationException("Missing signature header");
        }

        String sig = parseSignature(signatureHeader);
        validateTimestamp(timestampHeader, toleranceSeconds);

        String signedContent = timestampHeader + "." + payload;
        String expected = computeHmacSHA256(signedContent, secret);

        if (!MessageDigest.isEqual(
                sig.getBytes(StandardCharsets.UTF_8),
                expected.getBytes(StandardCharsets.UTF_8))) {
            throw new WebhookVerificationException("Signature does not match");
        }

        return true;
    }

    /**
     * Verify with default tolerance of 300 seconds.
     */
    public boolean verifySignature(
            String payload,
            String signatureHeader,
            String timestampHeader,
            String secret
    ) throws WebhookVerificationException {
        return verifySignature(payload, signatureHeader, timestampHeader, secret, DEFAULT_TOLERANCE_SECONDS);
    }

    /**
     * Verify a webhook signature against multiple secrets (for key rotation).
     *
     * <p>Tries each secret in order and returns true on the first match.
     *
     * @param payload          Raw request body
     * @param signatureHeader  Value of X-WaaS-Signature header ({@code v1=<hex>})
     * @param timestampHeader  Value of X-WaaS-Timestamp header (Unix seconds)
     * @param secrets          List of signing secrets to try
     * @param toleranceSeconds Max allowed timestamp age in seconds (0 to disable)
     * @return true if the signature matches any secret
     * @throws WebhookVerificationException if verification fails
     */
    public boolean verifyWithMultipleSecrets(
            String payload,
            String signatureHeader,
            String timestampHeader,
            List<String> secrets,
            long toleranceSeconds
    ) throws WebhookVerificationException {
        if (secrets == null || secrets.isEmpty()) {
            throw new WebhookVerificationException("No signing secrets configured");
        }
        if (signatureHeader == null || signatureHeader.isEmpty()) {
            throw new WebhookVerificationException("Missing signature header");
        }

        String sig = parseSignature(signatureHeader);
        validateTimestamp(timestampHeader, toleranceSeconds);

        String signedContent = timestampHeader + "." + payload;

        for (String secret : secrets) {
            String expected = computeHmacSHA256(signedContent, secret);
            if (MessageDigest.isEqual(
                    sig.getBytes(StandardCharsets.UTF_8),
                    expected.getBytes(StandardCharsets.UTF_8))) {
                return true;
            }
        }

        throw new WebhookVerificationException("Signature does not match");
    }

    /**
     * Verify with multiple secrets and default tolerance of 300 seconds.
     */
    public boolean verifyWithMultipleSecrets(
            String payload,
            String signatureHeader,
            String timestampHeader,
            List<String> secrets
    ) throws WebhookVerificationException {
        return verifyWithMultipleSecrets(payload, signatureHeader, timestampHeader, secrets, DEFAULT_TOLERANCE_SECONDS);
    }

    /**
     * Generate a signature for testing purposes.
     *
     * @param payload   Request body
     * @param timestamp Unix seconds string
     * @param secret    Signing secret
     * @return Signature in {@code v1=<hex>} format
     */
    public String sign(String payload, String timestamp, String secret) {
        String signedContent = timestamp + "." + payload;
        return "v1=" + computeHmacSHA256(signedContent, secret);
    }

    // ----- Internal helpers -----

    private String parseSignature(String header) throws WebhookVerificationException {
        String[] parts = header.split("=", 2);
        if (parts.length != 2 || !"v1".equals(parts[0])) {
            throw new WebhookVerificationException("Malformed signature header: expected v1=<hex>");
        }
        return parts[1];
    }

    private void validateTimestamp(String timestamp, long toleranceSeconds)
            throws WebhookVerificationException {
        if (timestamp == null || timestamp.isEmpty() || toleranceSeconds <= 0) {
            return;
        }

        long ts;
        try {
            ts = Long.parseLong(timestamp);
        } catch (NumberFormatException e) {
            throw new WebhookVerificationException("Malformed timestamp: " + timestamp);
        }

        long diff = Math.abs(Instant.now().getEpochSecond() - ts);
        if (diff > toleranceSeconds) {
            throw new WebhookVerificationException(
                    "Timestamp expired: " + diff + "s > " + toleranceSeconds + "s tolerance");
        }
    }

    private String computeHmacSHA256(String data, String secret) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(
                    secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            byte[] hash = mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
            StringBuilder sb = new StringBuilder(hash.length * 2);
            for (byte b : hash) {
                sb.append(String.format("%02x", b & 0xff));
            }
            return sb.toString();
        } catch (Exception e) {
            throw new RuntimeException("Failed to compute HMAC-SHA256", e);
        }
    }

    /**
     * Exception thrown when webhook signature verification fails.
     */
    public static class WebhookVerificationException extends Exception {
        public WebhookVerificationException(String message) {
            super(message);
        }
    }
}
