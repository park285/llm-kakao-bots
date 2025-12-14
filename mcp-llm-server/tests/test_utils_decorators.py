"""Tests for decorators."""

import logging

import pytest

from mcp_llm_server.utils.decorators import _truncate, logged_tool


class TestTruncate:
    """Tests for _truncate function."""

    def test_short_string(self) -> None:
        result = _truncate("hello", max_len=100)
        assert result == "hello"

    def test_exact_length(self) -> None:
        text = "a" * 100
        result = _truncate(text, max_len=100)
        assert result == text

    def test_truncated(self) -> None:
        text = "a" * 150
        result = _truncate(text, max_len=100)
        assert result == "a" * 100 + "..."
        assert len(result) == 103

    def test_default_max_len(self) -> None:
        text = "a" * 200
        result = _truncate(text)
        assert result == "a" * 100 + "..."

    def test_non_string(self) -> None:
        result = _truncate(12345, max_len=3)
        assert result == "123..."

    def test_list(self) -> None:
        result = _truncate([1, 2, 3], max_len=5)
        assert result == "[1, 2..."


class TestLoggedToolSync:
    """Tests for logged_tool decorator with sync functions."""

    def test_sync_function_success(self, caplog: pytest.LogCaptureFixture) -> None:
        @logged_tool
        def my_tool(x: int, y: int) -> int:
            return x + y

        with caplog.at_level(logging.INFO):
            result = my_tool(x=1, y=2)

        assert result == 3
        assert "[REQ] my_tool" in caplog.text
        assert "[RES] my_tool" in caplog.text

    def test_sync_function_error(self, caplog: pytest.LogCaptureFixture) -> None:
        @logged_tool
        def failing_tool() -> None:
            raise ValueError("test error")

        with (
            caplog.at_level(logging.ERROR),
            pytest.raises(ValueError, match="test error"),
        ):
            failing_tool()

        assert "[ERR] failing_tool" in caplog.text

    def test_preserves_function_name(self) -> None:
        @logged_tool
        def my_named_tool() -> str:
            return "ok"

        assert my_named_tool.__name__ == "my_named_tool"


class TestLoggedToolAsync:
    """Tests for logged_tool decorator with async functions."""

    @pytest.mark.asyncio
    async def test_async_function_success(
        self, caplog: pytest.LogCaptureFixture
    ) -> None:
        @logged_tool
        async def async_tool(msg: str) -> str:
            return f"echo: {msg}"

        with caplog.at_level(logging.INFO):
            result = await async_tool(msg="hello")

        assert result == "echo: hello"
        assert "[REQ] async_tool" in caplog.text
        assert "[RES] async_tool" in caplog.text

    @pytest.mark.asyncio
    async def test_async_function_error(self, caplog: pytest.LogCaptureFixture) -> None:
        @logged_tool
        async def async_failing() -> None:
            raise RuntimeError("async error")

        with (
            caplog.at_level(logging.ERROR),
            pytest.raises(RuntimeError, match="async error"),
        ):
            await async_failing()

        assert "[ERR] async_failing" in caplog.text

    @pytest.mark.asyncio
    async def test_async_preserves_function_name(self) -> None:
        @logged_tool
        async def my_async_tool() -> str:
            return "ok"

        assert my_async_tool.__name__ == "my_async_tool"
