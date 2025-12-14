"""Injection Guard - Score-based prompt injection detection"""

import asyncio
import inspect
import logging
from collections.abc import Awaitable, Callable
from pathlib import Path

from cachetools import TTLCache

from mcp_llm_server.config.settings import GuardSettings, get_settings
from mcp_llm_server.infra.rulepack_loader import (
    CompiledPack,
    load_and_compile_rulepacks,
)
from mcp_llm_server.models.guard import GuardEvaluation, GuardMatch
from mcp_llm_server.utils.text import normalize_text
from mcp_llm_server.utils.unicode import contains_emoji, is_jamo_only


log = logging.getLogger(__name__)


# 임계값 상수
DEFAULT_MALICIOUS_THRESHOLD = 0.7
# ANOMALY_SCORE_THRESHOLD moved to GuardSettings.anomaly_threshold


class InjectionGuard:
    """Score-based prompt injection detection guard

    Features:
    - Regex pattern matching
    - Aho-Corasick phrase matching
    - Jamo-only detection (Korean attack vector)
    - Emoji detection
    - Morphological anomaly detection (via Korean NLP)
    - Evaluation caching
    """

    def __init__(
        self,
        settings: GuardSettings | None = None,
        anomaly_scorer: Callable[[str], float | Awaitable[float]] | None = None,
    ) -> None:
        """Initialize guard

        Args:
            settings: Guard settings (default: from config)
            anomaly_scorer: Optional async function for morphological anomaly scoring
        """
        self._settings = settings or get_settings().guard
        self._anomaly_scorer: Callable[[str], float | Awaitable[float]] | None = (
            anomaly_scorer
        )
        self._compiled: list[CompiledPack] = []
        # Cache settings from configuration
        self._cache: TTLCache[str, GuardEvaluation] = TTLCache(
            maxsize=self._settings.cache_maxsize, ttl=self._settings.cache_ttl
        )
        self._lock = asyncio.Lock()
        self._inflight: dict[str, asyncio.Task[GuardEvaluation]] = {}

        if self._settings.enabled:
            self._load_rulepacks()

    def _load_rulepacks(self) -> None:
        """Load and compile rulepacks from configured directory"""
        # 패키지 내부 rulepacks 우선 사용
        package_dir = Path(__file__).parent.parent / "rulepacks"

        rulepacks_dir = Path(self._settings.rulepacks_dir)
        if rulepacks_dir.exists() and list(rulepacks_dir.glob("*.yml")):
            # 설정된 경로에 yml 파일이 있으면 사용
            pass
        elif package_dir.exists() and list(package_dir.glob("*.yml")):
            # 패키지 내부 rulepacks 사용
            rulepacks_dir = package_dir
        else:
            log.warning("Rulepacks directory not found or empty: %s", rulepacks_dir)
            return

        self._compiled = load_and_compile_rulepacks(rulepacks_dir)
        log.info(
            "InjectionGuard initialized: packs=%d, threshold=%.2f",
            len(self._compiled),
            self._settings.threshold,
        )

    async def evaluate(self, input_text: str) -> GuardEvaluation:
        """Evaluate input for potential injection attacks

        Args:
            input_text: User input to evaluate

        Returns:
            GuardEvaluation with score, hits, threshold, and malicious flag
        """
        if not self._settings.enabled:
            return GuardEvaluation(0.0, [], float("inf"))

        cached = self._cache.get(input_text)
        if cached is not None:
            return cached

        async with self._lock:
            cached = self._cache.get(input_text)
            if cached is not None:
                return cached

            task = self._inflight.get(input_text)
            if task is None:
                task = asyncio.create_task(self._evaluate_and_cache(input_text))
                self._inflight[input_text] = task

        return await asyncio.shield(task)

    async def _evaluate_and_cache(self, input_text: str) -> GuardEvaluation:
        """Evaluate and populate cache (in-flight de-duplication)."""
        try:
            result = await self._evaluate_internal(input_text)
            self._cache[input_text] = result
            return result
        finally:
            async with self._lock:
                self._inflight.pop(input_text, None)

    async def _evaluate_internal(self, input_text: str) -> GuardEvaluation:
        """Internal evaluation logic"""
        threshold = self._get_threshold()

        # 자모만 있는 입력 차단 (한국어 공격 벡터)
        if is_jamo_only(input_text):
            log.warning("InjectionGuard JAMO_ONLY_BLOCK: %s...", input_text[:50])
            return GuardEvaluation(
                threshold,
                [GuardMatch("jamo_only", threshold)],
                threshold,
            )

        # 이모지 차단
        if contains_emoji(input_text):
            log.warning("InjectionGuard EMOJI_BLOCK: %s...", input_text[:50])
            return GuardEvaluation(
                threshold,
                [GuardMatch("emoji_detected", threshold)],
                threshold,
            )

        # 텍스트 정규화
        normalized = normalize_text(input_text)

        # 규칙팩 평가
        base_score, base_hits = self._evaluate_packs(normalized)

        # 형태소 이상 점수 (선택적)
        anomaly_score, anomaly_hit = await self._compute_anomaly(input_text)

        total_score = base_score + anomaly_score
        hits = base_hits + ([anomaly_hit] if anomaly_hit else [])

        return GuardEvaluation(total_score, hits, threshold)

    def _get_threshold(self) -> float:
        """Get effective threshold"""
        if self._settings.threshold > 0:
            return self._settings.threshold
        if self._compiled:
            return max(p.threshold for p in self._compiled)
        return DEFAULT_MALICIOUS_THRESHOLD

    def _evaluate_packs(self, text: str) -> tuple[float, list[GuardMatch]]:
        """Evaluate all compiled packs against normalized text"""
        total = 0.0
        hits: list[GuardMatch] = []
        text_lower = text.lower()

        for pack in self._compiled:
            # Regex 평가
            for rule_id, pattern, weight in pack.regexes:
                if pattern.search(text):
                    total += weight
                    hits.append(GuardMatch(rule_id, weight))

            # Aho-Corasick 구문 매칭
            for _, phrase in pack.automaton.iter(text_lower):
                weight = pack.phrase_weights.get(phrase, 0.0)
                if weight > 0:
                    total += weight
                    hits.append(GuardMatch(f"phrase:{phrase}", weight))

        return total, hits

    async def _compute_anomaly(
        self, input_text: str
    ) -> tuple[float, GuardMatch | None]:
        """Compute morphological anomaly score"""
        if self._anomaly_scorer is None:
            return 0.0, None

        try:
            scorer = self._anomaly_scorer
            if inspect.iscoroutinefunction(scorer):
                score = await scorer(input_text)
            else:
                result = scorer(input_text)
                if inspect.iscoroutine(result):
                    score = await result
                else:
                    score = result

            if score > self._settings.anomaly_threshold:
                log.debug(
                    "InjectionGuard ANOMALY: score=%.2f, text=%s...",
                    score,
                    input_text[:30],
                )
                return score, GuardMatch("morphological_anomaly", score)

        except Exception as e:  # noqa: BLE001 allow graceful degradation
            log.warning("Anomaly detection failed: %s", e)

        return 0.0, None

    async def is_malicious(self, input_text: str) -> bool:
        """Convenience method to check if input is malicious"""
        evaluation = await self.evaluate(input_text)
        return evaluation.malicious

    def set_anomaly_scorer(
        self, scorer: Callable[[str], float | Awaitable[float]]
    ) -> None:
        """Set the anomaly scorer function (for dependency injection)"""
        self._anomaly_scorer = scorer


# Singleton instance
_guard: InjectionGuard | None = None


def get_injection_guard() -> InjectionGuard:
    """Get injection guard singleton"""
    global _guard
    if _guard is None:
        _guard = InjectionGuard()
    return _guard
