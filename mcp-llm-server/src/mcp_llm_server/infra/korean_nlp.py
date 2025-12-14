"""Korean NLP service using kiwipiepy

Replaces Komoran (JVM-based) with kiwipiepy (pure Python).

POS Tag Mapping (Komoran → Kiwi):
- UNK* → UN (Unknown/분석불능)
- NN* → NNG, NNP, NNB (명사류)
- VV → VV (동사)
- VA → VA (형용사)
- NR → NR (수사)

Anomaly Scoring Algorithm (ported from KomoranService.kt):
1. scoreUnknownTokens: 미지 토큰 비율
2. scoreTokenLength: 평균 토큰 길이
3. scoreIncompleteHangul: 불완전 한글 (자모만 있는 경우)
4. scoreContentRatio: 내용어 비율 (명사/동사/형용사/수사)
"""

import asyncio
import logging
import re
from collections.abc import Iterable
from dataclasses import dataclass
from threading import Lock
from typing import Final, Protocol, cast

from kiwipiepy import Kiwi


log = logging.getLogger(__name__)


class _KiwiToken(Protocol):
    form: str
    tag: str
    start: int
    length: int
    len: int


class KiwiProtocol(Protocol):
    def tokenize(self, text: str) -> Iterable[_KiwiToken]: ...


@dataclass(frozen=True)
class NlpToken:
    """Morphological analysis result token"""

    form: str  # 형태소
    tag: str  # POS 태그
    position: int  # 시작 위치
    length: int  # 길이


@dataclass(frozen=True)
class NlpHeuristics:
    """Heuristic analysis results for answer validation"""

    numeric_quantifier: bool  # 수사 포함 여부 (NR)
    unit_noun: bool  # 단위 명사 포함 여부
    boundary_ref: bool  # 경계 참조어 포함 여부
    comparison_word: bool  # 비교어 포함 여부


# 이상 탐지 임계값 상수 (KomoranService.kt와 동일)
UNKNOWN_RATIO_HIGH: Final[float] = 0.6
UNKNOWN_RATIO_MEDIUM: Final[float] = 0.4
UNKNOWN_RATIO_LOW: Final[float] = 0.2

UNKNOWN_SCORE_HIGH: Final[float] = 0.4
UNKNOWN_SCORE_MEDIUM: Final[float] = 0.3
UNKNOWN_SCORE_LOW: Final[float] = 0.1

TOKEN_LENGTH_LOW: Final[float] = 0.6
TOKEN_LENGTH_MEDIUM: Final[float] = 0.8
TOKEN_LENGTH_HIGH: Final[float] = 1.0

TOKEN_LENGTH_SCORE_HIGH: Final[float] = 0.3
TOKEN_LENGTH_SCORE_MEDIUM: Final[float] = 0.2
TOKEN_LENGTH_SCORE_LOW: Final[float] = 0.1

HANGUL_RATIO_LOW: Final[float] = 0.2
HANGUL_RATIO_MEDIUM: Final[float] = 0.4

HANGUL_SCORE_MEDIUM: Final[float] = 0.2
HANGUL_SCORE_LOW: Final[float] = 0.1

CONTENT_RATIO_THRESHOLD: Final[float] = 0.15
MIN_TOKEN_SIZE_FOR_CONTENT_CHECK: Final[int] = 3

DEFAULT_ANOMALY_SCORE: Final[float] = 0.5
EMPTY_TOKEN_ANOMALY_SCORE: Final[float] = 0.8
MIN_TEXT_LENGTH_FOR_ANOMALY: Final[int] = 3

# 패턴
INCOMPLETE_HANGUL_PATTERN: Final[re.Pattern[str]] = re.compile(r"[ㄱ-ㅎㅏ-ㅣ]{2,}")
EMOTICON_PATTERN: Final[re.Pattern[str]] = re.compile(r".*[ㅋㅎ]{2,}.*")

