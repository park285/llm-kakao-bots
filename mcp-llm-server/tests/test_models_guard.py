"""Tests for guard models."""

import pytest

from mcp_llm_server.models.guard import (
    GuardEvaluation,
    GuardMatch,
    PhrasesRule,
    RegexRule,
    Rulepack,
    parse_rule,
    parse_rulepack,
)


class TestGuardMatch:
    """Tests for GuardMatch dataclass."""

    def test_create(self) -> None:
        match = GuardMatch(id="rule1", weight=0.5)
        assert match.id == "rule1"
        assert match.weight == 0.5

    def test_frozen(self) -> None:
        match = GuardMatch(id="rule1", weight=0.5)
        with pytest.raises(AttributeError):
            match.id = "other"  # type: ignore[misc]


class TestGuardEvaluation:
    """Tests for GuardEvaluation dataclass."""

    def test_create(self) -> None:
        hits = [GuardMatch(id="r1", weight=0.3)]
        eval_result = GuardEvaluation(score=0.3, hits=hits, threshold=0.7)
        assert eval_result.score == 0.3
        assert eval_result.threshold == 0.7
        assert len(eval_result.hits) == 1

    def test_malicious_true(self) -> None:
        eval_result = GuardEvaluation(score=0.8, hits=[], threshold=0.7)
        assert eval_result.malicious is True

    def test_malicious_false(self) -> None:
        eval_result = GuardEvaluation(score=0.5, hits=[], threshold=0.7)
        assert eval_result.malicious is False

    def test_malicious_at_threshold(self) -> None:
        eval_result = GuardEvaluation(score=0.7, hits=[], threshold=0.7)
        assert eval_result.malicious is True

    def test_frozen(self) -> None:
        eval_result = GuardEvaluation(score=0.5, hits=[], threshold=0.7)
        with pytest.raises(AttributeError):
            eval_result.score = 1.0  # type: ignore[misc]


class TestRegexRule:
    """Tests for RegexRule dataclass."""

    def test_create(self) -> None:
        rule = RegexRule(id="regex1", pattern=r"test.*", weight=0.5)
        assert rule.id == "regex1"
        assert rule.pattern == r"test.*"
        assert rule.weight == 0.5
        assert rule.type == "regex"

    def test_default_type(self) -> None:
        rule = RegexRule(id="r", pattern="p", weight=0.1)
        assert rule.type == "regex"


class TestPhrasesRule:
    """Tests for PhrasesRule dataclass."""

    def test_create(self) -> None:
        rule = PhrasesRule(id="phrases1", phrases=["hello", "world"], weight=0.3)
        assert rule.id == "phrases1"
        assert rule.phrases == ["hello", "world"]
        assert rule.weight == 0.3
        assert rule.type == "phrases"

    def test_default_type(self) -> None:
        rule = PhrasesRule(id="p", phrases=[], weight=0.1)
        assert rule.type == "phrases"


class TestRulepack:
    """Tests for Rulepack dataclass."""

    def test_create(self) -> None:
        rules = [
            RegexRule(id="r1", pattern="test", weight=0.5),
            PhrasesRule(id="p1", phrases=["a", "b"], weight=0.3),
        ]
        pack = Rulepack(
            version=2,
            threshold=0.8,
            normalizers=["nfkc"],
            rules=rules,
        )
        assert pack.version == 2
        assert pack.threshold == 0.8
        assert pack.normalizers == ["nfkc"]
        assert len(pack.rules) == 2


class TestParseRule:
    """Tests for parse_rule function."""

    def test_parse_regex_rule(self) -> None:
        data = {
            "type": "regex",
            "id": "regex_test",
            "pattern": r"\btest\b",
            "weight": 0.4,
        }
        rule = parse_rule(data)
        assert isinstance(rule, RegexRule)
        assert rule.id == "regex_test"
        assert rule.pattern == r"\btest\b"
        assert rule.weight == 0.4

    def test_parse_phrases_rule(self) -> None:
        data = {
            "type": "phrases",
            "id": "phrases_test",
            "phrases": ["foo", "bar", "baz"],
            "weight": 0.2,
        }
        rule = parse_rule(data)
        assert isinstance(rule, PhrasesRule)
        assert rule.id == "phrases_test"
        assert rule.phrases == ["foo", "bar", "baz"]
        assert rule.weight == 0.2

    def test_unknown_rule_type_raises(self) -> None:
        data = {
            "type": "unknown",
            "id": "bad",
            "weight": 0.1,
        }
        with pytest.raises(ValueError, match="Unknown rule type"):
            parse_rule(data)

    def test_missing_type_raises(self) -> None:
        data = {"id": "no_type", "weight": 0.1}
        with pytest.raises(ValueError, match="Unknown rule type: None"):
            parse_rule(data)


class TestParseRulepack:
    """Tests for parse_rulepack function."""

    def test_parse_full_rulepack(self) -> None:
        data = {
            "version": 3,
            "threshold": 0.85,
            "normalizers": ["nfkc", "strip_zero_width"],
            "rules": [
                {"type": "regex", "id": "r1", "pattern": "test", "weight": 0.5},
                {"type": "phrases", "id": "p1", "phrases": ["a"], "weight": 0.3},
            ],
        }
        pack = parse_rulepack(data)
        assert pack.version == 3
        assert pack.threshold == 0.85
        assert pack.normalizers == ["nfkc", "strip_zero_width"]
        assert len(pack.rules) == 2
        assert isinstance(pack.rules[0], RegexRule)
        assert isinstance(pack.rules[1], PhrasesRule)

    def test_parse_with_defaults(self) -> None:
        data: dict = {}
        pack = parse_rulepack(data)
        assert pack.version == 1
        assert pack.threshold == 0.7
        assert pack.normalizers == ["nfkc", "strip_zero_width"]
        assert pack.rules == []

    def test_parse_empty_rules(self) -> None:
        data = {"version": 1, "rules": []}
        pack = parse_rulepack(data)
        assert len(pack.rules) == 0
