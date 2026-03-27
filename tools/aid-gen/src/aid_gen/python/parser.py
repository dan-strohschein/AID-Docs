"""Parse Python source code into AID model objects."""

from __future__ import annotations

import ast
import re
from typing import Sequence

from aid_gen.model import (
    AidFile,
    ConstEntry,
    Entry,
    Field_,
    FnEntry,
    ModuleHeader,
    Param,
    TraitEntry,
    TypeEntry,
    Variant,
)
from aid_gen.python.protocols import detect_protocols
from aid_gen.python.types import python_type_to_aid


def extract_module(source: str, module_name: str, version: str = "0.0.0") -> AidFile:
    """Parse Python source and extract an AID file."""
    tree = ast.parse(source)

    # Detect __all__ for export filtering
    all_names = _extract_all(tree)

    header = _build_header(tree, module_name, version)
    entries: list[Entry] = []

    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if _should_export(node.name, all_names):
                entries.append(_visit_function(node))

        elif isinstance(node, ast.ClassDef):
            if _should_export(node.name, all_names):
                # Check if it's a NamedTuple subclass
                base_names = [_get_base_name(b) for b in node.bases]
                if "NamedTuple" in base_names:
                    entries.append(_build_namedtuple_class(node))
                else:
                    class_entries = _visit_class(node)
                    entries.extend(class_entries)

        elif isinstance(node, (ast.Assign, ast.AnnAssign)):
            # Check for special type constructs before treating as constant
            type_entry = _visit_type_construct(node)
            if type_entry:
                name = type_entry.name
                if _should_export(name, all_names):
                    entries.append(type_entry)
                continue

            const = _visit_constant(node)
            if const:
                if _should_export(const.name, all_names):
                    entries.append(const)

    return AidFile(header=header, entries=entries)


def _build_header(tree: ast.Module, module_name: str, version: str) -> ModuleHeader:
    """Build the module header from the AST."""
    purpose = _get_docstring_first_line(tree)
    return ModuleHeader(
        module=module_name,
        lang="python",
        version=version,
        purpose=purpose,
        aid_version="0.1",
    )


def _visit_function(
    node: ast.FunctionDef | ast.AsyncFunctionDef,
    class_name: str | None = None,
) -> FnEntry:
    """Extract a function/method into a FnEntry."""
    name = f"{class_name}.{node.name}" if class_name else node.name
    is_async = isinstance(node, ast.AsyncFunctionDef)
    is_static = _has_decorator(node, "staticmethod")
    is_classmethod = _has_decorator(node, "classmethod")

    # Build signature
    sig = _build_signature(node, is_async, is_static, is_classmethod, class_name)

    # Build params (skip self/cls)
    params = _extract_params(node, class_name)

    # Return type
    returns_type = python_type_to_aid(node.returns) if node.returns else None
    returns = returns_type if returns_type and returns_type != "None" else None

    # Purpose from docstring
    purpose = _get_docstring_first_line(node)

    # Deprecated
    deprecated = None
    if _has_decorator(node, "deprecated"):
        deprecated = "Deprecated"

    entry = FnEntry(
        name=name,
        purpose=purpose,
        sigs=[sig],
        params=params if params else None,
        returns=returns,
        deprecated=deprecated,
    )

    return entry


def _build_signature(
    node: ast.FunctionDef | ast.AsyncFunctionDef,
    is_async: bool,
    is_static: bool,
    is_classmethod: bool,
    class_name: str | None,
) -> str:
    """Build the @sig string from a function's arguments and return type."""
    parts: list[str] = []

    args = node.args
    # Collect all positional args
    all_args = list(args.args)
    num_defaults = len(args.defaults)
    num_args = len(all_args)

    # Determine which args have defaults
    # defaults are right-aligned: if 3 args and 1 default, only the last has a default
    default_offset = num_args - num_defaults

    for i, arg in enumerate(all_args):
        # Skip self/cls for methods
        if i == 0 and class_name and not is_static:
            if arg.arg in ("self", "cls"):
                if arg.arg == "self" and not is_classmethod:
                    # Layer 1 defaults to `self` — Layer 2 determines mut
                    parts.append("self")
                continue

        type_str = python_type_to_aid(arg.annotation)
        has_default = i >= default_offset

        if has_default:
            parts.append(f"{arg.arg}?: {type_str}")
        else:
            parts.append(f"{arg.arg}: {type_str}")

    # *args
    if args.vararg:
        type_str = python_type_to_aid(args.vararg.annotation)
        parts.append(f"...{args.vararg.arg}: {type_str}")

    # keyword-only args
    kw_defaults = args.kw_defaults
    for i, arg in enumerate(args.kwonlyargs):
        type_str = python_type_to_aid(arg.annotation)
        has_default = kw_defaults[i] is not None
        if has_default:
            parts.append(f"{arg.arg}?: {type_str}")
        else:
            parts.append(f"{arg.arg}: {type_str}")

    # **kwargs
    if args.kwarg:
        type_str = python_type_to_aid(args.kwarg.annotation)
        parts.append(f"**{args.kwarg.arg}: {type_str}")

    # Return type
    return_type = python_type_to_aid(node.returns) if node.returns else "None"

    param_str = ", ".join(parts)
    prefix = "async " if is_async else ""

    return f"{prefix}({param_str}) -> {return_type}"


