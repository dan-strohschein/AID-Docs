"""Convert Python type annotations (AST nodes) to AID universal type notation."""

from __future__ import annotations

import ast


# Python type names that map directly to AID types
_DIRECT_MAP: dict[str, str] = {
    "str": "str",
    "int": "int",
    "float": "float",
    "bool": "bool",
    "bytes": "bytes",
    "None": "None",
    "NoneType": "None",
    "Any": "any",
    "object": "any",
}

# Generic container types with special AID notation
_CONTAINER_MAP: dict[str, str] = {
    "list": "list",
    "List": "list",
    "dict": "dict",
    "Dict": "dict",
    "set": "set",
    "Set": "set",
    "frozenset": "set",
    "FrozenSet": "set",
    "tuple": "tuple",
    "Tuple": "tuple",
    "Sequence": "list",
    "MutableSequence": "list",
    "Mapping": "dict",
    "MutableMapping": "dict",
    "Iterable": "list",
    "Iterator": "Iterator",
}


def python_type_to_aid(node: ast.expr | None) -> str:
    """Convert a Python type annotation AST node to AID type notation.

    Returns 'any' for missing or unrecognizable annotations.
    """
    if node is None:
        return "any"

    # Simple name: str, int, MyClass, etc.
    if isinstance(node, ast.Name):
        return _DIRECT_MAP.get(node.id, node.id)

    # Attribute access: typing.Optional, module.Type, etc.
    if isinstance(node, ast.Attribute):
        return _DIRECT_MAP.get(node.attr, node.attr)

    # String constant (forward reference): "ClassName"
    if isinstance(node, ast.Constant) and isinstance(node.value, str):
        return node.value

    # None constant
    if isinstance(node, ast.Constant) and node.value is None:
        return "None"

    # Subscript: list[int], dict[str, int], Optional[str], etc.
    if isinstance(node, ast.Subscript):
        return _convert_subscript(node)

    # BinOp with |: str | int (Python 3.10+ union syntax)
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.BitOr):
        return _convert_union_binop(node)

    # Fallback
    return "any"


def _convert_subscript(node: ast.Subscript) -> str:
    """Handle subscripted types: list[int], Optional[str], Callable[[A], B], etc."""
    base_name = _get_base_name(node.value)

    # Optional[T] → T?
    if base_name in ("Optional",):
        inner = python_type_to_aid(node.slice)
        return f"{inner}?"

    # Union[A, B] or Union[A, B, None]
    if base_name in ("Union",):
        return _convert_union_subscript(node.slice)

    # Callable[[A, B], C] → fn(A, B) -> C
    if base_name in ("Callable",):
        return _convert_callable(node.slice)

    # Container types
    container = _CONTAINER_MAP.get(base_name)
    if container:
        return _convert_container(container, node.slice)

    # Generic user type: MyClass[T] → MyClass[T]
    args = _get_subscript_args(node.slice)
    inner = ", ".join(python_type_to_aid(a) for a in args)
    return f"{base_name}[{inner}]"


def _convert_union_binop(node: ast.BinOp) -> str:
    """Handle A | B | None style unions."""
    types = _flatten_binop_union(node)
    type_strs = [python_type_to_aid(t) for t in types]

    # If one of the types is None, it's Optional
    if "None" in type_strs:
        non_none = [t for t in type_strs if t != "None"]
        if len(non_none) == 1:
            return f"{non_none[0]}?"
        # Multiple non-None types plus None: (A | B)?
        return f"({'|'.join(non_none)})?"

    return " | ".join(type_strs)


def _flatten_binop_union(node: ast.expr) -> list[ast.expr]:
    """Flatten nested A | B | C into a list."""
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.BitOr):
        return _flatten_binop_union(node.left) + _flatten_binop_union(node.right)
    return [node]


def _convert_union_subscript(slice_node: ast.expr) -> str:
    """Handle Union[A, B] and Union[A, B, None]."""
    args = _get_subscript_args(slice_node)
    type_strs = [python_type_to_aid(a) for a in args]

    if "None" in type_strs:
        non_none = [t for t in type_strs if t != "None"]
        if len(non_none) == 1:
            return f"{non_none[0]}?"
        return f"({'|'.join(non_none)})?"

    return " | ".join(type_strs)


def _convert_callable(slice_node: ast.expr) -> str:
    """Handle Callable[[A, B], C] → fn(A, B) -> C."""
    args = _get_subscript_args(slice_node)
    if len(args) != 2:
        return "fn() -> any"

    param_node, return_node = args
    return_type = python_type_to_aid(return_node)

    # First arg should be a list of parameter types
    if isinstance(param_node, ast.List):
        param_types = ", ".join(python_type_to_aid(p) for p in param_node.elts)
        return f"fn({param_types}) -> {return_type}"
    elif isinstance(param_node, ast.Constant) and param_node.value is Ellipsis:
        return f"fn(...) -> {return_type}"

    return f"fn() -> {return_type}"


def _convert_container(container: str, slice_node: ast.expr) -> str:
    """Handle list[T] → [T], dict[K, V] → dict[K, V], etc."""
    args = _get_subscript_args(slice_node)

    if container == "list":
        inner = python_type_to_aid(args[0]) if args else "any"
        return f"[{inner}]"

    if container == "dict":
        if len(args) >= 2:
            key = python_type_to_aid(args[0])
            val = python_type_to_aid(args[1])
            return f"dict[{key}, {val}]"
        return "dict[any, any]"

    if container == "set":
        inner = python_type_to_aid(args[0]) if args else "any"
        return f"set[{inner}]"

    if container == "tuple":
        if args:
            inner = ", ".join(python_type_to_aid(a) for a in args)
            return f"({inner})"
        return "()"

    # Fallback for other containers
    if args:
        inner = ", ".join(python_type_to_aid(a) for a in args)
        return f"{container}[{inner}]"
    return container


def _get_base_name(node: ast.expr) -> str:
    """Extract the name from a type expression (Name or Attribute)."""
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        return node.attr
    return ""


def _get_subscript_args(slice_node: ast.expr) -> list[ast.expr]:
    """Extract the argument list from a subscript's slice."""
    if isinstance(slice_node, ast.Tuple):
        return list(slice_node.elts)
    return [slice_node]
