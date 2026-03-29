// Tests Emit and orderEntries: in-memory AidFile serialized to .aid text.
package main

import (
	"strings"
	"testing"
)

func TestEmit_MinimalHeaderAndFn(t *testing.T) {
	f := &AidFile{
		Header: ModuleHeader{
			Module:     "demo/pkg",
			Lang:       "go",
			Version:    "1.2.3",
			Stability:  "stable",
			Purpose:    "Demonstration module.",
			Deps:       []string{"net/http", "context"},
			Source:     "https://example.com",
			AidVersion: "0.1",
		},
		Entries: []Entry{
			FnEntry{
				Name:    "Process",
				Purpose: "Runs the pipeline.",
				Sigs:    []string{"Process(ctx context.Context, name str) -> error"},
				Params: []Param{
					{Name: "ctx", Type: "context.Context", Desc: "Cancellation context."},
					{Name: "name", Type: "str"},
				},
				Returns:    "error",
				Errors:     []string{"ErrNotFound — when name is unknown"},
				Calls:      []string{"validate", "commit"},
				SourceFile: "proc.go",
				SourceLine: 42,
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@module demo/pkg") {
		t.Error("missing @module")
	}
	if !strings.Contains(out, "@lang go") {
		t.Error("missing @lang")
	}
	if !strings.Contains(out, "@stability stable") {
		t.Error("missing @stability")
	}
	if !strings.Contains(out, "@purpose Demonstration module.") {
		t.Error("missing header @purpose")
	}
	if !strings.Contains(out, "@deps [net/http, context]") {
		t.Error("missing @deps")
	}
	if !strings.Contains(out, "@source https://example.com") {
		t.Error("missing @source")
	}
	if !strings.Contains(out, "@fn Process") {
		t.Error("missing @fn Process")
	}
	if !strings.Contains(out, "@purpose Runs the pipeline.") {
		t.Error("missing fn @purpose")
	}
	if !strings.Contains(out, "@params") || !strings.Contains(out, "ctx:") {
		t.Error("missing @params / ctx")
	}
	if !strings.Contains(out, "@errors") || !strings.Contains(out, "ErrNotFound") {
		t.Error("missing @errors")
	}
	if !strings.Contains(out, "@calls [validate, commit]") {
		t.Error("missing @calls")
	}
	if !strings.Contains(out, "@source_file proc.go") || !strings.Contains(out, "@source_line 42") {
		t.Error("missing source location")
	}
	if !strings.Contains(out, "// [generated] Layer 1 mechanical extraction") {
		t.Error("missing generated banner")
	}
}

func TestEmit_TypeAndConst(t *testing.T) {
	f := &AidFile{
		Header: ModuleHeader{
			Module:     "x",
			Lang:       "go",
			Version:    "0.0.0",
			AidVersion: "0.1",
		},
		Entries: []Entry{
			ConstEntry{Name: "Max", Purpose: "Upper bound", Type: "int", Value: "10"},
			TypeEntry{
				Name:    "Widget",
				Kind:    "struct",
				Purpose: "A widget.",
				Fields: []Field{
					{Name: "ID", Type: "str", Desc: "Identifier"},
				},
				SourceFile: "widget.go",
				SourceLine: 1,
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@const Max") {
		t.Error("missing @const Max")
	}
	if !strings.Contains(out, "@type Widget") {
		t.Error("missing @type Widget")
	}
	if !strings.Contains(out, "@kind struct") {
		t.Error("missing @kind struct")
	}
	if !strings.Contains(out, "@fields") || !strings.Contains(out, "ID: str") {
		t.Error("missing fields block")
	}
	// Consts are ordered before types by orderEntries.
	constIdx := strings.Index(out, "@const Max")
	typeIdx := strings.Index(out, "@type Widget")
	if constIdx < 0 || typeIdx < 0 || constIdx > typeIdx {
		t.Errorf("want @const before @type: const@%d type@%d", constIdx, typeIdx)
	}
}

func TestEmit_OrderEntries_TypeBeforeMethod(t *testing.T) {
	f := &AidFile{
		Header: ModuleHeader{
			Module:     "srv",
			Lang:       "go",
			Version:    "1.0.0",
			AidVersion: "0.1",
		},
		Entries: []Entry{
			// Intentionally list method before type — emitter should group under type.
			FnEntry{Name: "Server.Listen", Sigs: []string{"Listen() -> error"}},
			TypeEntry{Name: "Server", Kind: "struct", Purpose: "HTTP server"},
			FnEntry{Name: "Run", Sigs: []string{"Run() -> None"}},
		},
	}
	out := Emit(f)
	typeIdx := strings.Index(out, "@type Server")
	methodIdx := strings.Index(out, "@fn Server.Listen")
	runIdx := strings.Index(out, "@fn Run")
	if typeIdx < 0 || methodIdx < 0 || runIdx < 0 {
		t.Fatalf("missing blocks: type@%d method@%d run@%d", typeIdx, methodIdx, runIdx)
	}
	if methodIdx < typeIdx {
		t.Error("expected @type Server before @fn Server.Listen")
	}
	if runIdx < methodIdx {
		t.Error("expected method grouped before standalone @fn Run")
	}
}

func TestEmit_Trait(t *testing.T) {
	f := &AidFile{
		Header: ModuleHeader{
			Module:     "api",
			Lang:       "go",
			Version:    "1.0.0",
			AidVersion: "0.1",
		},
		Entries: []Entry{
			TraitEntry{
				Name:     "Closer",
				Purpose:  "Releases resources.",
				Requires: []string{"Close() -> error"},
			},
		},
	}
	out := Emit(f)
	if !strings.Contains(out, "@trait Closer") || !strings.Contains(out, "@requires") {
		t.Errorf("trait output: %s", out)
	}
}
