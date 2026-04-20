package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePkg writes the given source into a fresh temp dir and returns the dir.
func writePkg(t *testing.T, name, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestExtract_PubFnOnly(t *testing.T) {
	src := `mod demo

// Adds two numbers.
pub fn add(a: i64, b: i64) -> i64 { a + b }

// Private helper.
fn helper(x: i64) -> i64 { x * 2 }
`
	dir := writePkg(t, "demo.aria", src)

	// Restore the global -internal flag after the test.
	orig := *includeInternal
	*includeInternal = false
	t.Cleanup(func() { *includeInternal = orig })

	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(aid.Entries) != 1 {
		t.Fatalf("want 1 entry (pub add only), got %d", len(aid.Entries))
	}
	fn, ok := aid.Entries[0].(FnEntry)
	if !ok {
		t.Fatalf("want FnEntry, got %T", aid.Entries[0])
	}
	if fn.Name != "add" {
		t.Errorf("name = %q, want add", fn.Name)
	}
	if !strings.Contains(fn.Sigs[0], "(a: i64, b: i64) -> i64") {
		t.Errorf("sig = %q", fn.Sigs[0])
	}
	if fn.Returns != "i64" {
		t.Errorf("returns = %q, want i64", fn.Returns)
	}
	if fn.Purpose != "Adds two numbers." {
		t.Errorf("purpose = %q", fn.Purpose)
	}
	if fn.SourceFile != "demo.aria" || fn.SourceLine == 0 {
		t.Errorf("source = %q:%d", fn.SourceFile, fn.SourceLine)
	}
}

func TestExtract_InternalFlagIncludesPrivate(t *testing.T) {
	src := `mod demo
pub fn a() {}
fn b() {}
`
	dir := writePkg(t, "demo.aria", src)
	orig := *includeInternal
	*includeInternal = true
	t.Cleanup(func() { *includeInternal = orig })

	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{Internal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(aid.Entries) != 2 {
		t.Fatalf("want 2 entries with -internal, got %d", len(aid.Entries))
	}
}

func TestExtract_FnWithEffectsAndErrors(t *testing.T) {
	src := `mod demo

pub fn fetch(url: str) -> str ! IoError with [Io, Net] {
    url
}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(aid.Entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(aid.Entries))
	}
	fn := aid.Entries[0].(FnEntry)
	if got := fn.Effects; len(got) != 2 || got[0] != "Io" || got[1] != "Net" {
		t.Errorf("effects = %v, want [Io Net]", got)
	}
	if got := fn.Errors; len(got) != 1 || got[0] != "IoError" {
		t.Errorf("errors = %v, want [IoError]", got)
	}
	if !strings.Contains(fn.Sigs[0], "! IoError") || !strings.Contains(fn.Sigs[0], "with [Io, Net]") {
		t.Errorf("sig missing error/effect clause: %q", fn.Sigs[0])
	}
}

func TestExtract_StructAndSumType(t *testing.T) {
	src := `mod demo

pub struct Point { x: i64, y: i64 }

pub type Shape =
    | Circle(f64)
    | Rect { w: i64, h: i64 }
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(aid.Entries) != 2 {
		t.Fatalf("want 2 type entries, got %d: %+v", len(aid.Entries), aid.Entries)
	}

	// Sorted by name: Point, Shape
	p := aid.Entries[0].(TypeEntry)
	if p.Name != "Point" || p.Kind != "struct" || len(p.Fields) != 2 {
		t.Errorf("Point = %+v", p)
	}

	s := aid.Entries[1].(TypeEntry)
	if s.Name != "Shape" || s.Kind != "union" || len(s.Variants) != 2 {
		t.Errorf("Shape = %+v", s)
	}
	if s.Variants[1].Payload == "" {
		t.Errorf("Shape.Rect expected struct payload, got empty")
	}
}

func TestExtract_DepsFromImports(t *testing.T) {
	src := `mod demo

use std.io
use std.fs

pub fn f() {}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	deps := aid.Header.Deps
	if len(deps) != 2 || deps[0] != "std.fs" || deps[1] != "std.io" {
		t.Errorf("deps = %v, want [std.fs std.io]", deps)
	}
}
