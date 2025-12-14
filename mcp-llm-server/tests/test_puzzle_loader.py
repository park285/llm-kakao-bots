"""Tests for puzzle loader."""

import json
from pathlib import Path

import pytest

from mcp_llm_server.domains.turtle_soup.puzzle_loader import (
    PresetPuzzle,
    PuzzleLoader,
    get_puzzle_loader,
)


def _expected_total_puzzles() -> int:
    """Calculate expected puzzle count from JSON fixtures."""
    puzzles_dir = (
        Path(__file__).parent.parent
        / "src"
        / "mcp_llm_server"
        / "domains"
        / "turtle_soup"
        / "puzzles"
    )
    total = 0
    for path in sorted(puzzles_dir.glob("*.json")):
        total += len(json.loads(path.read_text()))
    return total


class TestPresetPuzzle:
    """Tests for PresetPuzzle model."""

    def test_valid_puzzle(self) -> None:
        """Test creating a valid puzzle."""
        puzzle = PresetPuzzle(
            id=1,
            title="테스트",
            question="문제입니다",
            answer="정답입니다",
            difficulty=3,
        )
        assert puzzle.id == 1
        assert puzzle.title == "테스트"
        assert puzzle.difficulty == 3

    def test_difficulty_bounds(self) -> None:
        """Test difficulty bounds validation."""
        # Valid boundaries
        PresetPuzzle(id=1, title="t", question="q", answer="a", difficulty=1)
        PresetPuzzle(id=2, title="t", question="q", answer="a", difficulty=5)

        # Invalid: too low
        with pytest.raises(ValueError):
            PresetPuzzle(id=3, title="t", question="q", answer="a", difficulty=0)

        # Invalid: too high
        with pytest.raises(ValueError):
            PresetPuzzle(id=4, title="t", question="q", answer="a", difficulty=6)


class TestPuzzleLoader:
    """Tests for PuzzleLoader."""

    def test_load_puzzles(self) -> None:
        """Test loading puzzles from files."""
        loader = PuzzleLoader()
        count = loader.get_puzzle_count()
        expected = _expected_total_puzzles()
        assert count == expected, f"Expected {expected} puzzles, got {count}"

    def test_get_random_puzzle(self) -> None:
        """Test getting a random puzzle."""
        loader = PuzzleLoader()
        puzzle = loader.get_random_puzzle()
        assert isinstance(puzzle, PresetPuzzle)
        assert puzzle.id > 0

    def test_get_random_puzzle_by_difficulty(self) -> None:
        """Test getting a random puzzle by difficulty."""
        loader = PuzzleLoader()

        for difficulty in [1, 2, 3, 4, 5]:
            puzzle = loader.get_random_puzzle_by_difficulty(difficulty)
            assert puzzle.difficulty == difficulty

    def test_get_random_puzzle_invalid_difficulty(self) -> None:
        """Test getting puzzle with invalid difficulty."""
        loader = PuzzleLoader()

        with pytest.raises(ValueError, match="No puzzles found"):
            loader.get_random_puzzle_by_difficulty(99)

    def test_get_puzzle_by_id(self) -> None:
        """Test getting puzzle by ID."""
        loader = PuzzleLoader()

        puzzle = loader.get_puzzle_by_id(1)
        assert puzzle is not None
        assert puzzle.id == 1

        # Non-existent ID
        assert loader.get_puzzle_by_id(9999) is None

    def test_get_all_puzzles(self) -> None:
        """Test getting all puzzles."""
        loader = PuzzleLoader()
        puzzles = loader.get_all_puzzles()
        expected = _expected_total_puzzles()
        assert len(puzzles) == expected

    def test_get_puzzle_count_by_difficulty(self) -> None:
        """Test getting puzzle count by difficulty."""
        loader = PuzzleLoader()
        counts = loader.get_puzzle_count_by_difficulty()

        assert isinstance(counts, dict)
        total = sum(counts.values())
        assert total == _expected_total_puzzles()

    def test_reload(self) -> None:
        """Test reloading puzzles."""
        loader = PuzzleLoader()
        count = loader.reload()
        assert count == _expected_total_puzzles()

    def test_get_examples(self) -> None:
        """Test fetching examples for prompts."""
        loader = PuzzleLoader()
        examples = loader.get_examples(difficulty=1, max_examples=2)
        assert 1 <= len(examples) <= 2
        assert all(example.difficulty == 1 for example in examples)

    def test_singleton(self) -> None:
        """Test singleton pattern."""
        loader1 = get_puzzle_loader()
        loader2 = get_puzzle_loader()
        assert loader1 is loader2
