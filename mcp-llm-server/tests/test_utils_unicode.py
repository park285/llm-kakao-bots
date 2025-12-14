"""Tests for Unicode utilities."""

from mcp_llm_server.utils.unicode import (
    JAMO_BLOCK_REGEX,
    JAMO_ONLY_REGEX,
    UnicodeConstants,
    build_jamo_pattern,
    contains_emoji,
    is_emoji_codepoint,
    is_jamo_only,
)


class TestUnicodeConstants:
    """Tests for UnicodeConstants class."""

    def test_hangul_jamo_range(self) -> None:
        assert UnicodeConstants.HANGUL_JAMO_START == 0x1100
        assert UnicodeConstants.HANGUL_JAMO_END == 0x11FF

    def test_hangul_syllables_range(self) -> None:
        assert UnicodeConstants.HANGUL_SYLLABLES_START == 0xAC00
        assert UnicodeConstants.HANGUL_SYLLABLES_END == 0xD7A3

    def test_zero_width_joiner(self) -> None:
        assert UnicodeConstants.ZERO_WIDTH_JOINER == 0x200D

    def test_emoji_ranges_not_empty(self) -> None:
        assert len(UnicodeConstants.EMOJI_RANGES) > 0


class TestBuildJamoPattern:
    """Tests for build_jamo_pattern function."""

    def test_returns_bracket_pattern(self) -> None:
        pattern = build_jamo_pattern()
        assert pattern.startswith("[")
        assert pattern.endswith("]")

    def test_matches_jamo(self) -> None:
        import re

        pattern = re.compile(build_jamo_pattern())
        # 호환 자모
        assert pattern.search("ㄱ")
        assert pattern.search("ㄴ")
        assert pattern.search("ㅏ")


class TestIsEmojiCodepoint:
    """Tests for is_emoji_codepoint function."""

    def test_emoticon(self) -> None:
        # 😀 U+1F600
        assert is_emoji_codepoint(0x1F600)

    def test_zero_width_joiner(self) -> None:
        # ZWJ is considered emoji (for emoji sequences)
        assert is_emoji_codepoint(0x200D)

    def test_regular_ascii(self) -> None:
        assert not is_emoji_codepoint(ord("a"))
        assert not is_emoji_codepoint(ord("1"))

    def test_hangul(self) -> None:
        assert not is_emoji_codepoint(ord("가"))
        assert not is_emoji_codepoint(ord("ㄱ"))

    def test_misc_symbol(self) -> None:
        # ☀ U+2600
        assert is_emoji_codepoint(0x2600)


class TestContainsEmoji:
    """Tests for contains_emoji function."""

    def test_with_emoji(self) -> None:
        assert contains_emoji("hello 😀")
        assert contains_emoji("🎉")

    def test_without_emoji(self) -> None:
        assert not contains_emoji("hello")
        assert not contains_emoji("한글")
        assert not contains_emoji("123!@#")

    def test_empty_string(self) -> None:
        assert not contains_emoji("")

    def test_emoji_sequence(self) -> None:
        # 👨‍👩‍👧 (family emoji with ZWJ)
        assert contains_emoji("👨‍👩‍👧")


class TestIsJamoOnly:
    """Tests for is_jamo_only function."""

    def test_jamo_only(self) -> None:
        # 자모만 있는 텍스트
        assert is_jamo_only("ㄱㄴㄷ")
        assert is_jamo_only("ㅏㅓㅗ")

    def test_jamo_with_punctuation(self) -> None:
        # 자모 + 구두점/공백/숫자
        assert is_jamo_only("ㄱㄴㄷ 123!")

    def test_syllables_not_jamo_only(self) -> None:
        # 완성형 음절이 있으면 False
        assert not is_jamo_only("가나다")
        assert not is_jamo_only("ㄱ가ㄴ")

    def test_mixed_content(self) -> None:
        assert not is_jamo_only("ㄱㄴㄷ hello")  # ASCII 포함
        assert not is_jamo_only("안녕 ㄱㄴㄷ")  # 음절 포함

    def test_empty_string(self) -> None:
        assert not is_jamo_only("")

    def test_whitespace_only(self) -> None:
        assert not is_jamo_only("   ")


class TestJamoRegexPatterns:
    """Tests for pre-compiled regex patterns."""

    def test_jamo_block_regex_matches_jamo(self) -> None:
        assert JAMO_BLOCK_REGEX.search("ㄱ")
        assert JAMO_BLOCK_REGEX.search("ㅏ")

    def test_jamo_block_regex_no_match_syllable(self) -> None:
        # 완성형 음절은 자모 블록이 아님
        result = JAMO_BLOCK_REGEX.search("가")
        assert result is None

    def test_jamo_only_regex_full_match(self) -> None:
        assert JAMO_ONLY_REGEX.match("ㄱㄴㄷ")
        assert JAMO_ONLY_REGEX.match("ㄱ ㄴ ㄷ 123")

    def test_jamo_only_regex_no_match_with_syllable(self) -> None:
        assert not JAMO_ONLY_REGEX.match("가나다")
