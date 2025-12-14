"""Turtle Soup domain models - LangChain/Pydantic based."""

from enum import Enum

from pydantic import BaseModel, Field


class ValidationResult(str, Enum):
    """Result of solution validation."""

    YES = "YES"  # 정답 맞춤
    NO = "NO"  # 오답
    CLOSE = "CLOSE"  # 근접 (핵심은 맞았으나 설명 부족)

    @classmethod
    def from_text(cls, text: str) -> "ValidationResult | None":
        """Parse result from LLM response text."""
        text = text.strip().upper()
        for result in cls:
            if result.value in text:
                return result
        return None


class AnswerType(str, Enum):
    """Types of answers for player questions."""

    YES = "예"
    NO = "아니오"
    IRRELEVANT = "관계없습니다"
    IMPORTANT = "중요한 질문입니다!"
    SOMEWHAT = "조금은 관계있습니다"
    FALSE_PREMISE = "전제가 틀렸습니다"
    CANNOT_ANSWER = "답변할 수 없습니다"

    @classmethod
    def from_text(cls, text: str) -> "AnswerType | None":
        """Parse answer type from LLM response text."""
        text = text.strip()
        base_answers: list[AnswerType] = [
            cls.YES,
            cls.NO,
            cls.IRRELEVANT,
            cls.SOMEWHAT,
            cls.FALSE_PREMISE,
            cls.CANNOT_ANSWER,
        ]
        for answer in base_answers:
            if answer.value in text:
                return answer
        if cls.IMPORTANT.value in text:
            return cls.IMPORTANT
        return None


def format_answer_text(
    answer: AnswerType | None, is_important: bool, raw_text: str
) -> str:
    """Format final answer string including importance marker."""
    if answer is None:
        return raw_text
    if not is_important:
        return answer.value
    if answer == AnswerType.NO:
        return f"{AnswerType.NO.value} 하지만 {AnswerType.IMPORTANT.value}"
    return f"{answer.value}, {AnswerType.IMPORTANT.value}"


class PuzzleCategory(str, Enum):
    """Categories for lateral thinking puzzles."""

    MYSTERY = "MYSTERY"
    HORROR = "HORROR"
    ABSURD = "ABSURD"
    LOGIC = "LOGIC"


# Pydantic models for structured output


class HintOutput(BaseModel):
    """Structured output for hint generation."""

    hint: str = Field(description="Hint in Korean")


class PuzzleOutput(BaseModel):
    """Structured output for puzzle generation."""

    title: str = Field(description="Short title (3-8 characters)")
    scenario: str = Field(description="Puzzle scenario (3-5 sentences)")
    solution: str = Field(description="Explanation of the twist")
    category: PuzzleCategory = Field(description="Puzzle category")
    difficulty: int = Field(ge=1, le=5, description="Difficulty 1-5")
    hints: list[str] = Field(description="3 progressive hints")


# Response models (for MCP tool returns)


class QuestionHistoryItem(BaseModel):
    """Single Q&A history item."""

    question: str
    answer: str


class AnswerQuestionResponse(BaseModel):
    """Response for player question answering."""

    answer: str  # AnswerType value
    raw_text: str
    question_count: int  # Total Q&A count from session history
    history: list[QuestionHistoryItem] = []  # Q&A history for display


class HintResponse(BaseModel):
    """Response for hint generation."""

    hint: str
    level: int


class ValidationResponse(BaseModel):
    """Response for solution validation."""

    result: str  # ValidationResult value
    raw_text: str


class RevealResponse(BaseModel):
    """Response for solution reveal."""

    narrative: str


class GeneratePuzzleResponse(BaseModel):
    """Response for puzzle generation."""

    title: str
    scenario: str
    solution: str
    category: str
    difficulty: int
    hints: list[str]
    puzzle_id: int | None = None


class RewriteOutput(BaseModel):
    """Structured output for rewrite generation."""

    scenario: str = Field(description="Rewritten scenario in Korean")
    solution: str = Field(description="Rewritten solution in Korean")


class RewriteScenarioResponse(BaseModel):
    """Response for scenario rewriting."""

    scenario: str
    solution: str
    original_scenario: str
    original_solution: str