def _extract_params(
    node: ast.FunctionDef | ast.AsyncFunctionDef,
    class_name: str | None,
) -> list[Param]:
    """Extract parameter details for @params."""
    params: list[Param] = []
    args = node.args

    is_static = _has_decorator(node, "staticmethod")
    all_args = list(args.args)
    num_defaults = len(args.defaults)
    num_args = len(all_args)
    default_offset = num_args - num_defaults

    for i, arg in enumerate(all_args):
        # Skip self/cls
        if i == 0 and class_name and not is_static and arg.arg in ("self", "cls"):
            continue

        type_str = python_type_to_aid(arg.annotation) if arg.annotation else None
        has_default = i >= default_offset
        default_val = None
        if has_default:
            default_node = args.defaults[i - default_offset]
            default_val = _ast_to_str(default_node)

        params.append(Param(
            name=arg.arg,
            type=type_str,
            default=default_val,
        ))

    # *args
    if args.vararg:
        type_str = python_type_to_aid(args.vararg.annotation) if args.vararg.annotation else None
        params.append(Param(
            name=args.vararg.arg,
            type=type_str,
            is_variadic=True,
        ))

    # keyword-only args
    for i, arg in enumerate(args.kwonlyargs):
        type_str = python_type_to_aid(arg.annotation) if arg.annotation else None
        default_val = None
        if args.kw_defaults[i] is not None:
            default_val = _ast_to_str(args.kw_defaults[i])
        params.append(Param(
            name=arg.arg,
            type=type_str,
            default=default_val,
        ))

    return params


def _visit_class(node: ast.ClassDef) -> list[Entry]:
    """Extract a class into type/trait entries plus method entries."""
    entries: list[Entry] = []

    kind = _determine_class_kind(node)
    is_trait = kind in ("protocol", "abc")

    if is_trait:
        entries.append(_build_trait(node))
    else:
        entries.append(_build_type(node, kind))

    # Extract methods as separate FnEntry objects
    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if _is_public_method(item.name) and not _has_decorator(item, "property"):
                entries.append(_visit_function(item, class_name=node.name))

    return entries


def _build_type(node: ast.ClassDef, kind: str) -> TypeEntry:
    """Build a TypeEntry from a class definition."""
    purpose = _get_docstring_first_line(node)

    # Extends
    extends = _extract_bases(node)

    # Fields
    fields = _extract_fields(node, kind)

    # Variants (for enums)
    variants = _extract_variants(node) if kind == "enum" else None

    # Methods list
    methods = _extract_method_names(node)

    # Protocol detection
    protocols = detect_protocols(node)
    implements = protocols if protocols else None

    # Generic params
    generic_params = _extract_generic_params(node)

    # Constructors
    constructors = _extract_constructors(node)

    return TypeEntry(
        name=node.name,
        kind=kind,
        purpose=purpose,
        fields=fields if fields else None,
        variants=variants if variants else None,
        constructors=constructors,
        methods=methods if methods else None,
        extends=extends if extends else None,
        implements=implements,
        generic_params=generic_params,
    )


def _build_trait(node: ast.ClassDef) -> TraitEntry:
    """Build a TraitEntry from a Protocol/ABC class."""
    purpose = _get_docstring_first_line(node)
    is_protocol = "Protocol" in [_get_base_name(b) for b in node.bases]

    requires: list[str] = []
    provided: list[str] = []

    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if not _is_public_method(item.name):
                continue

            prefix = "async " if isinstance(item, ast.AsyncFunctionDef) else ""
            return_type = python_type_to_aid(item.returns) if item.returns else "None"

            # Build param list without self
            params: list[str] = []
            for i, arg in enumerate(item.args.args):
                if i == 0 and arg.arg == "self":
                    params.append("self")
                    continue
                type_str = python_type_to_aid(arg.annotation)
                params.append(f"{arg.arg}: {type_str}")

            method_sig = f"{prefix}fn {item.name}({', '.join(params)}) -> {return_type}"

            # Protocol methods are all required unless they have a real body.
            # ABC methods are required if decorated with @abstractmethod.
            if is_protocol or _has_decorator(item, "abstractmethod"):
                requires.append(method_sig)
            else:
                provided.append(method_sig)

    extends = _extract_bases(node)

    return TraitEntry(
        name=node.name,
        purpose=purpose,
        requires=requires,
        provided=provided if provided else None,
        extends=extends if extends else None,
    )


