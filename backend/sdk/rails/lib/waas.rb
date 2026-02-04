# frozen_string_literal: true

require 'waas/configuration'
require 'waas/client'
require 'waas/webhook_verifier'
require 'waas/middleware'
require 'waas/controller_helpers'
require 'waas/railtie' if defined?(Rails)

module WaaS
  class Error < StandardError; end
  class ConfigurationError < Error; end
  class SignatureVerificationError < Error; end

  class << self
    attr_accessor :configuration

    def configure
      self.configuration ||= Configuration.new
      yield(configuration) if block_given?
      configuration.validate!
      configuration
    end

    def client
      @client ||= Client.new(configuration)
    end

    def reset!
      @client = nil
      self.configuration = nil
    end
  end
end
