"""Tests for the AID parser."""
from __future__ import annotations

from aidkit.model import AidFile
from aidkit.parser import LineType, classify_line, parse_string


# ---------------------------------------------------------------------------
# classify_line tests
# ---------------------------------------------------------------------------

class TestClassifyLine:
    def test_blank(self):
        assert classify_line("")[0] == LineType.BLANK
        assert classify_line("   ")[0] == LineType.BLANK

    def test_separator(self):
        lt, fn, val = classify_line("---")
        assert lt == LineType.SEPARATOR

    def test_comment(self):
        lt, fn, val = classify_line("// This is a comment")
        assert lt == LineType.COMMENT
        assert val == "// This is a comment"

    def test_field_with_value(self):
        lt, fn, val = classify_line("@module mypackage")
        assert lt == LineType.FIELD
        assert fn == "module"
        assert val == "mypackage"

    def test_field_no_value(self):
        lt, fn, val = classify_line("@invariants")
        assert lt == LineType.FIELD
        assert fn == "invariants"
        assert val == ""

    def test_continuation(self):
        lt, fn, val = classify_line("  some continuation text")
        assert lt == LineType.CONTINUATION
        assert val == "some continuation text"

    def test_continuation_preserves_extra_indent(self):
        lt, fn, val = classify_line("    deeply indented")
        assert lt == LineType.CONTINUATION
        assert val == "  deeply indented"

    def test_tab_continuation(self):
        lt, fn, val = classify_line("\tcontent")
        assert lt == LineType.CONTINUATION


# ---------------------------------------------------------------------------
# Parser: header
# ---------------------------------------------------------------------------

class TestParserHeader:
    def test_basic_header(self):
        content = """\
@module mypackage
@lang go
@version 1.0.0
@purpose A test module
"""
        aid, warnings = parse_string(content)
        assert aid.header.module == "mypackage"
        assert aid.header.lang == "go"
        assert aid.header.version == "1.0.0"
        assert aid.header.purpose == "A test module"
        assert warnings == []

    def test_header_with_deps(self):
        content = "@deps [fmt, os, net/http]\n"
        aid, _ = parse_string(content)
        assert aid.header.deps == ["fmt", "os", "net/http"]

    def test_header_with_depends(self):
        content = "@depends [pkg/parser, pkg/emitter]\n"
        aid, _ = parse_string(content)
        assert aid.header.depends == ["pkg/parser", "pkg/emitter"]

    def test_header_extra_fields(self):
        content = "@module test\n@custom_field value\n"
        aid, _ = parse_string(content)
        assert aid.header.extra["custom_field"] == "value"

    def test_header_aid_fields(self):
        content = """\
@module test
@code_version git:abc1234
@aid_status draft
@aid_generated_by layer1
@aid_reviewed_by human
@aid_version 2.0
"""
        aid, _ = parse_string(content)
        assert aid.header.code_version == "git:abc1234"
        assert aid.header.aid_status == "draft"
        assert aid.header.aid_generated_by == "layer1"
        assert aid.header.aid_reviewed_by == "human"
        assert aid.header.aid_version == "2.0"

    def test_header_purpose_continuation(self):
        content = """\
@purpose First line
  continued here
"""
        aid, _ = parse_string(content)
        assert aid.header.purpose == "First line continued here"

    def test_comments(self):
        content = """\
// provenance marker
@module test
"""
        aid, _ = parse_string(content)
        assert "// provenance marker" in aid.comments


# ---------------------------------------------------------------------------
# Parser: entries
# ---------------------------------------------------------------------------

class TestParserEntries:
    def test_fn_entry(self):
        content = """\
@module test
@lang go
---
@fn DoSomething
@sig func DoSomething(ctx context.Context) error
@purpose Does something useful
"""
        aid, warnings = parse_string(content)
        assert len(aid.entries) == 1
        e = aid.entries[0]
        assert e.kind == "fn"
        assert e.name == "DoSomething"
        assert e.fields["sig"].inline_value == "func DoSomething(ctx context.Context) error"
        assert e.fields["purpose"].inline_value == "Does something useful"

    def test_type_entry(self):
        content = """\
@module test
@lang go
---
@type Config
@kind struct
@purpose Configuration options
@fields
  Name string — the config name
  Value int — the config value
"""
        aid, _ = parse_string(content)
        assert len(aid.entries) == 1
        e = aid.entries[0]
        assert e.kind == "type"
        assert e.name == "Config"
        assert e.fields["kind"].inline_value == "struct"
        assert len(e.fields["fields"].lines) == 2

    def test_trait_entry(self):
        content = """\
@module test
@lang go
---
@trait Stringer
@purpose Things that convert to string
@requires
  String() string
"""
        aid, _ = parse_string(content)
        assert aid.entries[0].kind == "trait"
        assert aid.entries[0].name == "Stringer"

    def test_const_entry(self):
        content = """\
@module test
@lang go
---
@const MaxSize
@type int
@value 1024
@purpose Maximum buffer size
"""
        aid, _ = parse_string(content)
        e = aid.entries[0]
        assert e.kind == "const"
        assert e.name == "MaxSize"

    def test_multiple_entries(self):
        content = """\
@module test
@lang go
---
@fn First
@sig func First()
@purpose First function
---
@fn Second
@sig func Second()
@purpose Second function
"""
        aid, _ = parse_string(content)
        assert len(aid.entries) == 2
        assert aid.entries[0].name == "First"
        assert aid.entries[1].name == "Second"


