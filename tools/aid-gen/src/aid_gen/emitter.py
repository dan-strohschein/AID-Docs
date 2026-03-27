"""Emit AID model objects as .aid formatted text."""

from __future__ import annotations

from aid_gen.model import (
    AidFile,
    ConstEntry,
    FnEntry,
    ModuleHeader,
    Param,
    PlatformNote,
    TraitEntry,
    TypeEntry,
    Workflow,
)


def emit(aid_file: AidFile, include_provenance: bool = True) -> str:
    """Convert an AidFile to .aid formatted text."""
    parts: list[str] = []

    if include_provenance:
        parts.append("// [generated] Layer 1 mechanical extraction — not yet reviewed")

    parts.append(_emit_header(aid_file.header))

    # Order entries: constants, types (with methods grouped after), traits, standalone functions
    ordered = _order_entries(aid_file.entries)

    for entry in ordered:
        parts.append("---")
        if isinstance(entry, FnEntry):
            parts.append(_emit_fn(entry))
        elif isinstance(entry, TypeEntry):
            parts.append(_emit_type(entry))
        elif isinstance(entry, TraitEntry):
            parts.append(_emit_trait(entry))
        elif isinstance(entry, ConstEntry):
            parts.append(_emit_const(entry))

    for workflow in aid_file.workflows:
        parts.append("---")
        parts.append(_emit_workflow(workflow))

    return "\n\n".join(parts) + "\n"


def _order_entries(entries: list) -> list:
    """Order entries: constants, types (with their methods), traits, standalone functions."""
    consts: list = []
    types_with_methods: list = []
    traits: list = []
    standalone_fns: list = []

    # Collect type names for grouping methods
    type_names: set[str] = set()
    for entry in entries:
        if isinstance(entry, TypeEntry):
            type_names.add(entry.name)

    # Collect methods grouped by their parent type
    method_map: dict[str, list] = {}
    for entry in entries:
        if isinstance(entry, FnEntry) and "." in entry.name:
            parent = entry.name.split(".")[0]
            if parent in type_names:
                method_map.setdefault(parent, []).append(entry)

    # Now sort
    for entry in entries:
        if isinstance(entry, ConstEntry):
            consts.append(entry)
        elif isinstance(entry, TypeEntry):
            types_with_methods.append(entry)
            # Add methods right after their parent type
            for method in method_map.get(entry.name, []):
                types_with_methods.append(method)
        elif isinstance(entry, TraitEntry):
            traits.append(entry)
        elif isinstance(entry, FnEntry):
            # Only add if not already grouped with a type
            if "." not in entry.name or entry.name.split(".")[0] not in type_names:
                standalone_fns.append(entry)

    return consts + types_with_methods + traits + standalone_fns


def _emit_header(header: ModuleHeader) -> str:
    lines: list[str] = []
    lines.append(f"@module {header.module}")
    lines.append(f"@lang {header.lang}")
    lines.append(f"@version {header.version}")
    if header.stability:
        lines.append(f"@stability {header.stability}")
    if header.purpose:
        lines.append(f"@purpose {header.purpose}")
    if header.deps:
        lines.append(f"@deps [{', '.join(header.deps)}]")
    if header.source:
        lines.append(f"@source {header.source}")
    lines.append(f"@aid_version {header.aid_version}")
    return "\n".join(lines)


def _emit_fn(entry: FnEntry) -> str:
    lines: list[str] = []
    lines.append(f"@fn {entry.name}")
    if entry.purpose:
        lines.append(f"@purpose {entry.purpose}")
    for sig in entry.sigs:
        lines.append(f"@sig {sig}")
    if entry.params:
        lines.append("@params")
        for param in entry.params:
            lines.extend(_emit_param(param, indent=2))
    if entry.returns and not _is_redundant_returns(entry):
        lines.append(f"@returns {entry.returns}")
    if entry.errors:
        lines.append("@errors")
        for error in entry.errors:
            lines.append(f"  {error}")
    if entry.pre:
        lines.append(f"@pre {entry.pre}")
    if entry.post:
        lines.append(f"@post {entry.post}")
    if entry.effects:
        lines.append(f"@effects [{', '.join(entry.effects)}]")
    if entry.thread_safety:
        lines.append(f"@thread_safety {entry.thread_safety}")
    if entry.complexity:
        lines.append(f"@complexity {entry.complexity}")
    if entry.since:
        lines.append(f"@since {entry.since}")
    if entry.deprecated:
        lines.append(f"@deprecated {entry.deprecated}")
    if entry.related:
        lines.append(f"@related {', '.join(entry.related)}")
    if entry.platform:
        lines.extend(_emit_platform(entry.platform))
    if entry.example:
        lines.append("@example")
        for line in entry.example.splitlines():
            lines.append(f"  {line}")
    return "\n".join(lines)


def _emit_param(param: Param, indent: int) -> list[str]:
    prefix = " " * indent
    lines: list[str] = []

    parts: list[str] = []
    if param.type:
        parts.append(param.type)
    if param.description:
        parts.append(param.description)
    if param.default:
        parts.append(f"Default {param.default}.")

    name = f"...{param.name}" if param.is_variadic else param.name
    detail = " — ".join(parts) if parts else ""

    if detail:
        lines.append(f"{prefix}{name}: {detail}")
    else:
        lines.append(f"{prefix}{name}:")

    for sub in param.sub_params:
        sub_parts: list[str] = []
        if sub.type:
            sub_parts.append(sub.type)
        if sub.description:
            sub_parts.append(sub.description)
        if sub.default:
            sub_parts.append(f"Default {sub.default}.")
        sub_detail = " — ".join(sub_parts) if sub_parts else ""
        if sub_detail:
            lines.append(f"{prefix}  .{sub.name}: {sub_detail}")
        else:
            lines.append(f"{prefix}  .{sub.name}:")

    return lines


