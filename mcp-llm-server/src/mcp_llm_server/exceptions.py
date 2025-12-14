"""Custom exception hierarchy for mcp-llm-server.

Exception 계층 구조:
- MCPLLMError (base)
  - LLMError: LLM 호출 관련
    - LLMTimeoutError: LLM 타임아웃
    - LLMRateLimitError: Rate limit
    - LLMParsingError: 응답 파싱 실패
    - LLMModelError: 모델 관련 에러
  - SessionError: 세션 관련
    - SessionNotFoundError: 세션 없음
    - SessionLimitExceededError: 세션 수 초과
    - SessionExpiredError: 세션 만료
  - GuardError: 인젝션 가드 관련
    - GuardBlockedError: 인젝션 차단
    - GuardConfigError: 설정 에러
  - ValidationError: 입력 검증 에러
    - InvalidInputError: 잘못된 입력
    - MissingFieldError: 필수 필드 누락
"""

from dataclasses import dataclass
from enum import Enum
from typing import Any


class ErrorCode(str, Enum):
    """표준화된 에러 코드."""

    # Generic
    INTERNAL_ERROR = "INTERNAL_ERROR"
    VALIDATION_ERROR = "VALIDATION_ERROR"

    # LLM
    LLM_ERROR = "LLM_ERROR"
    LLM_TIMEOUT = "LLM_TIMEOUT"
    LLM_RATE_LIMIT = "LLM_RATE_LIMIT"
    LLM_PARSING_ERROR = "LLM_PARSING_ERROR"
    LLM_MODEL_ERROR = "LLM_MODEL_ERROR"

    # Session
    SESSION_ERROR = "SESSION_ERROR"
    SESSION_NOT_FOUND = "SESSION_NOT_FOUND"
    SESSION_LIMIT_EXCEEDED = "SESSION_LIMIT_EXCEEDED"
    SESSION_EXPIRED = "SESSION_EXPIRED"

    # Guard
    GUARD_ERROR = "GUARD_ERROR"
    GUARD_BLOCKED = "GUARD_BLOCKED"
    GUARD_CONFIG_ERROR = "GUARD_CONFIG_ERROR"

    # Validation
    INVALID_INPUT = "INVALID_INPUT"
    MISSING_FIELD = "MISSING_FIELD"


@dataclass
class ErrorContext:
    """에러 컨텍스트 정보."""

    request_id: str | None = None
    session_id: str | None = None
    operation: str | None = None
    details: dict[str, Any] | None = None


class MCPLLMError(Exception):
    """Base exception for mcp-llm-server.

    모든 커스텀 예외의 부모 클래스.
    HTTP status code와 error code를 포함.
    """

    def __init__(
        self,
        message: str,
        error_code: ErrorCode = ErrorCode.INTERNAL_ERROR,
        status_code: int = 500,
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.error_code = error_code
        self.status_code = status_code
        self.context = context or ErrorContext()

    def to_dict(self) -> dict[str, Any]:
        """에러 정보를 딕셔너리로 변환."""
        return {
            "error_code": self.error_code.value,
            "error_type": self.__class__.__name__,
            "message": self.message,
            "request_id": self.context.request_id,
            "details": self.context.details,
        }


# =============================================================================
# LLM Errors
# =============================================================================


class LLMError(MCPLLMError):
    """LLM 호출 관련 에러."""

    def __init__(
        self,
        message: str,
        error_code: ErrorCode = ErrorCode.LLM_ERROR,
        status_code: int = 502,
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, error_code, status_code, context)


class LLMTimeoutError(LLMError):
    """LLM 호출 타임아웃."""

    def __init__(
        self,
        message: str = "LLM request timed out",
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, ErrorCode.LLM_TIMEOUT, 504, context)


class LLMRateLimitError(LLMError):
    """LLM API rate limit 초과."""

    def __init__(
        self,
        message: str = "LLM rate limit exceeded",
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, ErrorCode.LLM_RATE_LIMIT, 429, context)


class LLMParsingError(LLMError):
    """LLM 응답 파싱 실패."""

    def __init__(
        self,
        message: str = "Failed to parse LLM response",
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, ErrorCode.LLM_PARSING_ERROR, 502, context)


class LLMModelError(LLMError):
    """LLM 모델 관련 에러."""

    def __init__(
        self,
        message: str = "LLM model error",
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, ErrorCode.LLM_MODEL_ERROR, 502, context)


# =============================================================================
# Session Errors
# =============================================================================


class SessionError(MCPLLMError):
    """세션 관련 에러."""

    def __init__(
        self,
        message: str,
        error_code: ErrorCode = ErrorCode.SESSION_ERROR,
        status_code: int = 400,
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, error_code, status_code, context)


class SessionNotFoundError(SessionError):
    """세션을 찾을 수 없음."""

    def __init__(
        self,
        session_id: str,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.session_id = session_id
        super().__init__(
            f"Session '{session_id}' not found",
            ErrorCode.SESSION_NOT_FOUND,
            404,
            ctx,
        )


class SessionLimitExceededError(SessionError):
    """세션 수 제한 초과."""

    def __init__(
        self,
        max_sessions: int,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.details = {"max_sessions": max_sessions}
        super().__init__(
            f"Maximum sessions ({max_sessions}) reached",
            ErrorCode.SESSION_LIMIT_EXCEEDED,
            429,
            ctx,
        )


class SessionExpiredError(SessionError):
    """세션 만료."""

    def __init__(
        self,
        session_id: str,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.session_id = session_id
        super().__init__(
            f"Session '{session_id}' has expired",
            ErrorCode.SESSION_EXPIRED,
            410,
            ctx,
        )


# =============================================================================
# Guard Errors
# =============================================================================


class GuardError(MCPLLMError):
    """인젝션 가드 관련 에러."""

    def __init__(
        self,
        message: str,
        error_code: ErrorCode = ErrorCode.GUARD_ERROR,
        status_code: int = 400,
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, error_code, status_code, context)


class GuardBlockedError(GuardError):
    """인젝션 공격으로 차단됨."""

    def __init__(
        self,
        score: float,
        threshold: float,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.details = {"score": score, "threshold": threshold}
        super().__init__(
            f"Input blocked by injection guard (score={score:.2f}, threshold={threshold:.2f})",
            ErrorCode.GUARD_BLOCKED,
            400,
            ctx,
        )


class GuardConfigError(GuardError):
    """가드 설정 에러."""

    def __init__(
        self,
        message: str = "Guard configuration error",
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, ErrorCode.GUARD_CONFIG_ERROR, 500, context)


# =============================================================================
# Validation Errors
# =============================================================================


class ValidationError(MCPLLMError):
    """입력 검증 에러."""

    def __init__(
        self,
        message: str,
        error_code: ErrorCode = ErrorCode.VALIDATION_ERROR,
        status_code: int = 400,
        context: ErrorContext | None = None,
    ) -> None:
        super().__init__(message, error_code, status_code, context)


class InvalidInputError(ValidationError):
    """잘못된 입력값."""

    def __init__(
        self,
        field: str,
        reason: str,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.details = {"field": field, "reason": reason}
        super().__init__(
            f"Invalid input for field '{field}': {reason}",
            ErrorCode.INVALID_INPUT,
            400,
            ctx,
        )


class MissingFieldError(ValidationError):
    """필수 필드 누락."""

    def __init__(
        self,
        field: str,
        context: ErrorContext | None = None,
    ) -> None:
        ctx = context or ErrorContext()
        ctx.details = {"field": field}
        super().__init__(
            f"Required field '{field}' is missing",
            ErrorCode.MISSING_FIELD,
            400,
            ctx,
        )
