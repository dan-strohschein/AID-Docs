"""AID data model — structured representation of an AID document."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Union


@dataclass
class Param:
    """A function parameter with optional type, description, and constraints."""
    name: str
    type: str | None = None
    description: str | None = None
    default: str | None = None
    is_variadic: bool = False
    sub_params: list[Param] = field(default_factory=list)


@dataclass
class Field_:
    """A field on a struct/class type."""
    name: str
    type: str
    description: str | None = None
    default: str | None = None
    constraints: str | None = None


@dataclass
class Variant:
    """A variant of an enum/union type."""
    name: str
    payload: str | None = None
    description: str | None = None


@dataclass
class PlatformNote:
    """Platform-specific behavior note."""
    platform: str
    note: str


@dataclass
class ModuleHeader:
    """The module header section of an AID file."""
    module: str
    lang: str = "python"
    version: str = "0.0.0"
    stability: str | None = None
    purpose: str | None = None
    deps: list[str] | None = None
    source: str | None = None
    aid_version: str = "0.1"


@dataclass
class FnEntry:
    """A function or method entry."""
    name: str
    purpose: str | None = None
    sigs: list[str] = field(default_factory=list)
    params: list[Param] | None = None
    returns: str | None = None
    errors: list[str] | None = None
    pre: str | None = None
    post: str | None = None
    effects: list[str] | None = None
    thread_safety: str | None = None
    complexity: str | None = None
    since: str | None = None
    deprecated: str | None = None
    related: list[str] | None = None
    platform: list[PlatformNote] | None = None
    example: str | None = None


@dataclass
class TypeEntry:
    """A type entry (struct, class, enum, union, alias, newtype)."""
    name: str
    kind: str  # struct, enum, union, class, alias, newtype
    purpose: str | None = None
    fields: list[Field_] | None = None
    variants: list[Variant] | None = None
    invariants: list[str] | None = None
    constructors: str | None = None
    methods: list[str] | None = None
    extends: list[str] | None = None
    implements: list[str] | None = None
    generic_params: str | None = None
    platform: list[PlatformNote] | None = None
    since: str | None = None
    deprecated: str | None = None
    related: list[str] | None = None
    example: str | None = None


@dataclass
class TraitEntry:
    """A trait/interface/protocol entry."""
    name: str
    purpose: str | None = None
    requires: list[str] = field(default_factory=list)
    provided: list[str] | None = None
    implementors: list[str] | None = None
    extends: list[str] | None = None
    related: list[str] | None = None


@dataclass
class ConstEntry:
    """A constant entry."""
    name: str
    purpose: str | None = None
    type: str = "any"
    value: str | None = None
    since: str | None = None


@dataclass
class Workflow:
    """A workflow entry (largely empty from Layer 1)."""
    name: str
    purpose: str | None = None
    steps: list[str] = field(default_factory=list)
    errors_at: list[str] | None = None
    antipatterns: list[str] | None = None
    variants: list[str] | None = None
    example: str | None = None


# Union of all entry types
Entry = Union[FnEntry, TypeEntry, TraitEntry, ConstEntry]


@dataclass
class AidFile:
    """A complete AID document."""
    header: ModuleHeader
    entries: list[Entry] = field(default_factory=list)
    workflows: list[Workflow] = field(default_factory=list)
