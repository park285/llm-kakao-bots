"""Configuration settings for MCP LLM Server"""

import logging
import os
from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path

from dotenv import load_dotenv
from loguru import logger


log = logging.getLogger(__name__)

load_dotenv(override=True)  # .env 파일이 시스템 환경 변수보다 우선


class GeminiModel(str, Enum):
    """Supported Gemini models"""

    GEMINI_25_FLASH = "gemini-2.5-flash"
    GEMINI_25_FLASH_PREVIEW = "gemini-2.5-flash-preview-09-2025"
    GEMINI_25_PRO = "gemini-2.5-pro"
    GEMINI_3_PRO = "gemini-3-pro-preview"

    @classmethod
    def from_string(cls, model_name: str) -> "GeminiModel | str":
        for model in cls:
            if model.value == model_name:
                return model
        return model_name


class ThinkingLevel(str, Enum):
    """Gemini 3.0 thinking levels"""

    NONE = "none"
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"


def is_gemini_3(model: str) -> bool:
    """Check if model is Gemini 3.x"""
    return "gemini-3" in model.lower()


def is_gemini_25_preview(model: str) -> bool:
    """Check if model is Gemini 2.5 preview (supports thinking budget)"""
    return "2.5" in model and "preview" in model.lower()


GEMINI_3_FIXED_TEMPERATURE = 1.0


def _parse_api_keys() -> list[str]:
    """Parse multiple API keys from env"""
    keys_str = os.getenv("GOOGLE_API_KEYS", "")
    if keys_str:
        import re

        return [k.strip() for k in re.split(r"[,\s]+", keys_str) if k.strip()]
    single_key = os.getenv("GOOGLE_API_KEY", "")
    return [single_key] if single_key else []


def _get_int_or_none(key: str, default: int = 0) -> int | None:
    """Get int from env, return None if 0 or empty"""
    raw_value = os.getenv(key, str(default))
    try:
        val = int(raw_value)
    except ValueError:
        log.warning("Invalid int env %s=%s, using default=%s", key, raw_value, default)
        val = default
    return val if val > 0 else None


def _get_non_negative_int(key: str, default: int) -> int:
    """Get non-negative int from env, clamp negatives to 0."""
    raw_value = os.getenv(key, str(default))
    try:
        val = int(raw_value)
    except ValueError:
        log.warning("Invalid int env %s=%s, using default=%s", key, raw_value, default)
        return default
    return max(0, val)


@dataclass(frozen=True)
class ThinkingConfig:
    """Thinking configuration for different tasks"""

    # Gemini 3.0 thinking levels
    level_default: str = field(
        default_factory=lambda: os.getenv("GEMINI_THINKING_LEVEL", "low")
    )
    level_hints: str = field(
        default_factory=lambda: os.getenv("GEMINI_THINKING_LEVEL_HINTS", "low")
    )
    level_answer: str = field(
        default_factory=lambda: os.getenv("GEMINI_THINKING_LEVEL_ANSWER", "low")
    )
    level_verify: str = field(
        default_factory=lambda: os.getenv("GEMINI_THINKING_LEVEL_VERIFY", "low")
    )

    # Gemini 2.5 thinking budgets
    budget_default: int | None = field(
        default_factory=lambda: _get_int_or_none("GEMINI_THINKING_BUDGET", 0)
    )
    budget_hints: int | None = field(
        default_factory=lambda: _get_int_or_none("GEMINI_THINKING_BUDGET_HINTS", 8192)
    )
    budget_answer: int | None = field(
        default_factory=lambda: _get_int_or_none("GEMINI_THINKING_BUDGET_ANSWER", 4096)
    )
    budget_verify: int | None = field(
        default_factory=lambda: _get_int_or_none("GEMINI_THINKING_BUDGET_VERIFY", 2048)
    )

    def get_level(self, task: str | None = None) -> str:
        """Get thinking level for task"""
        levels = {
            "hints": self.level_hints,
            "answer": self.level_answer,
            "verify": self.level_verify,
        }
        return levels.get(task, self.level_default) if task else self.level_default

    def get_budget(self, task: str | None = None) -> int | None:
        """Get thinking budget for task"""
        budgets = {
            "hints": self.budget_hints,
            "answer": self.budget_answer,
            "verify": self.budget_verify,
        }
        return budgets.get(task, self.budget_default) if task else self.budget_default


