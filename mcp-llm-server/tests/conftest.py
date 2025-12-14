"""Pytest configuration and fixtures"""

import os

import pytest


# test 환경에서 API 호출 방지
os.environ.setdefault("GOOGLE_API_KEY", "test-api-key")
os.environ.setdefault("GEMINI_MODEL", "gemini-2.5-flash")


@pytest.fixture
def mock_settings() -> dict[str, object]:
    """Mock settings for testing"""
    from mcp_llm_server.config.settings import (
        GeminiSettings,
        GuardSettings,
        SessionSettings,
    )

    return {
        "gemini": GeminiSettings(),
        "session": SessionSettings(),
        "guard": GuardSettings(),
    }