def _emit_type(entry: TypeEntry) -> str:
    lines: list[str] = []
    lines.append(f"@type {entry.name}")
    lines.append(f"@kind {entry.kind}")
    if entry.purpose:
        lines.append(f"@purpose {entry.purpose}")
    if entry.generic_params:
        lines.append(f"@generic_params {entry.generic_params}")
    if entry.extends:
        lines.append(f"@extends {', '.join(entry.extends)}")
    if entry.fields:
        lines.append("@fields")
        for f in entry.fields:
            parts: list[str] = [f.type]
            if f.description:
                parts.append(f.description)
            if f.constraints:
                parts.append(f.constraints)
            if f.default:
                parts.append(f"Default {f.default}.")
            lines.append(f"  {f.name}: {' — '.join(parts)}" if len(parts) > 1
                         else f"  {f.name}: {parts[0]}")
    if entry.variants:
        lines.append("@variants")
        for v in entry.variants:
            payload = f"({v.payload})" if v.payload else ""
            desc = f" — {v.description}" if v.description else ""
            lines.append(f"  | {v.name}{payload}{desc}")
    if entry.invariants:
        lines.append("@invariants")
        for inv in entry.invariants:
            lines.append(f"  - {inv}")
    if entry.constructors:
        lines.append(f"@constructors {entry.constructors}")
    if entry.methods:
        lines.append(f"@methods {', '.join(entry.methods)}")
    if entry.implements:
        lines.append(f"@implements [{', '.join(entry.implements)}]")
    if entry.platform:
        lines.extend(_emit_platform(entry.platform))
    if entry.since:
        lines.append(f"@since {entry.since}")
    if entry.deprecated:
        lines.append(f"@deprecated {entry.deprecated}")
    if entry.related:
        lines.append(f"@related {', '.join(entry.related)}")
    if entry.example:
        lines.append("@example")
        for line in entry.example.splitlines():
            lines.append(f"  {line}")
    return "\n".join(lines)


def _emit_trait(entry: TraitEntry) -> str:
    lines: list[str] = []
    lines.append(f"@trait {entry.name}")
    if entry.purpose:
        lines.append(f"@purpose {entry.purpose}")
    if entry.extends:
        lines.append(f"@extends {', '.join(entry.extends)}")
    if entry.requires:
        lines.append("@requires")
        for req in entry.requires:
            lines.append(f"  {req}")
    if entry.provided:
        lines.append("@provided")
        for prov in entry.provided:
            lines.append(f"  {prov}")
    if entry.implementors:
        lines.append(f"@implementors [{', '.join(entry.implementors)}]")
    if entry.related:
        lines.append(f"@related {', '.join(entry.related)}")
    return "\n".join(lines)


def _emit_const(entry: ConstEntry) -> str:
    lines: list[str] = []
    lines.append(f"@const {entry.name}")
    if entry.purpose:
        lines.append(f"@purpose {entry.purpose}")
    lines.append(f"@type {entry.type}")
    if entry.value:
        lines.append(f"@value {entry.value}")
    if entry.since:
        lines.append(f"@since {entry.since}")
    return "\n".join(lines)


def _emit_workflow(workflow: Workflow) -> str:
    lines: list[str] = []
    lines.append(f"@workflow {workflow.name}")
    if workflow.purpose:
        lines.append(f"@purpose {workflow.purpose}")
    if workflow.steps:
        lines.append("@steps")
        for step in workflow.steps:
            lines.append(f"  {step}")
    if workflow.errors_at:
        lines.append("@errors_at")
        for err in workflow.errors_at:
            lines.append(f"  {err}")
    if workflow.antipatterns:
        lines.append("@antipatterns")
        for ap in workflow.antipatterns:
            lines.append(f"  - {ap}")
    if workflow.variants:
        lines.append("@variants")
        for var in workflow.variants:
            lines.append(f"  - {var}")
    if workflow.example:
        lines.append("@example")
        for line in workflow.example.splitlines():
            lines.append(f"  {line}")
    return "\n".join(lines)


def _emit_platform(notes: list[PlatformNote]) -> list[str]:
    lines: list[str] = ["@platform"]
    for note in notes:
        lines.append(f"  {note.platform}: {note.note}")
    return lines


def _is_redundant_returns(entry: FnEntry) -> bool:
    """Check if @returns is just a bare type name already visible in @sig.

    @returns is redundant when it's just repeating the return type from the sig
    without adding semantic information (e.g., "str" when sig shows "-> str").
    """
    if not entry.returns or not entry.sigs:
        return False

    returns = entry.returns.strip()

    # If the returns value contains spaces or descriptive words, it's not redundant
    # (e.g., "Response with status and body" is useful, "Response" alone is not)
    if " " in returns:
        return False

    # Check if the sig already shows this exact return type
    for sig in entry.sigs:
        if f"-> {returns}" in sig:
            return True

    return False
