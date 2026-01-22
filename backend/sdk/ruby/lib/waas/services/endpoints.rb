# frozen_string_literal: true

module WAAS
  module Services
    class Endpoints
      def initialize(client)
        @client = client
      end

      def create(url:, custom_headers: nil, retry_config: nil)
        body = { url: url }
        body[:custom_headers] = custom_headers if custom_headers
        body[:retry_config] = retry_config if retry_config

        response = @client.post("/webhooks/endpoints", body)
        Models::WebhookEndpoint.from_hash(response)
      end

      def get(endpoint_id)
        response = @client.get("/webhooks/endpoints/#{endpoint_id}")
        Models::WebhookEndpoint.from_hash(response)
      end

      def list(page: 1, per_page: 20)
        response = @client.get("/webhooks/endpoints?page=#{page}&per_page=#{per_page}")
        {
          endpoints: (response["endpoints"] || []).map { |e| Models::WebhookEndpoint.from_hash(e) },
          total: response["total"],
          page: response["page"],
          per_page: response["per_page"]
        }
      end

      def update(endpoint_id, **attributes)
        response = @client.patch("/webhooks/endpoints/#{endpoint_id}", attributes)
        Models::WebhookEndpoint.from_hash(response)
      end

      def delete(endpoint_id)
        @client.delete("/webhooks/endpoints/#{endpoint_id}")
        true
      end

      def rotate_secret(endpoint_id)
        response = @client.post("/webhooks/endpoints/#{endpoint_id}/rotate-secret")
        response["secret"]
      end
    end
  end
end
