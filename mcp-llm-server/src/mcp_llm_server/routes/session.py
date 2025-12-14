"""Session routes."""

import uuid
from typing import Any

from fastapi import APIRouter
from pydantic import BaseModel, Field

from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.exceptions import SessionNotFoundError
from mcp_llm_server.infra.gemini_client import get_gemini_client
from mcp_llm_server.infra.langgraph_session import get_langgraph_manager
from mcp_llm_server.routes.dependencies import ensure_safe_text
from mcp_llm_server.routes.llm import ChatResponse
from mcp_llm_server.utils.session import resolve_session_id


router = APIRouter(prefix="/api/sessions", tags=["Session"])


class SessionCreateRequest(BaseModel):
    """Session creation request"""

    session_id: str | None = Field(
        None, description="Session ID (auto-generated if not provided)"
    )
    chat_id: str | None = Field(
        None, description="Chat/room identifier for automatic session mapping"
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping"
    )
    model: str | None = None
    system_prompt: str | None = None


class SessionChatRequest(BaseModel):
    """Session chat request"""

    prompt: str


@router.post("")
async def api_session_create(request: SessionCreateRequest) -> dict[str, Any]:
    """Create a new chat session."""
    settings = get_settings()
    manager = await get_langgraph_manager()
    effective_model = request.model or settings.gemini.default_model

    derived_session_id = resolve_session_id(
        session_id=request.session_id,
        chat_id=request.chat_id,
        namespace=request.namespace,
        default_namespace="generic",
    )
    session_id = derived_session_id or str(uuid.uuid4())

    session = await manager.create_fresh_session(
        session_id=session_id,
        model=effective_model,
        system_prompt=request.system_prompt,
    )

    return {
        "session_id": session.session_id,
        "model": session.model,
        "created": True,
    }


@router.post("/{session_id}/messages", response_model=ChatResponse)
async def api_session_chat(
    session_id: str, request: SessionChatRequest
) -> ChatResponse:
    """Session-based chat."""
    await ensure_safe_text(request.prompt)

    manager = await get_langgraph_manager()
    session = await manager.get_session(session_id)

    if not session:
        raise SessionNotFoundError(session_id)

    await manager.add_message(session_id, "user", request.prompt)

    history = await manager.get_history_as_dicts(session_id)
    history = history[:-1] if history else []
    max_pairs = get_settings().session.history_max_pairs
    if max_pairs <= 0:
        history = []
    else:
        max_messages = max_pairs * 2
        if max_messages > 0 and len(history) > max_messages:
            history = history[-max_messages:]

    client = get_gemini_client()
    response = await client.chat(
        request.prompt, session.system_prompt, history, session.model
    )

    await manager.add_message(session_id, "assistant", response)

    return ChatResponse(response=response, model=session.model)


@router.delete("/{session_id}")
async def api_session_end(session_id: str) -> dict[str, Any]:
    """End a chat session."""
    manager = await get_langgraph_manager()
    removed = await manager.end_session(session_id)
    return {"session_id": session_id, "removed": removed}


@router.get("/{session_id}")
async def api_session_info(session_id: str) -> dict[str, Any]:
    """Get session information."""
    manager = await get_langgraph_manager()
    info = manager.get_session_info(session_id)

    if not info:
        raise SessionNotFoundError(session_id)
    return info
