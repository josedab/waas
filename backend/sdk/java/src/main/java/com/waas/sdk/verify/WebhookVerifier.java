package com.waas.sdk.verify;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.InvalidKeyException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.List;

/**
 * Standalone, zero-dependency WaaS webhook signature verification.
 *
 * <p>Usage:
 * <pre>{@code
 * WebhookVerifier verifier = new WebhookVerifier("whsec_your_secret");
 * verifier.verify(payloadBytes, signatureHeader, timestampHeader);
 * }</pre>
 *
 * <p>Spring integration:
 * <pre>{@code
 * @Bean
 * public FilterRegistrationBean<WebhookVerificationFilter> webhookFilter() {
 *     FilterRegistrationBean<WebhookVerificationFilter> bean = new FilterRegistrationBean<>();
 *     bean.setFilter(new WebhookVerificationFilter("whsec_..."));
 *     bean.addUrlPatterns("/webhooks/*");
 *     return bean;
 * }
 * }</pre>
 */
public class WebhookVerifier {

    public static final String SIGNATURE_HEADER = "X-WaaS-Signature";
    public static final String TIMESTAMP_HEADER = "X-WaaS-Timestamp";
    private static final int DEFAULT_TOLERANCE = 300;

    private final List<String> secrets;
    private final int timestampTolerance;

    public WebhookVerifier(String secret) {
        this(secret, List.of(), DEFAULT_TOLERANCE);
    }

    public WebhookVerifier(String secret, List<String> additionalSecrets, int timestampTolerance) {
        this.secrets = new ArrayList<>();
        this.secrets.add(secret);
        this.secrets.addAll(additionalSecrets);
        this.timestampTolerance = timestampTolerance;
    }

    /**
     * Verify a webhook signature.
     *
     * @param payload   Raw request body bytes.
     * @param signature Value of X-WaaS-Signature header (format: v1=hex).
     * @param timestamp Value of X-WaaS-Timestamp header (Unix seconds).
     * @throws VerificationException if verification fails.
     */
    public void verify(byte[] payload, String signature, String timestamp) throws VerificationException {
        if (signature == null || signature.isEmpty()) {
            throw new MissingSignatureException("Missing signature header");
        }

        verifyTimestamp(timestamp);

        byte[] sigBytes = parseSignature(signature);
        String signedPayload = timestamp + "." + new String(payload, StandardCharsets.UTF_8);

        for (String secret : secrets) {
            byte[] expected = computeHmac(secret, signedPayload);
            if (MessageDigest.isEqual(sigBytes, expected)) {
                return;
            }
        }

        throw new InvalidSignatureException("Signature does not match");
    }

    /** Generate a signature for testing purposes. */
    public String sign(byte[] payload, String timestamp) {
        String signedPayload = timestamp + "." + new String(payload, StandardCharsets.UTF_8);
        byte[] mac = computeHmac(secrets.get(0), signedPayload);
        return "v1=" + bytesToHex(mac);
    }

    private void verifyTimestamp(String timestamp) throws VerificationException {
        if (timestamp == null || timestamp.isEmpty() || timestampTolerance <= 0) {
            return;
        }
        try {
            long ts = Long.parseLong(timestamp);
            long diff = Math.abs(System.currentTimeMillis() / 1000 - ts);
            if (diff > timestampTolerance) {
                throw new TimestampExpiredException(
                        String.format("Timestamp expired: %ds > %ds tolerance", diff, timestampTolerance));
            }
        } catch (NumberFormatException e) {
            throw new VerificationException("Malformed timestamp: " + e.getMessage());
        }
    }

    private byte[] parseSignature(String header) throws VerificationException {
        String[] parts = header.split("=", 2);
        if (parts.length != 2) {
            throw new VerificationException("Malformed signature header");
        }
        return hexToBytes(parts[1]);
    }

    private static byte[] computeHmac(String secret, String data) {
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
            return mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
        } catch (NoSuchAlgorithmException | InvalidKeyException e) {
            throw new RuntimeException("HMAC-SHA256 not available", e);
        }
    }

    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }

    private static byte[] hexToBytes(String hex) throws VerificationException {
        if (hex.length() % 2 != 0) {
            throw new VerificationException("Invalid hex string in signature");
        }
        byte[] bytes = new byte[hex.length() / 2];
        for (int i = 0; i < hex.length(); i += 2) {
            bytes[i / 2] = (byte) Integer.parseInt(hex.substring(i, i + 2), 16);
        }
        return bytes;
    }

    // Exception classes
    public static class VerificationException extends Exception {
        public VerificationException(String message) { super(message); }
    }

    public static class MissingSignatureException extends VerificationException {
        public MissingSignatureException(String message) { super(message); }
    }

    public static class InvalidSignatureException extends VerificationException {
        public InvalidSignatureException(String message) { super(message); }
    }

    public static class TimestampExpiredException extends VerificationException {
        public TimestampExpiredException(String message) { super(message); }
    }
}
