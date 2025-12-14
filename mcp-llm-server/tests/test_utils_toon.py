"""Tests for TOON encoder."""

from mcp_llm_server.utils.toon import _is_primitive, encode, encode_secret


class TestIsPrimitive:
    """Tests for _is_primitive function."""

    def test_none(self) -> None:
        assert _is_primitive(None)

    def test_bool(self) -> None:
        assert _is_primitive(True)
        assert _is_primitive(False)

    def test_int(self) -> None:
        assert _is_primitive(0)
        assert _is_primitive(42)
        assert _is_primitive(-1)

    def test_float(self) -> None:
        assert _is_primitive(3.14)
        assert _is_primitive(-0.5)

    def test_str(self) -> None:
        assert _is_primitive("")
        assert _is_primitive("hello")

    def test_list_not_primitive(self) -> None:
        assert not _is_primitive([])
        assert not _is_primitive([1, 2, 3])

    def test_dict_not_primitive(self) -> None:
        assert not _is_primitive({})
        assert not _is_primitive({"a": 1})


class TestEncodePrimitives:
    """Tests for encoding primitive types."""

    def test_none(self) -> None:
        assert encode(None) == "null"

    def test_true(self) -> None:
        assert encode(True) == "true"

    def test_false(self) -> None:
        assert encode(False) == "false"

    def test_int(self) -> None:
        assert encode(42) == "42"
        assert encode(-1) == "-1"

    def test_float(self) -> None:
        assert encode(3.14) == "3.14"

    def test_string_simple(self) -> None:
        assert encode("hello") == "hello"

    def test_string_with_special_chars(self) -> None:
        # 특수문자 포함 시 따옴표로 감싸기
        assert encode("hello, world") == '"hello, world"'
        assert encode("key: value") == '"key: value"'
        assert encode("line1\nline2") == '"line1\nline2"'

    def test_string_with_quotes(self) -> None:
        result = encode('say "hello"')
        assert result == '"say \\"hello\\""'


class TestEncodeList:
    """Tests for encoding lists."""

    def test_empty_list(self) -> None:
        assert encode([]) == "[]"

    def test_primitive_list_csv_style(self) -> None:
        result = encode([1, 2, 3])
        assert result == "[3]: 1,2,3"

    def test_mixed_primitive_list(self) -> None:
        result = encode(["a", 1, True])
        assert result == "[3]: a,1,true"

    def test_list_of_dicts_table_style(self) -> None:
        data = [
            {"name": "Alice", "age": 30},
            {"name": "Bob", "age": 25},
        ]
        result = encode(data)
        assert "[2]{name,age}:" in result
        assert "Alice,30" in result
        assert "Bob,25" in result

    def test_list_of_mixed_dicts(self) -> None:
        # 키가 다른 dict들
        data = [{"a": 1}, {"b": 2}]
        result = encode(data)
        assert "[2]:" in result


class TestEncodeDict:
    """Tests for encoding dicts."""

    def test_empty_dict(self) -> None:
        assert encode({}) == "{}"

    def test_simple_dict(self) -> None:
        result = encode({"name": "test", "value": 42})
        assert "name: test" in result
        assert "value: 42" in result

    def test_nested_dict(self) -> None:
        data = {"outer": {"inner": "value"}}
        result = encode(data)
        assert "outer:" in result
        assert "inner: value" in result

    def test_dict_with_list_of_dicts(self) -> None:
        data = {
            "items": [
                {"id": 1, "name": "a"},
                {"id": 2, "name": "b"},
            ]
        }
        result = encode(data)
        assert "items[2]{id,name}:" in result


class TestEncodeSecret:
    """Tests for encode_secret function."""

    def test_basic(self) -> None:
        result = encode_secret("사과", "과일")
        assert "target: 사과" in result
        assert "category: 과일" in result

    def test_with_details(self) -> None:
        result = encode_secret("사과", "과일", {"color": "빨강"})
        assert "target: 사과" in result
        assert "category: 과일" in result
        assert "details:" in result
        assert "color: 빨강" in result

    def test_without_details(self) -> None:
        result = encode_secret("답", "카테고리")
        assert "details" not in result


class TestEncodeEdgeCases:
    """Edge case tests."""

    def test_unknown_type(self) -> None:
        # 알 수 없는 타입은 str()로 변환
        class Custom:
            def __str__(self) -> str:
                return "custom_value"

        result = encode(Custom())
        assert result == "custom_value"

    def test_deeply_nested(self) -> None:
        data = {"a": {"b": {"c": "deep"}}}
        result = encode(data)
        assert "c: deep" in result

    def test_list_with_nested_dicts(self) -> None:
        data = [{"a": {"b": 1}}, {"a": {"b": 2}}]
        result = encode(data)
        # 중첩 dict가 있는 리스트
        assert "[2]" in result
