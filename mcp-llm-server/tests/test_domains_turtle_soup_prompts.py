"""Tests for turtle_soup domain prompts."""

from mcp_llm_server.domains.turtle_soup.prompts import (
    get_turtle_soup_prompts,
)
from mcp_llm_server.utils.toon import encode_puzzle


class TestTurtleSoupPrompts:
    """Tests for TurtleSoupPrompts class."""

    def test_singleton(self) -> None:
        prompts1 = get_turtle_soup_prompts()
        prompts2 = get_turtle_soup_prompts()
        assert prompts1 is prompts2

    def test_answer_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.answer_system()
        assert len(system) > 0
        assert "Game Master" in system or "SECURITY" in system

    def test_answer_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        puzzle_toon = encode_puzzle("남자가 스프를 먹었다", "바다거북 스프")
        user = prompts.answer_user(
            puzzle_toon=puzzle_toon,
            question="스프가 중요한가요?",
            history="Q: 이전 질문\nA: 예",
        )
        # TOON-encoded puzzle is included
        assert "남자가 스프를 먹었다" in user or "scenario" in user
        assert "스프가 중요한가요?" in user

    def test_hint_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.hint_system()
        assert len(system) > 0
        assert "hint" in system.lower() or "SECURITY" in system

    def test_hint_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        puzzle_toon = encode_puzzle("시나리오", "솔루션")
        user = prompts.hint_user(
            puzzle_toon=puzzle_toon,
            level=2,
        )
        # TOON-encoded puzzle is included
        assert "시나리오" in user or "scenario" in user
        assert "2" in user

    def test_validate_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.validate_system()
        assert len(system) > 0
        assert "YES" in system or "NO" in system or "CLOSE" in system

    def test_validate_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        user = prompts.validate_user(
            solution="정답입니다",
            player_answer="플레이어 답변",
        )
        assert "정답입니다" in user
        assert "플레이어 답변" in user

    def test_reveal_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.reveal_system()
        assert len(system) > 0

    def test_reveal_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        puzzle_toon = encode_puzzle("시나리오", "솔루션")
        user = prompts.reveal_user(puzzle_toon=puzzle_toon)
        # TOON-encoded puzzle is included
        assert "시나리오" in user or "scenario" in user

    def test_generate_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.generate_system()
        assert len(system) > 0
        assert "Lateral Thinking" in system or "Turtle Soup" in system

    def test_generate_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        user = prompts.generate_user(
            category="MYSTERY",
            difficulty=3,
            theme="공포",
        )
        assert "MYSTERY" in user
        assert "3" in user
        assert "공포" in user
        assert "난이도는 반드시 정수 3" in user

    def test_generate_user_includes_examples(self) -> None:
        prompts = get_turtle_soup_prompts()
        user = prompts.generate_user(
            category="MYSTERY",
            difficulty=2,
            theme="테마",
            examples="- 제목: 예시\n  시나리오: 내용\n  정답: 해답\n  난이도: 2",
        )
        assert "예시" in user
        assert "시나리오" in user

    def test_rewrite_system_not_empty(self) -> None:
        prompts = get_turtle_soup_prompts()
        system = prompts.rewrite_system()
        assert len(system) > 0
        assert "SECURITY" in system or "rewrite" in system.lower()

    def test_rewrite_user_formatting(self) -> None:
        prompts = get_turtle_soup_prompts()
        user = prompts.rewrite_user(
            title="제목",
            scenario="원본",
            solution="솔루션",
            difficulty=2,
        )
        assert "제목" in user
        assert "원본" in user
        assert "솔루션" in user
        assert "2" in user
