# WaaS Rails Gem

Drop-in webhook integration for Ruby on Rails (7.x / 8.x).

## Installation

Since WaaS is self-hosted, add the gem from the local SDK directory in your `Gemfile`:

```ruby
gem 'waas-rails', path: '/path/to/waas/backend/sdk/rails'
```

Then run:

```bash
bundle install
rails generate waas:install
```

### Requirements

- Ruby ≥ 3.1
- Rails ≥ 7.0

## Quick Start

1. Configure WaaS in `config/initializers/waas.rb`:

```ruby
WaaS.configure do |config|
  config.api_url = "http://localhost:8080"
  config.api_key = ENV["WAAS_API_KEY"]
end
```

2. Send webhooks from anywhere in your app:

```ruby
WaaS.send(
  endpoint_id: "your-endpoint-id",
  payload: { event: "order.created", data: { id: 123 } }
)
```

3. Receive webhooks with signature verification:

```ruby
# config/routes.rb
post "/webhooks", to: "webhooks#receive"

# app/controllers/webhooks_controller.rb
class WebhooksController < ApplicationController
  skip_before_action :verify_authenticity_token

  def receive
    payload = WaaS.verify_webhook!(request)
    # Process the webhook payload
    head :ok
  end
end
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
