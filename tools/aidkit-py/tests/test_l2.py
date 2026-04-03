"""Tests for the L2 pipeline."""
from __future__ import annotations

from aidkit.model import AidFile, Entry, Field, Header
from aidkit.l2.generator import build_generator_prompt, extract_relevant_files
from aidkit.l2.diff import diff_l1_aids, L1Diff
from aidkit.l2.reviewer import build_reviewer_prompt
from aidkit.l2.incremental import build_incremental_prompt, build_incremental_generator_prompt
from aidkit.l2.staleness import StaleClaim, _parse_hunk_range
from aidkit.model import SourceRef


def _make_entry(kind: str, name: str, **field_vals) -> Entry:
    fields = {}
    for k, v in field_vals.items():
        if isinstance(v, str):
            fields[k] = Field(name=k, inline_value=v)
        else:
            fields[k] = v
    return Entry(kind=kind, name=name, fields=fields)


def _make_l1(**kwargs) -> AidFile:
    return AidFile(
        header=kwargs.get("header", Header(module="test", lang="go")),
        entries=kwargs.get("entries", []),
    )


# ---------------------------------------------------------------------------
# extract_relevant_files
# ---------------------------------------------------------------------------

class TestExtractRelevantFiles:
    def test_no_source_files(self):
        l1 = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        assert extract_relevant_files(l1) == []

    def test_direct_source_files(self):
        l1 = _make_l1(entries=[
            _make_entry("fn", "Foo", sig="func Foo()", source_file="foo.go"),
            _make_entry("fn", "Bar", sig="func Bar()", source_file="bar.go"),
        ])
        files = extract_relevant_files(l1)
        assert files == ["bar.go", "foo.go"]

    def test_callee_files_included(self):
        l1 = _make_l1(entries=[
            _make_entry("fn", "Foo", sig="func Foo()", source_file="foo.go", calls="Bar, Baz"),
            _make_entry("fn", "Bar", sig="func Bar()", source_file="bar.go"),
            _make_entry("fn", "Baz", sig="func Baz()", source_file="baz.go"),
        ])
        files = extract_relevant_files(l1)
        assert "bar.go" in files
        assert "baz.go" in files

    def test_method_short_name_resolution(self):
        l1 = _make_l1(entries=[
            _make_entry("fn", "Handler.Process", sig="func (h *Handler) Process()", source_file="handler.go"),
            _make_entry("fn", "Caller", sig="func Caller()", source_file="caller.go", calls="Process"),
        ])
        files = extract_relevant_files(l1)
        assert "handler.go" in files


# ---------------------------------------------------------------------------
# diff_l1_aids
# ---------------------------------------------------------------------------

class TestDiffL1Aids:
    def test_new_entry(self):
        old = _make_l1(entries=[])
        new = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        diff = diff_l1_aids(old, new)
        assert len(diff.new) == 1
        assert diff.new[0].name == "Foo"

    def test_removed_entry(self):
        old = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        new = _make_l1(entries=[])
        diff = diff_l1_aids(old, new)
        assert len(diff.removed) == 1

    def test_unchanged_entry(self):
        entry = _make_entry("fn", "Foo", sig="func Foo()", params="x int")
        old = _make_l1(entries=[entry])
        new = _make_l1(entries=[entry])
        diff = diff_l1_aids(old, new)
        assert len(diff.unchanged) == 1

    def test_modified_entry(self):
        old_entry = _make_entry("fn", "Foo", sig="func Foo(x int)")
        new_entry = _make_entry("fn", "Foo", sig="func Foo(x int, y int)")
        old = _make_l1(entries=[old_entry])
        new = _make_l1(entries=[new_entry])
        diff = diff_l1_aids(old, new)
        assert len(diff.modified) == 1
        assert diff.modified[0].old.name == "Foo"

    def test_reverse_call_propagation(self):
        """If B is modified and A calls B, A should also be marked modified."""
        old_a = _make_entry("fn", "A", sig="func A()", calls="B")
        old_b = _make_entry("fn", "B", sig="func B(x int)")
        new_a = _make_entry("fn", "A", sig="func A()", calls="B")
        new_b = _make_entry("fn", "B", sig="func B(x int, y int)")

        old = _make_l1(entries=[old_a, old_b])
        new = _make_l1(entries=[new_a, new_b])
        diff = diff_l1_aids(old, new)

        modified_names = {p.new.name for p in diff.modified}
        assert "A" in modified_names
        assert "B" in modified_names
        assert len(diff.unchanged) == 0


