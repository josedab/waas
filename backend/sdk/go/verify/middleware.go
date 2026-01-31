package verify

import (
	"fmt"
	"net/http"
	"io"
	"bytes"
)

// Middleware returns an HTTP middleware that verifies WaaS webhook signatures.
// Requests with invalid signatures receive a 401 response.
func Middleware(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			sig := r.Header.Get(SignatureHeaderName)
			ts := r.Header.Get(TimestampHeaderName)

			if err := v.Verify(body, sig, ts); err != nil {
				http.Error(w, fmt.Sprintf("webhook signature verification failed: %v", err), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GinMiddleware returns a middleware function compatible with the Gin framework.
// Usage: router.Use(verify.GinMiddleware(verifier))
// Note: This returns a standard http.Handler middleware. For Gin-native middleware,
// wrap with gin.WrapH or use the VerifyRequest helper directly.
func VerifyRequest(v *Verifier, body []byte, headers http.Header) error {
	sig := headers.Get(SignatureHeaderName)
	ts := headers.Get(TimestampHeaderName)
	return v.Verify(body, sig, ts)
}
