# frozen_string_literal: true

module WAAS
  class Configuration
    attr_accessor :api_key, :base_url, :timeout, :open_timeout

    def initialize
      @base_url = "https://api.waas-platform.com/api/v1"
      @timeout = 30
      @open_timeout = 10
    end
  end
end
