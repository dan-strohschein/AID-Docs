package emitter

import (
	"strings"
	"testing"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func TestEmitMinimalHeader(t *testing.T) {
	f := &parser.AidFile{
		Header: parser.Header{
			Module:     "test/mod",
			Lang:       "go",
			Version:    "1.0.0",
			AidVersion: "0.1",
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@module test/mod") {
		t.Errorf("missing @module in:\n%s", out)
	}
	if !strings.Contains(out, "@lang go") {
		t.Errorf("missing @lang")
	}
}

func TestEmitFnEntry(t *testing.T) {
	f := &parser.AidFile{
		Header: parser.Header{Module: "test", Lang: "go", Version: "1.0.0", AidVersion: "0.1"},
		Entries: []parser.Entry{
			{
				Kind: "fn",
				Name: "Get",
				Fields: map[string]parser.Field{
					"purpose": {Name: "purpose", InlineValue: "Get a value"},
					"sig":     {Name: "sig", InlineValue: "(key: str) -> str"},
					"params":  {Name: "params", Lines: []string{"key: The lookup key"}},
				},
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@fn Get") {
		t.Errorf("missing @fn Get in:\n%s", out)
	}
	if !strings.Contains(out, "@purpose Get a value") {
		t.Errorf("missing @purpose")
	}
	if !strings.Contains(out, "@sig (key: str) -> str") {
		t.Errorf("missing @sig")
	}
	if !strings.Contains(out, "  key: The lookup key") {
		t.Errorf("missing params continuation")
	}
}

func TestEmitAnnotation(t *testing.T) {
	f := &parser.AidFile{
		Header: parser.Header{Module: "test", Lang: "go", Version: "1.0.0", AidVersion: "0.1"},
		Annotations: []parser.Annotation{
			{
				Kind: "invariants",
				Fields: map[string]parser.Field{
					"invariants": {Name: "invariants", Lines: []string{
						"- BRIN is lossy [src: nodes.go:245]",
						"- Plan is immutable [src: planner.go:111]",
					}},
				},
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@invariants\n") {
		t.Errorf("missing @invariants block in:\n%s", out)
	}
	if !strings.Contains(out, "  - BRIN is lossy") {
		t.Errorf("missing invariant line")
	}
}

func TestEmitDecision(t *testing.T) {
	f := &parser.AidFile{
		Header: parser.Header{Module: "test", Lang: "go", Version: "1.0.0", AidVersion: "0.1"},
		Annotations: []parser.Annotation{
			{
				Kind: "decision",
				Name: "index_order",
				Fields: map[string]parser.Field{
					"purpose":   {Name: "purpose", InlineValue: "Why BTree first"},
					"chosen":    {Name: "chosen", InlineValue: "BTree first"},
					"rationale": {Name: "rationale", InlineValue: "BTree is exact"},
				},
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@decision index_order") {
		t.Errorf("missing @decision in:\n%s", out)
	}
	if !strings.Contains(out, "@chosen BTree first") {
		t.Errorf("missing @chosen")
	}
}

func TestEmitManifest(t *testing.T) {
	f := &parser.AidFile{
		IsManifest: true,
		Header:     parser.Header{AidVersion: "0.1", Extra: map[string]string{"project": "Test"}},
		Entries: []parser.Entry{
			{
				Kind: "package",
				Name: "query/planner",
				Fields: map[string]parser.Field{
					"aid_file": {Name: "aid_file", InlineValue: "planner.aid"},
					"purpose":  {Name: "purpose", InlineValue: "Query planning"},
					"layer":    {Name: "layer", InlineValue: "l2"},
				},
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@manifest") {
		t.Errorf("missing @manifest in:\n%s", out)
	}
	if !strings.Contains(out, "@package query/planner") {
		t.Errorf("missing @package")
	}
	if !strings.Contains(out, "@aid_file planner.aid") {
		t.Errorf("missing @aid_file")
	}
}

func TestRoundTrip(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig (key: str) -> str
@params
  key: The lookup key

---

@type Config
@kind struct
@purpose Configuration
@fields
  host: str
  port: int
`
	f, _, err := parser.ParseString(input)
	if err != nil {
		t.Fatal(err)
	}

	output := Emit(f)

	f2, _, err := parser.ParseString(output)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}

	// Verify key structures match
	if f2.Header.Module != f.Header.Module {
		t.Errorf("module mismatch: %q vs %q", f2.Header.Module, f.Header.Module)
	}
	if len(f2.Entries) != len(f.Entries) {
		t.Errorf("entry count: %d vs %d", len(f2.Entries), len(f.Entries))
	}
	for i, e := range f.Entries {
		if i < len(f2.Entries) {
			if f2.Entries[i].Kind != e.Kind || f2.Entries[i].Name != e.Name {
				t.Errorf("entry %d: %s/%s vs %s/%s", i, f2.Entries[i].Kind, f2.Entries[i].Name, e.Kind, e.Name)
			}
		}
	}
}
