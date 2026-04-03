"""Tests for the AID validator."""
from __future__ import annotations

from aidkit.model import AidFile, Annotation, Entry, Field, Header, Workflow
from aidkit.validator import Severity, validate


def _make_aid(**kwargs) -> AidFile:
    """Helper to build an AidFile with defaults."""
    return AidFile(
        header=kwargs.get("header", Header(module="test", lang="go", version="1.0")),
        entries=kwargs.get("entries", []),
        annotations=kwargs.get("annotations", []),
        workflows=kwargs.get("workflows", []),
        is_manifest=kwargs.get("is_manifest", False),
    )


def _make_fn(name: str, **extra_fields) -> Entry:
    fields = {"sig": Field(name="sig", inline_value=f"func {name}()"), "purpose": Field(name="purpose", inline_value="test")}
    fields.update({k: Field(name=k, inline_value=v) if isinstance(v, str) else v for k, v in extra_fields.items()})
    return Entry(kind="fn", name=name, fields=fields)


# ---------------------------------------------------------------------------
# Rule 1: header-complete
# ---------------------------------------------------------------------------

class TestHeaderComplete:
    def test_complete_header(self):
        aid = _make_aid()
        issues = [i for i in validate(aid) if i.rule == "header-complete"]
        assert issues == []

    def test_missing_module(self):
        aid = _make_aid(header=Header(lang="go", version="1.0"))
        issues = [i for i in validate(aid) if i.rule == "header-complete"]
        assert any(i.severity == Severity.ERROR and "@module" in i.message for i in issues)

    def test_missing_lang(self):
        aid = _make_aid(header=Header(module="test", version="1.0"))
        issues = [i for i in validate(aid) if i.rule == "header-complete"]
        assert any(i.severity == Severity.ERROR and "@lang" in i.message for i in issues)

    def test_missing_version_warning(self):
        aid = _make_aid(header=Header(module="test", lang="go"))
        issues = [i for i in validate(aid) if i.rule == "header-complete"]
        assert any(i.severity == Severity.WARNING and "@version" in i.message for i in issues)

    def test_manifest_skips(self):
        aid = _make_aid(header=Header(), is_manifest=True)
        issues = [i for i in validate(aid) if i.rule == "header-complete"]
        assert issues == []


# ---------------------------------------------------------------------------
# Rule 2: required-fields
# ---------------------------------------------------------------------------

class TestRequiredFields:
    def test_fn_needs_sig(self):
        entry = Entry(kind="fn", name="Foo", fields={"purpose": Field(name="purpose", inline_value="test")})
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@sig" in i.message]
        assert len(issues) == 1

    def test_fn_needs_purpose(self):
        entry = Entry(kind="fn", name="Foo", fields={"sig": Field(name="sig", inline_value="func Foo()")})
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@purpose" in i.message]
        assert len(issues) == 1

    def test_type_needs_kind(self):
        entry = Entry(kind="type", name="Config", fields={"purpose": Field(name="purpose", inline_value="test")})
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@kind" in i.message]
        assert len(issues) == 1

    def test_struct_needs_fields(self):
        entry = Entry(kind="type", name="Config", fields={
            "purpose": Field(name="purpose", inline_value="test"),
            "kind": Field(name="kind", inline_value="struct"),
        })
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@fields" in i.message]
        assert len(issues) == 1

    def test_enum_needs_variants(self):
        entry = Entry(kind="type", name="Status", fields={
            "purpose": Field(name="purpose", inline_value="test"),
            "kind": Field(name="kind", inline_value="enum"),
        })
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@variants" in i.message]
        assert len(issues) == 1

    def test_trait_needs_requires(self):
        entry = Entry(kind="trait", name="Stringer", fields={"purpose": Field(name="purpose", inline_value="test")})
        aid = _make_aid(entries=[entry])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@requires" in i.message]
        assert len(issues) == 1

    def test_workflow_needs_steps(self):
        wf = Workflow(name="flow", fields={"purpose": Field(name="purpose", inline_value="test")})
        aid = _make_aid(workflows=[wf])
        issues = [i for i in validate(aid) if i.rule == "required-fields" and "@steps" in i.message]
        assert len(issues) == 1


# ---------------------------------------------------------------------------
# Rule 3: method-binding
# ---------------------------------------------------------------------------

class TestMethodBinding:
    def test_method_without_type(self):
        fn = _make_fn("Config.Get")
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "method-binding"]
        assert any("no @type Config" in i.message for i in issues)

    def test_method_with_type(self):
        typ = Entry(kind="type", name="Config", fields={
            "purpose": Field(name="purpose", inline_value="test"),
            "kind": Field(name="kind", inline_value="struct"),
            "fields": Field(name="fields", lines=["Name string"]),
            "methods": Field(name="methods", inline_value="Get, Set"),
        })
        fn = _make_fn("Config.Get")
        aid = _make_aid(entries=[typ, fn])
        issues = [i for i in validate(aid) if i.rule == "method-binding" and i.severity == Severity.WARNING]
        assert issues == []

    def test_method_not_listed(self):
        typ = Entry(kind="type", name="Config", fields={
            "purpose": Field(name="purpose", inline_value="test"),
            "kind": Field(name="kind", inline_value="struct"),
            "fields": Field(name="fields", lines=["Name string"]),
            "methods": Field(name="methods", inline_value="Get"),
        })
        fn = _make_fn("Config.Set")
        aid = _make_aid(entries=[typ, fn])
        issues = [i for i in validate(aid) if i.rule == "method-binding" and i.severity == Severity.INFO]
        assert any("not listed" in i.message for i in issues)


# ---------------------------------------------------------------------------
# Rule 4: cross-references
# ---------------------------------------------------------------------------

