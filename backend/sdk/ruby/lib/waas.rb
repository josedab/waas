# frozen_string_literal: true

require "faraday"
require "faraday/retry"
require "json"

require_relative "waas/version"
require_relative "waas/errors"
require_relative "waas/configuration"
require_relative "waas/models"
require_relative "waas/services/endpoints"
require_relative "waas/services/deliveries"
require_relative "waas/services/webhooks"
require_relative "waas/client"

module WAAS
  class << self
    attr_accessor :configuration

    def configure
      self.configuration ||= Configuration.new
      yield(configuration)
    end
  end
end
