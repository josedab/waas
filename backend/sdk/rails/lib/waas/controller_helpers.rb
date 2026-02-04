# frozen_string_literal: true

module WaaS
  # Mixin for Rails controllers to handle incoming webhooks
  module ControllerHelpers
    extend ActiveSupport::Concern if defined?(ActiveSupport)

    def verify_waas_webhook!
      verifier = WaaS::WebhookVerifier.new
      signature = request.headers['X-Webhook-Signature']
      body = request.raw_post

      verifier.verify!(signature, body)
    rescue WaaS::SignatureVerificationError => e
      render json: { error: e.message }, status: :unauthorized
    end

    def waas_webhook_payload
      @waas_payload ||= JSON.parse(request.raw_post)
    end

    def send_waas_webhook(event_type:, payload:, **opts)
      WaaS.client.send_webhook(event_type: event_type, payload: payload, **opts)
    end
  end
end
