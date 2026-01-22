# frozen_string_literal: true

module WAAS
  module Services
    class Webhooks
      def initialize(client)
        @client = client
      end

      def send(endpoint_id:, payload:, headers: nil)
        body = { endpoint_id: endpoint_id, payload: payload }
        body[:headers] = headers if headers

        response = @client.post("/webhooks/send", body)
        Models::SendWebhookResponse.from_hash(response)
      end
    end
  end
end
