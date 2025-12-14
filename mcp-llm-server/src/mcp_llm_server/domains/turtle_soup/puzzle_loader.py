"""Puzzle loader for turtle soup domain."""

import logging
import random
from pathlib import Path

import orjson
from pydantic import BaseModel, Field


log = logging.getLogger(__name__)


class PresetPuzzle(BaseModel):
    """Puzzle preset loaded from JSON files."""

    id: int
    title: str
    question: str
    answer: str
    difficulty: int = Field(ge=1, le=5)


class PuzzleLoader:
    """Load and manage turtle soup puzzles from JSON files."""

    def __init__(self, puzzles_dir: Path | None = None) -> None:
        """Initialize puzzle loader.

        Args:
            puzzles_dir: Custom puzzles directory path
        """
        if puzzles_dir is None:
            puzzles_dir = Path(__file__).parent / "puzzles"
        self._puzzles_dir = puzzles_dir
        self._puzzles: list[PresetPuzzle] = []
        self._loaded = False

    def _ensure_loaded(self) -> None:
        """Ensure puzzles are loaded (lazy loading)."""
        if not self._loaded:
            self._load_all_puzzles()
            self._loaded = True

    def _load_all_puzzles(self) -> None:
        """Load all puzzles from JSON files."""
        self._puzzles = []
        file_paths = sorted(self._puzzles_dir.glob("*.json"))

        if not file_paths:
            log.warning("puzzle_files_not_found dir=%s", self._puzzles_dir)

        for file_path in file_paths:
            try:
                data = orjson.loads(file_path.read_bytes())
                loaded = [PresetPuzzle(**item) for item in data]
                self._puzzles.extend(loaded)
                log.info(
                    "puzzle_file_loaded file=%s count=%d",
                    file_path.name,
                    len(loaded),
                )
            except Exception:
                log.exception("puzzle_file_load_failed file=%s", file_path.name)

        log.info("total_puzzles_loaded count=%d", len(self._puzzles))

    def reload(self) -> int:
        """Reload puzzles from files (for hot reload).

        Returns:
            Number of puzzles loaded
        """
        self._loaded = False
        self._ensure_loaded()
        return len(self._puzzles)

    def get_random_puzzle(self) -> PresetPuzzle:
        """Get a random puzzle.

        Returns:
            Random puzzle

        Raises:
            ValueError: If no puzzles loaded
        """
        self._ensure_loaded()
        if not self._puzzles:
            raise ValueError("No puzzles loaded")
        return random.choice(self._puzzles)

    def get_random_puzzle_by_difficulty(self, difficulty: int) -> PresetPuzzle:
        """Get a random puzzle by difficulty.

        Args:
            difficulty: Difficulty level (1-5)

        Returns:
            Random puzzle with specified difficulty

        Raises:
            ValueError: If no puzzles found for difficulty
        """
        self._ensure_loaded()
        filtered = [p for p in self._puzzles if p.difficulty == difficulty]
        if not filtered:
            raise ValueError(f"No puzzles found for difficulty {difficulty}")
        return random.choice(filtered)

    def get_puzzle_by_id(self, puzzle_id: int) -> PresetPuzzle | None:
        """Get puzzle by ID.

        Args:
            puzzle_id: Puzzle ID

        Returns:
            Puzzle if found, None otherwise
        """
        self._ensure_loaded()
        for puzzle in self._puzzles:
            if puzzle.id == puzzle_id:
                return puzzle
        return None

    def get_all_puzzles(self) -> list[PresetPuzzle]:
        """Get all loaded puzzles.

        Returns:
            List of all puzzles
        """
        self._ensure_loaded()
        return self._puzzles.copy()

    def get_puzzle_count(self) -> int:
        """Get total puzzle count.

        Returns:
            Number of puzzles loaded
        """
        self._ensure_loaded()
        return len(self._puzzles)

    def get_puzzle_count_by_difficulty(self) -> dict[int, int]:
        """Get puzzle count by difficulty.

        Returns:
            Dict mapping difficulty to count
        """
        self._ensure_loaded()
        counts: dict[int, int] = {}
        for puzzle in self._puzzles:
            counts[puzzle.difficulty] = counts.get(puzzle.difficulty, 0) + 1
        return counts

    def get_examples(
        self, difficulty: int | None = None, max_examples: int = 3
    ) -> list[PresetPuzzle]:
        """Get sample puzzles for prompt few-shot."""
        self._ensure_loaded()
        candidates = self._puzzles
        if difficulty is not None:
            filtered = [p for p in candidates if p.difficulty == difficulty]
            if filtered:
                candidates = filtered
        if not candidates:
            return []
        if len(candidates) <= max_examples:
            return candidates
        return random.sample(candidates, k=max_examples)


# Singleton instance
_instance: PuzzleLoader | None = None


def get_puzzle_loader() -> PuzzleLoader:
    """Get singleton puzzle loader instance.

    Returns:
        PuzzleLoader instance
    """
    global _instance
    if _instance is None:
        _instance = PuzzleLoader()
    return _instance
