# WaaS Ruby on Rails Gem
#
# Drop-in webhook integration for Rails (7.x / 8.x)
#
# Setup (3 lines):
#   gem 'waas-rails'
#   rails generate waas:install
#   WaaS.configure { |c| c.api_key = ENV['WAAS_API_KEY'] }

Gem::Specification.new do |s|
  s.name        = 'waas-rails'
  s.version     = '1.0.0'
  s.summary     = 'WaaS webhook integration for Rails'
  s.description = 'Drop-in webhook sending and receiving for Ruby on Rails with automatic retries, signature verification, and admin UI.'
  s.authors     = ['WaaS Team']
  s.email       = 'support@waas.dev'
  s.homepage    = 'https://github.com/josedab/waas'
  s.license     = 'MIT'

  s.files       = Dir['lib/**/*', 'app/**/*', 'config/**/*', 'README.md']
  s.require_paths = ['lib']

  s.required_ruby_version = '>= 3.1'
  s.add_dependency 'rails', '>= 7.0', '< 9.0'
  s.add_dependency 'faraday', '>= 2.0'
end
