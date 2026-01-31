package com.waas.sdk.verify;

import javax.servlet.*;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;

/**
 * Servlet Filter for WaaS webhook signature verification.
 * Compatible with Spring Boot via FilterRegistrationBean.
 *
 * <pre>{@code
 * @Bean
 * public FilterRegistrationBean<WebhookVerificationFilter> webhookFilter() {
 *     FilterRegistrationBean<WebhookVerificationFilter> bean = new FilterRegistrationBean<>();
 *     bean.setFilter(new WebhookVerificationFilter("whsec_your_secret"));
 *     bean.addUrlPatterns("/webhooks/*");
 *     return bean;
 * }
 * }</pre>
 */
public class WebhookVerificationFilter implements Filter {

    private final WebhookVerifier verifier;

    public WebhookVerificationFilter(String secret) {
        this.verifier = new WebhookVerifier(secret);
    }

    @Override
    public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain)
            throws IOException, ServletException {
        HttpServletRequest httpReq = (HttpServletRequest) request;
        HttpServletResponse httpResp = (HttpServletResponse) response;

        if (!"POST".equalsIgnoreCase(httpReq.getMethod())) {
            chain.doFilter(request, response);
            return;
        }

        String signature = httpReq.getHeader(WebhookVerifier.SIGNATURE_HEADER);
        String timestamp = httpReq.getHeader(WebhookVerifier.TIMESTAMP_HEADER);

        if (signature == null || signature.isEmpty()) {
            chain.doFilter(request, response);
            return;
        }

        byte[] body = readBody(httpReq.getInputStream());

        try {
            verifier.verify(body, signature, timestamp);
        } catch (WebhookVerifier.VerificationException e) {
            httpResp.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
            httpResp.getWriter().write("{\"error\":\"" + e.getMessage() + "\"}");
            return;
        }

        chain.doFilter(request, response);
    }

    private byte[] readBody(InputStream is) throws IOException {
        ByteArrayOutputStream buf = new ByteArrayOutputStream();
        byte[] tmp = new byte[4096];
        int n;
        while ((n = is.read(tmp)) != -1) {
            buf.write(tmp, 0, n);
        }
        return buf.toByteArray();
    }
}