def _determine_class_kind(node: ast.ClassDef) -> str:
    """Determine the AID @kind for a Python class."""
    base_names = [_get_base_name(b) for b in node.bases]

    if "Enum" in base_names or "IntEnum" in base_names or "StrEnum" in base_names:
        return "enum"
    if "Protocol" in base_names:
        return "protocol"
    if "ABC" in base_names or any(
        _has_decorator(item, "abstractmethod")
        for item in node.body
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef))
    ):
        return "abc"
    if "TypedDict" in base_names:
        return "struct"
    if _has_decorator(node, "dataclass"):
        return "struct"

    return "class"


def _extract_bases(node: ast.ClassDef) -> list[str]:
    """Extract base classes, filtering out infrastructure bases."""
    skip = {"object", "ABC", "Protocol", "Enum", "IntEnum", "StrEnum",
            "TypedDict", "Generic", "BaseModel"}
    bases: list[str] = []
    for base in node.bases:
        name = _get_base_name(base)
        # Handle Generic[T] — skip Generic but extract type params elsewhere
        if isinstance(base, ast.Subscript):
            name = _get_base_name(base.value)
        if name and name not in skip:
            bases.append(name)
    return bases


def _extract_fields(node: ast.ClassDef, kind: str) -> list[Field_]:
    """Extract fields from a class."""
    fields: list[Field_] = []

    for item in node.body:
        # Annotated class attributes: name: Type
        if isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
            name = item.target.id
            if name.startswith("_"):
                continue
            type_str = python_type_to_aid(item.annotation)
            default = _ast_to_str(item.value) if item.value else None
            fields.append(Field_(name=name, type=type_str, default=default))

    # Also extract from __init__ if it exists and we don't already have fields
    if not fields and kind == "class":
        init = _find_method(node, "__init__")
        if init:
            for i, arg in enumerate(init.args.args):
                if i == 0 and arg.arg == "self":
                    continue
                if arg.arg.startswith("_"):
                    continue
                type_str = python_type_to_aid(arg.annotation) if arg.annotation else "any"
                fields.append(Field_(name=arg.arg, type=type_str))

    # Properties as fields
    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if _has_decorator(item, "property") and _is_public_method(item.name):
                type_str = python_type_to_aid(item.returns) if item.returns else "any"
                fields.append(Field_(name=item.name, type=type_str))

    return fields


def _extract_variants(node: ast.ClassDef) -> list[Variant]:
    """Extract enum variants from a class body."""
    variants: list[Variant] = []
    for item in node.body:
        if isinstance(item, ast.Assign):
            for target in item.targets:
                if isinstance(target, ast.Name) and not target.id.startswith("_"):
                    value = _ast_to_str(item.value) if item.value else None
                    variants.append(Variant(name=target.id, payload=value))
        elif isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
            if not item.target.id.startswith("_"):
                value = _ast_to_str(item.value) if item.value else None
                variants.append(Variant(name=item.target.id, payload=value))
    return variants


def _extract_method_names(node: ast.ClassDef) -> list[str]:
    """Get public method names from a class."""
    methods: list[str] = []
    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if _is_public_method(item.name) and not _has_decorator(item, "property"):
                methods.append(item.name)
    return methods


def _extract_generic_params(node: ast.ClassDef) -> str | None:
    """Extract generic type parameters from Generic[T, ...] base."""
    for base in node.bases:
        if isinstance(base, ast.Subscript):
            base_name = _get_base_name(base.value)
            if base_name == "Generic":
                args = []
                if isinstance(base.slice, ast.Tuple):
                    for elt in base.slice.elts:
                        args.append(python_type_to_aid(elt))
                else:
                    args.append(python_type_to_aid(base.slice))
                return ", ".join(args)
    return None


def _extract_constructors(node: ast.ClassDef) -> str | None:
    """Extract constructor info from __init__."""
    init = _find_method(node, "__init__")
    if not init:
        return None

    params: list[str] = []
    for i, arg in enumerate(init.args.args):
        if i == 0 and arg.arg == "self":
            continue
        type_str = python_type_to_aid(arg.annotation) if arg.annotation else "any"
        has_default = i >= len(init.args.args) - len(init.args.defaults)
        if has_default:
            params.append(f"{arg.arg}?: {type_str}")
        else:
            params.append(f"{arg.arg}: {type_str}")

    if params:
        return f"{node.name}({', '.join(params)})"
    return f"{node.name}()"


