# frozen_string_literal: true

require_relative "lib/waas/version"

Gem::Specification.new do |spec|
  spec.name = "waas-sdk"
  spec.version = WAAS::VERSION
  spec.authors = ["WAAS Team"]
  spec.email = ["sdk@waas-platform.com"]

  spec.summary = "Official Ruby SDK for the WAAS (Webhook-as-a-Service) Platform"
  spec.description = "Comprehensive Ruby client for managing webhooks, endpoints, deliveries, and analytics with the WAAS platform."
  spec.homepage = "https://github.com/waas-platform/waas-sdk-ruby"
  spec.license = "MIT"
  spec.required_ruby_version = ">= 3.0.0"

  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = spec.homepage
  spec.metadata["changelog_uri"] = "#{spec.homepage}/blob/main/CHANGELOG.md"

  spec.files = Dir.glob("{lib,sig}/**/*") + %w[README.md LICENSE.txt CHANGELOG.md]
  spec.require_paths = ["lib"]

  spec.add_dependency "faraday", "~> 2.0"
  spec.add_dependency "faraday-retry", "~> 2.0"

  spec.add_development_dependency "bundler", "~> 2.0"
  spec.add_development_dependency "rspec", "~> 3.0"
  spec.add_development_dependency "webmock", "~> 3.0"
  spec.add_development_dependency "rubocop", "~> 1.0"
end
