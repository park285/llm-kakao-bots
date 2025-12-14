"""Gemini client using LangChain ChatGoogleGenerativeAI

- thinking: Gemini 3는 thinking_level(low/high), 2.5는 thinking_budget 사용 (우회 구현)
- LangChain 기능 활용: structured output, usage metadata, retry/timeout, callbacks, tool binding
"""

import logging
import threading
from collections.abc import AsyncIterator, Awaitable, Callable
from dataclasses import dataclass
from typing import TYPE_CHECKING, NoReturn, TypeVar, cast


if TYPE_CHECKING:
    from mcp_llm_server.models.stream import (
        ChatResult,
        ContentBlock,
        StreamEvent,
        UsageInfo,
    )

from cachetools import LRUCache
from google.ai.generativelanguage_v1beta.types import generative_service
from google.api_core.exceptions import DeadlineExceeded, GoogleAPICallError
from google.genai.errors import APIError, ServerError
from langchain_core.callbacks import AsyncCallbackHandler
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage, SystemMessage
from langchain_core.tools import BaseTool
from langchain_google_genai import ChatGoogleGenerativeAI
from pydantic import BaseModel

from mcp_llm_server.config.settings import GeminiSettings, get_settings, is_gemini_3
from mcp_llm_server.exceptions import LLMModelError, LLMTimeoutError
from mcp_llm_server.infra.callbacks import get_default_callbacks
from mcp_llm_server.types import JSONMapping


log = logging.getLogger(__name__)
logging.getLogger("google_genai").setLevel(logging.WARNING)

T = TypeVar("T", bound=BaseModel)
ResultT = TypeVar("ResultT")

THINKING_LEVEL_SUPPORTED = (
    "thinking_level" in generative_service.ThinkingConfig._meta.fields
)
_THINKING_LEVEL_UNSUPPORTED_LOGGED = False


HTTP_STATUS_GATEWAY_TIMEOUT = 504


@dataclass
class ToolCall:
    """Tool call from LLM response"""

    name: str
    args: JSONMapping
    id: str = ""


@dataclass
class TokenUsage:
    """Token usage statistics"""

    input_tokens: int = 0
    output_tokens: int = 0
    total_tokens: int = 0
    reasoning_tokens: int = 0  # thinking mode


HistoryEntry = dict[str, str]