def _visit_constant(node: ast.Assign | ast.AnnAssign) -> ConstEntry | None:
    """Extract a module-level constant."""
    if isinstance(node, ast.AnnAssign):
        if not isinstance(node.target, ast.Name):
            return None
        name = node.target.id
        if not _is_upper_case(name):
            return None
        type_str = python_type_to_aid(node.annotation)
        value = _ast_to_str(node.value) if node.value else None
        return ConstEntry(name=name, type=type_str, value=value)

    if isinstance(node, ast.Assign):
        if len(node.targets) != 1 or not isinstance(node.targets[0], ast.Name):
            return None
        name = node.targets[0].id
        if not _is_upper_case(name):
            return None
        # Skip TypeVar, NewType, and other type construct calls
        if isinstance(node.value, ast.Call):
            call_name = _get_call_name(node.value)
            if call_name in ("TypeVar", "NewType", "ParamSpec", "TypeVarTuple"):
                return None
        type_str = _infer_type_from_value(node.value)
        value = _ast_to_str(node.value)
        return ConstEntry(name=name, type=type_str, value=value)

    return None


def _extract_all(tree: ast.Module) -> list[str] | None:
    """Extract __all__ list if defined. Returns None if no __all__."""
    for node in tree.body:
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == "__all__":
                    if isinstance(node.value, (ast.List, ast.Tuple)):
                        return [
                            elt.value for elt in node.value.elts
                            if isinstance(elt, ast.Constant) and isinstance(elt.value, str)
                        ]
    return None


def _should_export(name: str, all_names: list[str] | None) -> bool:
    """Check if a name should be exported based on __all__ and naming conventions."""
    if name.startswith("_"):
        return False
    if all_names is not None:
        return name in all_names
    return True


def _visit_type_construct(node: ast.Assign | ast.AnnAssign) -> TypeEntry | None:
    """Check if an assignment is a NewType, TypeAlias, or bare type alias."""
    if isinstance(node, ast.Assign):
        if len(node.targets) != 1 or not isinstance(node.targets[0], ast.Name):
            return None
        name = node.targets[0].id

        # Skip __all__, __version__, etc.
        if name.startswith("_"):
            return None

        # NewType("Name", BaseType)
        if (isinstance(node.value, ast.Call)
            and _get_call_name(node.value) == "NewType"
            and len(node.value.args) >= 2):
            base_type = python_type_to_aid(node.value.args[1])
            return TypeEntry(
                name=name,
                kind="newtype",
                purpose=f"Distinct type wrapping {base_type}",
                fields=[Field_(name="(inner)", type=base_type, description=f"The wrapped {base_type} value")],
            )

        # TypeVar — skip, not a type entry
        if (isinstance(node.value, ast.Call)
            and _get_call_name(node.value) == "TypeVar"):
            return None

        # Bare type alias: Name = dict[str, str] or Name = SomeType
        # Only if the value looks like a type (Name, Subscript), not a constant
        if _is_type_expression(node.value) and not _is_upper_case(name):
            aliased = python_type_to_aid(node.value)
            return TypeEntry(
                name=name,
                kind="alias",
                purpose=f"Alias for {aliased}",
            )

        return None

    if isinstance(node, ast.AnnAssign):
        if not isinstance(node.target, ast.Name):
            return None
        name = node.target.id

        # TypeAlias annotation: Name: TypeAlias = SomeType
        ann_name = ""
        if isinstance(node.annotation, ast.Name):
            ann_name = node.annotation.id
        elif isinstance(node.annotation, ast.Attribute):
            ann_name = node.annotation.attr

        if ann_name == "TypeAlias" and node.value:
            aliased = python_type_to_aid(node.value)
            return TypeEntry(
                name=name,
                kind="alias",
                purpose=f"Alias for {aliased}",
            )

        return None

    return None


def _build_namedtuple_class(node: ast.ClassDef) -> TypeEntry:
    """Build a TypeEntry from a NamedTuple subclass."""
    purpose = _get_docstring_first_line(node)
    fields: list[Field_] = []

    for item in node.body:
        if isinstance(item, ast.AnnAssign) and isinstance(item.target, ast.Name):
            name = item.target.id
            if name.startswith("_"):
                continue
            type_str = python_type_to_aid(item.annotation)
            default = _ast_to_str(item.value) if item.value else None
            fields.append(Field_(name=name, type=type_str, default=default))

    return TypeEntry(
        name=node.name,
        kind="struct",
        purpose=purpose,
        fields=fields if fields else None,
        constructors=f"{node.name}({', '.join(f'{f.name}: {f.type}' for f in fields)})" if fields else f"{node.name}()",
    )


