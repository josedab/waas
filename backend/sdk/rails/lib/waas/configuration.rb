# frozen_string_literal: true

module WaaS
  class Configuration
    attr_accessor :api_key, :api_url, :signing_secret, :timeout, :verify_signatures, :retry_count

    def initialize
      @api_url = ENV.fetch('WAAS_API_URL', 'http://localhost:8080')
      @api_key = ENV['WAAS_API_KEY']
      @signing_secret = ENV['WAAS_SIGNING_SECRET']
      @timeout = 30
      @verify_signatures = true
      @retry_count = 3
    end

    def validate!
      raise ConfigurationError, 'api_key is required' if api_key.nil? || api_key.empty?
    end
  end
end
