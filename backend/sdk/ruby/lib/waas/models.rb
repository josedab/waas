# frozen_string_literal: true

require "ostruct"
require "time"

module WAAS
  module Models
    class Base < OpenStruct
      def self.from_hash(hash)
        return nil if hash.nil?
        new(hash.transform_keys(&:to_sym))
      end
    end

    class WebhookEndpoint < Base
      def created_at
        Time.parse(self[:created_at]) if self[:created_at]
      end

      def updated_at
        Time.parse(self[:updated_at]) if self[:updated_at]
      end
    end

    class RetryConfiguration < Base; end

    class DeliveryAttempt < Base
      def scheduled_at
        Time.parse(self[:scheduled_at]) if self[:scheduled_at]
      end

      def delivered_at
        Time.parse(self[:delivered_at]) if self[:delivered_at]
      end

      def created_at
        Time.parse(self[:created_at]) if self[:created_at]
      end
    end

    class SendWebhookResponse < Base
      def scheduled_at
        Time.parse(self[:scheduled_at]) if self[:scheduled_at]
      end
    end
  end
end
