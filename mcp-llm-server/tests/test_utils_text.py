"""Tests for text processing utilities."""

from mcp_llm_server.utils.text import (
    STRIP_CHAR_TYPES,
    normalize_nfkc,
    normalize_text,
    strip_control_chars,
)


FULLWIDTH_ABC = "\uff21\uff22\uff23"
FULLWIDTH_123 = "\uff11\uff12\uff13"
FULLWIDTH_ABC_ZERO_WIDTH = "\uff21\u200b\uff22\uff23"
FULLWIDTH_AB_ZERO_WIDTH = "\uff21\u200b\uff22"


class TestNormalizeNfkc:
    """Tests for normalize_nfkc function."""

    def test_basic_ascii(self) -> None:
        assert normalize_nfkc("hello") == "hello"

    def test_fullwidth_to_ascii(self) -> None:
        # fullwidth letters -> ASCII
        assert normalize_nfkc(FULLWIDTH_ABC) == "ABC"
        assert normalize_nfkc(FULLWIDTH_123) == "123"

    def test_hangul_compatibility(self) -> None:
        # compatibility jamo -> first/last jamo (NFKC 변환)
        result = normalize_nfkc("ㄱㄴㄷ")
        # 호환 자모(U+3131~)가 첫가끝 자모(U+1100~)로 변환됨
        assert result == "ᄀᄂᄃ"

    def test_empty_string(self) -> None:
        assert normalize_nfkc("") == ""

    def test_mixed_content(self) -> None:
        # fullwidth -> ASCII, compatibility jamo -> first/last jamo
        assert normalize_nfkc(f"{FULLWIDTH_ABC} hello ㄱㄴㄷ") == "ABC hello ᄀᄂᄃ"


class TestStripControlChars:
    """Tests for strip_control_chars function."""

    def test_basic_text(self) -> None:
        assert strip_control_chars("hello") == "hello"

    def test_remove_zero_width_space(self) -> None:
        # zero-width space (U+200B)
        assert strip_control_chars("hel\u200blo") == "hello"

    def test_remove_zero_width_joiner(self) -> None:
        # zero-width joiner (U+200D)
        assert strip_control_chars("a\u200db") == "ab"

    def test_remove_soft_hyphen(self) -> None:
        # soft hyphen (U+00AD)
        assert strip_control_chars("soft\u00adhyphen") == "softhyphen"

    def test_preserve_regular_chars(self) -> None:
        # 일반 문자는 보존
        assert strip_control_chars("한글 English 123!@#") == "한글 English 123!@#"

    def test_empty_string(self) -> None:
        assert strip_control_chars("") == ""

    def test_multiple_control_chars(self) -> None:
        text = "\u200b\u200c\u200dhello\u200bworld"
        result = strip_control_chars(text)
        assert result == "helloworld"


class TestNormalizeText:
    """Tests for normalize_text function."""

    def test_default_normalizers(self) -> None:
        # 기본: nfkc + strip_zero_width
        text = FULLWIDTH_ABC_ZERO_WIDTH
        result = normalize_text(text)
        assert result == "ABC"

    def test_nfkc_only(self) -> None:
        text = FULLWIDTH_ABC_ZERO_WIDTH
        result = normalize_text(text, normalizers=["nfkc"])
        # nfkc만 적용, zero-width는 남음
        assert result == "A\u200bBC"

    def test_strip_only(self) -> None:
        text = FULLWIDTH_ABC_ZERO_WIDTH
        result = normalize_text(text, normalizers=["strip_zero_width"])
        # strip만 적용, fullwidth는 남음
        assert result == FULLWIDTH_ABC

    def test_empty_normalizers(self) -> None:
        text = FULLWIDTH_ABC_ZERO_WIDTH
        result = normalize_text(text, normalizers=[])
        assert result == text

    def test_unknown_normalizer_ignored(self) -> None:
        text = "hello"
        result = normalize_text(text, normalizers=["unknown", "nfkc"])
        assert result == "hello"

    def test_order_matters(self) -> None:
        # 순서 테스트
        text = FULLWIDTH_AB_ZERO_WIDTH
        r1 = normalize_text(text, normalizers=["nfkc", "strip_zero_width"])
        r2 = normalize_text(text, normalizers=["strip_zero_width", "nfkc"])
        assert r1 == r2 == "AB"


class TestStripCharTypes:
    """Tests for STRIP_CHAR_TYPES constant."""

    def test_contains_format(self) -> None:
        assert "Cf" in STRIP_CHAR_TYPES

    def test_contains_control(self) -> None:
        assert "Cc" in STRIP_CHAR_TYPES

    def test_is_set(self) -> None:
        assert isinstance(STRIP_CHAR_TYPES, set)
