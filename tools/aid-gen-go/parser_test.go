package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testdataDir(sub string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", sub)
}

func TestExtractCallsTransitiveClosure(t *testing.T) {
	dir := testdataDir("callchain")
	aid, err := ExtractPackage(dir, "callchain", "0.0.0", false)
	if err != nil {
		t.Fatal(err)
	}

	// Build a lookup of all FnEntry by name.
	fnByName := map[string]FnEntry{}
	for _, e := range aid.Entries {
		if fn, ok := e.(FnEntry); ok {
			fnByName[fn.Name] = fn
		}
	}

	// Exported must be present (it is exported).
	exported, ok := fnByName["Exported"]
	if !ok {
		t.Fatal("Exported function not found in entries")
	}
	if !containsStr(exported.Calls, "helper") {
		t.Errorf("Exported.Calls = %v, want it to contain %q", exported.Calls, "helper")
	}

	// helper must be emitted as a transitive callee of Exported.
	helper, ok := fnByName["helper"]
	if !ok {
		t.Fatal("helper not found in entries — transitive callee emission failed")
	}
	if helper.SourceFile == "" {
		t.Errorf("helper.SourceFile is empty, want a source file reference")
	}
	if !containsStr(helper.Calls, "deepHelper") {
		t.Errorf("helper.Calls = %v, want it to contain %q", helper.Calls, "deepHelper")
	}

	// deepHelper must also be emitted (transitive closure: Exported -> helper -> deepHelper).
	deep, ok := fnByName["deepHelper"]
	if !ok {
		t.Fatal("deepHelper not found in entries — transitive closure incomplete")
	}
	if deep.SourceFile == "" {
		t.Errorf("deepHelper.SourceFile is empty, want a source file reference")
	}
}

func TestExportedFunctionsAlwaysEmitted(t *testing.T) {
	dir := testdataDir("callchain")
	aid, err := ExtractPackage(dir, "callchain", "0.0.0", false)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, e := range aid.Entries {
		if fn, ok := e.(FnEntry); ok && fn.Name == "Exported" {
			found = true
			if fn.Purpose == "" {
				t.Errorf("Exported function missing purpose — exported functions should have full entries")
			}
			break
		}
	}
	if !found {
		t.Fatal("Exported function not found — exported functions must always be emitted")
	}
}

func containsStr(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

// TestExtractPackage_TestdataPackages runs mechanical extraction on each fixture
// package and checks emitted AID for high-signal symbols.
func TestExtractPackage_TestdataPackages(t *testing.T) {
	cases := []struct {
		name       string
		subdir     string
		moduleName string
		mustEmit   []string // substrings that must appear in Emit output
	}{
		{
			name:       "basic",
			subdir:     "basic",
			moduleName: "basic",
			mustEmit: []string{
				"@module basic",
				"@fn Get",
				"@fn Set",
				"@type Config",
				"@const MaxRetries",
			},
		},
		{
			name:       "generics",
			subdir:     "generics",
			moduleName: "generics",
			mustEmit: []string{
				"@module generics",
				"@type Pair",
				"@fn Min",
				"@fn Map",
			},
		},
		{
			name:       "interfaces",
			subdir:     "interfaces",
			moduleName: "interfaces",
			mustEmit: []string{
				"@module interfaces",
				"@trait Reader",
				"@type Connection",
				"@fn Connection.Read",
			},
		},
		{
			name:       "errors",
			subdir:     "errors",
			moduleName: "errors",
			mustEmit: []string{
				"@module errors",
				"@fn Wrap",
				"@type NotFoundError",
				"@type StatusCode",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := testdataDir(tc.subdir)
			aid, err := ExtractPackage(dir, tc.moduleName, "0.0.0", false)
			if err != nil {
				t.Fatalf("ExtractPackage: %v", err)
			}
			if aid == nil {
				t.Fatal("nil AidFile")
			}
			out := Emit(aid)
			for _, frag := range tc.mustEmit {
				if !strings.Contains(out, frag) {
					t.Errorf("emitted AID missing %q\n---\n%s", frag, out)
				}
			}
		})
	}
}
