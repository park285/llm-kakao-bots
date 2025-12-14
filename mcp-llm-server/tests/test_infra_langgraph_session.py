"""Tests for LangGraph session health."""

from datetime import datetime, timedelta
from types import SimpleNamespace

import pytest

from mcp_llm_server.exceptions import SessionExpiredError
from mcp_llm_server.infra import langgraph_session as lg


@pytest.fixture(autouse=True)
def restore_langgraph_globals() -> None:
    """Restore global state after each test."""
    prev_manager = lg._manager
    prev_saver = lg._redis_saver
    prev_cm = lg._redis_cm
    yield
    lg._manager = prev_manager
    lg._redis_saver = prev_saver
    lg._redis_cm = prev_cm


@pytest.mark.asyncio
async def test_get_langgraph_health_redis_connected(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Redis enabled and connected should report redis backend."""
    dummy_settings = SimpleNamespace(
        redis=SimpleNamespace(url="redis://test:6379", enabled=True),
        session=SimpleNamespace(session_ttl_minutes=30, max_sessions=50),
    )
    monkeypatch.setattr(lg, "get_settings", lambda: dummy_settings)

    lg._manager = SimpleNamespace(get_active_session_count=lambda: 5)
    lg._redis_saver = object()

    health = await lg.get_langgraph_health()

    assert health["backend"] == "redis"
    assert health["redis_connected"] is True
    assert health["session_count"] == 5
    assert health["redis_url"] == "redis://test:6379"
    assert health["session_ttl_minutes"] == 30


@pytest.mark.asyncio
async def test_get_langgraph_health_memory(monkeypatch: pytest.MonkeyPatch) -> None:
    """Redis disabled should report memory backend."""
    dummy_settings = SimpleNamespace(
        redis=SimpleNamespace(url="redis://test:6379", enabled=False),
        session=SimpleNamespace(session_ttl_minutes=10, max_sessions=10),
    )
    monkeypatch.setattr(lg, "get_settings", lambda: dummy_settings)

    lg._manager = SimpleNamespace(get_active_session_count=lambda: 0)
    lg._redis_saver = None

    health = await lg.get_langgraph_health()

    assert health["backend"] == "memory"
    assert health["redis_connected"] is False
    assert health["redis_enabled"] is False
    assert health["session_ttl_minutes"] == 10


@pytest.mark.asyncio
async def test_session_expired_on_access(monkeypatch: pytest.MonkeyPatch) -> None:
    """Expired sessions should be pruned and reported."""
    dummy_settings = SimpleNamespace(
        redis=SimpleNamespace(enabled=False, url=""),
        session=SimpleNamespace(session_ttl_minutes=1, max_sessions=5),
    )
    monkeypatch.setattr(lg, "get_settings", lambda: dummy_settings)

    manager = lg.LangGraphSessionManager(
        checkpointer=lg.MemorySaver(),
        max_sessions=5,
        session_ttl_minutes=1,
    )
    session = await manager.create_session("s1", "gemini-2.5-flash")
    session.last_accessed = datetime.now() - timedelta(minutes=2)

    with pytest.raises(SessionExpiredError):
        await manager.get_session("s1")

    assert manager.get_active_session_count() == 0