class TestCrossReferences:
    def test_valid_reference(self):
        fn1 = _make_fn("Foo", related=Field(name="related", inline_value="Bar"))
        fn2 = _make_fn("Bar")
        aid = _make_aid(entries=[fn1, fn2])
        issues = [i for i in validate(aid) if i.rule == "cross-references"]
        assert issues == []

    def test_invalid_reference(self):
        fn = _make_fn("Foo", related=Field(name="related", inline_value="NonExistent"))
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "cross-references"]
        assert len(issues) == 1

    def test_cross_module_skipped(self):
        fn = _make_fn("Foo", related=Field(name="related", inline_value="other/pkg.Bar"))
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "cross-references"]
        assert issues == []


# ---------------------------------------------------------------------------
# Rule 5: decision-fields
# ---------------------------------------------------------------------------

class TestDecisionFields:
    def test_complete_decision(self):
        ann = Annotation(kind="decision", name="use-mutex", fields={
            "purpose": Field(name="purpose", inline_value="Thread safety"),
            "chosen": Field(name="chosen", inline_value="sync.Mutex"),
            "rationale": Field(name="rationale", inline_value="Simple"),
        })
        aid = _make_aid(annotations=[ann])
        issues = [i for i in validate(aid) if i.rule == "decision-fields"]
        assert issues == []

    def test_incomplete_decision(self):
        ann = Annotation(kind="decision", name="use-mutex", fields={})
        aid = _make_aid(annotations=[ann])
        issues = [i for i in validate(aid) if i.rule == "decision-fields"]
        assert len(issues) == 3  # purpose, chosen, rationale


# ---------------------------------------------------------------------------
# Rule 6: manifest-fields
# ---------------------------------------------------------------------------

class TestManifestFields:
    def test_non_manifest_skips(self):
        aid = _make_aid()
        issues = [i for i in validate(aid) if i.rule == "manifest-fields"]
        assert issues == []

    def test_manifest_needs_aid_file(self):
        entry = Entry(kind="package", name="mypkg", fields={
            "purpose": Field(name="purpose", inline_value="test"),
        })
        aid = _make_aid(entries=[entry], is_manifest=True, header=Header())
        issues = [i for i in validate(aid) if i.rule == "manifest-fields"]
        assert any(i.severity == Severity.ERROR and "@aid_file" in i.message for i in issues)


# ---------------------------------------------------------------------------
# Rule 7: source-ref-format
# ---------------------------------------------------------------------------

class TestSourceRefFormat:
    def test_valid_ref(self):
        fn = _make_fn("Foo", errors=Field(name="errors", inline_value="err [src: main.go:42]"))
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "source-ref-format"]
        assert issues == []

    def test_malformed_ref(self):
        fn = _make_fn("Foo", errors=Field(name="errors", inline_value="err [src: no-line-number]"))
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "source-ref-format"]
        assert len(issues) == 1


# ---------------------------------------------------------------------------
# Rule 8: source-ref-security
# ---------------------------------------------------------------------------

class TestSourceRefSecurity:
    def test_absolute_path(self):
        from aidkit.model import SourceRef
        fn = Entry(kind="fn", name="Foo", fields={
            "sig": Field(name="sig", inline_value="func Foo()"),
            "purpose": Field(name="purpose", inline_value="test", source_refs=[
                SourceRef(file="/etc/passwd", start_line=1, end_line=1),
            ]),
        })
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "source-ref-security"]
        assert any("absolute path" in i.message for i in issues)

    def test_path_traversal(self):
        from aidkit.model import SourceRef
        fn = Entry(kind="fn", name="Foo", fields={
            "sig": Field(name="sig", inline_value="func Foo()"),
            "purpose": Field(name="purpose", inline_value="test", source_refs=[
                SourceRef(file="../../../etc/passwd", start_line=1, end_line=1),
            ]),
        })
        aid = _make_aid(entries=[fn])
        issues = [i for i in validate(aid) if i.rule == "source-ref-security"]
        assert any("path traversal" in i.message for i in issues)


# ---------------------------------------------------------------------------
# Rule 9: status-valid
# ---------------------------------------------------------------------------

class TestStatusValid:
    def test_valid_statuses(self):
        for status in ("draft", "reviewed", "approved", "stale"):
            aid = _make_aid(header=Header(module="t", lang="go", version="1", aid_status=status))
            issues = [i for i in validate(aid) if i.rule == "status-valid"]
            assert issues == [], f"Unexpected issue for status {status}"

    def test_invalid_status(self):
        aid = _make_aid(header=Header(module="t", lang="go", version="1", aid_status="bogus"))
        issues = [i for i in validate(aid) if i.rule == "status-valid"]
        assert len(issues) == 1

    def test_empty_status_ok(self):
        aid = _make_aid()
        issues = [i for i in validate(aid) if i.rule == "status-valid"]
        assert issues == []


# ---------------------------------------------------------------------------
# Rule 10: code-version-format
# ---------------------------------------------------------------------------

class TestCodeVersionFormat:
    def test_valid_format(self):
        aid = _make_aid(header=Header(module="t", lang="go", version="1", code_version="git:abc1234"))
        issues = [i for i in validate(aid) if i.rule == "code-version-format"]
        assert issues == []

    def test_invalid_format(self):
        aid = _make_aid(header=Header(module="t", lang="go", version="1", code_version="svn:12345"))
        issues = [i for i in validate(aid) if i.rule == "code-version-format"]
        assert len(issues) == 1

    def test_empty_ok(self):
        aid = _make_aid()
        issues = [i for i in validate(aid) if i.rule == "code-version-format"]
        assert issues == []
