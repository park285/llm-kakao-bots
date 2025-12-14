"""Text processing utilities"""

import unicodedata
from typing import Final


# Control character types to strip
STRIP_CHAR_TYPES: Final[set[str]] = {"Cf", "Cc"}  # Format, Control


def normalize_nfkc(text: str) -> str:
    """Apply NFKC normalization to text"""
    return unicodedata.normalize("NFKC", text)


def strip_control_chars(text: str) -> str:
    """Remove format and control characters from text"""
    return "".join(c for c in text if unicodedata.category(c) not in STRIP_CHAR_TYPES)


def normalize_text(text: str, normalizers: list[str] | None = None) -> str:
    """Apply specified normalizers to text

    Args:
        text: Input text
        normalizers: List of normalizer names ["nfkc", "strip_zero_width"]
    """
    if normalizers is None:
        normalizers = ["nfkc", "strip_zero_width"]

    result = text
    for norm in normalizers:
        if norm == "nfkc":
            result = normalize_nfkc(result)
        elif norm == "strip_zero_width":
            result = strip_control_chars(result)

    return result
