"""Tests for configuration settings."""

import os
from unittest.mock import patch

from mcp_llm_server.config.settings import (
    GEMINI_3_FIXED_TEMPERATURE,
    GeminiModel,
    GeminiSettings,
    GuardSettings,
    HttpSettings,
    LoggingSettings,
    RedisSettings,
    SessionSettings,
    Settings,
    ThinkingConfig,
    ThinkingLevel,
    _get_int_or_none,
    _parse_api_keys,
    get_settings,
    is_gemini_3,
    is_gemini_25_preview,
)


class TestGeminiModel:
    """Tests for GeminiModel enum."""

    def test_values(self) -> None:
        assert GeminiModel.GEMINI_25_FLASH.value == "gemini-2.5-flash"
        assert GeminiModel.GEMINI_3_PRO.value == "gemini-3-pro-preview"

    def test_from_string_known(self) -> None:
        model = GeminiModel.from_string("gemini-2.5-flash")
        assert model == GeminiModel.GEMINI_25_FLASH

    def test_from_string_unknown(self) -> None:
        # 알 수 없는 모델은 문자열 그대로 반환
        model = GeminiModel.from_string("unknown-model")
        assert model == "unknown-model"


class TestThinkingLevel:
    """Tests for ThinkingLevel enum."""

    def test_values(self) -> None:
        assert ThinkingLevel.NONE.value == "none"
        assert ThinkingLevel.LOW.value == "low"
        assert ThinkingLevel.MEDIUM.value == "medium"
        assert ThinkingLevel.HIGH.value == "high"


class TestIsGemini3:
    """Tests for is_gemini_3 function."""

    def test_gemini_3_pro(self) -> None:
        assert is_gemini_3("gemini-3-pro-preview")

    def test_gemini_3_case_insensitive(self) -> None:
        assert is_gemini_3("GEMINI-3-PRO")

    def test_gemini_25(self) -> None:
        assert not is_gemini_3("gemini-2.5-flash")

    def test_other_model(self) -> None:
        assert not is_gemini_3("gpt-4")


class TestIsGemini25Preview:
    """Tests for is_gemini_25_preview function."""

    def test_25_preview(self) -> None:
        assert is_gemini_25_preview("gemini-2.5-flash-preview-09-2025")

    def test_25_non_preview(self) -> None:
        assert not is_gemini_25_preview("gemini-2.5-flash")

    def test_3_preview(self) -> None:
        # 3.0은 2.5가 아니므로 False
        assert not is_gemini_25_preview("gemini-3-pro-preview")


class TestParseApiKeys:
    """Tests for _parse_api_keys function."""

    def test_single_key(self) -> None:
        with patch.dict(
            os.environ, {"GOOGLE_API_KEY": "key1", "GOOGLE_API_KEYS": ""}, clear=False
        ):
            keys = _parse_api_keys()
            assert keys == ["key1"]

    def test_multiple_keys_comma(self) -> None:
        with patch.dict(
            os.environ,
            {"GOOGLE_API_KEYS": "key1,key2,key3", "GOOGLE_API_KEY": ""},
            clear=False,
        ):
            keys = _parse_api_keys()
            assert keys == ["key1", "key2", "key3"]

    def test_multiple_keys_space(self) -> None:
        with patch.dict(
            os.environ,
            {"GOOGLE_API_KEYS": "key1 key2 key3", "GOOGLE_API_KEY": ""},
            clear=False,
        ):
            keys = _parse_api_keys()
            assert keys == ["key1", "key2", "key3"]

    def test_no_keys(self) -> None:
        with patch.dict(
            os.environ, {"GOOGLE_API_KEY": "", "GOOGLE_API_KEYS": ""}, clear=False
        ):
            keys = _parse_api_keys()
            assert keys == []