# ---------------------------------------------------------------------------
# Parser: workflows
# ---------------------------------------------------------------------------

class TestParserWorkflows:
    def test_workflow(self):
        content = """\
@module test
@lang go
---
@workflow request-handling
@purpose How a request is processed
@steps
  1. Parse request
  2. Validate input
  3. Process data
  4. Return response
"""
        aid, _ = parse_string(content)
        assert len(aid.workflows) == 1
        w = aid.workflows[0]
        assert w.name == "request-handling"
        assert len(w.fields["steps"].lines) == 4


# ---------------------------------------------------------------------------
# Parser: annotations
# ---------------------------------------------------------------------------

class TestParserAnnotations:
    def test_invariants(self):
        content = """\
@module test
@lang go
---
@invariants
  Buffer is never nil after init [src: buffer.go:42]
  Size >= 0 always [src: buffer.go:55]
"""
        aid, _ = parse_string(content)
        assert len(aid.annotations) == 1
        a = aid.annotations[0]
        assert a.kind == "invariants"
        assert len(a.fields["invariants"].lines) == 2

    def test_decision(self):
        content = """\
@module test
@lang go
---
@decision use-mutex
@purpose Thread safety strategy
@chosen sync.Mutex
@rationale Simple and sufficient for our workload
"""
        aid, _ = parse_string(content)
        a = aid.annotations[0]
        assert a.kind == "decision"
        assert a.name == "use-mutex"
        assert a.fields["purpose"].inline_value == "Thread safety strategy"
        assert a.fields["chosen"].inline_value == "sync.Mutex"


# ---------------------------------------------------------------------------
# Parser: manifest
# ---------------------------------------------------------------------------

class TestParserManifest:
    def test_manifest(self):
        content = """\
@manifest
@project MyProject
---
@package mypackage
@aid_file mypackage.aid
@purpose Core logic
"""
        aid, _ = parse_string(content)
        assert aid.is_manifest is True
        assert len(aid.entries) == 1
        e = aid.entries[0]
        assert e.kind == "package"
        assert e.name == "mypackage"
        assert e.fields["aid_file"].inline_value == "mypackage.aid"


# ---------------------------------------------------------------------------
# Parser: source refs
# ---------------------------------------------------------------------------

class TestSourceRefs:
    def test_inline_source_ref(self):
        content = """\
@module test
@lang go
---
@fn DoThing
@sig func DoThing()
@purpose Does a thing [src: main.go:42]
"""
        aid, _ = parse_string(content)
        refs = aid.entries[0].fields["purpose"].source_refs
        assert len(refs) == 1
        assert refs[0].file == "main.go"
        assert refs[0].start_line == 42
        assert refs[0].end_line == 42

    def test_range_source_ref(self):
        content = """\
@module test
@lang go
---
@fn DoThing
@sig func DoThing()
@errors
  Returns ErrNotFound if missing [src: handler.go:10-25]
"""
        aid, _ = parse_string(content)
        refs = aid.entries[0].fields["errors"].source_refs
        assert len(refs) == 1
        assert refs[0].start_line == 10
        assert refs[0].end_line == 25

    def test_source_ref_string(self):
        from aidkit.model import SourceRef
        ref = SourceRef(file="main.go", start_line=42, end_line=42)
        assert str(ref) == "[src: main.go:42]"
        ref2 = SourceRef(file="main.go", start_line=10, end_line=25)
        assert str(ref2) == "[src: main.go:10-25]"


# ---------------------------------------------------------------------------
# Parser: warnings
# ---------------------------------------------------------------------------

class TestParserWarnings:
    def test_field_before_entry(self):
        content = """\
@module test
@lang go
---
@purpose orphaned purpose
"""
        _, warnings = parse_string(content)
        assert len(warnings) == 1
        assert "before entry declaration" in warnings[0].message

    def test_continuation_outside_entry(self):
        content = """\
@module test
@lang go
---
  orphaned continuation
"""
        _, warnings = parse_string(content)
        assert len(warnings) == 1
        assert "continuation line outside an entry" in warnings[0].message
