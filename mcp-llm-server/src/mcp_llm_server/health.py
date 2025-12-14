"""Health check utilities for MCP LLM Server."""

from __future__ import annotations

import time
from dataclasses import dataclass
from typing import Any

from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.infra.langgraph_session import (
    get_langgraph_health,
    ping_langgraph_backend,
)


START_TIME_MONOTONIC = time.monotonic()


type ComponentDetail = dict[str, Any]
type ComponentPayload = dict[str, Any]
type HealthPayload = dict[str, Any]


@dataclass
class ComponentStatus:
    """Component status with details."""

    status: str
    detail: ComponentDetail

    def as_payload(self) -> ComponentPayload:
        """Convert to JSON-serializable payload."""
        return {"status": self.status, "detail": self.detail}


async def _build_langgraph_status(deep_checks: bool) -> ComponentStatus:
    """Collect LangGraph/Redis component health."""
    langgraph_detail = await get_langgraph_health()
    redis_enabled = bool(langgraph_detail.get("redis_enabled"))
    redis_connected = bool(langgraph_detail.get("redis_connected"))

    status = "ok"
    if redis_enabled and not redis_connected:
        status = "degraded"

    # Optional deep ping when Redis is enabled
    if deep_checks and redis_enabled and redis_connected:
        redis_reachable = await ping_langgraph_backend()
        if not redis_reachable:
            status = "degraded"
            langgraph_detail["redis_connected"] = False

    return ComponentStatus(status=status, detail=langgraph_detail)


def _build_app_status() -> ComponentStatus:
    """Collect application-level status (uptime only)."""
    uptime_seconds = int(time.monotonic() - START_TIME_MONOTONIC)
    return ComponentStatus(
        status="ok",
        detail={"uptime_seconds": uptime_seconds},
    )


def _build_gemini_status() -> ComponentStatus:
    """Collect Gemini API configuration health (non-invasive)."""
    settings = get_settings().gemini
    api_key_present = bool(settings.api_key)
    status = "ok" if api_key_present else "degraded"
    detail = {
        "api_key_present": api_key_present,
        "default_model": settings.default_model,
        "timeout_seconds": settings.timeout,
        "max_retries": settings.max_retries,
    }
    return ComponentStatus(status=status, detail=detail)


async def collect_health(deep_checks: bool = True) -> HealthPayload:
    """Collect full health payload.

    Args:
        deep_checks: Whether to perform external dependency checks (Redis ping).
    """
    components: dict[str, ComponentStatus] = {}

    app_status = _build_app_status()
    components["app"] = app_status

    langgraph_status = await _build_langgraph_status(deep_checks)
    components["langgraph"] = ComponentStatus(
        status=langgraph_status.status,
        detail={**langgraph_status.detail, "deep_checked": deep_checks},
    )

    gemini_status = _build_gemini_status()
    components["gemini"] = gemini_status

    overall_status = "ok"
    for component in components.values():
        if component.status != "ok":
            overall_status = "degraded"
            break

    return {
        "status": overall_status,
        "components": {key: value.as_payload() for key, value in components.items()},
        # Legacy compatibility: keep langgraph detail at top-level
        "langgraph": components["langgraph"].detail,
    }