class TestGetIntOrNone:
    """Tests for _get_int_or_none function."""

    def test_positive(self) -> None:
        with patch.dict(os.environ, {"TEST_VAR": "100"}):
            result = _get_int_or_none("TEST_VAR")
            assert result == 100

    def test_zero_returns_none(self) -> None:
        with patch.dict(os.environ, {"TEST_VAR": "0"}):
            result = _get_int_or_none("TEST_VAR")
            assert result is None

    def test_missing_uses_default(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            result = _get_int_or_none("MISSING_VAR", default=50)
            assert result == 50

    def test_default_zero_returns_none(self) -> None:
        with patch.dict(os.environ, {}, clear=True):
            result = _get_int_or_none("MISSING_VAR", default=0)
            assert result is None


class TestThinkingConfig:
    """Tests for ThinkingConfig dataclass."""

    def test_get_level_default(self) -> None:
        config = ThinkingConfig(
            level_default="medium",
            level_hints="high",
            level_answer="low",
            level_verify="none",
        )
        assert config.get_level() == "medium"
        assert config.get_level(None) == "medium"

    def test_get_level_specific(self) -> None:
        config = ThinkingConfig(
            level_default="medium",
            level_hints="high",
            level_answer="low",
            level_verify="none",
        )
        assert config.get_level("hints") == "high"
        assert config.get_level("answer") == "low"
        assert config.get_level("verify") == "none"

    def test_get_level_unknown_task(self) -> None:
        config = ThinkingConfig(level_default="medium")
        assert config.get_level("unknown") == "medium"

    def test_get_budget_default(self) -> None:
        config = ThinkingConfig(
            budget_default=1000,
            budget_hints=8192,
            budget_answer=4096,
            budget_verify=2048,
        )
        assert config.get_budget() == 1000
        assert config.get_budget(None) == 1000

    def test_get_budget_specific(self) -> None:
        config = ThinkingConfig(
            budget_default=1000,
            budget_hints=8192,
            budget_answer=4096,
            budget_verify=2048,
        )
        assert config.get_budget("hints") == 8192
        assert config.get_budget("answer") == 4096
        assert config.get_budget("verify") == 2048

    def test_get_budget_none(self) -> None:
        config = ThinkingConfig(budget_default=None)
        assert config.get_budget() is None


class TestGeminiSettings:
    """Tests for GeminiSettings dataclass."""

    def test_api_key_property(self) -> None:
        settings = GeminiSettings(api_keys=["key1", "key2"])
        assert settings.api_key == "key1"

    def test_api_key_empty(self) -> None:
        settings = GeminiSettings(api_keys=[])
        assert settings.api_key == ""

    def test_get_model_default(self) -> None:
        settings = GeminiSettings(default_model="gemini-2.5-flash")
        assert settings.get_model() == "gemini-2.5-flash"
        assert settings.get_model(None) == "gemini-2.5-flash"

    def test_get_model_specific(self) -> None:
        settings = GeminiSettings(
            default_model="gemini-2.5-flash",
            hints_model="gemini-3-pro-preview",
            answer_model="gemini-2.5-pro",
            verify_model="gemini-2.5-flash",
        )
        assert settings.get_model("hints") == "gemini-3-pro-preview"
        assert settings.get_model("answer") == "gemini-2.5-pro"
        assert settings.get_model("verify") == "gemini-2.5-flash"

    def test_get_model_fallback_when_empty(self) -> None:
        settings = GeminiSettings(
            default_model="gemini-2.5-flash",
            hints_model="",  # 빈 문자열
        )
        assert settings.get_model("hints") == "gemini-2.5-flash"

    def test_get_temperature_gemini3(self) -> None:
        settings = GeminiSettings(temperature=0.5)
        assert (
            settings.get_temperature("gemini-3-pro-preview")
            == GEMINI_3_FIXED_TEMPERATURE
        )

    def test_get_temperature_gemini25(self) -> None:
        settings = GeminiSettings(temperature=0.5)
        assert settings.get_temperature("gemini-2.5-flash") == 0.5


class TestSessionSettings:
    """Tests for SessionSettings dataclass."""

    def test_defaults(self) -> None:
        with patch.dict(
            os.environ, {"MAX_SESSIONS": "100", "SESSION_TTL_MINUTES": "60"}
        ):
            settings = SessionSettings()
            assert settings.max_sessions == 100
            assert settings.session_ttl_minutes == 60


class TestRedisSettings:
    """Tests for RedisSettings dataclass."""

    def test_enabled_true(self) -> None:
        with patch.dict(os.environ, {"LANGGRAPH_REDIS_ENABLED": "true"}):
            settings = RedisSettings()
            assert settings.enabled is True

    def test_enabled_false(self) -> None:
        with patch.dict(os.environ, {"LANGGRAPH_REDIS_ENABLED": "false"}):
            settings = RedisSettings()
            assert settings.enabled is False


class TestGuardSettings:
    """Tests for GuardSettings dataclass."""

    def test_defaults(self) -> None:
        with patch.dict(
            os.environ,
            {
                "GUARD_ENABLED": "true",
                "GUARD_THRESHOLD": "0.9",
                "RULEPACKS_DIR": "rules",
            },
        ):
            settings = GuardSettings()
            assert settings.enabled is True
            assert settings.threshold == 0.9
            assert settings.rulepacks_dir == "rules"


class TestLoggingSettings:
    """Tests for LoggingSettings dataclass."""

    def test_json_logs_true(self) -> None:
        with patch.dict(os.environ, {"LOG_JSON": "true"}):
            settings = LoggingSettings()
            assert settings.json_logs is True

    def test_console_enabled(self) -> None:
        with patch.dict(os.environ, {"LOG_CONSOLE": "true"}):
            settings = LoggingSettings()
            assert settings.console_enabled is True


class TestHttpSettings:
    """Tests for HttpSettings dataclass."""

    def test_defaults(self) -> None:
        with patch.dict(os.environ, {"HTTP_HOST": "0.0.0.0", "HTTP_PORT": "8080"}):
            settings = HttpSettings()
            assert settings.host == "0.0.0.0"
            assert settings.port == 8080


class TestSettings:
    """Tests for main Settings container."""

    def test_contains_all_subsettings(self) -> None:
        settings = Settings()
        assert hasattr(settings, "gemini")
        assert hasattr(settings, "session")
        assert hasattr(settings, "redis")
        assert hasattr(settings, "guard")
        assert hasattr(settings, "logging")
        assert hasattr(settings, "http")


class TestGetSettings:
    """Tests for get_settings singleton."""

    def test_returns_settings(self) -> None:
        settings = get_settings()
        assert isinstance(settings, Settings)

    def test_singleton(self) -> None:
        s1 = get_settings()
        s2 = get_settings()
        assert s1 is s2
