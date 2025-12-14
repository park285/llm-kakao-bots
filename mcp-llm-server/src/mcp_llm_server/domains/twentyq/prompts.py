"""Twenty Questions prompt loader and manager."""

import logging
from functools import lru_cache
from pathlib import Path

from mcp_llm_server.utils.prompts import load_yaml_mapping


log = logging.getLogger(__name__)

PROMPTS_DIR = Path(__file__).parent / "prompts"


def _escape_braces_for_langchain(
    text: str, keep_placeholders: list[str] | None = None
) -> str:
    """DEPRECATED: No longer needed - YAML files now use escaped braces.

    Kept for backward compatibility but simply returns text unchanged.
    JSON examples in YAML use {{ }} which LangChain treats as literal braces.
    """
    # 이제 YAML 파일에서 직접 {{ }} 사용하므로 이스케이프 불필요
    return text


@lru_cache(maxsize=16)
def _load_prompt_file(name: str) -> dict[str, str]:
    """Load a prompt YAML file."""
    path = PROMPTS_DIR / f"{name}.yml"

    log.debug("PROMPT_LOADED name=%s", name)
    return load_yaml_mapping(path)


class TwentyQPrompts:
    """Prompt templates for Twenty Questions game.

    Uses LangChain PromptTemplate for variable substitution.
    YAML files use {{ }} for literal braces in JSON examples.
    """

    # hint generation
    @staticmethod
    def hints_system(category: str | None = None) -> str:
        """Get system prompt for hint generation."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("hints")
        system = data["system"]

        if category:
            restriction = data.get("category_restriction", "")
            if restriction:
                forbidden = _get_forbidden_words(category)
                # LangChain PromptTemplate으로 변수 치환
                template = PromptTemplate.from_template(
                    restriction,
                    template_format="f-string",
                )
                restriction = template.format(
                    selectedCategory=category,
                    forbiddenWords=", ".join(forbidden),
                )
                system = system + "\n\n" + restriction

        return system

    @staticmethod
    def hints_user(secret: str) -> str:
        """Get user prompt for hint generation."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("hints")
        template = PromptTemplate.from_template(
            data["user"], template_format="f-string"
        )
        return template.format(toon=secret)

    # answer (yes/no scale)
    @staticmethod
    def answer_system() -> str:
        """Get system prompt for answering questions."""
        data = _load_prompt_file("answer")
        return data["system"]

    @staticmethod
    def answer_user(secret: str, question: str, history: str = "") -> str:
        """Get user prompt for answering questions."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("answer")
        template = PromptTemplate.from_template(
            data["user"], template_format="f-string"
        )
        result = template.format(toon=secret, question=question)
        if history:
            result = history + "\n\n" + result
        return result

    # verify guess
    @staticmethod
    def verify_system() -> str:
        """Get system prompt for verifying guesses."""
        data = _load_prompt_file("verify-answer")
        return data["system"]

    @staticmethod
    def verify_user(target: str, guess: str) -> str:
        """Get user prompt for verifying guesses."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("verify-answer")
        template = PromptTemplate.from_template(
            data["user"], template_format="f-string"
        )
        return template.format(target=target, guess=guess)

    # normalize question
    @staticmethod
    def normalize_system() -> str:
        """Get system prompt for normalizing questions."""
        data = _load_prompt_file("normalize")
        return data["system"]

    @staticmethod
    def normalize_user(question: str) -> str:
        """Get user prompt for normalizing questions."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("normalize")
        template = PromptTemplate.from_template(
            data["user"], template_format="f-string"
        )
        return template.format(question=question)

    # synonym check
    @staticmethod
    def synonym_system() -> str:
        """Get system prompt for synonym checking."""
        data = _load_prompt_file("synonym-check")
        return data["system"]

    @staticmethod
    def synonym_user(target: str, guess: str) -> str:
        """Get user prompt for synonym checking."""
        from langchain_core.prompts import PromptTemplate

        data = _load_prompt_file("synonym-check")
        template = PromptTemplate.from_template(
            data["user"], template_format="f-string"
        )
        return template.format(target=target, guess=guess)


def _get_forbidden_words(category: str) -> list[str]:
    """Get forbidden words for a category."""
    # 카테고리별 금지 단어 (정답 직접 노출 방지)
    category_forbidden: dict[str, list[str]] = {
        "음식": ["음식", "먹을 것", "식품"],
        "동물": ["동물", "생물", "생명체"],
        "사물": ["사물", "물건", "도구"],
        "장소": ["장소", "곳", "위치"],
        "인물": ["인물", "사람", "인간"],
        "개념": ["개념", "추상적", "관념"],
    }
    return category_forbidden.get(category, [category])


def get_twentyq_prompts() -> TwentyQPrompts:
    """Get the prompt manager instance."""
    return TwentyQPrompts()