# ---------------------------------------------------------------------------
# build_generator_prompt
# ---------------------------------------------------------------------------

class TestBuildGeneratorPrompt:
    def test_includes_l1_content(self):
        l1 = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        prompt = build_generator_prompt(l1, "/tmp/src")
        assert "@fn Foo" in prompt
        assert "Layer 1 AID" in prompt

    def test_includes_dep_aids(self):
        l1 = _make_l1()
        dep = _make_l1(header=Header(module="dep", lang="go"))
        prompt = build_generator_prompt(l1, "/tmp/src", dep_aids=[dep])
        assert "Related package AIDs" in prompt
        assert "@module dep" in prompt

    def test_conditional_error_map(self):
        l1 = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()", errors="ErrBad")])
        prompt = build_generator_prompt(l1, "/tmp/src")
        assert "@error_map" in prompt

    def test_no_error_map_without_errors(self):
        l1 = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        prompt = build_generator_prompt(l1, "/tmp/src")
        assert "@error_map" not in prompt


# ---------------------------------------------------------------------------
# build_reviewer_prompt
# ---------------------------------------------------------------------------

class TestBuildReviewerPrompt:
    def test_includes_draft(self):
        draft = _make_l1(entries=[_make_entry("fn", "Foo", sig="func Foo()")])
        prompt = build_reviewer_prompt(draft, "/tmp/project")
        assert "Reviewer" in prompt
        assert "@fn Foo" in prompt

    def test_includes_source_refs(self):
        entry = Entry(kind="fn", name="Foo", fields={
            "sig": Field(name="sig", inline_value="func Foo()"),
            "purpose": Field(name="purpose", inline_value="test", source_refs=[
                SourceRef(file="main.go", start_line=10, end_line=10),
            ]),
        })
        draft = _make_l1(entries=[entry])
        prompt = build_reviewer_prompt(draft, "/tmp/project")
        assert "main.go" in prompt


# ---------------------------------------------------------------------------
# build_incremental_prompt
# ---------------------------------------------------------------------------

class TestBuildIncrementalPrompt:
    def test_stale_claims(self):
        aid = _make_l1()
        claims = [StaleClaim(
            entry="Foo", field="purpose", reason="lines changed",
            ref=SourceRef(file="main.go", start_line=42, end_line=42),
            claim_text="Does stuff",
        )]
        prompt = build_incremental_prompt(aid, claims, "/tmp/project")
        assert "stale" in prompt.lower()
        assert "main.go" in prompt
        assert "Foo" in prompt


# ---------------------------------------------------------------------------
# build_incremental_generator_prompt
# ---------------------------------------------------------------------------

class TestBuildIncrementalGeneratorPrompt:
    def test_includes_change_summary(self):
        new_l1 = _make_l1(entries=[
            _make_entry("fn", "NewFn", sig="func NewFn()"),
            _make_entry("fn", "ModFn", sig="func ModFn(x int)"),
        ])
        diff = L1Diff(
            new=[_make_entry("fn", "NewFn", sig="func NewFn()")],
            modified=[],
            unchanged=[_make_entry("fn", "OldFn", sig="func OldFn()")],
            removed=[],
        )
        prompt = build_incremental_generator_prompt(new_l1, None, diff, "/tmp/src")
        assert "New entries: 1" in prompt
        assert "Unchanged entries: 1" in prompt


# ---------------------------------------------------------------------------
# parse_hunk_range
# ---------------------------------------------------------------------------

class TestParseHunkRange:
    def test_standard_hunk(self):
        start, end = _parse_hunk_range("@@ -10,5 +12,7 @@ func Foo()")
        assert start == 12
        assert end == 18

    def test_single_line_hunk(self):
        start, end = _parse_hunk_range("@@ -1 +1 @@")
        assert start == 1
        assert end == 1
