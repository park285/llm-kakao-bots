"""Unicode constants and utilities for Korean text processing"""

import re
from typing import Final


class UnicodeConstants:
    """Unicode code point ranges for Korean and emoji detection"""

    # Hangul Jamo ranges
    HANGUL_JAMO_START: Final[int] = 0x1100
    HANGUL_JAMO_END: Final[int] = 0x11FF

    HANGUL_COMPATIBILITY_JAMO_START: Final[int] = 0x3130
    HANGUL_COMPATIBILITY_JAMO_END: Final[int] = 0x318F

    HANGUL_JAMO_EXTENDED_A_START: Final[int] = 0xA960
    HANGUL_JAMO_EXTENDED_A_END: Final[int] = 0xA97F

    HANGUL_JAMO_EXTENDED_B_START: Final[int] = 0xD7B0
    HANGUL_JAMO_EXTENDED_B_END: Final[int] = 0xD7FF

    # Hangul Syllables
    HANGUL_SYLLABLES_START: Final[int] = 0xAC00
    HANGUL_SYLLABLES_END: Final[int] = 0xD7A3

    # Emoji ranges
    ZERO_WIDTH_JOINER: Final[int] = 0x200D
    EMOJI_RANGES: Final[list[range]] = [
        range(0x1F600, 0x1F64F + 1),  # Emoticons
        range(0x1F300, 0x1F5FF + 1),  # Misc Symbols and Pictographs
        range(0x1F680, 0x1F6FF + 1),  # Transport and Map
        range(0x1F1E0, 0x1F1FF + 1),  # Flags
        range(0x2600, 0x26FF + 1),  # Misc symbols
        range(0x2700, 0x27BF + 1),  # Dingbats
        range(0xFE00, 0xFE0F + 1),  # Variation Selectors
        range(0x1F900, 0x1F9FF + 1),  # Supplemental Symbols and Pictographs
        range(0x1FA00, 0x1FA6F + 1),  # Chess Symbols
        range(0x1FA70, 0x1FAFF + 1),  # Symbols and Pictographs Extended-A
    ]


def build_jamo_pattern() -> str:
    """Build regex character class for all Hangul Jamo blocks"""
    ranges = [
        (UnicodeConstants.HANGUL_JAMO_START, UnicodeConstants.HANGUL_JAMO_END),
        (
            UnicodeConstants.HANGUL_COMPATIBILITY_JAMO_START,
            UnicodeConstants.HANGUL_COMPATIBILITY_JAMO_END,
        ),
        (
            UnicodeConstants.HANGUL_JAMO_EXTENDED_A_START,
            UnicodeConstants.HANGUL_JAMO_EXTENDED_A_END,
        ),
        (
            UnicodeConstants.HANGUL_JAMO_EXTENDED_B_START,
            UnicodeConstants.HANGUL_JAMO_EXTENDED_B_END,
        ),
    ]
    parts = [f"{chr(start)}-{chr(end)}" for start, end in ranges]
    return "[" + "".join(parts) + "]"


# Pre-compiled regex patterns for jamo detection
JAMO_BLOCK_REGEX: Final[re.Pattern[str]] = re.compile(build_jamo_pattern())

# Build jamo-only pattern with whitespace, punctuation, and digits
# (Python re doesn't support \p{P} \p{N}, so we use explicit classes)
_PUNCT_DIGITS = r"\s\d!\"#$%&'()*+,\-./:;<=>?@\[\\\]^_`{|}~"
JAMO_ONLY_REGEX: Final[re.Pattern[str]] = re.compile(
    r"^[" + _PUNCT_DIGITS + build_jamo_pattern()[1:-1] + r"]+$", re.UNICODE
)


def is_emoji_codepoint(cp: int) -> bool:
    """Check if a codepoint is an emoji"""
    if cp == UnicodeConstants.ZERO_WIDTH_JOINER:
        return True
    return any(cp in r for r in UnicodeConstants.EMOJI_RANGES)


def contains_emoji(text: str) -> bool:
    """Check if text contains any emoji characters"""
    return any(is_emoji_codepoint(ord(c)) for c in text)


def is_jamo_only(text: str) -> bool:
    """Check if text contains only Hangul Jamo (no complete syllables) - potential attack vector"""
    t = text.strip()
    if not t:
        return False
    return bool(JAMO_BLOCK_REGEX.search(t)) and bool(JAMO_ONLY_REGEX.match(t))
