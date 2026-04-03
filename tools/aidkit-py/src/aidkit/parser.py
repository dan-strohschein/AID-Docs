"""State machine parser for AID files, per format.md section 7."""
from __future__ import annotations

import re
from enum import IntEnum, auto
from pathlib import Path
from typing import TextIO

from .model import AidFile, Annotation, Entry, Field, Header, SourceRef, Warning, Workflow


# ---------------------------------------------------------------------------
# Line classification
# ---------------------------------------------------------------------------

class LineType(IntEnum):
    FIELD = auto()
    CONTINUATION = auto()
    SEPARATOR = auto()
    COMMENT = auto()
    BLANK = auto()


def classify_line(line: str) -> tuple[LineType, str, str]:
    """Classify a single AID line.

    Returns (line_type, field_name, value).
    For non-field lines, field_name and value may be empty.
    """
    trimmed = line.rstrip(" \t\r\n")

    if trimmed == "":
        return LineType.BLANK, "", ""

    if trimmed == "---":
        return LineType.SEPARATOR, "", ""

    if trimmed.startswith("//"):
        return LineType.COMMENT, "", trimmed

    if trimmed.startswith("@"):
        rest = trimmed[1:]
        space_idx = rest.find(" ")
        if space_idx < 0:
            return LineType.FIELD, rest, ""
        field_name = rest[:space_idx]
        value = rest[space_idx + 1:].strip()
        return LineType.FIELD, field_name, value

    # Continuation -- starts with 2+ spaces
    if len(line) >= 2 and line[0] == " " and line[1] == " ":
        return LineType.CONTINUATION, "", line[2:]

    # Fallback -- any leading whitespace
    if line[0] in (" ", "\t"):
        return LineType.CONTINUATION, "", line.lstrip(" \t")

    # Unknown -- treat as continuation
    return LineType.CONTINUATION, "", trimmed


# ---------------------------------------------------------------------------
# Source reference extraction
# ---------------------------------------------------------------------------

_SRC_REF_PATTERN = re.compile(r"\[src:\s*([^\]]+)\]")
_LINE_REF_PATTERN = re.compile(r"^(.+?):(\d+)(?:-(\d+))?$")


def _extract_source_refs(text: str) -> list[SourceRef]:
    """Find all [src: file:line] references in a string."""
    matches = _SRC_REF_PATTERN.findall(text)
    if not matches:
        return []

    refs: list[SourceRef] = []
    for content in matches:
        parts = content.split(",")
        for part in parts:
            part = part.strip()
            m = _LINE_REF_PATTERN.match(part)
            if m is None:
                continue
            file = m.group(1).strip()
            start_line = int(m.group(2))
            end_line = int(m.group(3)) if m.group(3) else start_line
            refs.append(SourceRef(file=file, start_line=start_line, end_line=end_line))
    return refs


# ---------------------------------------------------------------------------
# List parsing
# ---------------------------------------------------------------------------

def _parse_list(value: str) -> list[str]:
    """Parse '[a, b, c]' or 'a, b, c' into a list of strings."""
    value = value.strip().lstrip("[").rstrip("]")
    if not value:
        return []
    return [p.strip() for p in value.split(",") if p.strip()]


# ---------------------------------------------------------------------------
# Parser states
# ---------------------------------------------------------------------------

class _State(IntEnum):
    HEADER = auto()
    ENTRY = auto()
    FIELD_VALUE = auto()
    DONE = auto()


# Entry-starting field names
_ENTRY_KINDS: dict[str, str] = {
    "fn": "fn",
    "type": "type",
    "trait": "trait",
    "const": "const",
    "workflow": "workflow",
}

# Annotation-starting field names (module-level Tier 2.5 blocks)
_ANNOTATION_KINDS: set[str] = {
    "invariants", "invariant",
    "antipatterns", "antipattern",
    "decision", "note",
    "error_map", "lock",
}

_MANIFEST_FIELD = "manifest"


# ---------------------------------------------------------------------------
# Header helpers
# ---------------------------------------------------------------------------

def _set_header_field(header: Header, name: str, value: str) -> None:
    _HEADER_SETTERS.get(name, lambda h, v: h.extra.__setitem__(name, v))(header, value)


_HEADER_SETTERS: dict[str, object] = {
    "module": lambda h, v: setattr(h, "module", v),
    "lang": lambda h, v: setattr(h, "lang", v),
    "version": lambda h, v: setattr(h, "version", v),
    "stability": lambda h, v: setattr(h, "stability", v),
    "purpose": lambda h, v: setattr(h, "purpose", v),
    "deps": lambda h, v: setattr(h, "deps", _parse_list(v)),
    "depends": lambda h, v: setattr(h, "depends", _parse_list(v)),
    "source": lambda h, v: setattr(h, "source", v),
    "code_version": lambda h, v: setattr(h, "code_version", v),
    "aid_status": lambda h, v: setattr(h, "aid_status", v),
    "aid_generated_by": lambda h, v: setattr(h, "aid_generated_by", v),
    "aid_reviewed_by": lambda h, v: setattr(h, "aid_reviewed_by", v),
    "aid_version": lambda h, v: setattr(h, "aid_version", v),
}


