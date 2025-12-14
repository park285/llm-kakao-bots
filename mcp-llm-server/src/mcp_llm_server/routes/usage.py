"""Usage-related routes."""

from datetime import date

from fastapi import APIRouter
from pydantic import BaseModel

from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.infra.usage_repository import DailyUsage, get_usage_repository


router = APIRouter(prefix="/api/usage", tags=["Usage"])


class UsageResponse(BaseModel):
    """Token usage response"""

    input_tokens: int
    output_tokens: int
    total_tokens: int
    reasoning_tokens: int | None
    model: str | None = None


class DailyUsageResponse(BaseModel):
    """Daily usage response"""

    usage_date: str
    input_tokens: int
    output_tokens: int
    total_tokens: int
    reasoning_tokens: int
    request_count: int
    model: str | None = None


class UsageListResponse(BaseModel):
    """List of daily usages"""

    usages: list[DailyUsageResponse]
    total_input_tokens: int
    total_output_tokens: int
    total_tokens: int
    total_request_count: int
    model: str | None = None


def _resolve_usage_model(model: str | None = None) -> str:
    return model or get_settings().gemini.default_model


def _build_daily_usage_response(
    usage: DailyUsage | None, model: str | None = None
) -> DailyUsageResponse:
    resolved_model = _resolve_usage_model(model)
    if usage is None:
        return DailyUsageResponse(
            usage_date=str(date.today()),
            input_tokens=0,
            output_tokens=0,
            total_tokens=0,
            reasoning_tokens=0,
            request_count=0,
            model=resolved_model,
        )
    return DailyUsageResponse(
        usage_date=usage.usage_date.isoformat(),
        input_tokens=usage.input_tokens,
        output_tokens=usage.output_tokens,
        total_tokens=usage.total_tokens,
        reasoning_tokens=usage.reasoning_tokens,
        request_count=usage.request_count,
        model=resolved_model,
    )


def _build_usage_list_response(
    usages: list[DailyUsage], model: str | None = None
) -> UsageListResponse:
    resolved_model = _resolve_usage_model(model)
    usage_list = [
        _build_daily_usage_response(usage, resolved_model) for usage in usages
    ]
    return UsageListResponse(
        usages=usage_list,
        total_input_tokens=sum(u.input_tokens for u in usages),
        total_output_tokens=sum(u.output_tokens for u in usages),
        total_tokens=sum(u.total_tokens for u in usages),
        total_request_count=sum(u.request_count for u in usages),
        model=resolved_model,
    )


@router.get("/daily", response_model=DailyUsageResponse)
async def api_usage_daily() -> DailyUsageResponse:
    """Get today's token usage from DB"""
    repository = get_usage_repository()
    usage = await repository.get_daily_usage()

    model = get_settings().gemini.default_model
    return _build_daily_usage_response(usage, model=model)


@router.get("/recent", response_model=UsageListResponse)
async def api_usage_recent(days: int = 7) -> UsageListResponse:
    """Get recent N days token usage from DB"""
    repository = get_usage_repository()
    usages = await repository.get_recent_usage(days=days)

    model = get_settings().gemini.default_model
    return _build_usage_list_response(usages, model=model)


@router.get("/total", response_model=UsageResponse)
async def api_usage_total(days: int = 30) -> UsageResponse:
    """Get aggregated token usage for N days from DB"""
    repository = get_usage_repository()
    usage = await repository.get_total_usage(days=days)

    return UsageResponse(
        input_tokens=usage.input_tokens,
        output_tokens=usage.output_tokens,
        total_tokens=usage.total_tokens,
        reasoning_tokens=usage.reasoning_tokens,
    )
