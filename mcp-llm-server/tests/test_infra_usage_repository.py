"""Tests for UsageRepository without real DB."""

from __future__ import annotations

from datetime import date
from types import SimpleNamespace
from typing import Any

import pytest

from mcp_llm_server.infra.usage_repository import (
    DailyUsage,
    TokenUsageRow,
    UsageRepository,
)


class FakePool:
    """Minimal asyncpg-like pool for tests."""

    def __init__(self) -> None:
        self.fetch_calls: list[tuple[str, tuple[Any, ...]]] = []
        self.fetchrow_calls: list[tuple[str, tuple[Any, ...]]] = []
        self.results = SimpleNamespace(fetch=[], fetchrow=None)

    async def fetch(self, query: str, *args: object) -> list[TokenUsageRow]:
        self.fetch_calls.append((query, args))
        return self.results.fetch

    async def fetchrow(self, query: str, *args: object) -> TokenUsageRow | None:
        self.fetchrow_calls.append((query, args))
        return self.results.fetchrow

    async def execute(self, query: str, *args: object) -> str:  # pragma: no cover
        raise NotImplementedError

    async def close(self) -> None:  # pragma: no cover
        return None


@pytest.fixture()
def repo(monkeypatch: pytest.MonkeyPatch) -> tuple[UsageRepository, FakePool]:
    repository = UsageRepository()
    fake_pool = FakePool()

    async def _get_pool() -> FakePool:
        return fake_pool

    monkeypatch.setattr(repository, "_get_pool", _get_pool)
    return repository, fake_pool


def test_row_to_daily_usage_defaults_today(
    repo: tuple[UsageRepository, FakePool],
) -> None:
    repository, _ = repo
    today = date(2025, 1, 1)
    row: TokenUsageRow = {
        "usage_date": today,
        "input_tokens": 1,
        "output_tokens": 2,
        "reasoning_tokens": 3,
        "request_count": 4,
    }

    result = repository._row_to_daily_usage(row)  # type: ignore[attr-defined]

    assert isinstance(result, DailyUsage)
    assert result.usage_date == today
    assert result.input_tokens == 1
    assert result.output_tokens == 2
    assert result.total_tokens == 3
    assert result.reasoning_tokens == 3
    assert result.request_count == 4


@pytest.mark.asyncio()
async def test_get_total_usage_when_empty(
    repo: tuple[UsageRepository, FakePool],
) -> None:
    repository, fake_pool = repo
    fake_pool.results.fetchrow = None

    result = await repository.get_total_usage(days=7)

    assert result.usage_date == date.today()
    assert result.input_tokens == 0
    assert result.output_tokens == 0
    assert result.reasoning_tokens == 0
    assert result.request_count == 0


@pytest.mark.asyncio()
async def test_get_recent_usage_maps_rows(
    repo: tuple[UsageRepository, FakePool],
) -> None:
    repository, fake_pool = repo
    fake_pool.results.fetch = [
        {
            "usage_date": date(2025, 1, 2),
            "input_tokens": 10,
            "output_tokens": 20,
            "reasoning_tokens": 30,
            "request_count": 40,
        },
        {
            "usage_date": date(2025, 1, 1),
            "input_tokens": 1,
            "output_tokens": 2,
            "reasoning_tokens": 3,
            "request_count": 4,
        },
    ]

    usages = await repository.get_recent_usage(days=2)

    assert len(usages) == 2
    assert usages[0].usage_date == date(2025, 1, 2)
    assert usages[0].total_tokens == 30
    assert usages[0].request_count == 40
    assert usages[1].usage_date == date(2025, 1, 1)
    assert usages[1].total_tokens == 3
    assert usages[1].request_count == 4
