"""Session ID resolution utilities.

Bots may omit explicit session_id and instead send chat_id + namespace.
This helper derives a stable session_id for LangGraph history tracking.
"""

from __future__ import annotations


def resolve_session_id(
    *,
    session_id: str | None,
    chat_id: str | None,
    namespace: str | None,
    default_namespace: str,
) -> str | None:
    """Resolve effective session_id for history tracking.

    Priority:
    1. Explicit session_id if provided
    2. Derived from namespace + chat_id if chat_id provided
    3. None (stateless)

    Args:
        session_id: Optional explicit session identifier.
        chat_id: Optional chat/room identifier.
        namespace: Optional bot namespace.
        default_namespace: Namespace used when namespace is not provided.

    Returns:
        Resolved session identifier or None.
    """
    if session_id:
        return session_id
    if chat_id:
        effective_namespace = namespace or default_namespace
        return f"{effective_namespace}:{chat_id}"
    return None
