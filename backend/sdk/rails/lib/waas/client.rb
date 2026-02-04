# frozen_string_literal: true

require 'faraday'
require 'json'

module WaaS
  class Client
    def initialize(config = WaaS.configuration)
      @config = config
      @conn = Faraday.new(url: config.api_url) do |f|
        f.request :json
        f.response :json
        f.headers['X-API-Key'] = config.api_key
        f.headers['Content-Type'] = 'application/json'
        f.options.timeout = config.timeout
      end
    end

    # Send a webhook event
    def send_webhook(event_type:, payload:, endpoint_ids: nil)
      body = { event_type: event_type, payload: payload }
      body[:endpoint_ids] = endpoint_ids if endpoint_ids
      response = @conn.post('/api/v1/webhooks/send', body)
      handle_response(response)
    end

    # Register a webhook endpoint
    def create_endpoint(url:, event_types: nil, **options)
      body = { url: url }
      body[:event_types] = event_types if event_types
      body.merge!(options)
      response = @conn.post('/api/v1/endpoints', body)
      handle_response(response)
    end

    # List delivery attempts
    def list_deliveries(limit: 20, offset: 0)
      response = @conn.get('/api/v1/webhooks/deliveries', { limit: limit, offset: offset })
      handle_response(response)
    end

    # Health check
    def health
      response = @conn.get('/health')
      handle_response(response)
    end

    private

    def handle_response(response)
      case response.status
      when 200..299
        response.body
      when 401
        raise Error, 'Unauthorized: check your API key'
      when 429
        raise Error, 'Rate limited: please retry later'
      else
        raise Error, "API error (#{response.status}): #{response.body}"
      end
    end
  end
end
