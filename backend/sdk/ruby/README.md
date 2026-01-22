# WAAS Ruby SDK

Official Ruby SDK for the WAAS (Webhook-as-a-Service) Platform.

## Requirements

- Ruby >= 3.0.0

## Installation

Add to your Gemfile:

```ruby
gem 'waas-sdk'
```

Or install directly:

```bash
gem install waas-sdk
```

## Quick Start

```ruby
require 'waas'

# Initialize client
client = WAAS::Client.new(api_key: 'your-api-key')

# Create an endpoint
endpoint = client.endpoints.create(
  url: 'https://your-server.com/webhook',
  retry_config: {
    max_attempts: 5,
    initial_delay_ms: 1000
  }
)
puts "Created endpoint: #{endpoint.id}"

# Send a webhook
result = client.webhooks.send(
  endpoint_id: endpoint.id,
  payload: { event: 'user.created', data: { id: 123 } }
)
puts "Delivery scheduled: #{result.delivery_id}"

# Check status
delivery = client.deliveries.get(result.delivery_id)
puts "Status: #{delivery.status}"
```

## Configuration

```ruby
WAAS.configure do |config|
  config.api_key = ENV['WAAS_API_KEY']
  config.base_url = 'https://api.waas-platform.com/api/v1'
  config.timeout = 30
end

client = WAAS::Client.new
```

## Error Handling

```ruby
begin
  client.endpoints.get('invalid-id')
rescue WAAS::NotFoundError => e
  puts "Endpoint not found"
rescue WAAS::AuthenticationError => e
  puts "Invalid API key"
rescue WAAS::RateLimitError => e
  puts "Rate limited, retry after #{e.retry_after} seconds"
rescue WAAS::APIError => e
  puts "API error: #{e.message} (#{e.code})"
end
```

## Documentation

For full documentation, visit [docs.waas-platform.com/sdks/ruby](https://docs.waas-platform.com/sdks/ruby)

## License

MIT License
