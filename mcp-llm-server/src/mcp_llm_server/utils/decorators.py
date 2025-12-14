"""Decorators for MCP tools."""

from __future__ import annotations

import functools
import inspect
import logging
import time
from typing import TYPE_CHECKING, ParamSpec, TypeVar, cast


if TYPE_CHECKING:
    from collections.abc import Awaitable, Callable


log = logging.getLogger(__name__)

P = ParamSpec("P")
R = TypeVar("R")


def _truncate(value: object, max_len: int = 100) -> str:
    """값을 로그용으로 축약"""
    s = str(value)
    return s[:max_len] + "..." if len(s) > max_len else s


def logged_tool(
    func: Callable[P, Awaitable[R] | R],
) -> Callable[P, Awaitable[R] | R]:
    """MCP tool 호출을 로깅하는 데코레이터"""

    @functools.wraps(func)
    async def async_wrapper(*args: P.args, **kwargs: P.kwargs) -> R:
        tool_name = func.__name__
        log.info("[REQ] %s args=%s", tool_name, _truncate(kwargs))
        start = time.perf_counter()
        try:
            result = await cast("Awaitable[R]", func(*args, **kwargs))
            elapsed = (time.perf_counter() - start) * 1000
            log.info("[RES] %s -> %s (%.1fms)", tool_name, _truncate(result), elapsed)
            return result
        except Exception as e:
            elapsed = (time.perf_counter() - start) * 1000
            log.error("[ERR] %s -> %s (%.1fms)", tool_name, e, elapsed)
            raise

    @functools.wraps(func)
    def sync_wrapper(*args: P.args, **kwargs: P.kwargs) -> Awaitable[R] | R:
        tool_name = func.__name__
        log.info("[REQ] %s args=%s", tool_name, _truncate(kwargs))
        start = time.perf_counter()
        try:
            result = func(*args, **kwargs)
            elapsed = (time.perf_counter() - start) * 1000
            log.info("[RES] %s -> %s (%.1fms)", tool_name, _truncate(result), elapsed)
            return result
        except Exception as e:
            elapsed = (time.perf_counter() - start) * 1000
            log.error("[ERR] %s -> %s (%.1fms)", tool_name, e, elapsed)
            raise

    if inspect.iscoroutinefunction(func):
        return cast("Callable[P, Awaitable[R]]", async_wrapper)
    return cast("Callable[P, R]", sync_wrapper)
