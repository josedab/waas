# frozen_string_literal: true

require 'openssl'

module WaaS
  class WebhookVerifier
    def initialize(secret = nil)
      @secret = secret || WaaS.configuration&.signing_secret
    end

    # Verify a webhook signature from request headers and body
    def verify!(signature_header, body, tolerance: 300)
      raise SignatureVerificationError, 'No signing secret configured' unless @secret
      raise SignatureVerificationError, 'Missing signature header' unless signature_header

      parts = signature_header.split(',').each_with_object({}) do |p, h|
        k, v = p.split('=', 2)
        h[k] = v
      end

      ts = parts['t']
      sig = parts['v1']
      raise SignatureVerificationError, 'Invalid signature format' unless ts && sig

      # Check timestamp tolerance
      age = Time.now.to_i - ts.to_i
      raise SignatureVerificationError, 'Timestamp expired' if tolerance.positive? && age > tolerance

      # Compute expected signature
      payload = "#{ts}.#{body}"
      expected = OpenSSL::HMAC.hexdigest('SHA256', @secret, payload)

      unless secure_compare(sig, expected)
        raise SignatureVerificationError, 'Signature mismatch'
      end

      true
    end

    # Returns true/false instead of raising
    def verify(signature_header, body, **opts)
      verify!(signature_header, body, **opts)
    rescue SignatureVerificationError
      false
    end

    private

    def secure_compare(a, b)
      return false unless a.bytesize == b.bytesize
      OpenSSL.fixed_length_secure_compare(a, b)
    end
  end
end
