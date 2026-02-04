# frozen_string_literal: true

require "openssl"

module Waas
  # Standalone, zero-dependency WaaS webhook signature verification for Rails.
  #
  # Algorithm:
  #   1. Parse X-WaaS-Signature header (format: v1=<hex_encoded_hmac>)
  #   2. Parse X-WaaS-Timestamp header (Unix seconds string)
  #   3. Validate timestamp is within tolerance (default 300s)
  #   4. Construct signed content: timestamp + "." + raw_payload_body
  #   5. Compute HMAC-SHA256 with secret key
  #   6. Compare using timing-safe comparison
  #
  # Usage:
  #   Waas::Verify.verify_signature(payload, sig_header, ts_header, secret)
  #   Waas::Verify.verify_with_multiple_secrets(payload, sig_header, ts_header, [s1, s2])
  #
  # Rails controller:
  #   include Waas::WebhookVerification
  #   before_action :verify_waas_signature!, only: :create
  module Verify
    SIGNATURE_HEADER = "X-WaaS-Signature"
    TIMESTAMP_HEADER = "X-WaaS-Timestamp"
    DEFAULT_TOLERANCE = 300

    class VerificationError < StandardError; end
    class MissingSignatureError < VerificationError; end
    class InvalidSignatureError < VerificationError; end
    class TimestampExpiredError < VerificationError; end

    # Verify a WaaS webhook signature.
    #
    # @param payload          [String] Raw request body
    # @param signature_header [String] Value of X-WaaS-Signature header (v1=<hex>)
    # @param timestamp_header [String] Value of X-WaaS-Timestamp header (Unix seconds)
    # @param secret           [String] Signing secret
    # @param tolerance_seconds [Integer] Max allowed timestamp age (default 300, 0 to disable)
    # @return [true]
    # @raise [VerificationError]
    def self.verify_signature(payload, signature_header, timestamp_header, secret, tolerance_seconds = DEFAULT_TOLERANCE)
      raise MissingSignatureError, "No signing secret configured" if secret.nil? || secret.empty?
      raise MissingSignatureError, "Missing signature header" if signature_header.nil? || signature_header.empty?

      sig_hex = parse_signature(signature_header)
      validate_timestamp(timestamp_header, tolerance_seconds)

      signed_content = "#{timestamp_header}.#{payload}"
      expected = OpenSSL::HMAC.hexdigest("SHA256", secret, signed_content)

      unless secure_compare(sig_hex, expected)
        raise InvalidSignatureError, "Signature does not match"
      end

      true
    end

    # Verify a webhook signature against multiple secrets (for key rotation).
    #
    # Tries each secret in order and returns true on the first match.
    #
    # @param payload          [String] Raw request body
    # @param signature_header [String] Value of X-WaaS-Signature header (v1=<hex>)
    # @param timestamp_header [String] Value of X-WaaS-Timestamp header (Unix seconds)
    # @param secrets          [Array<String>] Signing secrets to try
    # @param tolerance_seconds [Integer] Max allowed timestamp age (default 300, 0 to disable)
    # @return [true]
    # @raise [VerificationError]
    def self.verify_with_multiple_secrets(payload, signature_header, timestamp_header, secrets, tolerance_seconds = DEFAULT_TOLERANCE)
      raise MissingSignatureError, "No signing secrets configured" if secrets.nil? || secrets.empty?
      raise MissingSignatureError, "Missing signature header" if signature_header.nil? || signature_header.empty?

      sig_hex = parse_signature(signature_header)
      validate_timestamp(timestamp_header, tolerance_seconds)

      signed_content = "#{timestamp_header}.#{payload}"

      secrets.each do |secret|
        expected = OpenSSL::HMAC.hexdigest("SHA256", secret, signed_content)
        return true if secure_compare(sig_hex, expected)
      end

      raise InvalidSignatureError, "Signature does not match"
    end

    # Generate a signature for testing purposes.
    #
    # @param payload   [String] Request body
    # @param timestamp [String] Unix seconds
    # @param secret    [String] Signing secret
    # @return [String] Signature in v1=<hex> format
    def self.sign(payload, timestamp, secret)
      signed_content = "#{timestamp}.#{payload}"
      mac = OpenSSL::HMAC.hexdigest("SHA256", secret, signed_content)
      "v1=#{mac}"
    end

    def self.parse_signature(header)
      parts = header.split("=", 2)
      unless parts.length == 2 && parts[0] == "v1"
        raise VerificationError, "Malformed signature header: expected v1=<hex>"
      end
      parts[1]
    end
    private_class_method :parse_signature

    def self.validate_timestamp(timestamp, tolerance_seconds)
      return if timestamp.nil? || timestamp.empty? || tolerance_seconds <= 0

      ts = Integer(timestamp)
      diff = (Time.now.to_i - ts).abs
      if diff > tolerance_seconds
        raise TimestampExpiredError, "Timestamp expired: #{diff}s > #{tolerance_seconds}s tolerance"
      end
    rescue ArgumentError => e
      raise VerificationError, "Malformed timestamp: #{e.message}"
    end
    private_class_method :validate_timestamp

    def self.secure_compare(a, b)
      return false unless a.bytesize == b.bytesize
      OpenSSL.fixed_length_secure_compare(a, b)
    rescue StandardError
      # Fallback for older Ruby without OpenSSL.fixed_length_secure_compare
      l = a.unpack("C*")
      r = b.unpack("C*")
      result = 0
      l.zip(r) { |x, y| result |= x ^ y }
      result.zero?
    end
    private_class_method :secure_compare
  end

  # Rails controller concern for webhook signature verification.
  #
  # Usage:
  #   class WebhooksController < ApplicationController
  #     include Waas::WebhookVerification
  #     before_action :verify_waas_signature!, only: :create
  #
  #     def create
  #       event = JSON.parse(request.raw_post)
  #       head :ok
  #     end
  #   end
  module WebhookVerification
    def verify_waas_signature!
      secret = waas_signing_secret
      signature = request.headers[Verify::SIGNATURE_HEADER].to_s
      timestamp = request.headers[Verify::TIMESTAMP_HEADER].to_s
      body = request.raw_post

      Verify.verify_signature(body, signature, timestamp, secret)
    rescue Verify::VerificationError => e
      render json: { error: e.message }, status: :unauthorized
    end

    private

    # Override in your controller to customise secret retrieval.
    def waas_signing_secret
      if defined?(Rails)
        Rails.application.credentials.dig(:waas, :signing_secret) ||
          ENV.fetch("WAAS_SIGNING_SECRET", nil)
      else
        ENV.fetch("WAAS_SIGNING_SECRET", nil)
      end
    end
  end

  # Rack middleware for webhook signature verification.
  #
  # Usage in Rails:
  #   config.middleware.use Waas::WebhookVerificationMiddleware,
  #     secret: ENV["WAAS_SIGNING_SECRET"],
  #     path: "/webhooks/waas"
  class WebhookVerificationMiddleware
    def initialize(app, secret:, path: "/webhooks/waas", secrets: [], tolerance_seconds: Verify::DEFAULT_TOLERANCE)
      @app = app
      @path = path
      @secrets = ([secret] + Array(secrets)).compact.reject(&:empty?)
      @tolerance_seconds = tolerance_seconds
    end

    def call(env)
      if should_verify?(env)
        body = env["rack.input"].read
        env["rack.input"].rewind

        signature = env["HTTP_X_WAAS_SIGNATURE"].to_s
        timestamp = env["HTTP_X_WAAS_TIMESTAMP"].to_s

        begin
          Verify.verify_with_multiple_secrets(body, signature, timestamp, @secrets, @tolerance_seconds)
          env["waas.verified"] = true
          env["waas.payload"] = body
        rescue Verify::VerificationError => e
          return [401, { "Content-Type" => "application/json" }, [%({"error":"#{e.message}"})]]
        end
      end

      @app.call(env)
    end

    private

    def should_verify?(env)
      env["REQUEST_METHOD"] == "POST" && env["PATH_INFO"] == @path
    end
  end
end
