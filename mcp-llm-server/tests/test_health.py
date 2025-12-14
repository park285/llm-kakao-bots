from unittest.mock import AsyncMock

import pytest

from mcp_llm_server.health import collect_health


class DummyGeminiSettings:
    api_key = "dummy"
    default_model = "gemini-test"
    timeout = 10
    max_retries = 1


class DummySettings:
    gemini = DummyGeminiSettings()


@pytest.mark.asyncio
async def test_collect_health_marks_degraded_on_redis_ping(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(
        "mcp_llm_server.health.get_settings",
        lambda: DummySettings(),
    )
    monkeypatch.setattr(
        "mcp_llm_server.health.get_langgraph_health",
        AsyncMock(
            return_value={
                "redis_enabled": True,
                "redis_connected": True,
                "backend": "redis",
                "session_count": 1,
                "redis_url": "redis://localhost:46379",
                "session_ttl_minutes": 120,
            }
        ),
    )
    ping_mock = AsyncMock(return_value=False)
    monkeypatch.setattr("mcp_llm_server.health.ping_langgraph_backend", ping_mock)

    result = await collect_health(deep_checks=True)

    ping_mock.assert_awaited_once()
    assert result["status"] == "degraded"
    assert result["components"]["langgraph"]["status"] == "degraded"
    assert result["components"]["langgraph"]["detail"]["redis_connected"] is False
    assert result["components"]["langgraph"]["detail"]["deep_checked"] is True


@pytest.mark.asyncio
async def test_collect_health_live_skips_redis_ping(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(
        "mcp_llm_server.health.get_settings",
        lambda: DummySettings(),
    )
    monkeypatch.setattr(
        "mcp_llm_server.health.get_langgraph_health",
        AsyncMock(
            return_value={
                "redis_enabled": True,
                "redis_connected": True,
                "backend": "redis",
                "session_count": 1,
                "redis_url": "redis://localhost:46379",
                "session_ttl_minutes": 120,
            }
        ),
    )
    ping_mock = AsyncMock(return_value=False)
    monkeypatch.setattr("mcp_llm_server.health.ping_langgraph_backend", ping_mock)

    result = await collect_health(deep_checks=False)

    ping_mock.assert_not_awaited()
    assert result["status"] == "ok"
    assert result["components"]["langgraph"]["detail"]["deep_checked"] is False
