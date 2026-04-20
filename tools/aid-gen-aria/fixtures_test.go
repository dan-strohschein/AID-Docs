// Structural parity tests over the testdata/ fixtures. These mirror
// aid-gen-go's testdata-based checks: rather than byte-exact golden files
// (which would break on every cosmetic emitter tweak), we assert on
// specific entry counts, names, call-graph contents, and field presence.
//
// The fixtures are also the input for M8 — so any change here should be
// considered against its effect on the stdlib/compiler AID outputs.
package main

import (
	"strings"
	"testing"
)

func extractFixture(t *testing.T, dir string, opts ExtractOptions) *AidFile {
	t.Helper()
	e := NewAriaExtractor()
	aid, err := e.Extract(dir, "", "0.0.0", opts)
	if err != nil {
		t.Fatalf("extract %s: %v", dir, err)
	}
	return aid
}

func findEntry(aid *AidFile, name string) Entry {
	for _, e := range aid.Entries {
		if entryName(e) == name {
			return e
		}
	}
	return nil
}

func hasString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// --- basic ---

func TestFixture_Basic(t *testing.T) {
	aid := extractFixture(t, "testdata/basic", ExtractOptions{})

	// Public surface: 2 consts + Config (with 2 methods) + 3 free fns = 7 entries.
	// internal_helper is private and not referenced, so no closure backfill.
	for _, want := range []string{"MaxRetries", "DefaultTimeout", "Config", "Config.validate", "Config.set_host", "new_config", "get", "set"} {
		if findEntry(aid, want) == nil {
			t.Errorf("basic: missing entry %q (have %v)", want, entryNames(aid))
		}
	}
	if findEntry(aid, "internal_helper") != nil {
		t.Errorf("basic: internal_helper should not be emitted (not referenced)")
	}

	cfg := findEntry(aid, "Config").(TypeEntry)
	if cfg.Kind != "struct" || len(cfg.Fields) != 3 {
		t.Errorf("basic: Config expected 3 fields, got %+v", cfg.Fields)
	}
}

// --- callchain ---

func TestFixture_Callchain_TransitiveClosure(t *testing.T) {
	aid := extractFixture(t, "testdata/callchain", ExtractOptions{})
	names := entryNames(aid)

	// exported → helper → deep_helper reached via closure; never_called dropped.
	for _, want := range []string{"exported", "helper", "deep_helper"} {
		if findEntry(aid, want) == nil {
			t.Errorf("callchain: missing %q (have %v)", want, names)
		}
	}
	if findEntry(aid, "never_called") != nil {
		t.Errorf("callchain: never_called leaked into output")
	}

	// Exported carries full details; helper / deep_helper are minimal.
	exp := findEntry(aid, "exported").(FnEntry)
	if !hasString(exp.Calls, "helper") {
		t.Errorf("exported.Calls = %v, want helper", exp.Calls)
	}
	h := findEntry(aid, "helper").(FnEntry)
	if h.Purpose != "" {
		t.Errorf("helper (backfilled) should have empty @purpose, got %q", h.Purpose)
	}
	if !hasString(h.Calls, "deep_helper") {
		t.Errorf("helper.Calls = %v, want deep_helper", h.Calls)
	}
}

// --- errors ---

func TestFixture_Errors_CategoriesFromImpls(t *testing.T) {
	aid := extractFixture(t, "testdata/errors", ExtractOptions{})

	nf, ok := findEntry(aid, "NotFoundError").(TypeEntry)
	if !ok {
		t.Fatalf("errors: NotFoundError not emitted; have %v", entryNames(aid))
	}
	if !hasString(nf.ErrorCategories, "Permanent") || !hasString(nf.ErrorCategories, "UserFault") {
		t.Errorf("NotFoundError.ErrorCategories = %v, want [Permanent, UserFault]", nf.ErrorCategories)
	}

	ve := findEntry(aid, "ValidationError").(TypeEntry)
	if !hasString(ve.ErrorCategories, "UserFault") {
		t.Errorf("ValidationError.ErrorCategories = %v, want [UserFault]", ve.ErrorCategories)
	}

	// StatusCode enum has 4 variants.
	sc := findEntry(aid, "StatusCode").(TypeEntry)
	if sc.Kind != "enum" || len(sc.Variants) != 4 {
		t.Errorf("StatusCode expected enum with 4 variants, got kind=%s variants=%d", sc.Kind, len(sc.Variants))
	}
}

// --- generics ---

