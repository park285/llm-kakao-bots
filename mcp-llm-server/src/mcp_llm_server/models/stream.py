"""Streaming and chat response models."""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Any


class StreamEventType(str, Enum):
    """Stream event types."""

    TOKEN = "token"  # 텍스트 토큰
    REASONING = "reasoning"  # thinking/reasoning 블록
    TOOL_CALL = "tool_call"  # 도구 호출
    USAGE = "usage"  # 토큰 사용량 (스트림 종료 시)
    ERROR = "error"  # 오류
    DONE = "done"  # 스트림 완료


class ContentBlockType(str, Enum):
    """Content block types from Gemini response."""

    TEXT = "text"
    REASONING = "reasoning"
    TOOL_CALL = "tool_call"
    TOOL_RESULT = "tool_result"
    UNKNOWN = "unknown"


@dataclass
class ContentBlock:
    """Parsed content block from LLM response."""

    type: ContentBlockType
    content: str = ""
    # tool_call 전용 필드
    tool_name: str | None = None
    tool_args: dict[str, Any] | None = None
    tool_id: str | None = None


@dataclass
class UsageInfo:
    """Token usage information."""

    input_tokens: int = 0
    output_tokens: int = 0
    total_tokens: int = 0
    reasoning_tokens: int = 0  # thinking mode 토큰


@dataclass
class StreamEvent:
    """Single streaming event.

    모니터링 및 실시간 UI 업데이트용 이벤트 구조.
    """

    type: StreamEventType
    content: str = ""
    # usage 이벤트 전용
    usage: UsageInfo | None = None
    # error 이벤트 전용
    error: str | None = None
    # 메타데이터
    metadata: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        """Convert to JSON-serializable dict."""
        result: dict[str, Any] = {"type": self.type.value}

        if self.content:
            result["content"] = self.content
        if self.usage:
            result["usage"] = {
                "input_tokens": self.usage.input_tokens,
                "output_tokens": self.usage.output_tokens,
                "total_tokens": self.usage.total_tokens,
                "reasoning_tokens": self.usage.reasoning_tokens,
            }
        if self.error:
            result["error"] = self.error
        if self.metadata:
            result["metadata"] = self.metadata

        return result


@dataclass
class ChatResult:
    """Extended chat response with usage and content blocks.

    chat_with_usage() 반환용 - 텍스트 + 메타데이터.
    """

    text: str
    usage: UsageInfo
    # Content blocks (reasoning 포함)
    blocks: list[ContentBlock] = field(default_factory=list)
    # raw reasoning text (있는 경우)
    reasoning: str = ""

    @property
    def has_reasoning(self) -> bool:
        """Check if response contains reasoning."""
        return bool(self.reasoning) or any(
            b.type == ContentBlockType.REASONING for b in self.blocks
        )