def _get_call_name(node: ast.Call) -> str:
    """Get the function name from a Call node."""
    if isinstance(node.func, ast.Name):
        return node.func.id
    if isinstance(node.func, ast.Attribute):
        return node.func.attr
    return ""


def _is_type_expression(node: ast.expr) -> bool:
    """Check if an expression looks like a type reference (not a value)."""
    # Subscript like dict[str, str], list[int]
    if isinstance(node, ast.Subscript):
        return True
    # A capitalized name that's not UPPER_CASE (type, not constant)
    if isinstance(node, ast.Name):
        return node.id[0:1].isupper() and not _is_upper_case(node.id)
    # Attribute like typing.Dict
    if isinstance(node, ast.Attribute):
        return True
    # BinOp with | (union type)
    if isinstance(node, ast.BinOp) and isinstance(node.op, ast.BitOr):
        return True
    return False


# --- Helpers ---


def _is_public(name: str) -> bool:
    """Check if a name is public (doesn't start with _)."""
    return not name.startswith("_")


def _is_public_method(name: str) -> bool:
    """Check if a method name is public (not private, but dunders are handled separately)."""
    if name.startswith("__") and name.endswith("__"):
        return False  # Dunders handled by protocol detection
    return not name.startswith("_")


def _is_upper_case(name: str) -> bool:
    """Check if a name looks like a constant (UPPER_CASE)."""
    return bool(re.match(r'^[A-Z][A-Z0-9_]*$', name))


def _has_decorator(node: ast.FunctionDef | ast.AsyncFunctionDef | ast.ClassDef, name: str) -> bool:
    """Check if a function/class has a specific decorator."""
    for dec in node.decorator_list:
        if isinstance(dec, ast.Name) and dec.id == name:
            return True
        if isinstance(dec, ast.Attribute) and dec.attr == name:
            return True
        if isinstance(dec, ast.Call):
            if isinstance(dec.func, ast.Name) and dec.func.id == name:
                return True
            if isinstance(dec.func, ast.Attribute) and dec.func.attr == name:
                return True
    return False


def _get_base_name(node: ast.expr) -> str:
    """Get the name from a base class expression."""
    if isinstance(node, ast.Name):
        return node.id
    if isinstance(node, ast.Attribute):
        return node.attr
    if isinstance(node, ast.Subscript):
        return _get_base_name(node.value)
    return ""


def _find_method(node: ast.ClassDef, name: str) -> ast.FunctionDef | ast.AsyncFunctionDef | None:
    """Find a method by name in a class body."""
    for item in node.body:
        if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)) and item.name == name:
            return item
    return None


def _get_docstring_first_line(node: ast.AST) -> str | None:
    """Extract the first line of a docstring from a module, class, or function."""
    if isinstance(node, (ast.Module, ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)):
        if (node.body
            and isinstance(node.body[0], ast.Expr)
            and isinstance(node.body[0].value, ast.Constant)
            and isinstance(node.body[0].value.value, str)):
            docstring = node.body[0].value.value.strip()
            first_line = docstring.split("\n")[0].strip()
            # Truncate to 120 chars per AID spec
            if len(first_line) > 120:
                first_line = first_line[:117] + "..."
            return first_line
    return None


def _ast_to_str(node: ast.expr | None) -> str | None:
    """Convert an AST expression to a string representation."""
    if node is None:
        return None
    try:
        return ast.unparse(node)
    except Exception:
        return None


def _infer_type_from_value(node: ast.expr) -> str:
    """Infer an AID type from a constant's value."""
    if isinstance(node, ast.Constant):
        if isinstance(node.value, str):
            return "str"
        if isinstance(node.value, bool):  # bool before int — bool is subclass of int
            return "bool"
        if isinstance(node.value, int):
            return "int"
        if isinstance(node.value, float):
            return "float"
        if isinstance(node.value, bytes):
            return "bytes"
        if node.value is None:
            return "None"
    if isinstance(node, (ast.List, ast.ListComp)):
        return "[any]"
    if isinstance(node, (ast.Dict, ast.DictComp)):
        return "dict[any, any]"
    if isinstance(node, (ast.Set, ast.SetComp)):
        return "set[any]"
    if isinstance(node, ast.Tuple):
        return f"({', '.join('any' for _ in node.elts)})"
    return "any"
