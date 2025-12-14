"""Tests for rulepack loader and compiler."""

import tempfile
from pathlib import Path

import pytest

from mcp_llm_server.infra.rulepack_loader import (
    CompiledPack,
    RulepackCompiler,
    RulepackLoader,
    load_and_compile_rulepacks,
)
from mcp_llm_server.models.guard import PhrasesRule, RegexRule, Rulepack


@pytest.fixture
def sample_rulepack() -> Rulepack:
    """Create a sample rulepack for testing."""
    return Rulepack(
        version=1,
        threshold=0.7,
        normalizers=["nfkc"],
        rules=[
            RegexRule(id="r1", pattern=r"\btest\b", weight=0.3),
            RegexRule(id="r2", pattern=r"hello.*world", weight=0.4),
            PhrasesRule(id="p1", phrases=["attack", "malicious"], weight=0.5),
        ],
    )


@pytest.fixture
def temp_rulepacks_dir() -> Path:
    """Create a temporary directory with rulepack files."""
    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)

        # rulepack1.yml
        (tmppath / "rulepack1.yml").write_text(
            """
version: 1
threshold: 0.8
normalizers:
  - nfkc
rules:
  - type: regex
    id: regex1
    pattern: "dangerous"
    weight: 0.5
  - type: phrases
    id: phrases1
    phrases:
      - "attack"
      - "exploit"
    weight: 0.4
""",
            encoding="utf-8",
        )

        # rulepack2.yaml
        (tmppath / "rulepack2.yaml").write_text(
            """
version: 2
threshold: 0.9
rules:
  - type: regex
    id: regex2
    pattern: "hack"
    weight: 0.6
""",
            encoding="utf-8",
        )

        # invalid.txt (not yaml)
        (tmppath / "invalid.txt").write_text("not yaml", encoding="utf-8")

        yield tmppath


class TestCompiledPack:
    """Tests for CompiledPack dataclass."""

    def test_create(self) -> None:
        import ahocorasick

        automaton = ahocorasick.Automaton()
        automaton.make_automaton()

        pack = CompiledPack(
            threshold=0.8,
            regexes=[],
            automaton=automaton,
            phrase_weights={},
        )
        assert pack.threshold == 0.8
        assert pack.regexes == []


class TestRulepackLoader:
    """Tests for RulepackLoader class."""

    def test_load_from_directory(self, temp_rulepacks_dir: Path) -> None:
        loader = RulepackLoader(temp_rulepacks_dir)
        packs = loader.load_from_directory()

        # yml과 yaml 모두 로드
        assert len(packs) == 2

    def test_load_specific_pattern(self, temp_rulepacks_dir: Path) -> None:
        loader = RulepackLoader(temp_rulepacks_dir)
        packs = loader.load_from_directory(patterns=["*.yml"])

        assert len(packs) == 1

    def test_load_empty_directory(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            loader = RulepackLoader(tmpdir)
            packs = loader.load_from_directory()
            assert packs == []


class TestRulepackCompiler:
    """Tests for RulepackCompiler class."""

    def test_compile_regex_rules(self, sample_rulepack: Rulepack) -> None:
        compiled = RulepackCompiler.compile(sample_rulepack)

        # 2 regex rules
        assert len(compiled.regexes) == 2

        # 첫 번째 regex 확인
        rule_id, pattern, weight = compiled.regexes[0]
        assert rule_id == "r1"
        assert weight == 0.3
        assert pattern.search("this is a test")

    def test_compile_phrase_rules(self, sample_rulepack: Rulepack) -> None:
        compiled = RulepackCompiler.compile(sample_rulepack)

        # phrase weights 확인
        assert "attack" in compiled.phrase_weights
        assert "malicious" in compiled.phrase_weights
        assert compiled.phrase_weights["attack"] == 0.5

    def test_automaton_search(self, sample_rulepack: Rulepack) -> None:
        compiled = RulepackCompiler.compile(sample_rulepack)

        # Aho-Corasick 검색 테스트
        text = "this is an attack on the system"
        matches = list(compiled.automaton.iter(text.lower()))

        # "attack" 매칭
        assert len(matches) >= 1
        # (end_index, matched_word)
        assert any(m[1] == "attack" for m in matches)

    def test_compile_invalid_regex(self) -> None:
        pack = Rulepack(
            version=1,
            threshold=0.7,
            normalizers=[],
            rules=[
                RegexRule(id="bad", pattern=r"[invalid(", weight=0.5),  # 잘못된 regex
                RegexRule(id="good", pattern=r"valid", weight=0.3),
            ],
        )
        compiled = RulepackCompiler.compile(pack)

        # 유효한 것만 컴파일됨
        assert len(compiled.regexes) == 1
        assert compiled.regexes[0][0] == "good"

    def test_compile_empty_rulepack(self) -> None:
        pack = Rulepack(version=1, threshold=0.5, normalizers=[], rules=[])
        compiled = RulepackCompiler.compile(pack)

        assert compiled.threshold == 0.5
        assert compiled.regexes == []
        assert compiled.phrase_weights == {}


class TestLoadAndCompileRulepacks:
    """Tests for load_and_compile_rulepacks function."""

    def test_load_and_compile(self, temp_rulepacks_dir: Path) -> None:
        compiled_packs = load_and_compile_rulepacks(temp_rulepacks_dir)

        assert len(compiled_packs) == 2
        assert all(isinstance(p, CompiledPack) for p in compiled_packs)

    def test_load_with_pattern(self, temp_rulepacks_dir: Path) -> None:
        compiled_packs = load_and_compile_rulepacks(
            temp_rulepacks_dir, patterns=["*.yaml"]
        )

        assert len(compiled_packs) == 1

    def test_string_path(self, temp_rulepacks_dir: Path) -> None:
        # str 경로도 지원
        compiled_packs = load_and_compile_rulepacks(str(temp_rulepacks_dir))
        assert len(compiled_packs) == 2
