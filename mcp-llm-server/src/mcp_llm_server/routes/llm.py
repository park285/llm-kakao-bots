"""LLM-related routes and schemas."""

from collections.abc import AsyncIterator
from types import GenericAlias, UnionType
from typing import Any, TypeVar, cast

from fastapi import APIRouter
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field, ValidationError, create_model

from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.exceptions import InvalidInputError, MCPLLMError
from mcp_llm_server.infra.callbacks import get_metrics_handler
from mcp_llm_server.infra.gemini_client import get_gemini_client
from mcp_llm_server.routes.dependencies import ensure_safe_text


router = APIRouter(prefix="/api/llm", tags=["LLM"])

T = TypeVar("T", bound=BaseModel)


class ChatRequest(BaseModel):
    """LLM chat request"""

    prompt: str = Field(..., description="User message")
    system_prompt: str | None = Field(None, description="System instruction")
    history: list[dict[str, str]] | None = Field(
        None,
        description='Conversation history [{"role": "user"|"assistant", "content": "..."}]',
    )
    model: str | None = Field(None, description="Model override")
    task: str | None = Field(None, description="Task (hints|answer|verify)")


class ChatResponse(BaseModel):
    """LLM chat response"""

    response: str
    model: str


class StructuredRequest(BaseModel):
    """Structured output request"""

    prompt: str
    json_schema: dict[str, Any] = Field(..., description="JSON schema for output")
    system_prompt: str | None = None
    history: list[dict[str, str]] | None = None
    model: str | None = None


class ChatWithUsageRequest(BaseModel):
    """Request for chat with usage info."""

    prompt: str
    system_prompt: str | None = None
    history: list[dict[str, str]] | None = None
    model: str | None = None
    task: str | None = None  # hints, answer, verify


class ChatWithUsageResponse(BaseModel):
    """Response with text, usage, and optional reasoning."""

    text: str
    usage: dict[str, int]
    reasoning: str = ""
    has_reasoning: bool = False


class StreamEventsRequest(BaseModel):
    """Request for event streaming."""

    prompt: str
    system_prompt: str | None = None
    history: list[dict[str, str]] | None = None
    model: str | None = None
    task: str | None = None


class UsageResponse(BaseModel):
    """Token usage response"""

    input_tokens: int
    output_tokens: int
    total_tokens: int
    reasoning_tokens: int | None
    model: str | None = None


FieldAnnotation = type[Any] | GenericAlias | UnionType

STRUCTURED_ARRAY_TYPE_MAP: dict[str, FieldAnnotation] = {
    "string": list[str],
    "integer": list[int],
    "number": list[float],
    "boolean": list[bool],
    "object": list[dict[str, Any]],
}

STRUCTURED_TYPE_MAP: dict[str, type[Any]] = {
    "string": str,
    "integer": int,
    "number": float,
    "boolean": bool,
}


def _resolve_structured_field_type(name: str, prop: dict[str, Any]) -> FieldAnnotation:
    prop_type_raw = prop.get("type")

    if isinstance(prop_type_raw, list):
        allow_none = "null" in prop_type_raw
        prop_type = next(
            (t for t in prop_type_raw if isinstance(t, str) and t != "null"), None
        )
    elif isinstance(prop_type_raw, str):
        allow_none = False
        prop_type = prop_type_raw
    else:
        raise InvalidInputError(name, "type must be string or list")

    if prop_type is None:
        raise InvalidInputError(name, "type is required")

    resolved: FieldAnnotation
    if prop_type == "array":
        items = prop.get("items")
        if not isinstance(items, dict):
            raise InvalidInputError(name, "items must be an object")

        item_type = items.get("type")
        if not isinstance(item_type, str):
            raise InvalidInputError(name, "items.type is required")

        mapped = STRUCTURED_ARRAY_TYPE_MAP.get(item_type)
        if mapped is None:
            raise InvalidInputError(
                name,
                "array items.type must be one of string|integer|number|boolean|object",
            )
        resolved = mapped
    elif prop_type == "object":
        resolved = dict[str, Any]
    else:
        base = STRUCTURED_TYPE_MAP.get(prop_type)
        if base is None:
            raise InvalidInputError(name, f"unsupported type '{prop_type}'")
        resolved = base

    return resolved | None if allow_none else resolved


def schema_to_pydantic(schema: dict[str, Any]) -> type[BaseModel]:
    """Convert JSON schema to Pydantic model"""
    if schema.get("type") not in (None, "object"):
        raise InvalidInputError("json_schema", "only object schemas are supported")

    properties = schema.get("properties") or {}
    if not isinstance(properties, dict):
        raise InvalidInputError("json_schema", "properties must be an object")

    required = set(schema.get("required", []))
    field_definitions: dict[str, tuple[FieldAnnotation, object]] = {}
    for name, prop in properties.items():
        if not isinstance(prop, dict):
            raise InvalidInputError(name, "property definition must be an object")
        field_type = _resolve_structured_field_type(name, prop)
        default: object = ... if name in required else None
        field_definitions[name] = (field_type, default)

    if not field_definitions:
        raise InvalidInputError("json_schema", "at least one property is required")

    model = create_model("DynamicModel", **field_definitions)  # type: ignore[call-overload]
    return cast("type[BaseModel]", model)


