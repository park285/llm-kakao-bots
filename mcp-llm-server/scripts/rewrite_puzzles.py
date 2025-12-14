#!/usr/bin/env python3
"""Batch rewrite puzzles using LLM.

Usage:
    # Rewrite all puzzles
    python scripts/rewrite_puzzles.py

    # Rewrite specific range
    python scripts/rewrite_puzzles.py --start 1 --end 50

    # Dry run (no save)
    python scripts/rewrite_puzzles.py --dry-run

    # Rewrite specific IDs
    python scripts/rewrite_puzzles.py --ids 17,39,48
"""

import argparse
import asyncio
import json
import logging
import pathlib
import sys


# Add src to path
sys.path.insert(0, str(pathlib.Path(__file__).parent.parent / "src"))

from mcp_llm_server.domains.turtle_soup.puzzle_loader import PresetPuzzle, PuzzleLoader


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)
log = logging.getLogger(__name__)


async def rewrite_puzzle(puzzle: PresetPuzzle) -> dict[str, str] | None:
    """Rewrite a single puzzle using LLM.

    Args:
        puzzle: Puzzle to rewrite

    Returns:
        Dict with rewritten scenario and solution, or None on failure
    """
    import httpx

    url = "http://localhost:20527/api/turtle-soup/rewrites"
    payload = {
        "title": puzzle.title,
        "scenario": puzzle.question,
        "solution": puzzle.answer,
        "difficulty": puzzle.difficulty,
    }

    try:
        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.post(url, json=payload)
            response.raise_for_status()
            data = response.json()
            return {
                "scenario": data.get("scenario", puzzle.question),
                "solution": data.get("solution", puzzle.answer),
            }
    except Exception as e:
        log.error("Failed to rewrite puzzle %d: %s", puzzle.id, e)
        return None


async def batch_rewrite(
    puzzles: list[PresetPuzzle],
    dry_run: bool = False,
    delay: float = 1.0,
) -> list[dict]:
    """Batch rewrite puzzles.

    Args:
        puzzles: List of puzzles to rewrite
        dry_run: If True, don't actually call API
        delay: Delay between API calls (seconds)

    Returns:
        List of rewritten puzzle dicts
    """
    results = []

    for i, puzzle in enumerate(puzzles, 1):
        log.info(
            "[%d/%d] Rewriting puzzle %d: %s",
            i,
            len(puzzles),
            puzzle.id,
            puzzle.title,
        )

        if dry_run:
            # Just copy original
            result = puzzle.model_dump()
        else:
            rewritten = await rewrite_puzzle(puzzle)
            if rewritten:
                result = puzzle.model_dump()
                result["question"] = rewritten["scenario"]
                result["answer"] = rewritten["solution"]
                log.info("  -> Rewritten successfully")
            else:
                result = puzzle.model_dump()
                log.warning("  -> Failed, keeping original")

            # Rate limiting
            if i < len(puzzles):
                await asyncio.sleep(delay)

        results.append(result)

    return results


def save_puzzles(puzzles: list[dict], output_dir: pathlib.Path) -> None:
    """Save puzzles to JSON files.

    Args:
        puzzles: List of puzzle dicts
        output_dir: Output directory
    """
    # Split into 3 files like original
    file_ranges = [
        ("1.json", 1, 50),
        ("2.json", 51, 100),
        ("3.json", 101, 200),
    ]

    for filename, start_id, end_id in file_ranges:
        file_puzzles = [p for p in puzzles if start_id <= p["id"] <= end_id]
        if file_puzzles:
            output_path = output_dir / filename
            with output_path.open("w", encoding="utf-8") as f:
                json.dump(file_puzzles, f, ensure_ascii=False, indent=2)
            log.info("Saved %d puzzles to %s", len(file_puzzles), output_path)


async def main() -> None:
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Batch rewrite puzzles")
    parser.add_argument("--start", type=int, default=1, help="Start puzzle ID")
    parser.add_argument("--end", type=int, default=200, help="End puzzle ID")
    parser.add_argument("--ids", type=str, help="Specific IDs (comma-separated)")
    parser.add_argument("--dry-run", action="store_true", help="Don't call API")
    parser.add_argument("--delay", type=float, default=1.0, help="Delay between calls")
    parser.add_argument(
        "--output",
        type=str,
        default="src/mcp_llm_server/domains/turtle_soup/puzzles",
        help="Output directory",
    )
    args = parser.parse_args()

    # Load puzzles
    loader = PuzzleLoader()
    all_puzzles = loader.get_all_puzzles()

    # Filter by range or IDs
    if args.ids:
        target_ids = {int(x.strip()) for x in args.ids.split(",")}
        puzzles = [p for p in all_puzzles if p.id in target_ids]
    else:
        puzzles = [p for p in all_puzzles if args.start <= p.id <= args.end]

    if not puzzles:
        log.error("No puzzles found for given criteria")
        return

    log.info("Found %d puzzles to rewrite", len(puzzles))
    log.info("Dry run: %s", args.dry_run)

    # Rewrite
    rewritten = await batch_rewrite(puzzles, args.dry_run, args.delay)

    # Merge with original (keep unrewritten puzzles)
    result_map = {p["id"]: p for p in rewritten}
    final_puzzles = []
    for p in all_puzzles:
        if p.id in result_map:
            final_puzzles.append(result_map[p.id])
        else:
            final_puzzles.append(p.model_dump())

    # Save
    if not args.dry_run:
        output_dir = pathlib.Path(args.output)
        save_puzzles(final_puzzles, output_dir)
        log.info("Done! Total puzzles: %d", len(final_puzzles))
    else:
        log.info("Dry run complete. No files saved.")


if __name__ == "__main__":
    asyncio.run(main())
