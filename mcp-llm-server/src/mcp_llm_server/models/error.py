"""Error response models for HTTP API.

표준화된 에러 응답 포맷을 정의.
봇단에서 파싱하기 쉬운 구조.
"""

from typing import Any

from pydantic import BaseModel, Field


class ErrorResponse(BaseModel):
    """표준 에러 응답 모델.

    모든 에러 응답은 이 포맷을 따름.
    봇에서 error_code로 에러 타입을 구분 가능.
    """

    error_code: str = Field(
        ..., description="에러 코드 (예: LLM_TIMEOUT, SESSION_NOT_FOUND)"
    )
    error_type: str = Field(..., description="Exception 클래스명")
    message: str = Field(..., description="사람이 읽을 수 있는 에러 메시지")
    request_id: str | None = Field(None, description="요청 추적용 ID (X-Request-ID)")
    details: dict[str, Any] | None = Field(None, description="추가 에러 상세 정보")

    model_config = {
        "json_schema_extra": {
            "example": {
                "error_code": "SESSION_NOT_FOUND",
                "error_type": "SessionNotFoundError",
                "message": "Session 'abc-123' not found",
                "request_id": "req-xyz-456",
                "details": None,
            }
        }
    }


class ErrorDetail(BaseModel):
    """Validation 에러 상세 정보."""

    field: str = Field(..., description="에러 발생 필드")
    message: str = Field(..., description="에러 메시지")
    value: Any | None = Field(None, description="입력값 (선택적)")


class ValidationErrorResponse(ErrorResponse):
    """입력 검증 에러 응답.

    여러 필드의 검증 에러를 한번에 반환.
    """

    errors: list[ErrorDetail] = Field(
        default_factory=list, description="필드별 에러 목록"
    )

    model_config = {
        "json_schema_extra": {
            "example": {
                "error_code": "VALIDATION_ERROR",
                "error_type": "ValidationError",
                "message": "Input validation failed",
                "request_id": "req-xyz-456",
                "details": None,
                "errors": [
                    {"field": "target", "message": "Field required", "value": None},
                    {
                        "field": "category",
                        "message": "must be one of: 사물, 음식, 동물",
                        "value": "invalid",
                    },
                ],
            }
        }
    }
