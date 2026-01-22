<?php

declare(strict_types=1);

namespace WAAS;

use GuzzleHttp\Client as GuzzleClient;
use GuzzleHttp\Exception\GuzzleException;
use GuzzleHttp\Exception\RequestException;
use WAAS\Exceptions\{
    WAASException,
    AuthenticationException,
    NotFoundException,
    RateLimitException,
    ValidationException
};
use WAAS\Services\{Endpoints, Deliveries, Webhooks};

class Client
{
    private GuzzleClient $httpClient;
    private Config $config;

    public function __construct(string|Config $configOrApiKey)
    {
        if (is_string($configOrApiKey)) {
            $this->config = new Config(apiKey: $configOrApiKey);
        } else {
            $this->config = $configOrApiKey;
        }

        $this->httpClient = new GuzzleClient([
            'base_uri' => $this->config->baseUrl,
            'timeout' => $this->config->timeout,
            'connect_timeout' => $this->config->connectTimeout,
            'headers' => [
                'X-API-Key' => $this->config->apiKey,
                'Content-Type' => 'application/json',
                'Accept' => 'application/json',
                'User-Agent' => 'waas-sdk-php/1.0.0',
            ],
        ]);
    }

    public function endpoints(): Endpoints
    {
        return new Endpoints($this);
    }

    public function deliveries(): Deliveries
    {
        return new Deliveries($this);
    }

    public function webhooks(): Webhooks
    {
        return new Webhooks($this);
    }

    /**
     * @return array<string, mixed>
     */
    public function get(string $path): array
    {
        return $this->request('GET', $path);
    }

    /**
     * @param array<string, mixed> $body
     * @return array<string, mixed>
     */
    public function post(string $path, array $body = []): array
    {
        return $this->request('POST', $path, $body);
    }

    /**
     * @param array<string, mixed> $body
     * @return array<string, mixed>
     */
    public function patch(string $path, array $body = []): array
    {
        return $this->request('PATCH', $path, $body);
    }

    public function delete(string $path): void
    {
        $this->request('DELETE', $path);
    }

    /**
     * @param array<string, mixed>|null $body
     * @return array<string, mixed>
     */
    private function request(string $method, string $path, ?array $body = null): array
    {
        $options = [];
        if ($body !== null) {
            $options['json'] = $body;
        }

        try {
            $response = $this->httpClient->request($method, $path, $options);
            $content = $response->getBody()->getContents();
            
            if (empty($content)) {
                return [];
            }

            return json_decode($content, true, 512, JSON_THROW_ON_ERROR);
        } catch (RequestException $e) {
            $this->handleRequestException($e);
        } catch (GuzzleException $e) {
            throw new WAASException('Request failed: ' . $e->getMessage(), 0, $e);
        }
    }

    private function handleRequestException(RequestException $e): never
    {
        $response = $e->getResponse();
        $statusCode = $response?->getStatusCode() ?? 0;
        $body = [];

        if ($response) {
            $content = $response->getBody()->getContents();
            if (!empty($content)) {
                try {
                    $body = json_decode($content, true, 512, JSON_THROW_ON_ERROR);
                } catch (\JsonException) {
                    // Ignore JSON parse errors
                }
            }
        }

        $message = $body['message'] ?? $e->getMessage();
        $code = $body['code'] ?? 'UNKNOWN_ERROR';

        match ($statusCode) {
            401 => throw new AuthenticationException($message),
            404 => throw new NotFoundException($message),
            422 => throw new ValidationException($message, $body['details'] ?? null),
            429 => throw new RateLimitException(
                $message,
                isset($response) ? (int) $response->getHeaderLine('Retry-After') ?: null : null
            ),
            default => throw new WAASException($message, $statusCode),
        };
    }
}
