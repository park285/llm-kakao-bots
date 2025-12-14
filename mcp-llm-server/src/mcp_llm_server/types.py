"""프로젝트 전반에서 사용하는 JSON 타입 별칭."""

from __future__ import annotations


__all__ = ["JSONMapping", "JSONPrimitive", "JSONValue"]

# JSON 직렬화 가능 타입
type JSONPrimitive = None | bool | int | float | str
type JSONValue = JSONPrimitive | list["JSONValue"] | dict[str, "JSONValue"]
type JSONMapping = dict[str, JSONValue]
