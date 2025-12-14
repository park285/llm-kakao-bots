"""Turtle Soup prompt loader and templates."""

import logging
from functools import lru_cache
from pathlib import Path

from mcp_llm_server.utils.prompts import load_yaml_dir


log = logging.getLogger(__name__)

# 프롬프트 디렉토리
PROMPTS_DIR = Path(__file__).parent / "prompts"


@lru_cache(maxsize=1)
def _load_prompts() -> dict[str, dict[str, str]]:
    """Load all prompt YAML files."""
    prompts = load_yaml_dir(PROMPTS_DIR)
    log.info("TURTLE_SOUP_PROMPTS_LOADED count=%d", len(prompts))
    return prompts


class TurtleSoupPrompts:
    """Prompt templates for Turtle Soup game."""

    def __init__(self) -> None:
        self._prompts = _load_prompts()

    def answer_system(self) -> str:
        """Get system prompt for answering player questions."""
        return self._prompts.get("answer", {}).get("system", "")

    def answer_user(self, puzzle_toon: str, question: str, history: str = "") -> str:
        """Format user prompt for question answering."""
        template = self._prompts.get("answer", {}).get("user", "{puzzle}\n{question}")
        return template.format(puzzle=puzzle_toon, question=question, history=history)

    def hint_system(self) -> str:
        """Get system prompt for hint generation."""
        return self._prompts.get("hint", {}).get("system", "")

    def hint_user(self, puzzle_toon: str, level: int) -> str:
        """Format user prompt for hint generation."""
        template = self._prompts.get("hint", {}).get("user", "{puzzle}\n{level}")
        return template.format(puzzle=puzzle_toon, level=level)

    def validate_system(self) -> str:
        """Get system prompt for solution validation."""
        return self._prompts.get("validate", {}).get("system", "")

    def validate_user(self, solution: str, player_answer: str) -> str:
        """Format user prompt for validation."""
        template = self._prompts.get("validate", {}).get(
            "user", "{solution}\n{player_answer}"
        )
        return template.format(solution=solution, player_answer=player_answer)

    def reveal_system(self) -> str:
        """Get system prompt for solution reveal."""
        return self._prompts.get("reveal", {}).get("system", "")

    def reveal_user(self, puzzle_toon: str) -> str:
        """Format user prompt for reveal."""
        template = self._prompts.get("reveal", {}).get("user", "{puzzle}")
        return template.format(puzzle=puzzle_toon)

    def generate_system(self) -> str:
        """Get system prompt for puzzle generation."""
        return self._prompts.get("generate", {}).get("system", "")

    def generate_user(
        self, category: str, difficulty: int, theme: str, examples: str = ""
    ) -> str:
        """Format user prompt for puzzle generation."""
        template = self._prompts.get("generate", {}).get(
            "user",
            "카테고리: {category}, 난이도: {difficulty}, 테마: {theme}\n{examples}",
        )
        return template.format(
            category=category,
            difficulty=difficulty,
            theme=theme,
            examples=examples,
        )

    def rewrite_system(self) -> str:
        """Get system prompt for scenario rewriting."""
        return self._prompts.get("rewrite", {}).get("system", "")

    def rewrite_user(
        self, title: str, scenario: str, solution: str, difficulty: int
    ) -> str:
        """Format user prompt for rewriting."""
        template = self._prompts.get("rewrite", {}).get(
            "user",
            "제목: {title}\n원본 시나리오: {scenario}\n정답: {solution}\n난이도: {difficulty}",
        )
        return template.format(
            title=title, scenario=scenario, solution=solution, difficulty=difficulty
        )


@lru_cache(maxsize=1)
def get_turtle_soup_prompts() -> TurtleSoupPrompts:
    """Get singleton prompts instance."""
    return TurtleSoupPrompts()
