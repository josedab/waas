# frozen_string_literal: true

module WAAS
  module Services
    class Deliveries
      def initialize(client)
        @client = client
      end

      def get(delivery_id)
        response = @client.get("/webhooks/deliveries/#{delivery_id}")
        Models::DeliveryAttempt.from_hash(response)
      end

      def list(endpoint_id: nil, status: nil, page: 1, per_page: 20)
        params = ["page=#{page}", "per_page=#{per_page}"]
        params << "endpoint_id=#{endpoint_id}" if endpoint_id
        params << "status=#{status}" if status

        response = @client.get("/webhooks/deliveries?#{params.join("&")}")
        {
          deliveries: (response["deliveries"] || []).map { |d| Models::DeliveryAttempt.from_hash(d) },
          total: response["total"],
          page: response["page"],
          per_page: response["per_page"]
        }
      end

      def retry(delivery_id)
        response = @client.post("/webhooks/deliveries/#{delivery_id}/retry")
        Models::SendWebhookResponse.from_hash(response)
      end
    end
  end
end
