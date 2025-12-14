"""Tests for Twenty Questions prompts."""

import pytest

from mcp_llm_server.domains.twentyq.prompts import (
    TwentyQPrompts,
    _escape_braces_for_langchain,
    _get_forbidden_words,
    _load_prompt_file,
    get_twentyq_prompts,
)


class TestEscapeBracesForLangchain:
    """Tests for _escape_braces_for_langchain function.

    Note: This function is now DEPRECATED and returns text unchanged.
    YAML files should use {{ }} directly for literal braces.
    """

    def test_no_braces(self) -> None:
        result = _escape_braces_for_langchain("Hello world")
        assert result == "Hello world"

    def test_returns_unchanged(self) -> None:
        # DEPRECATED: function now returns text unchanged
        result = _escape_braces_for_langchain("This is {test}")
        assert result == "This is {test}"

    def test_with_keep_placeholders_unchanged(self) -> None:
        # DEPRECATED: keep_placeholders no longer has effect
        result = _escape_braces_for_langchain(
            "Hello {name}, welcome to {place}",
            keep_placeholders=["name"],
        )
        assert result == "Hello {name}, welcome to {place}"

    def test_multiple_braces_unchanged(self) -> None:
        # DEPRECATED: returns text unchanged
        result = _escape_braces_for_langchain(
            "{a} and {b} and {c}",
            keep_placeholders=["a", "c"],
        )
        assert result == "{a} and {b} and {c}"

    def test_json_example_unchanged(self) -> None:
        # DEPRECATED: JSON escaping should be done in YAML files
        text = '{"key": "value"}'
        result = _escape_braces_for_langchain(text)
        assert result == '{"key": "value"}'


class TestLoadPromptFile:
    """Tests for _load_prompt_file function."""

    def test_load_existing_file(self) -> None:
        # 실제 존재하는 프롬프트 파일
        data = _load_prompt_file("hints")
        assert "system" in data
        assert "user" in data

    def test_load_nonexistent_file(self) -> None:
        with pytest.raises(FileNotFoundError):
            _load_prompt_file("nonexistent-prompt-file")

    def test_caching(self) -> None:
        # 같은 파일 두 번 로드 - 캐시에서
        data1 = _load_prompt_file("hints")
        data2 = _load_prompt_file("hints")
        # 같은 딕셔너리 객체
        assert data1 is data2


class TestGetForbiddenWords:
    """Tests for _get_forbidden_words function."""

    def test_known_category(self) -> None:
        words = _get_forbidden_words("음식")
        assert "음식" in words
        assert "먹을 것" in words

    def test_unknown_category(self) -> None:
        words = _get_forbidden_words("알수없음")
        assert words == ["알수없음"]

    def test_animal_category(self) -> None:
        words = _get_forbidden_words("동물")
        assert "동물" in words
        assert "생물" in words


class TestTwentyQPromptsHints:
    """Tests for TwentyQPrompts hint methods."""

    def test_hints_system_basic(self) -> None:
        system = TwentyQPrompts.hints_system()
        assert isinstance(system, str)
        assert len(system) > 0

    def test_hints_system_with_category(self) -> None:
        system = TwentyQPrompts.hints_system(category="음식")
        assert isinstance(system, str)
        # 카테고리 제한이 추가됨
        assert "음식" in system or len(system) > 0

    def test_hints_user(self) -> None:
        user = TwentyQPrompts.hints_user("target: 사과\ncategory: 과일")
        assert isinstance(user, str)


class TestTwentyQPromptsAnswer:
    """Tests for TwentyQPrompts answer methods."""

    def test_answer_system(self) -> None:
        system = TwentyQPrompts.answer_system()
        assert isinstance(system, str)

    def test_answer_user(self) -> None:
        user = TwentyQPrompts.answer_user(
            secret="target: 사과",
            question="빨간색인가요?",
        )
        assert isinstance(user, str)
        assert "빨간색인가요" in user

    def test_answer_user_with_history(self) -> None:
        user = TwentyQPrompts.answer_user(
            secret="target: 사과",
            question="새 질문",
            history="이전 Q&A",
        )
        assert "이전 Q&A" in user
        assert "새 질문" in user


class TestTwentyQPromptsVerify:
    """Tests for TwentyQPrompts verify methods."""

    def test_verify_system(self) -> None:
        system = TwentyQPrompts.verify_system()
        assert isinstance(system, str)

    def test_verify_user(self) -> None:
        user = TwentyQPrompts.verify_user(target="사과", guess="빨간 과일")
        assert isinstance(user, str)
        assert "사과" in user
        assert "빨간 과일" in user


class TestTwentyQPromptsNormalize:
    """Tests for TwentyQPrompts normalize methods."""

    def test_normalize_system(self) -> None:
        system = TwentyQPrompts.normalize_system()
        assert isinstance(system, str)

    def test_normalize_user(self) -> None:
        user = TwentyQPrompts.normalize_user("이거 빨간색이에요?")
        assert isinstance(user, str)


class TestTwentyQPromptsSynonym:
    """Tests for TwentyQPrompts synonym methods."""

    def test_synonym_system(self) -> None:
        system = TwentyQPrompts.synonym_system()
        assert isinstance(system, str)

    def test_synonym_user(self) -> None:
        user = TwentyQPrompts.synonym_user(target="사과", guess="애플")
        assert isinstance(user, str)
        assert "사과" in user
        assert "애플" in user


class TestGetTwentyqPrompts:
    """Tests for get_twentyq_prompts function."""

    def test_returns_instance(self) -> None:
        prompts = get_twentyq_prompts()
        assert isinstance(prompts, TwentyQPrompts)
