package main

import (
	"strings"
	"testing"
)

func TestExtractTests_CollectsMocksAndHelpers(t *testing.T) {
	src := `mod demo

pub fn real_fn() -> i64 { 42 }

struct MockClient { calls: i64 }
struct PlainStruct { x: i64 }

fn setup_fixture() -> i64 { 1 }
fn teardown() {}
fn helper_build() -> i64 { 2 }
fn unrelated() -> i64 { 3 }

test "adds correctly" {
    real_fn()
}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.ExtractTests(dir, "demo_test", "0.0.0")
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]Entry{}
	for _, ent := range aid.Entries {
		names[entryName(ent)] = ent
	}

	// Scaffolding type kept, plain struct dropped.
	if _, ok := names["MockClient"]; !ok {
		t.Errorf("MockClient missing; entries: %v", names)
	}
	if _, ok := names["PlainStruct"]; ok {
		t.Errorf("PlainStruct should be filtered")
	}

	// Helper fns kept, unrelated fn dropped, production real_fn dropped.
	for _, want := range []string{"setup_fixture", "teardown", "helper_build"} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing helper %q", want)
		}
	}
	for _, reject := range []string{"unrelated", "real_fn"} {
		if _, ok := names[reject]; ok {
			t.Errorf("%q should not be in test AID", reject)
		}
	}

	// Test block synthesised as test_<slug> with @calls → real_fn.
	tb, ok := names["test_adds_correctly"].(FnEntry)
	if !ok {
		t.Fatalf("test block entry missing: %v", names)
	}
	if tb.Purpose != `Test: adds correctly` {
		t.Errorf("purpose = %q", tb.Purpose)
	}
	found := false
	for _, c := range tb.Calls {
		if c == "real_fn" {
			found = true
		}
	}
	if !found {
		t.Errorf("test_adds_correctly.Calls = %v, want real_fn", tb.Calls)
	}
}

func TestExtractTests_EmitterRendersSynthEntry(t *testing.T) {
	src := `mod demo
pub fn real_fn() {}
test "smoke" { real_fn() }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.ExtractTests(dir, "demo_test", "0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	out := Emit(aid)
	if !strings.Contains(out, "@fn test_smoke") {
		t.Errorf("expected @fn test_smoke in:\n%s", out)
	}
	if !strings.Contains(out, `@purpose Test: smoke`) {
		t.Errorf("expected Test: smoke purpose")
	}
}

func TestExtractTests_NoScaffoldReturnsError(t *testing.T) {
	src := `mod demo
pub fn f() {}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	_, err := e.ExtractTests(dir, "demo_test", "0.0.0")
	if err == nil {
		t.Fatal("expected error when no test scaffolding present")
	}
}

func TestExtract_TestBlocksFilteredFromMainAID(t *testing.T) {
	src := `mod demo
pub fn real_fn() {}
test "smoke" { real_fn() }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range aid.Entries {
		name := entryName(ent)
		if strings.HasPrefix(name, "test_") || strings.Contains(name, "smoke") {
			t.Errorf("test entry %q leaked into main AID", name)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct{ in, want string }{
		{"adds correctly", "adds_correctly"},
		{"handles !!! bang", "handles_bang"},
		{"UPPER case", "upper_case"},
		{"  spaced  ", "spaced"},
		{"!!!", "anon"},
		{"", "anon"},
	}
	for _, tt := range tests {
		if got := slugify(tt.in); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
