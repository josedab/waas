# frozen_string_literal: true

module WAAS
  class Error < StandardError; end

  class APIError < Error
    attr_reader :status_code, :code, :details

    def initialize(message, status_code: nil, code: nil, details: nil)
      @status_code = status_code
      @code = code
      @details = details
      super(message)
    end
  end

  class AuthenticationError < APIError
    def initialize(message = "Authentication failed")
      super(message, status_code: 401, code: "AUTHENTICATION_FAILED")
    end
  end

  class NotFoundError < APIError
    def initialize(message = "Resource not found")
      super(message, status_code: 404, code: "NOT_FOUND")
    end
  end

  class RateLimitError < APIError
    attr_reader :retry_after

    def initialize(message = "Rate limit exceeded", retry_after: nil)
      @retry_after = retry_after
      super(message, status_code: 429, code: "RATE_LIMIT_EXCEEDED")
    end
  end

  class ValidationError < APIError
    def initialize(message = "Validation failed", details: nil)
      super(message, status_code: 422, code: "VALIDATION_ERROR", details: details)
    end
  end
end
