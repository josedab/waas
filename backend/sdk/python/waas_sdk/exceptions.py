"""WAAS SDK exceptions."""

from typing import Any, Optional


class WAASError(Exception):
    """Base exception for WAAS SDK."""

    def __init__(self, message: str) -> None:
        self.message = message
        super().__init__(message)


class WAASAPIError(WAASError):
    """API error from WAAS service."""

    def __init__(
        self,
        message: str,
        status_code: int,
        code: Optional[str] = None,
        details: Optional[dict[str, Any]] = None,
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.details = details or {}

    def __str__(self) -> str:
        return f"WAASAPIError({self.status_code}): {self.code} - {self.message}"


class WAASAuthenticationError(WAASAPIError):
    """Authentication failed."""

    def __init__(self, message: str = "Authentication failed") -> None:
        super().__init__(message, status_code=401, code="AUTHENTICATION_FAILED")


class WAASRateLimitError(WAASAPIError):
    """Rate limit exceeded."""

    def __init__(
        self,
        message: str = "Rate limit exceeded",
        retry_after: Optional[int] = None,
    ) -> None:
        super().__init__(message, status_code=429, code="RATE_LIMIT_EXCEEDED")
        self.retry_after = retry_after


class WAASValidationError(WAASAPIError):
    """Validation error."""

    def __init__(
        self,
        message: str,
        details: Optional[dict[str, Any]] = None,
    ) -> None:
        super().__init__(message, status_code=400, code="VALIDATION_ERROR", details=details)


class WAASNotFoundError(WAASAPIError):
    """Resource not found."""

    def __init__(self, message: str = "Resource not found") -> None:
        super().__init__(message, status_code=404, code="NOT_FOUND")


class WAASConnectionError(WAASError):
    """Connection error."""

    pass


class WAASTimeoutError(WAASError):
    """Request timeout."""

    pass
