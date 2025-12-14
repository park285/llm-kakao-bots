"""Injection guard routes."""

from typing import Any

from fastapi import APIRouter
from pydantic import BaseModel

from mcp_llm_server.infra.injection_guard import get_injection_guard


router = APIRouter(prefix="/api/guard", tags=["Guard"])


class GuardRequest(BaseModel):
    """Guard evaluation request"""

    input_text: str


class GuardResponse(BaseModel):
    """Guard evaluation response"""

    score: float
    malicious: bool
    threshold: float
    hits: list[dict[str, Any]]


@router.post("/evaluations", response_model=GuardResponse)
async def api_guard_evaluate(request: GuardRequest) -> GuardResponse:
    """Evaluate input for injection attacks"""
    guard = get_injection_guard()
    evaluation = await guard.evaluate(request.input_text)

    return GuardResponse(
        score=evaluation.score,
        malicious=evaluation.malicious,
        threshold=evaluation.threshold,
        hits=[{"id": h.id, "weight": h.weight} for h in evaluation.hits],
    )


@router.post("/checks")
async def api_guard_is_malicious(request: GuardRequest) -> dict[str, bool]:
    """Quick malicious check"""
    guard = get_injection_guard()
    result = await guard.is_malicious(request.input_text)
    return {"malicious": result}