# 단위명사 및 경계어 세트 (KomoranService.kt와 동일)
UNIT_NOUNS: Final[frozenset[str]] = frozenset(
    {
        "글자",
        "자",
        "음절",
        "문자",
        "토큰",
        "개",
        "번",
        "번째",
        "회",
        "차례",
        "모음",
        "자음",
        "초성",
        "중성",
        "종성",
        "받침",
    }
)

BOUNDARY_WORDS: Final[frozenset[str]] = frozenset(
    {"처음", "끝", "마지막", "시작", "중간", "가운데", "초성", "중성", "종성", "받침"}
)

COMPARISON_WORDS: Final[frozenset[str]] = frozenset(
    {"이상", "이하", "초과", "미만", "넘", "이내"}
)


def _is_unknown_tag(tag: str) -> bool:
    """Check if POS tag indicates unknown token (Kiwi: UN, Komoran: UNK*)"""
    return tag == "UN" or tag.startswith("UNK")


def _is_content_tag(tag: str) -> bool:
    """Check if POS tag indicates content word (명사/동사/형용사/수사)"""
    return (
        tag.startswith("NN")  # NNG, NNP, NNB
        or tag.startswith("VV")  # 동사
        or tag.startswith("VA")  # 형용사
        or tag == "NR"  # 수사
    )


def score_unknown_tokens(tokens: list[NlpToken]) -> float:
    """Score based on unknown token ratio"""
    if not tokens:
        return 0.0

    unknown_count = sum(1 for t in tokens if _is_unknown_tag(t.tag))
    unknown_ratio = unknown_count / len(tokens)

    if unknown_ratio > UNKNOWN_RATIO_HIGH:
        return UNKNOWN_SCORE_HIGH
    if unknown_ratio > UNKNOWN_RATIO_MEDIUM:
        return UNKNOWN_SCORE_MEDIUM
    if unknown_ratio > UNKNOWN_RATIO_LOW:
        return UNKNOWN_SCORE_LOW
    return 0.0


def score_token_length(tokens: list[NlpToken]) -> float:
    """Score based on average token length (shorter = more suspicious)"""
    if not tokens:
        return 0.0

    avg_length = sum(t.length for t in tokens) / len(tokens)

    if avg_length < TOKEN_LENGTH_LOW:
        return TOKEN_LENGTH_SCORE_HIGH
    if avg_length < TOKEN_LENGTH_MEDIUM:
        return TOKEN_LENGTH_SCORE_MEDIUM
    if avg_length < TOKEN_LENGTH_HIGH:
        return TOKEN_LENGTH_SCORE_LOW
    return 0.0


def score_incomplete_hangul(text: str) -> float:
    """Score based on incomplete hangul (자모) presence"""
    if not text:
        return 0.0

    # 한글 완성형 비율 계산
    hangul_count = sum(1 for c in text if "가" <= c <= "힣")
    hangul_ratio = hangul_count / len(text) if text else 0

    # 불완전 한글 패턴 확인 (이모티콘 제외)
    has_incomplete = bool(INCOMPLETE_HANGUL_PATTERN.search(text))
    is_emoticon = bool(EMOTICON_PATTERN.match(text))

    if has_incomplete and not is_emoticon:
        if hangul_ratio < HANGUL_RATIO_LOW:
            return HANGUL_SCORE_MEDIUM
        if hangul_ratio < HANGUL_RATIO_MEDIUM:
            return HANGUL_SCORE_LOW
    return 0.0


def score_content_ratio(tokens: list[NlpToken]) -> float:
    """Score based on content word ratio (lower = more suspicious)"""
    if len(tokens) <= MIN_TOKEN_SIZE_FOR_CONTENT_CHECK:
        return 0.0

    content_count = sum(1 for t in tokens if _is_content_tag(t.tag))
    content_ratio = content_count / len(tokens)

    if content_ratio < CONTENT_RATIO_THRESHOLD:
        return CONTENT_RATIO_THRESHOLD
    return 0.0


