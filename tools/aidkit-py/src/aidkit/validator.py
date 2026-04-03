"""Validation rules for AID files.

All checks produce warnings, not errors -- a file with warnings is still usable.
"""
from __future__ import annotations

import re
from abc import ABC, abstractmethod
from dataclasses import dataclass
from enum import IntEnum, auto

from .model import AidFile, Field


# ---------------------------------------------------------------------------
# Severity
# ---------------------------------------------------------------------------

class Severity(IntEnum):
    INFO = auto()
    WARNING = auto()
    ERROR = auto()

    def __str__(self) -> str:
        return self.name


# ---------------------------------------------------------------------------
# Issue
# ---------------------------------------------------------------------------

@dataclass
class Issue:
    rule: str
    severity: Severity
    message: str
    entry: str = ""
    field: str = ""

    def __str__(self) -> str:
        loc = ""
        if self.entry:
            loc = self.entry
            if self.field:
                loc += "." + self.field
            loc += ": "
        return f"[{self.severity}] {loc}{self.message} ({self.rule})"


# ---------------------------------------------------------------------------
# Rule interface
# ---------------------------------------------------------------------------

class Rule(ABC):
    @abstractmethod
    def name(self) -> str: ...

    @abstractmethod
    def check(self, file: AidFile) -> list[Issue]: ...


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def validate(file: AidFile) -> list[Issue]:
    """Run all rules against a parsed AID file."""
    issues: list[Issue] = []
    for rule in all_rules():
        issues.extend(rule.check(file))
    return issues


def all_rules() -> list[Rule]:
    """Return all built-in validation rules."""
    return [
        HeaderCompleteRule(),
        RequiredFieldsRule(),
        MethodBindingRule(),
        CrossReferencesRule(),
        DecisionFieldsRule(),
        ManifestFieldsRule(),
        SourceRefFormatRule(),
        SourceRefSecurityRule(),
        StatusValidRule(),
        CodeVersionFormatRule(),
    ]


# ---------------------------------------------------------------------------
# Rule 1: header-complete
# ---------------------------------------------------------------------------

class HeaderCompleteRule(Rule):
    def name(self) -> str:
        return "header-complete"

    def check(self, file: AidFile) -> list[Issue]:
        if file.is_manifest:
            return []
        issues: list[Issue] = []
        if not file.header.module:
            issues.append(Issue(
                rule=self.name(), severity=Severity.ERROR,
                message="@module is required",
            ))
        if not file.header.lang:
            issues.append(Issue(
                rule=self.name(), severity=Severity.ERROR,
                message="@lang is required",
            ))
        if not file.header.version:
            issues.append(Issue(
                rule=self.name(), severity=Severity.WARNING,
                message="@version is missing",
            ))
        return issues


# ---------------------------------------------------------------------------
# Rule 2: required-fields
# ---------------------------------------------------------------------------

class RequiredFieldsRule(Rule):
    def name(self) -> str:
        return "required-fields"

    def check(self, file: AidFile) -> list[Issue]:
        issues: list[Issue] = []
        for e in file.entries:
            if "purpose" not in e.fields:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.WARNING,
                    entry=e.name, message="@purpose is missing",
                ))

            if e.kind == "fn":
                if "sig" not in e.fields:
                    issues.append(Issue(
                        rule=self.name(), severity=Severity.ERROR,
                        entry=e.name, message="@sig is required for @fn entries",
                    ))

            elif e.kind == "type":
                kind_val = ""
                if "kind" in e.fields:
                    kind_val = e.fields["kind"].inline_value
                else:
                    issues.append(Issue(
                        rule=self.name(), severity=Severity.ERROR,
                        entry=e.name, message="@kind is required for @type entries",
                    ))
                if kind_val in ("struct", "class"):
                    if "fields" not in e.fields:
                        issues.append(Issue(
                            rule=self.name(), severity=Severity.WARNING,
                            entry=e.name, message="@fields is expected for struct/class types",
                        ))
                elif kind_val in ("enum", "union"):
                    if "variants" not in e.fields:
                        issues.append(Issue(
                            rule=self.name(), severity=Severity.WARNING,
                            entry=e.name, message="@variants is expected for enum/union types",
                        ))

            elif e.kind == "trait":
                if "requires" not in e.fields:
                    issues.append(Issue(
                        rule=self.name(), severity=Severity.WARNING,
                        entry=e.name, message="@requires is expected for @trait entries",
                    ))

        for w in file.workflows:
            if "purpose" not in w.fields:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.WARNING,
                    entry=f"workflow:{w.name}", message="@purpose is missing",
                ))
            if "steps" not in w.fields:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.WARNING,
                    entry=f"workflow:{w.name}",
                    message="@steps is required for @workflow entries",
                ))
        return issues


