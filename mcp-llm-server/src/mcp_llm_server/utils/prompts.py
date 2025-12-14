"""Shared helpers for loading prompt YAML files."""

from __future__ import annotations

from typing import TYPE_CHECKING

import yaml

from mcp_llm_server.config.logging import get_logger


if TYPE_CHECKING:
    from pathlib import Path


log = get_logger(__name__)


def load_yaml_mapping(path: Path) -> dict[str, str]:
    """Load a single YAML file as a string-to-string mapping."""
    if not path.exists():
        raise FileNotFoundError(f"Prompt file not found: {path}")

    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, dict):
        raise ValueError(f"{path.name} must contain a mapping")

    return {
        str(key): str(value) if value is not None else ""
        for key, value in data.items()
        if isinstance(key, str)
    }


def load_yaml_dir(directory: Path) -> dict[str, dict[str, str]]:
    """Load all YAML files in a directory into a nested mapping."""
    prompts: dict[str, dict[str, str]] = {}
    for yaml_file in directory.glob("*.yml"):
        if not yaml_file.is_file():
            continue
        prompts[yaml_file.stem] = load_yaml_mapping(yaml_file)
    log.info("PROMPTS_LOADED dir={} count={}", directory, len(prompts))
    return prompts
