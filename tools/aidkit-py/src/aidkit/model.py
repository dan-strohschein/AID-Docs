"""Data model for parsed AID files, per format.md section 7."""
from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class SourceRef:
    """A parsed [src: file:line] reference linking a claim to code."""

    file: str
    start_line: int
    end_line: int  # Same as start_line for single-line refs

    def __str__(self) -> str:
        if self.start_line == self.end_line:
            return f"[src: {self.file}:{self.start_line}]"
        return f"[src: {self.file}:{self.start_line}-{self.end_line}]"


@dataclass
class Field:
    """A single @field and its value(s)."""

    name: str
    inline_value: str = ""
    lines: list[str] = field(default_factory=list)
    source_refs: list[SourceRef] = field(default_factory=list)

    def value(self) -> str:
        """Return the full field value -- inline value plus any continuation lines."""
        if not self.lines:
            return self.inline_value
        result = self.inline_value
        for line in self.lines:
            if result:
                result += "\n"
            result += line
        return result


@dataclass
class Header:
    """Module-level metadata from the first section."""

    module: str = ""
    lang: str = ""
    version: str = ""
    stability: str = ""
    purpose: str = ""
    deps: list[str] = field(default_factory=list)
    depends: list[str] = field(default_factory=list)
    source: str = ""
    code_version: str = ""
    aid_status: str = ""
    aid_generated_by: str = ""
    aid_reviewed_by: str = ""
    aid_version: str = ""
    extra: dict[str, str] = field(default_factory=dict)


@dataclass
class Entry:
    """A single API entry: @fn, @type, @trait, or @const."""

    kind: str  # "fn", "type", "trait", "const"
    name: str
    fields: dict[str, Field] = field(default_factory=dict)


@dataclass
class Annotation:
    """A module-level annotation block (Tier 2.5)."""

    kind: str  # "invariants", "antipatterns", "decision", "note"
    name: str = ""  # For decision/note: the identifier
    fields: dict[str, Field] = field(default_factory=dict)


@dataclass
class Workflow:
    """A @workflow block."""

    name: str
    fields: dict[str, Field] = field(default_factory=dict)


@dataclass
class Warning:
    """A non-fatal issue found during parsing or validation."""

    line: int
    message: str

    def __str__(self) -> str:
        if self.line > 0:
            return f"line {self.line}: {self.message}"
        return self.message


@dataclass
class AidFile:
    """Parsed representation of a complete .aid document."""

    header: Header = field(default_factory=Header)
    entries: list[Entry] = field(default_factory=list)
    annotations: list[Annotation] = field(default_factory=list)
    workflows: list[Workflow] = field(default_factory=list)
    comments: list[str] = field(default_factory=list)
    is_manifest: bool = False