class KoreanNlpService:
    """Korean morphological analysis service using kiwipiepy

    Provides:
    - Morphological analysis (analyze)
    - Anomaly score calculation (calculate_anomaly_score)
    - Heuristic analysis for answer validation (analyze_heuristics)
    """

    def __init__(self) -> None:
        self._kiwi: KiwiProtocol | None = None
        self._lock: Lock = Lock()

    def _get_kiwi(self) -> KiwiProtocol:
        """Lazy initialization of Kiwi instance"""
        if self._kiwi is None:
            with self._lock:
                if self._kiwi is None:
                    log.info("Initializing Kiwi Korean NLP")
                    self._kiwi = cast("KiwiProtocol", Kiwi())
                    log.info("Kiwi initialized successfully")
        return self._kiwi

    def analyze(self, text: str) -> list[NlpToken]:
        """Perform morphological analysis

        Args:
            text: Input text to analyze

        Returns:
            List of NlpToken with form, tag, position, length
        """
        if not text or not text.strip():
            return []

        try:
            kiwi = self._get_kiwi()
            result = kiwi.tokenize(text)

            return [
                NlpToken(
                    form=token.form,
                    tag=token.tag,
                    position=token.start,
                    length=token.len,
                )
                for token in result
            ]
        except Exception as e:  # noqa: BLE001 tolerate NLP lib failure
            log.error("Failed to analyze text: %s", e)
            return []

    async def analyze_async(self, text: str) -> list[NlpToken]:
        """Async wrapper for analyze to offload CPU-bound tokenization."""
        return await asyncio.to_thread(self.analyze, text)

    def calculate_anomaly_score(self, text: str) -> float:
        """Calculate anomaly score for input text

        Higher score indicates higher likelihood of injection attack.
        Score range: 0.0 ~ 1.0

        Algorithm (ported from KomoranService.kt):
        - Score unknown tokens ratio
        - Score average token length
        - Score incomplete hangul presence
        - Score content word ratio
        """
        if len(text) < MIN_TEXT_LENGTH_FOR_ANOMALY:
            return 0.0

        try:
            tokens = self.analyze(text)

            if not tokens:
                return EMPTY_TOKEN_ANOMALY_SCORE

            score = (
                score_unknown_tokens(tokens)
                + score_token_length(tokens)
                + score_incomplete_hangul(text)
                + score_content_ratio(tokens)
            )

            return min(max(score, 0.0), 1.0)  # clamp to [0, 1]

        except Exception as e:  # noqa: BLE001 - scoring should degrade gracefully
            log.warning("calculate_anomaly_score failed: %s", e)
            return DEFAULT_ANOMALY_SCORE

    async def calculate_anomaly_score_async(self, text: str) -> float:
        """Async wrapper for anomaly scoring to avoid event loop blocking."""
        return await asyncio.to_thread(self.calculate_anomaly_score, text)

    def analyze_heuristics(self, text: str) -> NlpHeuristics:
        """Analyze text for answer validation heuristics

        Returns:
            NlpHeuristics with flags for various patterns
        """
        default = NlpHeuristics(
            numeric_quantifier=False,
            unit_noun=False,
            boundary_ref=False,
            comparison_word=False,
        )

        try:
            tokens = self.analyze(text)
            if not tokens:
                return default

            forms = {t.form for t in tokens}
            tags = {t.tag for t in tokens}

            return NlpHeuristics(
                numeric_quantifier="NR" in tags,
                unit_noun=bool(forms & UNIT_NOUNS),
                boundary_ref=bool(forms & BOUNDARY_WORDS),
                comparison_word=bool(forms & COMPARISON_WORDS),
            )

        except Exception as e:  # noqa: BLE001 heuristics should degrade gracefully
            log.warning("analyze_heuristics failed: %s", e)
            return default

    async def analyze_heuristics_async(self, text: str) -> NlpHeuristics:
        """Async wrapper for heuristic analysis."""
        return await asyncio.to_thread(self.analyze_heuristics, text)


# Singleton instance
_service: KoreanNlpService | None = None


def get_korean_nlp_service() -> KoreanNlpService:
    """Get Korean NLP service singleton"""
    global _service
    if _service is None:
        _service = KoreanNlpService()
    return _service
