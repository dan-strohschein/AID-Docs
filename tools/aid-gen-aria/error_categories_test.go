package main

import (
	"strings"
	"testing"
)

func TestExtract_ErrorCategoriesFromImpls(t *testing.T) {
	src := `mod demo

pub type IoError =
    | Timeout
    | NotFound

impl Retryable for IoError {}
impl Transient for IoError {}
impl Permanent for IoError {}

// Not a category trait — should land in @implements only.
impl Display for IoError {}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(aid.Entries) == 0 {
		t.Fatal("expected at least one entry")
	}

	var te TypeEntry
	for _, ent := range aid.Entries {
		if t, ok := ent.(TypeEntry); ok && t.Name == "IoError" {
			te = t
			break
		}
	}
	if te.Name == "" {
		t.Fatal("IoError TypeEntry not found")
	}

	want := map[string]bool{"Retryable": true, "Transient": true, "Permanent": true}
	if len(te.ErrorCategories) != len(want) {
		t.Fatalf("error_categories = %v, want 3 entries", te.ErrorCategories)
	}
	for _, c := range te.ErrorCategories {
		if !want[c] {
			t.Errorf("unexpected category %q", c)
		}
	}

	// Display must appear in @implements but not in @error_categories.
	foundDisplay := false
	for _, i := range te.Implements {
		if i == "Display" {
			foundDisplay = true
		}
		if isKnownErrorCategory(i) && !want[i] {
			t.Errorf("implements leaked unknown category %q", i)
		}
	}
	if !foundDisplay {
		t.Errorf("Display missing from @implements: %v", te.Implements)
	}

	// Emitter renders @error_categories.
	out := Emit(aid)
	if !strings.Contains(out, "@error_categories [") {
		t.Errorf("emitter missing @error_categories line:\n%s", out)
	}
}

func TestExtract_DuplicateCategoryImplsDedupe(t *testing.T) {
	src := `mod demo
pub type E = | A | B
impl Retryable for E {}
impl Retryable for E {}
`
	dir := writePkg(t, "demo.aria", src)
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "demo", "0.0.0", ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var te TypeEntry
	for _, ent := range aid.Entries {
		if t, ok := ent.(TypeEntry); ok && t.Name == "E" {
			te = t
		}
	}
	if len(te.ErrorCategories) != 1 {
		t.Errorf("want 1 category after dedupe, got %v", te.ErrorCategories)
	}
}