class GeminiClient:
    """LangChain-based Gemini client with multi-model support

    Features:
    - Multi-model support with caching
    - Structured output with Pydantic models
    - Token usage tracking
    - Retry/timeout handling
    - Callback support for logging/monitoring

    Gemini 3 모델: 온도 1.0 고정, thinking_budget 사용 (향후 thinking_level로 변경)
    Gemini 2.5 모델: 설정된 온도 사용, thinking_budget 사용
    """

    def __init__(
        self,
        settings: GeminiSettings | None = None,
        callbacks: list[AsyncCallbackHandler] | None = None,
    ) -> None:
        self._settings = settings or get_settings().gemini
        # (model, task) -> LLM instance, cache size from settings
        self._model_cache: LRUCache[tuple[str, str | None], ChatGoogleGenerativeAI] = (
            LRUCache(maxsize=self._settings.model_cache_size)
        )
        self._callbacks = callbacks or []  # 누적 사용량
        self._cache_lock = threading.Lock()
        self._api_key_lock = threading.Lock()
        self._api_key_index = 0

    @staticmethod
    def _normalize_thinking_level(level: str | None) -> str | None:
        """Normalize configured thinking_level to LangChain-supported values."""
        if not level:
            return None
        normalized = level.lower()
        if normalized in {"low", "high"}:
            return normalized
        if normalized == "medium":
            return "high"
        if normalized == "none":
            return None

        log.warning("Unsupported thinking_level=%s, skipping", level)
        return None

    def _build_thinking_kwargs(self, model: str, task: str | None) -> dict[str, object]:
        """Resolve thinking configuration per model/task."""
        if is_gemini_3(model):
            level = self._normalize_thinking_level(
                self._settings.thinking.get_level(task)
            )
            if level:
                if THINKING_LEVEL_SUPPORTED:
                    log.debug(
                        "Gemini 3 thinking_level: model=%s, task=%s, level=%s",
                        model,
                        task,
                        level,
                    )
                    return {"thinking_level": level}
                global _THINKING_LEVEL_UNSUPPORTED_LOGGED
                if not _THINKING_LEVEL_UNSUPPORTED_LOGGED:
                    log.info(
                        "Gemini 3 thinking_level unsupported by client library, "
                        "ignoring: model=%s task=%s level=%s",
                        model,
                        task,
                        level,
                    )
                    _THINKING_LEVEL_UNSUPPORTED_LOGGED = True
            return {}

        thinking_budget = self._settings.thinking.get_budget(task)
        if thinking_budget:
            log.debug(
                "Gemini 2.x thinking_budget: model=%s, task=%s, budget=%d",
                model,
                task,
                thinking_budget,
            )
            return {"thinking_budget": thinking_budget}

        return {}

    def _select_api_key(self) -> str:
        """Select next API key (round-robin)."""
        keys = self._settings.api_keys
        if not keys:
            raise LLMModelError("No Gemini API keys configured")

        with self._api_key_lock:
            key = keys[self._api_key_index % len(keys)]
            self._api_key_index += 1
        return key

    def _get_llm(
        self, model: str | None = None, task: str | None = None
    ) -> ChatGoogleGenerativeAI:
        """Get or create LLM instance for specific model and task (cached).

        Args:
            model: Model name override (default: settings.default_model)
            task: Task type for thinking config (hints, answer, verify)
        """
        model_name = model or self._settings.default_model
        cache_key = (model_name, task)

        with self._cache_lock:
            cached = self._model_cache.get(cache_key)
            if cached is None:
                cached = self._create_llm(model_name, task)
                self._model_cache[cache_key] = cached

        return cached

    def _create_llm(
        self, model: str, task: str | None = None
    ) -> ChatGoogleGenerativeAI:
        """Create LangChain Gemini model instance with model/task-specific settings.

        Args:
            model: Model name
            task: Task type for thinking config (hints, answer, verify)
        """
        # Gemini 3은 온도 1.0 고정 (변경 시 루핑 현상 발생)
        temperature = self._settings.get_temperature(model)
        api_key = self._select_api_key()

        kwargs: dict[str, object] = {
            "model": model,
            "temperature": temperature,
            "max_output_tokens": self._settings.max_output_tokens,
            "google_api_key": api_key,
            # Retry/timeout from settings
            "max_retries": self._settings.max_retries,
            "timeout": self._settings.timeout,
            # Callbacks for logging/monitoring
            "callbacks": self._callbacks if self._callbacks else None,
        }

        kwargs.update(self._build_thinking_kwargs(model, task))

        return ChatGoogleGenerativeAI(**kwargs)

    def _handle_llm_exception(self, exc: Exception) -> NoReturn:
        """Translate LangChain/Google API exceptions into structured LLM errors."""
        if isinstance(exc, DeadlineExceeded):
            log.warning("Gemini deadline exceeded (%s)", exc)
            raise LLMTimeoutError(f"Gemini deadline exceeded: {exc}") from exc
        if isinstance(exc, ServerError):
            status_code = getattr(exc, "status_code", None) or getattr(
                exc, "code", None
            )
            message = str(exc)
            if (
                status_code == HTTP_STATUS_GATEWAY_TIMEOUT
                or "DEADLINE_EXCEEDED" in message
            ):
                log.warning("Gemini server deadline exceeded (%s)", exc)
                raise LLMTimeoutError(f"Gemini deadline exceeded: {exc}") from exc
            log.warning("Gemini server error (%s)", exc)
            raise LLMModelError(f"Gemini server error: {exc}") from exc
        if isinstance(exc, APIError):
            log.warning("Gemini API error (%s)", exc)
            raise LLMModelError(f"Gemini API error: {exc}") from exc

        if isinstance(exc, GoogleAPICallError):
            log.warning("Gemini API error (%s)", exc)
            raise LLMModelError(f"Gemini API error: {exc}") from exc

        log.warning("Unexpected Gemini error (%s)", exc)
        raise LLMModelError(f"Unexpected Gemini error: {exc}") from exc

    async def _invoke_with_error_handling(
        self, operation: Awaitable[ResultT]
    ) -> ResultT:
        try:
            return await operation
        except (DeadlineExceeded, GoogleAPICallError, APIError, ServerError) as exc:
            self._handle_llm_exception(exc)
        except Exception as exc:  # noqa: BLE001
            self._handle_llm_exception(exc)

    async def _stream_with_error_handling(
        self, iterator: AsyncIterator[BaseMessage]
    ) -> AsyncIterator[BaseMessage]:
        try:
            async for chunk in iterator:
                yield chunk
        except (DeadlineExceeded, GoogleAPICallError) as exc:
            self._handle_llm_exception(exc)
        except Exception as exc:  # noqa: BLE001
            self._handle_llm_exception(exc)

    def _build_messages(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
    ) -> list[BaseMessage]:
        """Build LangChain message list from input"""
        messages: list[BaseMessage] = []

        if system_prompt:
            messages.append(SystemMessage(content=system_prompt))

        # history: [{"role": "user"|"assistant", "content": "..."}]
        if history:
            for msg in history:
                role = msg.get("role", "user")
                content = msg.get("content", "")
                if role == "user":
                    messages.append(HumanMessage(content=content))
                elif role == "assistant":
                    messages.append(AIMessage(content=content))

        messages.append(HumanMessage(content=prompt))
        return messages

    def _extract_text(self, response: BaseMessage) -> str:
        """Extract text from response (handles Gemini 3 list format)"""
        # Gemini 3은 thought signatures 포함으로 list 반환 가능
        content = response.content
        if isinstance(content, list):
            # 텍스트 부분만 추출
            return "".join(
                item if isinstance(item, str) else item.get("text", "")
                for item in content
            )
        return content if isinstance(content, str) else str(content)

    async def chat(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
    ) -> str:
        """Stateless chat - single request/response

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override (default: settings.default_model)

        Returns:
            LLM response text
        """
        llm = self._get_llm(model)
        messages = self._build_messages(prompt, system_prompt, history)
        response = await self._invoke_with_error_handling(llm.ainvoke(messages))
        return self._extract_text(response)

    async def chat_structured(
        self,
        prompt: str,
        output_schema: type[T],
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
    ) -> T:
        """Chat with structured output using Pydantic model

        Args:
            prompt: User message
            output_schema: Pydantic model class for response structure
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override

        Returns:
            Pydantic model instance with structured response
        """
        llm = self._get_llm(model)
        structured_llm = llm.with_structured_output(output_schema)
        messages = self._build_messages(prompt, system_prompt, history)
        response = await self._invoke_with_error_handling(
            structured_llm.ainvoke(messages)
        )
        return cast("T", response)

    async def stream(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
    ) -> AsyncIterator[str]:
        """Streaming chat - yields chunks

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
        model: Optional model override (default: settings.default_model)
        """
        llm = self._get_llm(model)
        messages = self._build_messages(prompt, system_prompt, history)
        async for chunk in self._stream_with_error_handling(llm.astream(messages)):
            text = self._extract_text(chunk)
            if text:
                yield text

    async def chat_with_tools(
        self,
        prompt: str,
        tools: list[BaseTool | Callable[..., object]],
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
    ) -> tuple[str, list[ToolCall]]:
        """Chat with tool binding - LLM can request tool calls

        Args:
            prompt: User message
            tools: List of LangChain tools or callables
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override

        Returns:
            Tuple of (response_text, tool_calls)
            - response_text: LLM response (may be empty if only tool calls)
            - tool_calls: List of ToolCall objects the LLM wants to invoke
        """
        llm = self._get_llm(model)
        llm_with_tools = llm.bind_tools(tools)
        messages = self._build_messages(prompt, system_prompt, history)

        response = await self._invoke_with_error_handling(
            llm_with_tools.ainvoke(messages)
        )

        # extract tool calls from response
        tool_calls: list[ToolCall] = []
        if hasattr(response, "tool_calls") and response.tool_calls:
            for tc in response.tool_calls:
                args = tc.get("args", {})
                if not isinstance(args, dict):
                    args = {}
                tool_calls.append(
                    ToolCall(
                        name=tc.get("name", ""),
                        args=args,
                        id=tc.get("id") or "",
                    )
                )

        text = self._extract_text(response)
        return text, tool_calls

    # =========================================================================
    # Public API (for external use without accessing private methods)
    # =========================================================================

    def get_llm_for_task(self, task: str | None = None) -> ChatGoogleGenerativeAI:
        """Get LLM instance configured for specific task.

        This is the public API for accessing task-specific LLM instances.
        Use this instead of _get_llm() for better encapsulation.

        Args:
            task: Task type (hints, answer, verify) for thinking config

        Returns:
            ChatGoogleGenerativeAI instance with task-specific settings
        """
        model = self._settings.get_model(task)
        return self._get_llm(model, task)

    def extract_text(self, response: BaseMessage) -> str:
        """Extract text from LLM response (public API).

        Handles Gemini 3's list format for thought signatures.

        Args:
            response: LLM response object

        Returns:
            Extracted text content
        """
        return self._extract_text(response)

    # =========================================================================
    # Content Blocks & Usage Extraction
    # =========================================================================

    def _parse_content_blocks(self, response: BaseMessage) -> list["ContentBlock"]:
        """Parse content blocks from LLM response.

        Gemini 3.x returns list with reasoning/text blocks.
        """
        from mcp_llm_server.models.stream import ContentBlock, ContentBlockType

        blocks: list[ContentBlock] = []
        content = response.content

        if isinstance(content, str):
            # 단순 텍스트 응답
            blocks.append(ContentBlock(type=ContentBlockType.TEXT, content=content))
        elif isinstance(content, list):
            # Gemini 3 list format (reasoning + text blocks)
            for item in content:
                if isinstance(item, str):
                    blocks.append(
                        ContentBlock(type=ContentBlockType.TEXT, content=item)
                    )
                elif isinstance(item, dict):
                    block_type = item.get("type", "")
                    if block_type == "reasoning" or "reasoning" in item:
                        reasoning_text = item.get("reasoning") or item.get("text") or ""
                        blocks.append(
                            ContentBlock(
                                type=ContentBlockType.REASONING,
                                content=str(reasoning_text),
                            )
                        )
                    elif block_type == "text" or "text" in item:
                        blocks.append(
                            ContentBlock(
                                type=ContentBlockType.TEXT,
                                content=item.get("text", ""),
                            )
                        )
                    elif block_type == "tool_call" or "tool_calls" in item:
                        # tool call 블록 처리
                        blocks.append(
                            ContentBlock(
                                type=ContentBlockType.TOOL_CALL,
                                tool_name=item.get("name", ""),
                                tool_args=item.get("args", {}),
                                tool_id=item.get("id", ""),
                            )
                        )
                    else:
                        # Unknown block type
                        blocks.append(
                            ContentBlock(
                                type=ContentBlockType.UNKNOWN,
                                content=str(item),
                            )
                        )

        return blocks

    def _extract_usage(self, response: BaseMessage) -> "UsageInfo":
        """Extract token usage from LLM response."""
        from mcp_llm_server.models.stream import UsageInfo

        usage = UsageInfo()

        # AIMessage.usage_metadata에서 추출
        if hasattr(response, "usage_metadata") and response.usage_metadata:
            meta = response.usage_metadata
            usage.input_tokens = meta.get("input_tokens", 0)
            usage.output_tokens = meta.get("output_tokens", 0)
            usage.total_tokens = meta.get("total_tokens", 0)

            # reasoning tokens (thinking mode)
            if "output_token_details" in meta:
                usage.reasoning_tokens = meta["output_token_details"].get(
                    "reasoning", 0
                )

        return usage

    async def chat_with_usage(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
        task: str | None = None,
    ) -> "ChatResult":
        """Chat with extended response including usage and content blocks.

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override
            task: Task type for thinking config (hints, answer, verify)

        Returns:
            ChatResult with text, usage, and content blocks
        """
        from mcp_llm_server.models.stream import ChatResult, ContentBlockType

        llm = self._get_llm(model, task)
        messages = self._build_messages(prompt, system_prompt, history)
        response = await llm.ainvoke(messages)

        # Content blocks 파싱
        blocks = self._parse_content_blocks(response)

        # 텍스트 추출 (TEXT 블록만)
        text_parts = [b.content for b in blocks if b.type == ContentBlockType.TEXT]
        text = "".join(text_parts)

        # Reasoning 추출
        reasoning_parts = [
            b.content for b in blocks if b.type == ContentBlockType.REASONING
        ]
        reasoning = "\n".join(reasoning_parts)

        # Usage 추출
        usage = self._extract_usage(response)

        return ChatResult(
            text=text,
            usage=usage,
            blocks=blocks,
            reasoning=reasoning,
        )

    async def stream_events(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[HistoryEntry] | None = None,
        model: str | None = None,
        task: str | None = None,
    ) -> AsyncIterator["StreamEvent"]:
        """Stream events with detailed event types for monitoring.

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override
            task: Task type for thinking config

        Yields:
            StreamEvent objects (TOKEN, REASONING, USAGE, DONE)
        """
        from mcp_llm_server.models.stream import StreamEvent, StreamEventType

        llm = self._get_llm(model, task)
        messages = self._build_messages(prompt, system_prompt, history)

        accumulated_text = ""
        last_response = None

        try:
            async for chunk in self._stream_with_error_handling(llm.astream(messages)):
                last_response = chunk
                content = chunk.content

                if isinstance(content, str) and content:
                    accumulated_text += content
                    yield StreamEvent(type=StreamEventType.TOKEN, content=content)
                elif isinstance(content, list):
                    # Gemini 3 list format
                    for item in content:
                        if isinstance(item, str) and item:
                            accumulated_text += item
                            yield StreamEvent(type=StreamEventType.TOKEN, content=item)
                        elif isinstance(item, dict):
                            if "reasoning" in item:
                                yield StreamEvent(
                                    type=StreamEventType.REASONING,
                                    content=item.get("reasoning", ""),
                                )
                            elif "text" in item:
                                text = item.get("text", "")
                                if text:
                                    accumulated_text += text
                                    yield StreamEvent(
                                        type=StreamEventType.TOKEN, content=text
                                    )

            if last_response:
                usage = self._extract_usage(last_response)
                yield StreamEvent(type=StreamEventType.USAGE, usage=usage)

            yield StreamEvent(
                type=StreamEventType.DONE,
                metadata={"total_length": len(accumulated_text)},
            )
        except LLMModelError as exc:
            log.warning("Stream error: %s", exc)
            yield StreamEvent(type=StreamEventType.ERROR, error=str(exc))
        except Exception as exc:
            log.exception("Stream error")
            yield StreamEvent(type=StreamEventType.ERROR, error=str(exc))


# singleton instance
_client: GeminiClient | None = None


def get_gemini_client() -> GeminiClient:
    """Get Gemini client singleton (with default callbacks)"""
    global _client
    if _client is None:
        _client = GeminiClient(callbacks=get_default_callbacks())
    return _client