# ---------------------------------------------------------------------------
# Rule 3: method-binding
# ---------------------------------------------------------------------------

class MethodBindingRule(Rule):
    def name(self) -> str:
        return "method-binding"

    def check(self, file: AidFile) -> list[Issue]:
        type_names: set[str] = set()
        type_methods: dict[str, set[str]] = {}

        for e in file.entries:
            if e.kind in ("type", "trait"):
                type_names.add(e.name)
                if "methods" in e.fields:
                    methods_val = e.fields["methods"].inline_value
                    type_methods[e.name] = {
                        m.strip() for m in methods_val.split(",") if m.strip()
                    }

        issues: list[Issue] = []
        for e in file.entries:
            if e.kind != "fn":
                continue
            dot_idx = e.name.find(".")
            if dot_idx < 0:
                continue
            type_name = e.name[:dot_idx]
            method_name = e.name[dot_idx + 1:]

            if type_name not in type_names:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.WARNING,
                    entry=e.name,
                    message=f"method on type {type_name!r} but no @type {type_name} entry exists",
                ))
                continue

            if type_name in type_methods and method_name not in type_methods[type_name]:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.INFO,
                    entry=e.name,
                    message=f"method {method_name!r} not listed in @type {type_name} @methods",
                ))
        return issues


# ---------------------------------------------------------------------------
# Rule 4: cross-references
# ---------------------------------------------------------------------------

class CrossReferencesRule(Rule):
    def name(self) -> str:
        return "cross-references"

    def check(self, file: AidFile) -> list[Issue]:
        names: set[str] = {e.name for e in file.entries}
        names.update(w.name for w in file.workflows)

        issues: list[Issue] = []
        for e in file.entries:
            if "related" not in e.fields:
                continue
            refs = e.fields["related"].inline_value.split(",")
            for ref in refs:
                ref = ref.strip()
                if not ref or "/" in ref:
                    continue
                if ref not in names:
                    issues.append(Issue(
                        rule=self.name(), severity=Severity.INFO,
                        entry=e.name, field="related",
                        message=f"@related reference {ref!r} not found in this file",
                    ))
        return issues


# ---------------------------------------------------------------------------
# Rule 5: decision-fields
# ---------------------------------------------------------------------------

class DecisionFieldsRule(Rule):
    def name(self) -> str:
        return "decision-fields"

    def check(self, file: AidFile) -> list[Issue]:
        issues: list[Issue] = []
        for a in file.annotations:
            if a.kind != "decision":
                continue
            entry_name = f"decision:{a.name}"
            for required in ("purpose", "chosen", "rationale"):
                if required not in a.fields:
                    issues.append(Issue(
                        rule=self.name(), severity=Severity.WARNING,
                        entry=entry_name,
                        message=f"@{required} is required for @decision blocks",
                    ))
        return issues


# ---------------------------------------------------------------------------
# Rule 6: manifest-fields
# ---------------------------------------------------------------------------

class ManifestFieldsRule(Rule):
    def name(self) -> str:
        return "manifest-fields"

    def check(self, file: AidFile) -> list[Issue]:
        if not file.is_manifest:
            return []
        issues: list[Issue] = []
        for e in file.entries:
            if e.kind != "package":
                continue
            entry_name = f"package:{e.name}"
            if "aid_file" not in e.fields:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.ERROR,
                    entry=entry_name,
                    message="@aid_file is required in manifest entries",
                ))
            if "purpose" not in e.fields:
                issues.append(Issue(
                    rule=self.name(), severity=Severity.WARNING,
                    entry=entry_name,
                    message="@purpose is recommended in manifest entries",
                ))
        return issues


