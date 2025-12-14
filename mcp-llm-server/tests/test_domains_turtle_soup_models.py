"""Tests for turtle_soup domain models."""

import pytest

from mcp_llm_server.domains.turtle_soup.models import (
    AnswerQuestionResponse,
    AnswerType,
    GeneratePuzzleResponse,
    HintOutput,
    HintResponse,
    PuzzleCategory,
    PuzzleOutput,
    RevealResponse,
    RewriteScenarioResponse,
    ValidationResponse,
    ValidationResult,
    format_answer_text,
)


class TestValidationResult:
    """Tests for ValidationResult enum."""

    def test_values(self) -> None:
        assert ValidationResult.YES.value == "YES"
        assert ValidationResult.NO.value == "NO"
        assert ValidationResult.CLOSE.value == "CLOSE"

    def test_from_text_exact(self) -> None:
        assert ValidationResult.from_text("YES") == ValidationResult.YES
        assert ValidationResult.from_text("NO") == ValidationResult.NO
        assert ValidationResult.from_text("CLOSE") == ValidationResult.CLOSE

    def test_from_text_with_whitespace(self) -> None:
        assert ValidationResult.from_text("  YES  ") == ValidationResult.YES
        assert ValidationResult.from_text("\nNO\n") == ValidationResult.NO

    def test_from_text_lowercase(self) -> None:
        assert ValidationResult.from_text("yes") == ValidationResult.YES
        assert ValidationResult.from_text("no") == ValidationResult.NO

    def test_from_text_invalid(self) -> None:
        assert ValidationResult.from_text("INVALID") is None
        assert ValidationResult.from_text("") is None


class TestAnswerType:
    """Tests for AnswerType enum."""

    def test_values(self) -> None:
        assert AnswerType.YES.value == "예"
        assert AnswerType.NO.value == "아니오"
        assert AnswerType.IRRELEVANT.value == "관계없습니다"
        assert AnswerType.IMPORTANT.value == "중요한 질문입니다!"
        assert AnswerType.SOMEWHAT.value == "조금은 관계있습니다"
        assert AnswerType.FALSE_PREMISE.value == "전제가 틀렸습니다"
        assert AnswerType.CANNOT_ANSWER.value == "답변할 수 없습니다"

    def test_from_text_exact(self) -> None:
        assert AnswerType.from_text("예") == AnswerType.YES
        assert AnswerType.from_text("아니오") == AnswerType.NO
        assert AnswerType.from_text("관계없습니다") == AnswerType.IRRELEVANT

    def test_from_text_in_sentence(self) -> None:
        assert AnswerType.from_text("네, 예입니다") == AnswerType.YES
        assert AnswerType.from_text("그건 관계없습니다") == AnswerType.IRRELEVANT

    def test_from_text_invalid(self) -> None:
        assert AnswerType.from_text("INVALID") is None
        assert AnswerType.from_text("") is None


class TestFormatAnswerText:
    """Tests for format_answer_text helper."""

    def test_non_important_returns_base(self) -> None:
        assert format_answer_text(AnswerType.YES, False, "예, raw") == "예"

    def test_important_yes_uses_suffix(self) -> None:
        assert (
            format_answer_text(AnswerType.YES, True, "예, raw")
            == "예, 중요한 질문입니다!"
        )

    def test_important_no_uses_contrastive_phrase(self) -> None:
        assert (
            format_answer_text(AnswerType.NO, True, "아니오, raw")
            == "아니오 하지만 중요한 질문입니다!"
        )

    def test_no_answer_falls_back_to_raw(self) -> None:
        assert format_answer_text(None, True, "원문 응답") == "원문 응답"


class TestPuzzleCategory:
    """Tests for PuzzleCategory enum."""

    def test_values(self) -> None:
        assert PuzzleCategory.MYSTERY.value == "MYSTERY"
        assert PuzzleCategory.HORROR.value == "HORROR"
        assert PuzzleCategory.ABSURD.value == "ABSURD"
        assert PuzzleCategory.LOGIC.value == "LOGIC"


class TestResponseModels:
    """Tests for response Pydantic models."""

    def test_answer_question_response(self) -> None:
        response = AnswerQuestionResponse(
            answer="예", raw_text="예, 맞습니다", question_count=1
        )
        assert response.answer == "예"
        assert response.raw_text == "예, 맞습니다"
        assert response.question_count == 1
        assert response.history == []

    def test_hint_response(self) -> None:
        response = HintResponse(hint="시간에 주목하세요", level=1)
        assert response.hint == "시간에 주목하세요"
        assert response.level == 1

    def test_validation_response(self) -> None:
        response = ValidationResponse(result="YES", raw_text="YES")
        assert response.result == "YES"
        assert response.raw_text == "YES"

    def test_reveal_response(self) -> None:
        response = RevealResponse(narrative="진실은...")
        assert response.narrative == "진실은..."

    def test_generate_puzzle_response(self) -> None:
        response = GeneratePuzzleResponse(
            title="미스터리",
            scenario="남자가...",
            solution="사실은...",
            category="MYSTERY",
            difficulty=3,
            hints=["힌트1", "힌트2", "힌트3"],
            puzzle_id=42,
        )
        assert response.title == "미스터리"
        assert response.difficulty == 3
        assert len(response.hints) == 3
        assert response.puzzle_id == 42

    def test_rewrite_scenario_response(self) -> None:
        response = RewriteScenarioResponse(
            scenario="새로운 시나리오",
            solution="새로운 솔루션",
            original_scenario="원본 시나리오",
            original_solution="원본 솔루션",
        )
        assert response.scenario == "새로운 시나리오"
        assert response.solution == "새로운 솔루션"
        assert response.original_scenario == "원본 시나리오"
        assert response.original_solution == "원본 솔루션"


class TestOutputModels:
    """Tests for structured output Pydantic models."""

    def test_hint_output(self) -> None:
        output = HintOutput(hint="힌트입니다")
        assert output.hint == "힌트입니다"

    def test_puzzle_output(self) -> None:
        output = PuzzleOutput(
            title="제목",
            scenario="시나리오",
            solution="솔루션",
            category=PuzzleCategory.MYSTERY,
            difficulty=3,
            hints=["힌트1", "힌트2", "힌트3"],
        )
        assert output.title == "제목"
        assert output.category == PuzzleCategory.MYSTERY
        assert output.difficulty == 3

    def test_puzzle_output_difficulty_validation(self) -> None:
        # difficulty는 1-5 범위
        with pytest.raises(ValueError):
            PuzzleOutput(
                title="제목",
                scenario="시나리오",
                solution="솔루션",
                category=PuzzleCategory.MYSTERY,
                difficulty=0,  # Invalid
                hints=[],
            )

        with pytest.raises(ValueError):
            PuzzleOutput(
                title="제목",
                scenario="시나리오",
                solution="솔루션",
                category=PuzzleCategory.MYSTERY,
                difficulty=6,  # Invalid
                hints=[],
            )
