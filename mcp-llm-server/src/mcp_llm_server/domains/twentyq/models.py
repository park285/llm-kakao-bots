"""Twenty Questions domain models - LangChain/Pydantic based."""

from enum import Enum

from pydantic import BaseModel, Field


class AnswerScale(str, Enum):
    """5-scale answer for yes/no questions."""

    YES = "예"
    PROBABLY_YES = "아마도 예"
    PROBABLY_NO = "아마도 아니오"
    NO = "아니오"

    @classmethod
    def from_text(cls, text: str) -> "AnswerScale | None":
        """Parse scale from LLM response text."""
        text = text.strip()
        for scale in cls:
            if scale.value in text:
                return scale
        return None


class VerifyResult(str, Enum):
    """Result of guess verification."""

    ACCEPT = "정답"
    CLOSE = "근접"
    REJECT = "오답"


class SynonymResult(str, Enum):
    """Result of synonym check."""

    EQUIVALENT = "동일"
    NOT_EQUIVALENT = "상이"


# Pydantic models for structured output (LangChain with_structured_output)


class HintsOutput(BaseModel):
    """Structured output for hint generation."""

    hints: list[str] = Field(description="List of hints in Korean")


class NormalizeOutput(BaseModel):
    """Structured output for question normalization."""

    normalized: str = Field(description="Normalized question in Korean")


class VerifyOutput(BaseModel):
    """Structured output for guess verification."""

    result: VerifyResult = Field(description="정답, 근접, or 오답")


class SynonymOutput(BaseModel):
    """Structured output for synonym check."""

    result: SynonymResult = Field(description="동일 or 상이")


class AnswerOutput(BaseModel):
    """Structured output for yes/no question answering."""

    answer: str = Field(description="One of: 예, 아마도 예, 아마도 아니오, 아니오")


# Response models (for MCP tool returns)


class HintsResponse(BaseModel):
    """Response for hint generation."""

    hints: list[str]
    thought_signature: str | None = None


class AnswerResponse(BaseModel):
    """Response for question answering."""

    scale: str | None  # AnswerScale value
    raw_text: str
    thought_signature: str | None = None


class VerifyResponse(BaseModel):
    """Response for guess verification."""

    result: str | None  # VerifyResult value
    raw_text: str


class NormalizeResponse(BaseModel):
    """Response for question normalization."""

    normalized: str
    original: str


class SynonymResponse(BaseModel):
    """Response for synonym check."""

    result: str | None  # SynonymResult value
    raw_text: str