# ---------------------------------------------------------------------------
# Rule 7: source-ref-format
# ---------------------------------------------------------------------------

_VALID_SRC_REF = re.compile(r"^[a-zA-Z0-9_./-]+:\d+(-\d+)?$")
_SRC_REF_INLINE = re.compile(r"\[src:\s*([^\]]+)\]")


class SourceRefFormatRule(Rule):
    def name(self) -> str:
        return "source-ref-format"

    def check(self, file: AidFile) -> list[Issue]:
        issues: list[Issue] = []

        def check_fields(entry_name: str, fields: dict[str, Field]) -> None:
            for field in fields.values():
                full_text = field.value()
                for match in _SRC_REF_INLINE.finditer(full_text):
                    parts = match.group(1).split(",")
                    for part in parts:
                        part = part.strip()
                        if not _VALID_SRC_REF.match(part):
                            issues.append(Issue(
                                rule=self.name(), severity=Severity.WARNING,
                                entry=entry_name, field=field.name,
                                message=f"malformed source reference: {part!r}",
                            ))

        for e in file.entries:
            check_fields(e.name, e.fields)
        for w in file.workflows:
            check_fields(f"workflow:{w.name}", w.fields)
        return issues


# ---------------------------------------------------------------------------
# Rule 8: source-ref-security
# ---------------------------------------------------------------------------

class SourceRefSecurityRule(Rule):
    def name(self) -> str:
        return "source-ref-security"

    def check(self, file: AidFile) -> list[Issue]:
        issues: list[Issue] = []

        def check_refs(entry_name: str, fields: dict[str, Field]) -> None:
            for field in fields.values():
                for ref in field.source_refs:
                    if ref.file.startswith("/"):
                        issues.append(Issue(
                            rule=self.name(), severity=Severity.ERROR,
                            entry=entry_name, field=field.name,
                            message=f"absolute path in [src:] reference: {ref.file}",
                        ))
                    if ".." in ref.file:
                        issues.append(Issue(
                            rule=self.name(), severity=Severity.ERROR,
                            entry=entry_name, field=field.name,
                            message=f"path traversal in [src:] reference: {ref.file}",
                        ))

        for e in file.entries:
            check_refs(e.name, e.fields)
        for w in file.workflows:
            check_refs(f"workflow:{w.name}", w.fields)
        for a in file.annotations:
            check_refs(f"{a.kind}:{a.name}", a.fields)
        return issues


# ---------------------------------------------------------------------------
# Rule 9: status-valid
# ---------------------------------------------------------------------------

_VALID_STATUSES = {"draft", "reviewed", "approved", "stale"}


class StatusValidRule(Rule):
    def name(self) -> str:
        return "status-valid"

    def check(self, file: AidFile) -> list[Issue]:
        if not file.header.aid_status:
            return []
        if file.header.aid_status not in _VALID_STATUSES:
            return [Issue(
                rule=self.name(), severity=Severity.WARNING,
                message=f"@aid_status {file.header.aid_status!r} is not one of: draft, reviewed, approved, stale",
            )]
        return []


# ---------------------------------------------------------------------------
# Rule 10: code-version-format
# ---------------------------------------------------------------------------

_CODE_VERSION_PATTERN = re.compile(r"^git:[a-f0-9]{7,40}$")


class CodeVersionFormatRule(Rule):
    def name(self) -> str:
        return "code-version-format"

    def check(self, file: AidFile) -> list[Issue]:
        if not file.header.code_version:
            return []
        if not _CODE_VERSION_PATTERN.match(file.header.code_version):
            return [Issue(
                rule=self.name(), severity=Severity.WARNING,
                message=f"@code_version {file.header.code_version!r} doesn't match format git:HASH",
            )]
        return []
