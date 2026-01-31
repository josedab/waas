# frozen_string_literal: true

require "openssl"

module Waas
  # Standalone, zero-dependency WaaS webhook signature verification.
  #
  # Usage:
  #   verifier = Waas::WebhookVerifier.new("whsec_your_secret")
  #   verifier.verify!(payload, signature_header, timestamp_header)
  #
  # Rails middleware:
  #   config.middleware.use Waas::WebhookVerifier::RackMiddleware, "whsec_..."
  #
  class WebhookVerifier
    SIGNATURE_HEADER = "X-WaaS-Signature"
    TIMESTAMP_HEADER = "X-WaaS-Timestamp"
    DEFAULT_TOLERANCE = 300

    class VerificationError < StandardError; end
    class MissingSignatureError < VerificationError; end
    class InvalidSignatureError < VerificationError; end
    class TimestampExpiredError < VerificationError; end

    # @param secret [String] Primary signing secret
    # @param secrets [Array<String>] Additional secrets for key rotation
    # @param timestamp_tolerance [Integer] Max allowed age in seconds
    def initialize(secret, secrets: [], timestamp_tolerance: DEFAULT_TOLERANCE)
      @secrets = [secret] + secrets
      @timestamp_tolerance = timestamp_tolerance
    end

    # Verify a webhook signature.
    #
    # @param payload [String] Raw request body
    # @param signature [String] Value of X-WaaS-Signature header
    # @param timestamp [String] Value of X-WaaS-Timestamp header
    # @return [true] if valid
    # @raise [VerificationError] if invalid
    def verify!(payload, signature, timestamp = "")
      raise MissingSignatureError, "Missing signature header" if signature.nil? || signature.empty?

      verify_timestamp!(timestamp)

      sig_bytes = parse_signature(signature)
      signed_payload = "#{timestamp}.#{payload}"

      @secrets.each do |secret|
        expected = OpenSSL::HMAC.digest("SHA256", secret, signed_payload)
        return true if secure_compare(sig_bytes, expected)
      end

      raise InvalidSignatureError, "Signature does not match"
    end

    # Generate a signature for testing.
    def sign(payload, timestamp)
      signed_payload = "#{timestamp}.#{payload}"
      mac = OpenSSL::HMAC.hexdigest("SHA256", @secrets.first, signed_payload)
      "v1=#{mac}"
    end

    private

    def verify_timestamp!(timestamp)
      return if timestamp.nil? || timestamp.empty? || @timestamp_tolerance <= 0

      ts = Integer(timestamp)
      diff = (Time.now.to_i - ts).abs
      if diff > @timestamp_tolerance
        raise TimestampExpiredError, "Timestamp expired: #{diff}s > #{@timestamp_tolerance}s tolerance"
      end
    rescue ArgumentError => e
      raise VerificationError, "Malformed timestamp: #{e.message}"
    end

    def parse_signature(header)
      parts = header.split("=", 2)
      raise VerificationError, "Malformed signature header" if parts.length != 2

      [parts[1]].pack("H*")
    end

    def secure_compare(a, b)
      return false unless a.bytesize == b.bytesize

      OpenSSL.fixed_length_secure_compare(a, b)
    rescue StandardError
      # Fallback for older Ruby versions
      a == b
    end

    # Rack/Rails middleware for webhook signature verification.
    #
    # Usage in Rails:
    #   config.middleware.use Waas::WebhookVerifier::RackMiddleware, "whsec_..."
    class RackMiddleware
      def initialize(app, secret, **options)
        @app = app
        @verifier = WebhookVerifier.new(secret, **options)
      end

      def call(env)
        return @app.call(env) unless env["REQUEST_METHOD"] == "POST"

        signature = env["HTTP_X_WAAS_SIGNATURE"]
        return @app.call(env) if signature.nil? || signature.empty?

        timestamp = env["HTTP_X_WAAS_TIMESTAMP"] || ""
        body = env["rack.input"].read
        env["rack.input"].rewind

        begin
          @verifier.verify!(body, signature, timestamp)
          @app.call(env)
        rescue VerificationError => e
          [401, { "Content-Type" => "application/json" }, [%({"error":"#{e.message}"})]]
        end
      end
    end
  end
end
