"""LangChain async callback handlers for logging, metrics, and usage tracking."""

from __future__ import annotations

from datetime import datetime
from typing import TYPE_CHECKING

from langchain_core.callbacks import AsyncCallbackHandler

from mcp_llm_server.config.logging import get_logger


if TYPE_CHECKING:
    from collections.abc import Mapping
    from uuid import UUID

    from langchain_core.outputs import LLMResult
    from langchain_core.runnables import RunnableConfig

    from mcp_llm_server.infra.usage_repository import UsageRepository


log = get_logger(__name__)


class LoggingCallbackHandler(AsyncCallbackHandler):
    """Async LangChain callback handler for structured logging.

    로그 LLM 호출 시작/종료, 토큰 사용량, 오류 등을 기록.
    """

    def __init__(self, log_level: str = "DEBUG") -> None:
        super().__init__()
        self._log_level = log_level.upper()
        self._start_times: dict[str, datetime] = {}

    def _log(self, message: str, *args: object) -> None:
        """Log message at configured level."""
        log.log(self._log_level, message, *args)

    async def on_llm_start(  # noqa: PLR0913 - signature defined by AsyncCallbackHandler
        self,
        serialized: Mapping[str, object],
        prompts: list[str],
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        tags: list[str] | None = None,  # noqa: ARG002
        metadata: Mapping[str, object] | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Run when LLM starts."""
        self._start_times[str(run_id)] = datetime.now()
        model_name = serialized.get("name", "unknown")
        prompt_len = sum(len(p) for p in prompts)
        self._log(
            "LLM start: model={}, prompts={}, chars={}",
            model_name,
            len(prompts),
            prompt_len,
        )

    async def on_llm_end(
        self,
        response: LLMResult,
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Run when LLM ends."""
        run_id_str = str(run_id)
        duration_ms = 0.0
        if run_id_str in self._start_times:
            start = self._start_times.pop(run_id_str)
            duration_ms = (datetime.now() - start).total_seconds() * 1000

        # 토큰 사용량 추출 (generation_info에서 먼저 시도)
        token_usage = {}
        if response.generations and response.generations[0]:
            gen = response.generations[0][0] if response.generations[0] else None
            if gen and hasattr(gen, "generation_info") and gen.generation_info:
                usage = gen.generation_info.get("usage_metadata", {})
                if usage:
                    token_usage = {
                        "input": usage.get("input_tokens", 0),
                        "output": usage.get("output_tokens", 0),
                        "total": usage.get("total_tokens", 0),
                    }

        # Fallback: llm_output
        if not token_usage and response.llm_output:
            usage = response.llm_output.get("usage_metadata", {})
            if usage:
                token_usage = {
                    "input": usage.get("input_tokens", 0),
                    "output": usage.get("output_tokens", 0),
                    "total": usage.get("total_tokens", 0),
                }

        self._log(
            "LLM end: duration={:.1f}ms, tokens={}",
            duration_ms,
            token_usage or "N/A",
        )

    async def on_llm_error(
        self,
        error: BaseException,
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Run when LLM errors."""
        log.warning("LLM error: {}", str(error))
        self._start_times.pop(str(run_id), None)


class MetricsCallbackHandler(AsyncCallbackHandler):
    """Async LangChain callback handler for metrics and DB usage tracking.

    메모리 메트릭 수집 + PostgreSQL DB에 일별 사용량 저장.
    """

    def __init__(self, repository: UsageRepository | None = None) -> None:
        super().__init__()
        self._repository = repository
        self._total_input_tokens: int = 0
        self._total_output_tokens: int = 0
        self._total_reasoning_tokens: int = 0
        self._total_calls: int = 0
        self._total_errors: int = 0
        self._total_duration_ms: float = 0.0
        self._start_times: dict[str, datetime] = {}

    def set_repository(self, repository: UsageRepository) -> None:
        """Set usage repository for DB persistence."""
        self._repository = repository

    async def on_llm_start(
        self,
        serialized: Mapping[str, object],  # noqa: ARG002
        prompts: list[str],  # noqa: ARG002
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Record LLM start time."""
        self._start_times[str(run_id)] = datetime.now()
        self._total_calls += 1

    async def on_llm_end(
        self,
        response: LLMResult,
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Record LLM metrics and save to DB."""
        run_id_str = str(run_id)
        if run_id_str in self._start_times:
            start = self._start_times.pop(run_id_str)
            self._total_duration_ms += (datetime.now() - start).total_seconds() * 1000

        # 토큰 사용량 추출 및 누적
        input_tokens = 0
        output_tokens = 0
        reasoning_tokens = 0

        # LangChain Google GenAI: ChatGeneration.message (AIMessage)에 usage_metadata 있음
        if response.generations and response.generations[0]:
            gen = response.generations[0][0] if response.generations[0] else None
            # ChatGeneration인 경우 message 속성에서 usage_metadata 추출
            if gen and hasattr(gen, "message"):
                msg = gen.message
                if hasattr(msg, "usage_metadata") and msg.usage_metadata:
                    usage = msg.usage_metadata
                    input_tokens = usage.get("input_tokens", 0)
                    output_tokens = usage.get("output_tokens", 0)
                    self._total_input_tokens += input_tokens
                    self._total_output_tokens += output_tokens

                    if "output_token_details" in usage:
                        reasoning_tokens = usage["output_token_details"].get(
                            "reasoning", 0
                        )
                        self._total_reasoning_tokens += reasoning_tokens
                    log.debug(
                        "USAGE_EXTRACTED in={}, out={}, reasoning={}",
                        input_tokens,
                        output_tokens,
                        reasoning_tokens,
                    )

            # Fallback: generation_info에서 시도
            if (
                input_tokens == 0
                and output_tokens == 0
                and gen is not None
                and hasattr(gen, "generation_info")
                and gen.generation_info
            ):
                usage = gen.generation_info.get("usage_metadata", {})
                if usage:
                    input_tokens = usage.get("input_tokens", 0)
                    output_tokens = usage.get("output_tokens", 0)
                    self._total_input_tokens += input_tokens
                    self._total_output_tokens += output_tokens

                    if "output_token_details" in usage:
                        reasoning_tokens = usage["output_token_details"].get(
                            "reasoning", 0
                        )
                        self._total_reasoning_tokens += reasoning_tokens

        # Fallback: llm_output에서 시도
        if input_tokens == 0 and output_tokens == 0 and response.llm_output:
            usage = response.llm_output.get("usage_metadata", {})
            if usage:
                input_tokens = usage.get("input_tokens", 0)
                output_tokens = usage.get("output_tokens", 0)
                self._total_input_tokens += input_tokens
                self._total_output_tokens += output_tokens

                if "output_token_details" in usage:
                    reasoning_tokens = usage["output_token_details"].get("reasoning", 0)
                    self._total_reasoning_tokens += reasoning_tokens

        # DB에 저장 (repository가 설정된 경우)
        if self._repository and (input_tokens > 0 or output_tokens > 0):
            try:
                await self._repository.record_usage(
                    input_tokens=input_tokens,
                    output_tokens=output_tokens,
                    reasoning_tokens=reasoning_tokens,
                )
                log.debug(
                    "USAGE_DB_SAVED in={}, out={}, reasoning={}",
                    input_tokens,
                    output_tokens,
                    reasoning_tokens,
                )
            except Exception as e:  # noqa: BLE001 allow logging-only DB failure
                log.error("USAGE_DB_SAVE_FAILED: {}", e)

    async def on_llm_error(
        self,
        error: BaseException,  # noqa: ARG002
        *,
        run_id: UUID,
        parent_run_id: UUID | None = None,  # noqa: ARG002
        **_kwargs: object,
    ) -> None:
        """Record error occurrence."""
        self._total_errors += 1
        self._start_times.pop(str(run_id), None)

    def get_metrics(self) -> dict[str, float | int]:
        """Get collected metrics (in-memory)."""
        return {
            "total_calls": self._total_calls,
            "total_errors": self._total_errors,
            "total_input_tokens": self._total_input_tokens,
            "total_output_tokens": self._total_output_tokens,
            "total_reasoning_tokens": self._total_reasoning_tokens,
            "total_tokens": self._total_input_tokens + self._total_output_tokens,
            "total_duration_ms": round(self._total_duration_ms, 1),
            "avg_duration_ms": round(
                self._total_duration_ms / max(self._total_calls, 1), 1
            ),
        }

    def reset(self) -> None:
        """Reset all metrics."""
        self._total_input_tokens = 0
        self._total_output_tokens = 0
        self._total_reasoning_tokens = 0
        self._total_calls = 0
        self._total_errors = 0
        self._total_duration_ms = 0.0


# singleton instances
_logging_handler: LoggingCallbackHandler | None = None
_metrics_handler: MetricsCallbackHandler | None = None


def get_logging_handler(log_level: str = "DEBUG") -> LoggingCallbackHandler:
    """Get singleton LoggingCallbackHandler."""
    global _logging_handler
    if _logging_handler is None:
        _logging_handler = LoggingCallbackHandler(log_level)
    return _logging_handler


def get_metrics_handler() -> MetricsCallbackHandler:
    """Get singleton MetricsCallbackHandler."""
    global _metrics_handler
    if _metrics_handler is None:
        _metrics_handler = MetricsCallbackHandler()
    return _metrics_handler


def get_default_callbacks() -> list[AsyncCallbackHandler]:
    """Get default callback handlers for LLM operations."""
    return [get_logging_handler(), get_metrics_handler()]


def get_callback_config() -> RunnableConfig:
    """Get RunnableConfig with default callbacks for chain.ainvoke()."""
    from langchain_core.runnables import RunnableConfig

    return RunnableConfig(callbacks=list(get_default_callbacks()))