func TestFixture_Generics(t *testing.T) {
	aid := extractFixture(t, "testdata/generics", ExtractOptions{})

	pair := findEntry(aid, "Pair").(TypeEntry)
	if pair.GenericParams != "[A, B]" {
		t.Errorf("Pair.GenericParams = %q, want [A, B]", pair.GenericParams)
	}

	min := findEntry(aid, "min").(FnEntry)
	if !strings.Contains(min.Sigs[0], "[T: Ordered]") {
		t.Errorf("min.Sig = %q missing generic bound", min.Sigs[0])
	}

	mapFn := findEntry(aid, "map_items").(FnEntry)
	if !strings.Contains(mapFn.Sigs[0], "fn(T) -> U") {
		t.Errorf("map_items.Sig = %q missing fn-type param", mapFn.Sigs[0])
	}
}

// --- interfaces ---

func TestFixture_Interfaces_TraitImpls(t *testing.T) {
	aid := extractFixture(t, "testdata/interfaces", ExtractOptions{})

	conn := findEntry(aid, "Connection").(TypeEntry)
	for _, want := range []string{"Reader", "Writer", "Closer"} {
		if !hasString(conn.Implements, want) {
			t.Errorf("Connection.Implements = %v, missing %q", conn.Implements, want)
		}
	}

	// Trait with supertraits.
	rw := findEntry(aid, "ReadWriter").(TraitEntry)
	for _, want := range []string{"Reader", "Writer"} {
		if !hasString(rw.Extends, want) {
			t.Errorf("ReadWriter.Extends = %v, missing %q", rw.Extends, want)
		}
	}

	// Methods attached to Connection via impls appear as Connection.* FnEntries.
	for _, m := range []string{"Connection.read", "Connection.write", "Connection.close"} {
		if findEntry(aid, m) == nil {
			t.Errorf("interfaces: missing method entry %q", m)
		}
	}
}

// --- testpkg ---

func TestFixture_Testpkg_MainAID(t *testing.T) {
	aid := extractFixture(t, "testdata/testpkg", ExtractOptions{})

	// Main AID: traits + production types + helper, but NO test block entry.
	for _, want := range []string{"BundleService", "Index", "MockBundleService", "StubIndex", "setup_test_index"} {
		if findEntry(aid, want) == nil {
			t.Errorf("testpkg main: missing %q (have %v)", want, entryNames(aid))
		}
	}
	for _, bad := range []string{"test_get_bundle_by_name_returns_stubbed_value"} {
		if findEntry(aid, bad) != nil {
			t.Errorf("testpkg main: test block %q leaked into main AID", bad)
		}
	}
}

func TestFixture_Testpkg_TestAID(t *testing.T) {
	e := NewAriaExtractor()
	aid, err := e.ExtractTests("testdata/testpkg", "testpkg_test", "0.0.0")
	if err != nil {
		t.Fatal(err)
	}
	// Test AID: Mock/Stub types, helper, and the synthetic test block.
	for _, want := range []string{"MockBundleService", "StubIndex", "setup_test_index", "test_get_bundle_by_name_returns_stubbed_value"} {
		if findEntry(aid, want) == nil {
			t.Errorf("testpkg test: missing %q (have %v)", want, entryNames(aid))
		}
	}
	// BundleService (a regular trait, not Mock-prefixed) filtered out.
	if findEntry(aid, "BundleService") != nil {
		t.Errorf("testpkg test: production trait BundleService should not appear")
	}

	tb := findEntry(aid, "test_get_bundle_by_name_returns_stubbed_value").(FnEntry)
	if !hasString(tb.Calls, "setup_test_index") {
		t.Errorf("test-block @calls missing setup_test_index; got %v", tb.Calls)
	}
}

// --- CLI flag matrix (minimal end-to-end coverage) ---

func TestFixture_AllFlag_IncludesPrivate(t *testing.T) {
	aid := extractFixture(t, "testdata/basic", ExtractOptions{All: true})
	if findEntry(aid, "internal_helper") == nil {
		t.Errorf("-all should include private fns; have %v", entryNames(aid))
	}
}

func TestFixture_EmitterSmokes(t *testing.T) {
	// Verify every fixture round-trips through the emitter without panics
	// and produces a header with the expected module name.
	cases := []struct{ dir, want string }{
		{"testdata/basic", "@module basic"},
		{"testdata/callchain", "@module callchain"},
		{"testdata/errors", "@module errors"},
		{"testdata/generics", "@module generics"},
		{"testdata/interfaces", "@module interfaces"},
		{"testdata/testpkg", "@module testpkg"},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			aid := extractFixture(t, c.dir, ExtractOptions{})
			out := Emit(aid)
			if !strings.Contains(out, c.want) {
				t.Errorf("%s missing %q in output", c.dir, c.want)
			}
			if !strings.Contains(out, "@aid_version 0.2") {
				t.Errorf("%s missing @aid_version 0.2", c.dir)
			}
		})
	}
}

// entryNames is a small helper for readable failure output.
func entryNames(aid *AidFile) []string {
	out := make([]string, 0, len(aid.Entries))
	for _, e := range aid.Entries {
		out = append(out, entryName(e))
	}
	return out
}
