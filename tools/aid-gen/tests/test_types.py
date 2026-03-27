"""Tests for Python type → AID type mapping."""

import ast
from aid_gen.python.types import python_type_to_aid


def _parse_annotation(code: str) -> ast.expr:
    """Parse a type annotation string into an AST node."""
    tree = ast.parse(f"x: {code}", mode="exec")
    return tree.body[0].annotation


def test_basic_types():
    assert python_type_to_aid(_parse_annotation("str")) == "str"
    assert python_type_to_aid(_parse_annotation("int")) == "int"
    assert python_type_to_aid(_parse_annotation("float")) == "float"
    assert python_type_to_aid(_parse_annotation("bool")) == "bool"
    assert python_type_to_aid(_parse_annotation("bytes")) == "bytes"
    assert python_type_to_aid(_parse_annotation("None")) == "None"


def test_any():
    assert python_type_to_aid(_parse_annotation("Any")) == "any"
    assert python_type_to_aid(None) == "any"


def test_user_defined_type():
    assert python_type_to_aid(_parse_annotation("Response")) == "Response"
    assert python_type_to_aid(_parse_annotation("MyClass")) == "MyClass"


def test_list():
    assert python_type_to_aid(_parse_annotation("list[int]")) == "[int]"
    assert python_type_to_aid(_parse_annotation("list[str]")) == "[str]"
    assert python_type_to_aid(_parse_annotation("List[int]")) == "[int]"


def test_dict():
    assert python_type_to_aid(_parse_annotation("dict[str, int]")) == "dict[str, int]"
    assert python_type_to_aid(_parse_annotation("Dict[str, Any]")) == "dict[str, any]"


def test_set():
    assert python_type_to_aid(_parse_annotation("set[int]")) == "set[int]"
    assert python_type_to_aid(_parse_annotation("Set[str]")) == "set[str]"


def test_tuple():
    assert python_type_to_aid(_parse_annotation("tuple[int, str]")) == "(int, str)"
    assert python_type_to_aid(_parse_annotation("Tuple[int, str, bool]")) == "(int, str, bool)"


def test_optional():
    assert python_type_to_aid(_parse_annotation("Optional[str]")) == "str?"
    assert python_type_to_aid(_parse_annotation("Optional[int]")) == "int?"


def test_union_with_none_is_optional():
    # Python 3.10+ syntax
    assert python_type_to_aid(_parse_annotation("str | None")) == "str?"
    assert python_type_to_aid(_parse_annotation("int | None")) == "int?"


def test_union_without_none():
    assert python_type_to_aid(_parse_annotation("str | int")) == "str | int"


def test_union_typing_module():
    assert python_type_to_aid(_parse_annotation("Union[str, int]")) == "str | int"
    assert python_type_to_aid(_parse_annotation("Union[str, None]")) == "str?"


def test_callable():
    result = python_type_to_aid(_parse_annotation("Callable[[str, int], bool]"))
    assert result == "fn(str, int) -> bool"

    result = python_type_to_aid(_parse_annotation("Callable[[], None]"))
    assert result == "fn() -> None"


def test_nested_generics():
    result = python_type_to_aid(_parse_annotation("list[dict[str, int]]"))
    assert result == "[dict[str, int]]"

    result = python_type_to_aid(_parse_annotation("dict[str, list[int]]"))
    assert result == "dict[str, [int]]"


def test_forward_reference():
    assert python_type_to_aid(_parse_annotation('"ClassName"')) == "ClassName"


def test_generic_user_type():
    result = python_type_to_aid(_parse_annotation("MyGeneric[int, str]"))
    assert result == "MyGeneric[int, str]"


def test_optional_complex():
    result = python_type_to_aid(_parse_annotation("Optional[list[str]]"))
    assert result == "[str]?"


def test_sequence_maps_to_list():
    result = python_type_to_aid(_parse_annotation("Sequence[int]"))
    assert result == "[int]"


def test_mapping_maps_to_dict():
    result = python_type_to_aid(_parse_annotation("Mapping[str, int]"))
    assert result == "dict[str, int]"
