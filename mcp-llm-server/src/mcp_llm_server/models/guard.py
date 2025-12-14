"""Guard evaluation models"""

from collections.abc import Mapping
from dataclasses import dataclass
from typing import Literal, cast

from mcp_llm_server.types import JSONValue


@dataclass(frozen=True)
class GuardMatch:
    """A single rule match in guard evaluation"""

    id: str
    weight: float


@dataclass(frozen=True)
class GuardEvaluation:
    """Result of injection guard evaluation"""

    score: float
    hits: list[GuardMatch]
    threshold: float

    @property
    def malicious(self) -> bool:
        """Check if score exceeds threshold"""
        return self.score >= self.threshold


@dataclass
class RegexRule:
    """Regex-based detection rule"""

    id: str
    pattern: str
    weight: float
    type: Literal["regex"] = "regex"


@dataclass
class PhrasesRule:
    """Phrase-based detection rule using Aho-Corasick"""

    id: str
    phrases: list[str]
    weight: float
    type: Literal["phrases"] = "phrases"


# Union type for rules
Rule = RegexRule | PhrasesRule


@dataclass
class Rulepack:
    """A collection of detection rules"""

    version: int
    threshold: float
    normalizers: list[str]
    rules: list[Rule]


def _require_float(value: JSONValue, field: str) -> float:
    if isinstance(value, (int, float)):
        return float(value)
    raise ValueError(f"{field} must be a number")


def parse_rule(data: Mapping[str, JSONValue]) -> Rule:
    """Parse a rule from dictionary (YAML)"""
    rule_type = data.get("type")
    if rule_type == "regex":
        rule_id = data.get("id")
        pattern = data.get("pattern")
        weight = data.get("weight")
        if not isinstance(rule_id, str) or not isinstance(pattern, str):
            raise ValueError("regex rule requires id and pattern")
        return RegexRule(
            id=rule_id,
            pattern=pattern,
            weight=_require_float(weight, "weight"),
        )
    if rule_type == "phrases":
        rule_id = data.get("id")
        phrases = data.get("phrases")
        weight = data.get("weight")
        if not isinstance(rule_id, str):
            raise ValueError("phrases rule requires id")
        if not isinstance(phrases, list):
            raise ValueError("phrases must be a list")
        if not all(isinstance(p, str) for p in phrases):
            raise ValueError("phrases entries must be strings")
        phrase_list = cast("list[str]", phrases)
        return PhrasesRule(
            id=rule_id,
            phrases=list(phrase_list),
            weight=_require_float(weight, "weight"),
        )
    raise ValueError(f"Unknown rule type: {rule_type}")


def parse_rulepack(data: Mapping[str, JSONValue]) -> Rulepack:
    """Parse a rulepack from dictionary (YAML)"""
    rules_raw = data.get("rules", [])
    if not isinstance(rules_raw, list):
        raise ValueError("rules must be a list")

    rules: list[Rule] = []
    for rule in rules_raw:
        if not isinstance(rule, dict):
            raise ValueError("each rule must be an object")
        rules.append(parse_rule(rule))

    normalizers_raw = data.get("normalizers", ["nfkc", "strip_zero_width"])
    if isinstance(normalizers_raw, list):
        normalizer_list = [str(n) for n in normalizers_raw]
    else:
        normalizer_list = ["nfkc", "strip_zero_width"]

    version = data.get("version", 1)
    if not isinstance(version, int):
        raise ValueError("version must be an integer")

    return Rulepack(
        version=version,
        threshold=_require_float(data.get("threshold", 0.7), "threshold"),
        normalizers=normalizer_list,
        rules=rules,
    )
