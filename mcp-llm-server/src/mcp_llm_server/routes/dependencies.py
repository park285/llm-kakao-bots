"""Common FastAPI dependencies and utilities for routes."""

from mcp_llm_server.exceptions import GuardBlockedError
from mcp_llm_server.infra.injection_guard import get_injection_guard


async def ensure_safe_text(input_text: str) -> None:
    """Run injection guard on provided text."""
    guard = get_injection_guard()
    evaluation = await guard.evaluate(input_text)
    if evaluation.malicious:
        raise GuardBlockedError(evaluation.score, evaluation.threshold)
