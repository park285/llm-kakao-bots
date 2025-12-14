"""Health and model config routes."""

from fastapi import APIRouter
from pydantic import BaseModel

from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.health import collect_health


router = APIRouter(tags=["Health"])


@router.get("/health")
async def health() -> dict[str, object]:
    """Health check with deep dependency checks."""
    return await collect_health(deep_checks=True)


@router.get("/health/ready")
async def health_ready() -> dict[str, object]:
    """Readiness probe (deep checks)."""
    return await collect_health(deep_checks=True)


@router.get("/health/live")
async def health_live() -> dict[str, object]:
    """Liveness probe (shallow checks)."""
    return await collect_health(deep_checks=False)


class ModelConfigResponse(BaseModel):
    """Model configuration snapshot for debugging."""

    model_default: str
    model_hints: str | None
    model_answer: str | None
    model_verify: str | None
    temperature: float
    timeout_seconds: int
    max_retries: int
    http2_enabled: bool


@router.get("/health/models", response_model=ModelConfigResponse)
async def health_models() -> ModelConfigResponse:
    """Return Gemini model configuration snapshot."""
    settings = get_settings().gemini
    http = get_settings().http
    return ModelConfigResponse(
        model_default=settings.default_model,
        model_hints=settings.hints_model or settings.default_model,
        model_answer=settings.answer_model or settings.default_model,
        model_verify=settings.verify_model or settings.default_model,
        temperature=settings.temperature,
        timeout_seconds=settings.timeout,
        max_retries=settings.max_retries,
        http2_enabled=http.http2_enabled,
    )