@dataclass(frozen=True)
class GeminiSettings:
    """Gemini API settings"""

    api_keys: list[str] = field(default_factory=_parse_api_keys)
    default_model: str = field(
        default_factory=lambda: os.getenv(
            "GEMINI_MODEL", "gemini-2.5-flash-preview-09-2025"
        )
    )
    hints_model: str = field(
        default_factory=lambda: os.getenv("GEMINI_HINTS_MODEL", "")
    )
    answer_model: str = field(
        default_factory=lambda: os.getenv("GEMINI_ANSWER_MODEL", "")
    )
    verify_model: str = field(
        default_factory=lambda: os.getenv("GEMINI_VERIFY_MODEL", "")
    )
    temperature: float = field(
        default_factory=lambda: float(os.getenv("GEMINI_TEMPERATURE", "0.7"))
    )
    max_output_tokens: int = field(
        default_factory=lambda: int(os.getenv("GEMINI_MAX_TOKENS", "8192"))
    )
    thinking: ThinkingConfig = field(default_factory=ThinkingConfig)
    # LLM client settings
    max_retries: int = field(
        default_factory=lambda: max(1, int(os.getenv("GEMINI_MAX_RETRIES", "6")))
    )
    timeout: int = field(default_factory=lambda: int(os.getenv("GEMINI_TIMEOUT", "60")))
    model_cache_size: int = field(
        default_factory=lambda: int(os.getenv("GEMINI_MODEL_CACHE_SIZE", "20"))
    )
    failover_attempts: int = field(
        default_factory=lambda: max(1, int(os.getenv("GEMINI_FAILOVER_ATTEMPTS", "2")))
    )

    @property
    def api_key(self) -> str:
        """Get primary API key"""
        return self.api_keys[0] if self.api_keys else ""

    def get_model(self, task: str | None = None) -> str:
        """Get model for task"""
        if task == "hints" and self.hints_model:
            return self.hints_model
        if task == "answer" and self.answer_model:
            return self.answer_model
        if task == "verify" and self.verify_model:
            return self.verify_model
        return self.default_model

    def get_temperature(self, model: str) -> float:
        """Get temperature for model (Gemini 3.0 = 1.0 fixed)"""
        if is_gemini_3(model):
            return GEMINI_3_FIXED_TEMPERATURE
        return self.temperature


@dataclass(frozen=True)
class SessionSettings:
    """Session management settings"""

    max_sessions: int = field(
        default_factory=lambda: int(os.getenv("MAX_SESSIONS", "50"))
    )
    session_ttl_minutes: int = field(
        default_factory=lambda: int(os.getenv("SESSION_TTL_MINUTES", "1440"))
    )
    history_max_pairs: int = field(
        default_factory=lambda: _get_non_negative_int("SESSION_HISTORY_MAX_PAIRS", 10)
    )


@dataclass(frozen=True)
class RedisSettings:
    """Redis Stack settings for LangGraph checkpointer"""

    url: str = field(
        default_factory=lambda: os.getenv("REDIS_URL", "redis://localhost:46379")
    )
    enabled: bool = field(
        default_factory=lambda: os.getenv("LANGGRAPH_REDIS_ENABLED", "true").lower()
        == "true"
    )


@dataclass(frozen=True)
class GuardSettings:
    """Injection guard settings"""

    enabled: bool = field(
        default_factory=lambda: os.getenv("GUARD_ENABLED", "true").lower() == "true"
    )
    threshold: float = field(
        default_factory=lambda: float(os.getenv("GUARD_THRESHOLD", "0.85"))
    )
    rulepacks_dir: str = field(
        default_factory=lambda: os.getenv("RULEPACKS_DIR", "rulepacks")
    )
    # Cache settings
    cache_maxsize: int = field(
        default_factory=lambda: int(os.getenv("GUARD_CACHE_SIZE", "10000"))
    )
    cache_ttl: int = field(
        default_factory=lambda: int(os.getenv("GUARD_CACHE_TTL", "3600"))
    )
    # Anomaly detection
    anomaly_threshold: float = field(
        default_factory=lambda: float(os.getenv("GUARD_ANOMALY_THRESHOLD", "0.5"))
    )


