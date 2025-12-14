"""NLP routes."""

from typing import Any

from fastapi import APIRouter
from pydantic import BaseModel

from mcp_llm_server.infra.korean_nlp import get_korean_nlp_service


router = APIRouter(prefix="/api/nlp", tags=["NLP"])


class NlpRequest(BaseModel):
    """NLP analysis request"""

    text: str


@router.post("/analyses")
async def api_nlp_analyze(request: NlpRequest) -> list[dict[str, Any]]:
    """Korean morphological analysis"""
    service = get_korean_nlp_service()
    tokens = await service.analyze_async(request.text)
    return [
        {
            "form": t.form,
            "tag": t.tag,
            "position": t.position,
            "length": t.length,
        }
        for t in tokens
    ]


@router.post("/anomaly-scores")
async def api_nlp_anomaly_score(request: NlpRequest) -> dict[str, float]:
    """Calculate anomaly score"""
    service = get_korean_nlp_service()
    score = await service.calculate_anomaly_score_async(request.text)
    return {"score": score}


@router.post("/heuristics")
async def api_nlp_heuristics(request: NlpRequest) -> dict[str, bool]:
    """Analyze text for heuristics"""
    service = get_korean_nlp_service()
    heuristics = await service.analyze_heuristics_async(request.text)
    return {
        "numeric_quantifier": heuristics.numeric_quantifier,
        "unit_noun": heuristics.unit_noun,
        "boundary_ref": heuristics.boundary_ref,
        "comparison_word": heuristics.comparison_word,
    }
