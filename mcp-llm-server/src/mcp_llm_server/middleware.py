"""HTTP middleware for request tracking and error handling.

기능:
- X-Request-ID 헤더 추적 및 자동 생성
- request_id를 contextvars로 전파
- 응답 헤더에 request_id 포함
"""

import uuid
from collections.abc import Callable
from contextvars import ContextVar
from typing import Any

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response

from mcp_llm_server.config.logging import get_logger


log = get_logger(__name__)

# Context variable for request_id (thread-safe)
_request_id_ctx: ContextVar[str | None] = ContextVar("request_id", default=None)


def get_request_id() -> str | None:
    """현재 요청의 request_id 반환.

    Exception에서 context 정보 추가 시 사용.
    """
    return _request_id_ctx.get()


def set_request_id(request_id: str) -> None:
    """request_id 설정 (테스트용)."""
    _request_id_ctx.set(request_id)


class RequestIdMiddleware(BaseHTTPMiddleware):
    """X-Request-ID 추적 미들웨어.

    1. X-Request-ID 헤더가 있으면 사용
    2. 없으면 UUID 자동 생성
    3. contextvars에 저장하여 전파
    4. 응답 헤더에 X-Request-ID 포함
    """

    async def dispatch(
        self,
        request: Request,
        call_next: Callable[[Request], Any],
    ) -> Response:
        # X-Request-ID 추출 또는 생성
        request_id = request.headers.get("X-Request-ID")
        if not request_id:
            request_id = str(uuid.uuid4())

        # contextvars에 저장
        token = _request_id_ctx.set(request_id)

        # 로그에 request_id 포함
        log.debug(
            "REQUEST_START method={} path={} request_id={}",
            request.method,
            request.url.path,
            request_id,
        )

        try:
            response: Response = await call_next(request)

            # 응답 헤더에 request_id 추가
            response.headers["X-Request-ID"] = request_id

            log.debug(
                "REQUEST_END method={} path={} status={} request_id={}",
                request.method,
                request.url.path,
                response.status_code,
                request_id,
            )

            return response

        finally:
            # context 복원
            _request_id_ctx.reset(token)
