<?php
/**
 * WaaS Laravel Integration
 *
 * Drop-in webhook sending and receiving for Laravel (10.x / 11.x).
 *
 * Quick setup:
 *   composer require waas/laravel
 *   php artisan vendor:publish --tag=waas-config
 *   WAAS_API_KEY=your-key-here in .env
 */

namespace WaaS\Laravel;

use Illuminate\Support\ServiceProvider;

class WaaSServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->mergeConfigFrom(__DIR__ . '/../config/waas.php', 'waas');

        $this->app->singleton(WaaSClient::class, function ($app) {
            return new WaaSClient(
                config('waas.api_key'),
                config('waas.api_url', 'http://localhost:8080')
            );
        });

        $this->app->singleton(WebhookVerifier::class, function ($app) {
            return new WebhookVerifier(config('waas.signing_secret'));
        });
    }

    public function boot(): void
    {
        $this->publishes([
            __DIR__ . '/../config/waas.php' => config_path('waas.php'),
        ], 'waas-config');
    }
}