@router.post("/chat", response_model=ChatResponse)
async def api_chat(request: ChatRequest) -> ChatResponse:
    """Stateless LLM chat"""
    await ensure_safe_text(request.prompt)

    client = get_gemini_client()
    settings = get_settings()

    response = await client.chat(
        prompt=request.prompt,
        system_prompt=request.system_prompt,
        history=request.history,
        model=request.model,
    )
    return ChatResponse(
        response=response,
        model=request.model or settings.gemini.default_model,
    )


@router.post("/stream")
async def api_stream(request: ChatRequest) -> StreamingResponse:
    """Streaming LLM chat"""
    await ensure_safe_text(request.prompt)
    client = get_gemini_client()

    async def generate() -> AsyncIterator[str]:
        async for chunk in client.stream(
            prompt=request.prompt,
            system_prompt=request.system_prompt,
            history=request.history,
            model=request.model,
        ):
            yield chunk

    return StreamingResponse(generate(), media_type="text/plain")


@router.post("/chat-with-usage", response_model=ChatWithUsageResponse)
async def api_chat_with_usage(
    request: ChatWithUsageRequest,
) -> ChatWithUsageResponse:
    """Chat with extended response including usage and reasoning info."""
    await ensure_safe_text(request.prompt)
    client = get_gemini_client()
    result = await client.chat_with_usage(
        prompt=request.prompt,
        system_prompt=request.system_prompt,
        history=request.history,
        model=request.model,
        task=request.task,
    )
    return ChatWithUsageResponse(
        text=result.text,
        usage={
            "input_tokens": result.usage.input_tokens,
            "output_tokens": result.usage.output_tokens,
            "total_tokens": result.usage.total_tokens,
            "reasoning_tokens": result.usage.reasoning_tokens,
        },
        reasoning=result.reasoning,
        has_reasoning=result.has_reasoning,
    )


@router.post("/stream-events")
async def api_stream_events(request: StreamEventsRequest) -> StreamingResponse:
    """Stream events with detailed type info (TOKEN, REASONING, USAGE, DONE).

    Returns Server-Sent Events (SSE) format for real-time monitoring.
    Each line is a JSON object with event type and content.
    """
    import json

    await ensure_safe_text(request.prompt)
    client = get_gemini_client()

    async def generate() -> AsyncIterator[str]:
        async for event in client.stream_events(
            prompt=request.prompt,
            system_prompt=request.system_prompt,
            history=request.history,
            model=request.model,
            task=request.task,
        ):
            yield json.dumps(event.to_dict(), ensure_ascii=False) + "\n"

    return StreamingResponse(
        generate(),
        media_type="application/x-ndjson",
        headers={"X-Content-Type-Options": "nosniff"},
    )


@router.post("/structured")
async def api_structured(request: StructuredRequest) -> dict[str, Any]:
    """Structured output chat"""
    try:
        output_model = schema_to_pydantic(request.json_schema)
        await ensure_safe_text(request.prompt)
        client = get_gemini_client()
        result: BaseModel = await client.chat_structured(
            prompt=request.prompt,
            output_schema=output_model,
            system_prompt=request.system_prompt,
            history=request.history,
            model=request.model,
        )
        return result.model_dump()
    except MCPLLMError:
        raise
    except (ValueError, TypeError, ValidationError) as e:
        from mcp_llm_server.exceptions import LLMParsingError

        raise LLMParsingError(f"Structured output failed: {e}") from e


@router.get("/usage", response_model=UsageResponse)
async def api_usage() -> UsageResponse:
    """Get in-memory metrics since server start (for debugging)"""
    metrics = get_metrics_handler().get_metrics()
    model = get_settings().gemini.default_model

    return UsageResponse(
        input_tokens=int(metrics["total_input_tokens"]),
        output_tokens=int(metrics["total_output_tokens"]),
        total_tokens=int(metrics["total_tokens"]),
        reasoning_tokens=(
            int(metrics["total_reasoning_tokens"])
            if metrics.get("total_reasoning_tokens") is not None
            else None
        ),
        model=model,
    )


@router.get("/usage/total", response_model=UsageResponse)
async def api_total_usage() -> UsageResponse:
    """Get cumulative token usage from DB (default: last 30 days)"""
    from mcp_llm_server.infra.usage_repository import get_usage_repository

    repository = get_usage_repository()
    usage = await repository.get_total_usage(days=30)
    model = get_settings().gemini.default_model

    return UsageResponse(
        input_tokens=usage.input_tokens,
        output_tokens=usage.output_tokens,
        total_tokens=usage.total_tokens,
        reasoning_tokens=usage.reasoning_tokens,
        model=model,
    )


@router.get("/metrics")
async def api_metrics() -> dict[str, Any]:
    """Get aggregated LLM metrics (total calls, tokens, duration)"""
    handler = get_metrics_handler()
    return handler.get_metrics()
