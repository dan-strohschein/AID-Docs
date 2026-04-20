package main

import (
	"reflect"
	"testing"
)

func TestCallGraph_FreeFunctionCalls(t *testing.T) {
	src := `mod demo

pub fn a() -> i64 {
    x := b()
    c(x) + d()
}

fn b() -> i64 { 1 }
fn c(x: i64) -> i64 { x + 1 }
fn d() -> i64 { 2 }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var a FnEntry
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok && fn.Name == "a" {
			a = fn
		}
	}
	if a.Name == "" {
		t.Fatal("a not emitted")
	}
	want := []string{"b", "c", "d"}
	if !reflect.DeepEqual(a.Calls, want) {
		t.Errorf("a.Calls = %v, want %v", a.Calls, want)
	}
}

func TestCallGraph_TransitiveClosure(t *testing.T) {
	src := `mod demo

pub fn start() -> i64 { helper() }
fn helper() -> i64 { deep_helper() }
fn deep_helper() -> i64 { 42 }
fn never_called() -> i64 { 0 }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok {
			names[fn.Name] = true
		}
	}
	// start is pub, helper and deep_helper reached via closure, never_called skipped.
	for _, want := range []string{"start", "helper", "deep_helper"} {
		if !names[want] {
			t.Errorf("expected %q in entries, got %v", want, names)
		}
	}
	if names["never_called"] {
		t.Errorf("never_called should not be emitted")
	}
}

func TestCallGraph_BackfilledEntriesAreMinimal(t *testing.T) {
	src := `mod demo

pub fn start() -> i64 { helper() }

// This doc should NOT appear on the backfilled helper entry.
fn helper() -> i64 { 42 }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var h FnEntry
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok && fn.Name == "helper" {
			h = fn
		}
	}
	if h.Name == "" {
		t.Fatal("helper not backfilled")
	}
	if h.Purpose != "" {
		t.Errorf("minimal entry should not carry @purpose, got %q", h.Purpose)
	}
	if len(h.Params) != 0 {
		t.Errorf("minimal entry should not carry @params, got %v", h.Params)
	}
	if h.SourceLine == 0 {
		t.Errorf("minimal entry should still carry source position")
	}
}

func TestCallGraph_MethodReceiverResolution(t *testing.T) {
	src := `mod demo

pub struct Checker { depth: i64 }

impl Checker {
    fn check(self) -> i64 {
        self.helper()
    }

    fn helper(self) -> i64 { self.depth }
}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var check FnEntry
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok && fn.Name == "Checker.check" {
			check = fn
		}
	}
	if check.Name == "" {
		t.Fatal("Checker.check not emitted")
	}
	want := []string{"Checker.helper"}
	if !reflect.DeepEqual(check.Calls, want) {
		t.Errorf("calls = %v, want %v", check.Calls, want)
	}
}

func TestCallGraph_QualifiedPathCall(t *testing.T) {
	src := `mod demo

pub fn f() -> str { std.fs.read("x") }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var f FnEntry
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok && fn.Name == "f" {
			f = fn
		}
	}
	if len(f.Calls) != 1 || f.Calls[0] != "std.fs.read" {
		t.Errorf("calls = %v, want [std.fs.read]", f.Calls)
	}
}

func TestCallGraph_DedupeAndSort(t *testing.T) {
	src := `mod demo

pub fn f() -> i64 {
    a()
    b()
    a()
    a()
}
fn a() -> i64 { 1 }
fn b() -> i64 { 2 }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var f FnEntry
	for _, ent := range aid.Entries {
		if fn, ok := ent.(FnEntry); ok && fn.Name == "f" {
			f = fn
		}
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(f.Calls, want) {
		t.Errorf("calls = %v, want %v (sorted, deduped)", f.Calls, want)
	}
}
