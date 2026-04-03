"""Emitter converts parsed AID data structures back to .aid text format."""
from __future__ import annotations

from .model import AidFile, Annotation, Entry, Field, Header, Workflow


def emit(f: AidFile) -> str:
    """Convert an AidFile to .aid formatted text."""
    parts: list[str] = []

    for c in f.comments:
        parts.append(c + "\n")
    if f.comments:
        parts.append("\n")

    parts.append(_emit_header(f.header, f.is_manifest))

    for e in f.entries:
        parts.append("\n---\n\n")
        parts.append(_emit_entry(e))

    for a in f.annotations:
        parts.append("\n---\n\n")
        parts.append(_emit_annotation(a))

    for w in f.workflows:
        parts.append("\n---\n\n")
        parts.append(_emit_workflow(w))

    return "".join(parts)


def _emit_header(h: Header, is_manifest: bool) -> str:
    lines: list[str] = []
    if is_manifest:
        lines.append("@manifest\n")
    if h.module:
        lines.append(f"@module {h.module}\n")
    if h.lang:
        lines.append(f"@lang {h.lang}\n")
    if h.version:
        lines.append(f"@version {h.version}\n")
    if h.stability:
        lines.append(f"@stability {h.stability}\n")
    if h.purpose:
        lines.append(f"@purpose {h.purpose}\n")
    if h.deps:
        lines.append(f"@deps [{', '.join(h.deps)}]\n")
    if h.depends:
        lines.append(f"@depends [{', '.join(h.depends)}]\n")
    if h.source:
        lines.append(f"@source {h.source}\n")
    if h.code_version:
        lines.append(f"@code_version {h.code_version}\n")
    if h.aid_status:
        lines.append(f"@aid_status {h.aid_status}\n")
    if h.aid_generated_by:
        lines.append(f"@aid_generated_by {h.aid_generated_by}\n")
    if h.aid_reviewed_by:
        lines.append(f"@aid_reviewed_by {h.aid_reviewed_by}\n")
    if h.aid_version:
        lines.append(f"@aid_version {h.aid_version}\n")
    for k, v in h.extra.items():
        lines.append(f"@{k} {v}\n")
    return "".join(lines)


def _emit_entry(e: Entry) -> str:
    lines: list[str] = [f"@{e.kind} {e.name}\n"]
    lines.append(_emit_fields(e.fields, e.kind))
    return "".join(lines)


def _emit_annotation(a: Annotation) -> str:
    lines: list[str] = []
    if a.name:
        lines.append(f"@{a.kind} {a.name}\n")
    else:
        lines.append(f"@{a.kind}\n")

    # Block-style annotations: content stored under field with same key as kind
    if a.kind in a.fields:
        block_field = a.fields[a.kind]
        for line in block_field.lines:
            lines.append(f"  {line}\n")

    lines.append(_emit_fields(a.fields, a.kind))
    return "".join(lines)


def _emit_workflow(w: Workflow) -> str:
    lines: list[str] = [f"@workflow {w.name}\n"]
    lines.append(_emit_fields(w.fields, "workflow"))
    return "".join(lines)


def _emit_fields(fields: dict[str, Field], skip_key: str) -> str:
    order = _field_order(fields, skip_key)
    lines: list[str] = []
    for name in order:
        field = fields[name]
        if field.inline_value:
            lines.append(f"@{name} {field.inline_value}\n")
        elif field.lines:
            lines.append(f"@{name}\n")
        for line in field.lines:
            lines.append(f"  {line}\n")
    return "".join(lines)


# Priority fields in preferred order
_PRIORITY_FIELDS = [
    "purpose", "kind", "generic_params", "extends",
    "sig", "params", "returns", "errors",
    "fields", "variants", "invariants",
    "constructors", "methods", "implements",
    "requires", "provided", "implementors",
    "pre", "post", "effects", "thread_safety", "complexity",
    "steps", "errors_at", "antipatterns",
    "context", "chosen", "rejected", "rationale", "tradeoff",
    "since", "deprecated", "related", "platform",
    "aid_file", "aid_status", "depends", "layer", "key_risks",
    "type", "value",
    "example",
]


def _field_order(fields: dict[str, Field], skip_key: str) -> list[str]:
    """Return field names in a sensible emit order."""
    seen: set[str] = {skip_key}
    result: list[str] = []

    for name in _PRIORITY_FIELDS:
        if name in fields and name not in seen:
            result.append(name)
            seen.add(name)

    # Remaining fields alphabetically
    for name in sorted(fields):
        if name not in seen:
            result.append(name)

    return result
