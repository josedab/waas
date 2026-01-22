"""Pydantic models for WAAS SDK."""

from datetime import datetime
from enum import Enum
from typing import Any, Optional
from uuid import UUID

from pydantic import BaseModel, Field, HttpUrl


class DeliveryStatus(str, Enum):
    """Webhook delivery status."""

    PENDING = "pending"
    PROCESSING = "processing"
    SUCCESS = "success"
    FAILED = "failed"
    RETRYING = "retrying"


class RetryConfiguration(BaseModel):
    """Retry configuration for webhook endpoints."""

    max_attempts: int = Field(default=5, ge=1, le=10)
    initial_delay_ms: int = Field(default=1000, ge=100)
    max_delay_ms: int = Field(default=300000, ge=1000)
    backoff_multiplier: int = Field(default=2, ge=1, le=5)


class WebhookEndpoint(BaseModel):
    """Webhook endpoint model."""

    id: UUID
    tenant_id: UUID
    url: str
    is_active: bool = True
    retry_config: RetryConfiguration
    custom_headers: dict[str, str] = Field(default_factory=dict)
    created_at: datetime
    updated_at: datetime


class CreateEndpointRequest(BaseModel):
    """Request to create a webhook endpoint."""

    url: HttpUrl
    custom_headers: dict[str, str] = Field(default_factory=dict)
    retry_config: Optional[RetryConfiguration] = None


class UpdateEndpointRequest(BaseModel):
    """Request to update a webhook endpoint."""

    url: Optional[HttpUrl] = None
    is_active: Optional[bool] = None
    custom_headers: Optional[dict[str, str]] = None
    retry_config: Optional[RetryConfiguration] = None


class DeliveryAttempt(BaseModel):
    """Webhook delivery attempt model."""

    id: UUID
    endpoint_id: UUID
    payload_hash: str
    payload_size: int
    status: DeliveryStatus
    http_status: Optional[int] = None
    response_body: Optional[str] = None
    error_message: Optional[str] = None
    attempt_number: int
    scheduled_at: datetime
    delivered_at: Optional[datetime] = None
    created_at: datetime


class SendWebhookRequest(BaseModel):
    """Request to send a webhook."""

    endpoint_id: Optional[UUID] = None
    payload: Any
    headers: dict[str, str] = Field(default_factory=dict)


class SendWebhookResponse(BaseModel):
    """Response from sending a webhook."""

    delivery_id: UUID
    endpoint_id: UUID
    status: str
    scheduled_at: datetime


class BatchSendResponse(BaseModel):
    """Response from batch webhook send."""

    deliveries: list[SendWebhookResponse]
    total: int
    queued: int
    failed: int


class Tenant(BaseModel):
    """Tenant model."""

    id: UUID
    name: str
    subscription_tier: str
    monthly_quota: int
    rate_limit_per_minute: int
    is_active: bool
    created_at: datetime
    updated_at: datetime


class AnalyticsSummary(BaseModel):
    """Analytics summary model."""

    total_deliveries: int
    successful_deliveries: int
    failed_deliveries: int
    success_rate: float
    avg_latency_ms: float
    p95_latency_ms: float
    p99_latency_ms: float


class QuotaUsage(BaseModel):
    """Quota usage model."""

    id: UUID
    tenant_id: UUID
    month: datetime
    request_count: int
    success_count: int
    failure_count: int
    overage_count: int


class TimeSeriesDataPoint(BaseModel):
    """Time series data point."""

    timestamp: datetime
    value: float


class TransformConfig(BaseModel):
    """Transformation configuration."""

    timeout_ms: int = 5000
    max_memory_mb: int = 64
    allow_http: bool = False
    enable_logging: bool = True


class Transformation(BaseModel):
    """Transformation model."""

    id: UUID
    tenant_id: UUID
    name: str
    description: Optional[str] = None
    script: str
    enabled: bool = True
    version: int
    config: TransformConfig
    created_at: datetime
    updated_at: datetime


class CreateTransformationRequest(BaseModel):
    """Request to create a transformation."""

    name: str = Field(min_length=1, max_length=255)
    description: Optional[str] = None
    script: str
    enabled: bool = True
    config: Optional[TransformConfig] = None


class TestTransformationRequest(BaseModel):
    """Request to test a transformation."""

    script: str
    input_payload: Any


class TestTransformationResponse(BaseModel):
    """Response from testing a transformation."""

    success: bool
    output_payload: Optional[Any] = None
    error: Optional[str] = None
    execution_time_ms: int
    logs: list[str] = Field(default_factory=list)


class PaginatedResponse(BaseModel):
    """Paginated response wrapper."""

    data: list[Any]
    total: int
    page: int
    per_page: int
    total_pages: int


class TestWebhookRequest(BaseModel):
    """Request to test a webhook."""

    url: HttpUrl
    payload: Any
    headers: dict[str, str] = Field(default_factory=dict)
    method: str = "POST"
    timeout: int = 30


class TestWebhookResponse(BaseModel):
    """Response from testing a webhook."""

    test_id: UUID
    url: str
    status: str
    http_status: Optional[int] = None
    response_body: Optional[str] = None
    error_message: Optional[str] = None
    latency_ms: Optional[int] = None
    request_id: str
    tested_at: datetime
