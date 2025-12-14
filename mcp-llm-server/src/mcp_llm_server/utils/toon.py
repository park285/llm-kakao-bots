"""TOON (Token-Oriented Object Notation) encoder.

TOON is a compact, human-readable encoding that minimizes tokens for LLM prompts.
See: https://github.com/toon-format/toon
"""

from typing import cast

from mcp_llm_server.types import JSONMapping


def encode(data: object, indent: int = 0) -> str:
    """Encode Python object to TOON format.

    Args:
        data: Python object (dict, list, or primitive)
        indent: Current indentation level

    Returns:
        TOON formatted string
    """
    prefix = " " * indent

    if data is None:
        return "null"

    if isinstance(data, bool):
        return "true" if data else "false"

    if isinstance(data, (int, float)):
        return str(data)

    if isinstance(data, str):
        # 문자열에 특수문자가 있으면 따옴표로 감싸기
        if any(c in data for c in [",", ":", "\n", '"', "'"]):
            # 내부 따옴표 이스케이프
            escaped = data.replace('"', '\\"')
            return f'"{escaped}"'
        return data

    if isinstance(data, list):
        data_list = data
        if not data_list:
            return "[]"

        # 모든 요소가 primitive인 경우 CSV 스타일
        if all(_is_primitive(item) for item in data_list):
            items = [encode(item) for item in data_list]
            return f"[{len(data_list)}]: {','.join(items)}"

        # 모든 요소가 동일한 키를 가진 dict인 경우 테이블 스타일
        if data_list and all(isinstance(item, dict) for item in data_list):
            dict_items = cast("list[JSONMapping]", data_list)
            keys = list(dict_items[0].keys())
            if all(set(item.keys()) == set(keys) for item in dict_items):
                # 테이블 형식
                header = f"[{len(data_list)}]{{{','.join(keys)}}}:"
                rows = []
                for item in dict_items:
                    row_values = [encode(item[k]) for k in keys]
                    rows.append(",".join(row_values))
                return header + "\n" + "\n".join(f"{prefix} {row}" for row in rows)

        # 일반 배열
        lines = [f"[{len(data_list)}]:"]
        for item in data_list:
            lines.append(f"{prefix} - {encode(item, indent + 2)}")
        return "\n".join(lines)

    if isinstance(data, dict):
        if not data:
            return "{}"

        lines = []
        mapping: dict[str, object] = cast("dict[str, object]", data)
        for key, value in mapping.items():
            if isinstance(value, dict) and value:
                # 중첩 객체
                lines.append(f"{key}:")
                for sub_key, sub_value in value.items():
                    encoded_value = encode(sub_value, indent + 2)
                    lines.append(f"{prefix}  {sub_key}: {encoded_value}")
            elif (
                isinstance(value, list)
                and value
                and all(isinstance(item, dict) for item in value)
            ):
                # 객체 배열 - 테이블 형식
                value_dicts = cast("list[JSONMapping]", value)
                keys = list(value_dicts[0].keys())
                if all(set(item.keys()) == set(keys) for item in value_dicts):
                    header = f"{key}[{len(value_dicts)}]{{{','.join(keys)}}}:"
                    lines.append(header)
                    for item in value_dicts:
                        row_values = [encode(item[k]) for k in keys]
                        lines.append(f"{prefix}  {','.join(row_values)}")
                else:
                    lines.append(f"{key}: {encode(value, indent)}")
            else:
                encoded_value = encode(value, indent)
                lines.append(f"{key}: {encoded_value}")

        return "\n".join(lines)

    # 기타 타입
    return str(data)


def _is_primitive(value: object) -> bool:
    """Check if value is a primitive type."""
    return value is None or isinstance(value, (bool, int, float, str))


def encode_secret(
    target: str, category: str, details: JSONMapping | None = None
) -> str:
    """Encode secret info for 20Q game in TOON format.

    Args:
        target: Secret answer
        category: Category name
        details: Optional additional details

    Returns:
        TOON formatted string
    """
    data: JSONMapping = {
        "target": target,
        "category": category,
    }
    if details:
        data["details"] = details

    return encode(data)


def encode_puzzle(
    scenario: str,
    solution: str,
    category: str | None = None,
    difficulty: int | None = None,
) -> str:
    """Encode puzzle info for Turtle Soup game in TOON format.

    Args:
        scenario: The puzzle scenario presented to player
        solution: The hidden solution
        category: Optional category name
        difficulty: Optional difficulty level (1-5)

    Returns:
        TOON formatted string
    """
    data: JSONMapping = {
        "scenario": scenario,
        "solution": solution,
    }
    if category:
        data["category"] = category
    if difficulty is not None:
        data["difficulty"] = difficulty

    return encode(data)