def _append_header_field(header: Header, name: str, value: str) -> None:
    if name == "purpose":
        header.purpose += " " + value
    elif name in header.extra:
        header.extra[name] += "\n" + value


# ---------------------------------------------------------------------------
# Block field helpers
# ---------------------------------------------------------------------------

def _append_block_field(
    entry: Entry | None,
    workflow: Workflow | None,
    annotation: Annotation | None,
    field_name: str,
    value: str,
) -> None:
    fields: dict[str, Field] | None = None
    if entry is not None:
        fields = entry.fields
    elif workflow is not None:
        fields = workflow.fields
    elif annotation is not None:
        fields = annotation.fields

    if fields is None:
        return

    if field_name not in fields:
        fields[field_name] = Field(name=field_name)
    field = fields[field_name]
    field.lines.append(value)
    field.source_refs.extend(_extract_source_refs(value))


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def parse_file(path: str | Path) -> tuple[AidFile, list[Warning]]:
    """Read and parse an AID file from disk."""
    with open(path, "r", encoding="utf-8") as f:
        return parse(f)


def parse_string(content: str) -> tuple[AidFile, list[Warning]]:
    """Parse an AID document from a string."""
    import io
    return parse(io.StringIO(content))


def parse(reader: TextIO) -> tuple[AidFile, list[Warning]]:
    """Parse an AID document from a text reader."""
    result = AidFile()
    warnings: list[Warning] = []
    state = _State.HEADER

    current_entry: Entry | None = None
    current_workflow: Workflow | None = None
    current_annotation: Annotation | None = None
    current_field_name = ""

    def finish_block() -> None:
        nonlocal current_entry, current_workflow, current_annotation, current_field_name
        if current_entry is not None:
            result.entries.append(current_entry)
            current_entry = None
        if current_workflow is not None:
            result.workflows.append(current_workflow)
            current_workflow = None
        if current_annotation is not None:
            result.annotations.append(current_annotation)
            current_annotation = None
        current_field_name = ""

    line_num = 0
    for raw_line in reader:
        raw_line = raw_line.rstrip("\n")
        line_num += 1
        line_type, field_name, value = classify_line(raw_line)

        if state == _State.HEADER:
            if line_type == LineType.FIELD:
                if field_name == _MANIFEST_FIELD:
                    result.is_manifest = True
                _set_header_field(result.header, field_name, value)
                current_field_name = field_name

            elif line_type == LineType.CONTINUATION:
                if current_field_name:
                    _append_header_field(result.header, current_field_name, value)

            elif line_type == LineType.SEPARATOR:
                state = _State.ENTRY
                current_field_name = ""

            elif line_type == LineType.COMMENT:
                result.comments.append(value)

            # LineType.BLANK: skip

        elif state == _State.ENTRY:
            if line_type == LineType.FIELD:
                is_entry_kind = field_name in _ENTRY_KINDS
                is_annotation = field_name in _ANNOTATION_KINDS

                if result.is_manifest and field_name == "package":
                    current_entry = Entry(kind="package", name=value)
                    current_field_name = field_name
                    state = _State.FIELD_VALUE

                elif is_annotation:
                    current_annotation = Annotation(kind=field_name, name=value)
                    current_field_name = field_name
                    state = _State.FIELD_VALUE

                elif is_entry_kind:
                    kind = _ENTRY_KINDS[field_name]
                    if kind == "workflow":
                        current_workflow = Workflow(name=value)
                        current_field_name = field_name
                    else:
                        current_entry = Entry(kind=kind, name=value)
                        current_field_name = field_name
                    state = _State.FIELD_VALUE

                else:
                    warnings.append(Warning(
                        line=line_num,
                        message=f"field @{field_name} before entry declaration",
                    ))

            elif line_type == LineType.CONTINUATION:
                warnings.append(Warning(
                    line=line_num,
                    message="continuation line outside an entry",
                ))

            # COMMENT, BLANK, SEPARATOR: skip

        elif state == _State.FIELD_VALUE:
            if line_type == LineType.FIELD:
                current_field_name = field_name
                new_field = Field(
                    name=field_name,
                    inline_value=value,
                    source_refs=_extract_source_refs(value),
                )
                if current_entry is not None:
                    current_entry.fields[field_name] = new_field
                elif current_workflow is not None:
                    current_workflow.fields[field_name] = new_field
                elif current_annotation is not None:
                    current_annotation.fields[field_name] = new_field

            elif line_type == LineType.CONTINUATION:
                if current_field_name:
                    _append_block_field(
                        current_entry, current_workflow, current_annotation,
                        current_field_name, value,
                    )

            elif line_type == LineType.SEPARATOR:
                finish_block()
                state = _State.ENTRY

            # COMMENT, BLANK: skip

    # Finalize any open block
    finish_block()

    return result, warnings