@dataclass(frozen=True)
class LoggingSettings:
    """Logging settings"""

    level: str = field(default_factory=lambda: os.getenv("LOG_LEVEL", "INFO"))
    log_dir: str = field(default_factory=lambda: os.getenv("LOG_DIR", "logs"))
    rotation: str = field(default_factory=lambda: os.getenv("LOG_ROTATION", "10 MB"))
    retention: str = field(default_factory=lambda: os.getenv("LOG_RETENTION", "7 days"))
    compression: str = field(default_factory=lambda: os.getenv("LOG_COMPRESSION", "gz"))
    json_logs: bool = field(
        default_factory=lambda: os.getenv("LOG_JSON", "false").lower() == "true"
    )
    console_enabled: bool = field(
        default_factory=lambda: os.getenv("LOG_CONSOLE", "false").lower() == "true"
    )


@dataclass(frozen=True)
class HttpSettings:
    """HTTP server settings"""

    host: str = field(default_factory=lambda: os.getenv("HTTP_HOST", "127.0.0.1"))
    port: int = field(default_factory=lambda: int(os.getenv("HTTP_PORT", "40527")))
    # HTTP/2 (h2c) 사용 여부
    http2_enabled: bool = field(
        default_factory=lambda: os.getenv("HTTP2_ENABLED", "true").lower() == "true"
    )


@dataclass(frozen=True)
class DatabaseSettings:
    """PostgreSQL database settings for usage tracking"""

    host: str = field(default_factory=lambda: os.getenv("DB_HOST", "localhost"))
    port: int = field(default_factory=lambda: int(os.getenv("DB_PORT", "5432")))
    database: str = field(default_factory=lambda: os.getenv("DB_NAME", "twentyq"))
    user: str = field(default_factory=lambda: os.getenv("DB_USER", "twentyq"))
    password: str = field(default_factory=lambda: os.getenv("DB_PASSWORD", ""))
    min_pool_size: int = field(
        default_factory=lambda: int(os.getenv("DB_MIN_POOL", "1"))
    )
    max_pool_size: int = field(
        default_factory=lambda: int(os.getenv("DB_MAX_POOL", "5"))
    )

    @property
    def dsn(self) -> str:
        """PostgreSQL connection DSN"""
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.database}"


@dataclass(frozen=True)
class Settings:
    """Main settings container"""

    gemini: GeminiSettings = field(default_factory=GeminiSettings)
    session: SessionSettings = field(default_factory=SessionSettings)
    redis: RedisSettings = field(default_factory=RedisSettings)
    guard: GuardSettings = field(default_factory=GuardSettings)
    logging: LoggingSettings = field(default_factory=LoggingSettings)
    http: HttpSettings = field(default_factory=HttpSettings)
    database: DatabaseSettings = field(default_factory=DatabaseSettings)


_settings: Settings | None = None


def get_settings() -> Settings:
    """Get application settings singleton"""
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings


SECRET_MASK_FULL_LENGTH = 4


def _mask_secret(value: str) -> str:
    """Mask secrets for safe logging."""
    if not value:
        return "<missing>"
    if len(value) <= SECRET_MASK_FULL_LENGTH:
        return "*" * len(value)
    return f"{value[:2]}***{value[-2:]}"


def log_env_status(settings: Settings) -> None:
    """Log key env status (masked) to verify loading."""
    env_file_present = Path(".env").exists()
    gemini_keys = settings.gemini.api_keys
    primary_key = _mask_secret(gemini_keys[0]) if gemini_keys else "<missing>"
    keys_count = len(gemini_keys)

    logger.bind(name="env").info(
        "ENV_STATUS env_file={} gemini_keys={} primary_key={} model={} timeout={} redis={} db_host={} db_name={} session_ttl={} history_pairs={}",
        env_file_present,
        keys_count,
        primary_key,
        settings.gemini.default_model,
        settings.gemini.timeout,
        settings.redis.url,
        settings.database.host,
        settings.database.database,
        settings.session.session_ttl_minutes,
        settings.session.history_max_pairs,
    )

    if keys_count == 0:
        logger.bind(name="env").error("ENV_MISSING_GOOGLE_API_KEY")
