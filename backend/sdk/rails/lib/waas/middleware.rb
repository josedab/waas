# frozen_string_literal: true

module WaaS
  # Rails middleware for automatic webhook signature verification
  class Middleware
    def initialize(app, path: '/webhooks/waas', secret: nil)
      @app = app
      @path = path
      @verifier = WebhookVerifier.new(secret)
    end

    def call(env)
      request = Rack::Request.new(env)

      if request.path == @path && request.post?
        body = request.body.read
        request.body.rewind

        signature = env['HTTP_X_WEBHOOK_SIGNATURE']
        unless @verifier.verify(signature, body)
          return [401, { 'Content-Type' => 'application/json' }, ['{"error":"Invalid webhook signature"}']]
        end

        env['waas.verified'] = true
        env['waas.payload'] = body
      end

      @app.call(env)
    end
  end
end
