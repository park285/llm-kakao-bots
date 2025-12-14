"""Usage repository for PostgreSQL token usage tracking.

봇의 token_usage 테이블을 공유하여 일별 사용량을 저장/조회합니다.
"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass
from datetime import date
from typing import TYPE_CHECKING, Protocol, TypedDict, cast

from mcp_llm_server.config.logging import get_logger
from mcp_llm_server.config.settings import get_settings


if TYPE_CHECKING:
    from collections.abc import Mapping


log = get_logger(__name__)


class AsyncpgPool(Protocol):
    async def execute(self, query: str, *args: object) -> str: ...
    async def fetchrow(
        self, query: str, *args: object
    ) -> Mapping[str, object] | None: ...
    async def fetch(self, query: str, *args: object) -> list[Mapping[str, object]]: ...
    async def close(self) -> None: ...


class TokenUsageRow(TypedDict):
    usage_date: date
    input_tokens: int
    output_tokens: int
    reasoning_tokens: int
    request_count: int


@dataclass
class DailyUsage:
    """일별 토큰 사용량"""

    usage_date: date
    input_tokens: int
    output_tokens: int
    reasoning_tokens: int
    request_count: int

    @property
    def total_tokens(self) -> int:
        return self.input_tokens + self.output_tokens


class UsageRepository:
    """PostgreSQL 기반 토큰 사용량 저장소

    봇의 token_usage 테이블 구조:
    - id: SERIAL PK
    - usage_date: DATE (일별 유니크)
    - input_tokens: BIGINT
    - output_tokens: BIGINT
    - reasoning_tokens: BIGINT
    - request_count: BIGINT
    - version: BIGINT (optimistic locking)
    """

    def __init__(self) -> None:
        self._pool: AsyncpgPool | None = None
        self._settings = get_settings().database
        self._lock = asyncio.Lock()

    def _row_to_daily_usage(
        self, row: TokenUsageRow, *, default_date: date | None = None
    ) -> DailyUsage:
        usage_date = row.get("usage_date", default_date or date.today())
        return DailyUsage(
            usage_date=usage_date,
            input_tokens=int(row["input_tokens"]),
            output_tokens=int(row["output_tokens"]),
            reasoning_tokens=int(row["reasoning_tokens"]),
            request_count=int(row["request_count"]),
        )

    async def _get_pool(self) -> AsyncpgPool:
        """Get or create connection pool"""
        if self._pool is None:
            async with self._lock:
                if self._pool is None:
                    import asyncpg

                    pool = await asyncpg.create_pool(
                        host=self._settings.host,
                        port=self._settings.port,
                        database=self._settings.database,
                        user=self._settings.user,
                        password=self._settings.password,
                        min_size=self._settings.min_pool_size,
                        max_size=self._settings.max_pool_size,
                    )
                    self._pool = cast("AsyncpgPool", pool)
                    log.info(
                        "DB pool created: {}:{}/{}",
                        self._settings.host,
                        self._settings.port,
                        self._settings.database,
                    )
        return self._pool

    async def close(self) -> None:
        """Close connection pool"""
        if self._pool:
            await self._pool.close()
            self._pool = None
            log.info("DB pool closed")

    async def record_usage(
        self,
        input_tokens: int,
        output_tokens: int,
        reasoning_tokens: int = 0,
    ) -> None:
        """Record token usage for today (upsert with optimistic locking)

        Args:
            input_tokens: 입력 토큰 수
            output_tokens: 출력 토큰 수
            reasoning_tokens: reasoning 토큰 수 (thinking mode)
        """
        if input_tokens <= 0 and output_tokens <= 0:
            return

        pool = await self._get_pool()
        today = date.today()

        # Upsert: INSERT or UPDATE with optimistic locking
        # PostgreSQL ON CONFLICT로 atomic upsert 구현
        query = """
            INSERT INTO token_usage (
                usage_date,
                input_tokens,
                output_tokens,
                reasoning_tokens,
                request_count,
                version
            )
            VALUES ($1, $2, $3, $4, 1, 0)
            ON CONFLICT (usage_date)
            DO UPDATE SET
                input_tokens = token_usage.input_tokens + EXCLUDED.input_tokens,
                output_tokens = token_usage.output_tokens + EXCLUDED.output_tokens,
                reasoning_tokens = token_usage.reasoning_tokens + EXCLUDED.reasoning_tokens,
                request_count = token_usage.request_count + 1,
                version = token_usage.version + 1
        """

        try:
            await pool.execute(
                query, today, input_tokens, output_tokens, reasoning_tokens
            )
            log.debug(
                "USAGE_RECORDED date={}, in={}, out={}, reasoning={}",
                today,
                input_tokens,
                output_tokens,
                reasoning_tokens,
            )
        except Exception as e:
            log.error("USAGE_RECORD_FAILED: {}", e)
            raise

    async def get_daily_usage(
        self, target_date: date | None = None
    ) -> DailyUsage | None:
        """Get usage for a specific date (default: today)"""
        pool = await self._get_pool()
        target = target_date or date.today()

        query = """
            SELECT usage_date, input_tokens, output_tokens, reasoning_tokens, request_count
            FROM token_usage
            WHERE usage_date = $1
        """

        row = await pool.fetchrow(query, target)
        if row:
            usage_row = cast("TokenUsageRow", row)
            return self._row_to_daily_usage(usage_row)
        return None

    async def get_usage_range(
        self, start_date: date, end_date: date
    ) -> list[DailyUsage]:
        """Get usage for a date range"""
        pool = await self._get_pool()

        query = """
            SELECT usage_date, input_tokens, output_tokens, reasoning_tokens, request_count
            FROM token_usage
            WHERE usage_date >= $1 AND usage_date <= $2
            ORDER BY usage_date DESC
        """

        rows = await pool.fetch(query, start_date, end_date)
        usage_rows = cast("list[TokenUsageRow]", rows)
        return [self._row_to_daily_usage(row) for row in usage_rows]

    async def get_recent_usage(self, days: int = 7) -> list[DailyUsage]:
        """Get usage for recent N days"""
        pool = await self._get_pool()

        query = """
            SELECT usage_date, input_tokens, output_tokens, reasoning_tokens, request_count
            FROM token_usage
            ORDER BY usage_date DESC
            LIMIT $1
        """

        rows = await pool.fetch(query, days)
        usage_rows = cast("list[TokenUsageRow]", rows)
        return [self._row_to_daily_usage(row) for row in usage_rows]

    async def get_total_usage(self, days: int = 30) -> DailyUsage:
        """Get aggregated total usage for recent N days"""
        pool = await self._get_pool()

        query = """
            SELECT
                COALESCE(SUM(input_tokens), 0) as input_tokens,
                COALESCE(SUM(output_tokens), 0) as output_tokens,
                COALESCE(SUM(reasoning_tokens), 0) as reasoning_tokens,
                COALESCE(SUM(request_count), 0) as request_count
            FROM token_usage
            WHERE usage_date >= CURRENT_DATE - $1::int
        """

        row = await pool.fetchrow(query, days)
        usage_row = cast("TokenUsageRow | None", row) if row else None
        if not usage_row:
            return DailyUsage(
                usage_date=date.today(),
                input_tokens=0,
                output_tokens=0,
                reasoning_tokens=0,
                request_count=0,
            )
        return self._row_to_daily_usage(usage_row, default_date=date.today())


# Singleton
_repository: UsageRepository | None = None


def get_usage_repository() -> UsageRepository:
    """Get singleton UsageRepository instance"""
    global _repository
    if _repository is None:
        _repository = UsageRepository()
    return _repository
