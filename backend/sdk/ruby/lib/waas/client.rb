# frozen_string_literal: true

module WAAS
  class Client
    attr_reader :endpoints, :deliveries, :webhooks

    def initialize(api_key: nil, base_url: nil, timeout: nil)
      @api_key = api_key || WAAS.configuration&.api_key
      @base_url = base_url || WAAS.configuration&.base_url || "https://api.waas-platform.com/api/v1"
      @timeout = timeout || WAAS.configuration&.timeout || 30

      raise ArgumentError, "API key is required" unless @api_key

      @connection = build_connection

      @endpoints = Services::Endpoints.new(self)
      @deliveries = Services::Deliveries.new(self)
      @webhooks = Services::Webhooks.new(self)
    end

    def get(path)
      request(:get, path)
    end

    def post(path, body = {})
      request(:post, path, body)
    end

    def patch(path, body = {})
      request(:patch, path, body)
    end

    def delete(path)
      request(:delete, path)
    end

    private

    def build_connection
      Faraday.new(url: @base_url) do |f|
        f.request :json
        f.request :retry, max: 2, interval: 0.5, backoff_factor: 2
        f.response :json, content_type: /\bjson$/
        f.adapter Faraday.default_adapter
        f.headers["X-API-Key"] = @api_key
        f.headers["User-Agent"] = "waas-sdk-ruby/#{VERSION}"
        f.options.timeout = @timeout
      end
    end

    def request(method, path, body = nil)
      response = @connection.public_send(method, path) do |req|
        req.body = body if body && %i[post put patch].include?(method)
      end

      handle_response(response)
    rescue Faraday::ConnectionFailed => e
      raise Error, "Connection failed: #{e.message}"
    rescue Faraday::TimeoutError => e
      raise Error, "Request timed out: #{e.message}"
    end

    def handle_response(response)
      return response.body if response.success?

      body = response.body.is_a?(Hash) ? response.body : {}
      message = body["message"] || "API error"
      code = body["code"]

      case response.status
      when 401
        raise AuthenticationError, message
      when 404
        raise NotFoundError, message
      when 429
        retry_after = response.headers["Retry-After"]&.to_i
        raise RateLimitError.new(message, retry_after: retry_after)
      when 422
        raise ValidationError.new(message, details: body["details"])
      else
        raise APIError.new(message, status_code: response.status, code: code)
      end
    end
  end
end
