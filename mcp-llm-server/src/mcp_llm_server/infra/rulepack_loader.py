"""Rulepack loader and compiler"""

import logging
import re
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol, cast

import ahocorasick
import yaml

from mcp_llm_server.models.guard import (
    PhrasesRule,
    RegexRule,
    Rulepack,
    parse_rulepack,
)


log = logging.getLogger(__name__)


class AutomatonProtocol(Protocol):
    def add_word(self, key: str, value: str) -> None: ...
    def make_automaton(self) -> None: ...
    def iter(self, text: str) -> Iterable[tuple[int, str]]: ...


@dataclass
class CompiledPack:
    """Compiled rulepack for efficient evaluation"""

    threshold: float
    regexes: list[tuple[str, re.Pattern[str], float]]  # (id, pattern, weight)
    automaton: AutomatonProtocol  # Aho-Corasick for phrases
    phrase_weights: dict[str, float]  # phrase -> weight


class RulepackLoader:
    """Load and compile rulepacks from YAML files"""

    def __init__(self, rulepacks_dir: str | Path) -> None:
        self._dir = Path(rulepacks_dir)

    def load_from_directory(self, patterns: list[str] | None = None) -> list[Rulepack]:
        """Load rulepacks matching patterns from directory

        Args:
            patterns: Glob patterns to match (default: ["*.yml", "*.yaml"])
        """
        if patterns is None:
            patterns = ["*.yml", "*.yaml"]

        packs = []
        for pattern in patterns:
            for path in self._dir.glob(pattern):
                try:
                    pack = self._load_file(path)
                    packs.append(pack)
                    log.info(
                        "Loaded rulepack: %s (%d rules)", path.name, len(pack.rules)
                    )
                except (OSError, ValueError, yaml.YAMLError) as e:
                    log.error("Failed to load rulepack %s: %s", path, e)

        return packs

    def _load_file(self, path: Path) -> Rulepack:
        """Load a single rulepack file"""
        with path.open(encoding="utf-8") as f:
            data = yaml.safe_load(f) or {}
        if not isinstance(data, dict):
            raise ValueError(f"{path.name} must contain a mapping")
        return parse_rulepack(data)


class RulepackCompiler:
    """Compile rulepacks for efficient evaluation"""

    @staticmethod
    def compile(pack: Rulepack) -> CompiledPack:
        """Compile a rulepack into efficient data structures"""
        regexes = RulepackCompiler._build_regexes(pack)
        automaton, phrase_weights = RulepackCompiler._build_automaton(pack)

        return CompiledPack(
            threshold=pack.threshold,
            regexes=regexes,
            automaton=automaton,
            phrase_weights=phrase_weights,
        )

    @staticmethod
    def _build_regexes(pack: Rulepack) -> list[tuple[str, re.Pattern[str], float]]:
        """Compile regex rules"""
        result = []
        for rule in pack.rules:
            if isinstance(rule, RegexRule):
                try:
                    pattern = re.compile(rule.pattern, re.IGNORECASE | re.UNICODE)
                    result.append((rule.id, pattern, rule.weight))
                except re.error as e:
                    log.warning("Invalid regex in rule %s: %s", rule.id, e)
        return result

    @staticmethod
    def _build_automaton(
        pack: Rulepack,
    ) -> tuple[AutomatonProtocol, dict[str, float]]:
        """Build Aho-Corasick automaton for phrase matching"""
        automaton: AutomatonProtocol = cast(
            "AutomatonProtocol", ahocorasick.Automaton()
        )
        phrase_weights: dict[str, float] = {}

        for rule in pack.rules:
            if isinstance(rule, PhrasesRule):
                for phrase in rule.phrases:
                    key = phrase.lower()
                    automaton.add_word(key, key)
                    phrase_weights[key] = rule.weight

        automaton.make_automaton()
        return automaton, phrase_weights


def load_and_compile_rulepacks(
    rulepacks_dir: str | Path,
    patterns: list[str] | None = None,
) -> list[CompiledPack]:
    """Convenience function to load and compile all rulepacks

    Args:
        rulepacks_dir: Directory containing rulepack YAML files
        patterns: Optional glob patterns to match
    """
    loader = RulepackLoader(rulepacks_dir)
    packs = loader.load_from_directory(patterns)
    return [RulepackCompiler.compile(pack) for pack in packs]
