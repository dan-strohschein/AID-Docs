package parser

import (
	"strings"
	"testing"
)

func TestParseMinimalHeader(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1
`
	f, warns, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) > 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if f.Header.Module != "test/mod" {
		t.Errorf("module = %q, want %q", f.Header.Module, "test/mod")
	}
	if f.Header.Lang != "go" {
		t.Errorf("lang = %q, want %q", f.Header.Lang, "go")
	}
	if f.Header.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", f.Header.Version, "1.0.0")
	}
}

func TestParseHeaderWithL2Fields(t *testing.T) {
	input := `@module query/planner
@lang go
@version 2.0.0
@code_version git:979fe97
@aid_status reviewed
@aid_generated_by layer2-generator
@aid_reviewed_by layer2-reviewer
@depends [syndrQL, domain/index, domain/models]
@aid_version 0.1
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if f.Header.CodeVersion != "git:979fe97" {
		t.Errorf("code_version = %q, want %q", f.Header.CodeVersion, "git:979fe97")
	}
	if f.Header.AidStatus != "reviewed" {
		t.Errorf("aid_status = %q, want %q", f.Header.AidStatus, "reviewed")
	}
	if f.Header.AidGeneratedBy != "layer2-generator" {
		t.Errorf("aid_generated_by = %q", f.Header.AidGeneratedBy)
	}
	if len(f.Header.Depends) != 3 {
		t.Errorf("depends len = %d, want 3", len(f.Header.Depends))
	}
}

func TestParseFnEntry(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Retrieve a value by key
@sig (key: str) -> str? ! error
@params
  key: The lookup key. Required.
@returns Value if found, None if absent
@errors
  NotFoundError — key does not exist
@related Set, Delete
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(f.Entries))
	}
	e := f.Entries[0]
	if e.Kind != "fn" {
		t.Errorf("kind = %q, want %q", e.Kind, "fn")
	}
	if e.Name != "Get" {
		t.Errorf("name = %q, want %q", e.Name, "Get")
	}
	if purpose, ok := e.Fields["purpose"]; !ok || purpose.InlineValue != "Retrieve a value by key" {
		t.Errorf("purpose = %v", e.Fields["purpose"])
	}
	if sig, ok := e.Fields["sig"]; !ok || sig.InlineValue != "(key: str) -> str? ! error" {
		t.Errorf("sig = %v", e.Fields["sig"])
	}
	// Check multi-line params
	if params, ok := e.Fields["params"]; !ok || len(params.Lines) != 1 {
		t.Errorf("params lines = %d, want 1", len(e.Fields["params"].Lines))
	}
	// Check related
	if related, ok := e.Fields["related"]; !ok || related.InlineValue != "Set, Delete" {
		t.Errorf("related = %v", e.Fields["related"])
	}
}

func TestParseTypeEntry(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@type Config
@kind struct
@purpose Configuration options
@fields
  host: str — Server hostname
  port: int — Server port number
@methods Validate, SetHost
@implements [Debug, Display]
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(f.Entries))
	}
	e := f.Entries[0]
	if e.Kind != "type" {
		t.Errorf("kind = %q", e.Kind)
	}
	if e.Name != "Config" {
		t.Errorf("name = %q", e.Name)
	}
	if kind := e.Fields["kind"]; kind.InlineValue != "struct" {
		t.Errorf("kind value = %q", kind.InlineValue)
	}
	if fields := e.Fields["fields"]; len(fields.Lines) != 2 {
		t.Errorf("fields lines = %d, want 2", len(fields.Lines))
	}
}

func TestParseWorkflow(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@workflow basic_usage
@purpose Create, configure, and use the client
@steps
  1. Create: NewClient(host) — connect
  2. Configure: client.SetTimeout(30s)
  3. Execute: client.Get(key)
@antipatterns
  - Don't forget to close the client
  - Don't reuse after close
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Workflows) != 1 {
		t.Fatalf("workflows = %d, want 1", len(f.Workflows))
	}
	w := f.Workflows[0]
	if w.Name != "basic_usage" {
		t.Errorf("name = %q", w.Name)
	}
	if steps := w.Fields["steps"]; len(steps.Lines) != 3 {
		t.Errorf("steps lines = %d, want 3", len(steps.Lines))
	}
	if ap := w.Fields["antipatterns"]; len(ap.Lines) != 2 {
		t.Errorf("antipatterns lines = %d, want 2", len(ap.Lines))
	}
}

func TestParseMultipleEntries(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig (key: str) -> str

---

@fn Set
@purpose Set a value
@sig (key: str, value: str) -> None

---

@type Config
@kind struct
@purpose Config options
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(f.Entries))
	}
	if f.Entries[0].Kind != "fn" || f.Entries[0].Name != "Get" {
		t.Errorf("entry 0: %s %s", f.Entries[0].Kind, f.Entries[0].Name)
	}
	if f.Entries[1].Kind != "fn" || f.Entries[1].Name != "Set" {
		t.Errorf("entry 1: %s %s", f.Entries[1].Kind, f.Entries[1].Name)
	}
	if f.Entries[2].Kind != "type" || f.Entries[2].Name != "Config" {
		t.Errorf("entry 2: %s %s", f.Entries[2].Kind, f.Entries[2].Name)
	}
}

func TestParseSourceRefs(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig (key: str) -> str
@invariants
  - Must hold lock before calling [src: pkg/store.go:45]
  - Returns nil for missing keys [src: pkg/store.go:50-60]
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	inv := f.Entries[0].Fields["invariants"]
	if len(inv.SourceRefs) != 2 {
		t.Fatalf("source refs = %d, want 2", len(inv.SourceRefs))
	}
	ref0 := inv.SourceRefs[0]
	if ref0.File != "pkg/store.go" || ref0.StartLine != 45 || ref0.EndLine != 45 {
		t.Errorf("ref0 = %v", ref0)
	}
	ref1 := inv.SourceRefs[1]
	if ref1.File != "pkg/store.go" || ref1.StartLine != 50 || ref1.EndLine != 60 {
		t.Errorf("ref1 = %v", ref1)
	}
}

func TestParseComments(t *testing.T) {
	input := `// [generated] Layer 1 mechanical extraction
// Another comment

@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Comments) != 2 {
		t.Errorf("comments = %d, want 2", len(f.Comments))
	}
	if !strings.Contains(f.Comments[0], "[generated]") {
		t.Errorf("comment 0 = %q", f.Comments[0])
	}
}

func TestParseDeps(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@deps [ssl, dns, url]
@aid_version 0.1
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Header.Deps) != 3 {
		t.Fatalf("deps = %d, want 3", len(f.Header.Deps))
	}
	if f.Header.Deps[0] != "ssl" || f.Header.Deps[1] != "dns" || f.Header.Deps[2] != "url" {
		t.Errorf("deps = %v", f.Header.Deps)
	}
}

func TestParseTraitEntry(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@trait Reader
@purpose Can read data from a source
@requires
  fn Read(p: bytes) -> int ! error
@extends Closer
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("entries = %d", len(f.Entries))
	}
	e := f.Entries[0]
	if e.Kind != "trait" {
		t.Errorf("kind = %q", e.Kind)
	}
	if e.Name != "Reader" {
		t.Errorf("name = %q", e.Name)
	}
}

func TestParseConstEntry(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@const MaxRetries
@purpose Maximum retry attempts
@type int
@value 3
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Fatalf("entries = %d", len(f.Entries))
	}
	e := f.Entries[0]
	if e.Kind != "const" {
		t.Errorf("kind = %q", e.Kind)
	}
	if e.Fields["value"].InlineValue != "3" {
		t.Errorf("value = %q", e.Fields["value"].InlineValue)
	}
}

func TestParseUnknownFieldsForwardCompat(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@future_field some_value
@aid_version 0.1
`
	f, warns, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	// Unknown fields should be stored, not produce warnings
	if len(warns) > 0 {
		t.Errorf("expected no warnings for unknown fields, got %v", warns)
	}
	if f.Header.Extra["future_field"] != "some_value" {
		t.Errorf("extra = %v", f.Header.Extra)
	}
}

func TestClassifyLine(t *testing.T) {
	tests := []struct {
		input     string
		wantType  LineType
		wantField string
		wantValue string
	}{
		{"@fn Get", LineField, "fn", "Get"},
		{"@purpose Do something", LineField, "purpose", "Do something"},
		{"@params", LineField, "params", ""},
		{"  key: str — description", LineContinuation, "", "key: str — description"},
		{"    .timeout: Duration", LineContinuation, "", "  .timeout: Duration"},
		{"---", LineSeparator, "", ""},
		{"// comment", LineComment, "", "// comment"},
		{"", LineBlank, "", ""},
		{"   ", LineBlank, "", ""},
	}
	for _, tt := range tests {
		typ, field, val := ClassifyLine(tt.input)
		if typ != tt.wantType {
			t.Errorf("ClassifyLine(%q): type = %d, want %d", tt.input, typ, tt.wantType)
		}
		if field != tt.wantField {
			t.Errorf("ClassifyLine(%q): field = %q, want %q", tt.input, field, tt.wantField)
		}
		if val != tt.wantValue {
			t.Errorf("ClassifyLine(%q): value = %q, want %q", tt.input, val, tt.wantValue)
		}
	}
}

func TestExtractSourceRefs(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"no refs here", 0},
		{"something [src: file.go:42]", 1},
		{"both [src: a.go:1-10] and [src: b.go:20]", 2},
		{"multi [src: a.go:1, b.go:2]", 2},
	}
	for _, tt := range tests {
		refs := extractSourceRefs(tt.input)
		if len(refs) != tt.want {
			t.Errorf("extractSourceRefs(%q) = %d refs, want %d", tt.input, len(refs), tt.want)
		}
	}
}

func TestParseInvariantsAnnotation(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig (key: str) -> str

---

@invariants
  - BRIN is lossy [src: nodes.go:245]
  - Plan is immutable after creation [src: planner.go:111]
`
	f, warns, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warns {
		t.Logf("warning: %s", w)
	}
	if len(f.Annotations) != 1 {
		t.Fatalf("annotations = %d, want 1", len(f.Annotations))
	}
	a := f.Annotations[0]
	if a.Kind != "invariants" {
		t.Errorf("kind = %q, want invariants", a.Kind)
	}
	// Check source refs were extracted from continuation lines
	inv := a.Fields["invariants"]
	if len(inv.SourceRefs) != 2 {
		t.Errorf("source refs = %d, want 2", len(inv.SourceRefs))
	}
}

func TestParseDecisionAnnotation(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@decision index_selection_order
@purpose Why BTree is checked before BRIN
@chosen BTree first, BRIN fallback
@rejected Cost-based selection
@rationale BTree gives exact results
  [src: query_router.go:973-1022]
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Annotations) != 1 {
		t.Fatalf("annotations = %d, want 1", len(f.Annotations))
	}
	d := f.Annotations[0]
	if d.Kind != "decision" {
		t.Errorf("kind = %q", d.Kind)
	}
	if d.Name != "index_selection_order" {
		t.Errorf("name = %q", d.Name)
	}
	if d.Fields["chosen"].InlineValue != "BTree first, BRIN fallback" {
		t.Errorf("chosen = %q", d.Fields["chosen"].InlineValue)
	}
	if d.Fields["rejected"].InlineValue != "Cost-based selection" {
		t.Errorf("rejected = %q", d.Fields["rejected"].InlineValue)
	}
}

func TestParseNoteAnnotation(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@note adapter-deprecation
@purpose ExpressionAdapter is a migration bridge
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Annotations) != 1 {
		t.Fatalf("annotations = %d, want 1", len(f.Annotations))
	}
	n := f.Annotations[0]
	if n.Kind != "note" || n.Name != "adapter-deprecation" {
		t.Errorf("note = %q %q", n.Kind, n.Name)
	}
}

func TestParseManifest(t *testing.T) {
	input := `@manifest
@project SyndrDB
@aid_version 0.1

---

@package query/planner
@aid_file planner.aid
@aid_status reviewed
@depends [syndrQL, domain/index]
@purpose Query planning and optimization
@layer l2

---

@package domain/index/brinindex
@aid_file brinindex.aid
@purpose BRIN index implementation
@layer l1
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsManifest {
		t.Error("expected IsManifest = true")
	}
	if len(f.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(f.Entries))
	}
	e0 := f.Entries[0]
	if e0.Kind != "package" || e0.Name != "query/planner" {
		t.Errorf("entry 0 = %q %q", e0.Kind, e0.Name)
	}
	if e0.Fields["aid_file"].InlineValue != "planner.aid" {
		t.Errorf("aid_file = %q", e0.Fields["aid_file"].InlineValue)
	}
	if e0.Fields["layer"].InlineValue != "l2" {
		t.Errorf("layer = %q", e0.Fields["layer"].InlineValue)
	}
	e1 := f.Entries[1]
	if e1.Kind != "package" || e1.Name != "domain/index/brinindex" {
		t.Errorf("entry 1 = %q %q", e1.Kind, e1.Name)
	}
}

func TestParseMultipleAnnotations(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn Get
@purpose Get a value
@sig () -> str

---

@invariants
  - Always lock before read

---

@antipatterns
  - Don't call Get without init

---

@workflow basic_usage
@purpose Use the module
@steps
  1. Init
  2. Get
`
	f, _, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(f.Entries))
	}
	if len(f.Annotations) != 2 {
		t.Errorf("annotations = %d, want 2", len(f.Annotations))
	}
	if len(f.Workflows) != 1 {
		t.Errorf("workflows = %d, want 1", len(f.Workflows))
	}
	if f.Annotations[0].Kind != "invariants" {
		t.Errorf("annotation 0 kind = %q", f.Annotations[0].Kind)
	}
	if f.Annotations[1].Kind != "antipatterns" {
		t.Errorf("annotation 1 kind = %q", f.Annotations[1].Kind)
	}
}

func TestParseErrorMap(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@error_map sample_rejection
@purpose Documents all sample rejection paths
@entries
  ErrOutOfOrder — timestamp before series max | retriable | out_of_order_total | silently drops [src: head.go:686]
  ErrTooOld — outside OOO window | fatal_for_batch | too_old_total | breaks loop [src: head.go:681]
`
	f, warns, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warns {
		t.Logf("warning: %s", w)
	}
	if len(f.Annotations) != 1 {
		t.Fatalf("annotations = %d, want 1", len(f.Annotations))
	}
	a := f.Annotations[0]
	if a.Kind != "error_map" {
		t.Errorf("kind = %q, want %q", a.Kind, "error_map")
	}
	if a.Name != "sample_rejection" {
		t.Errorf("name = %q, want %q", a.Name, "sample_rejection")
	}
	if purpose, ok := a.Fields["purpose"]; !ok || purpose.InlineValue != "Documents all sample rejection paths" {
		t.Errorf("purpose = %v", a.Fields["purpose"])
	}
	entries, ok := a.Fields["entries"]
	if !ok {
		t.Fatal("missing entries field")
	}
	if len(entries.Lines) != 2 {
		t.Fatalf("entries lines = %d, want 2", len(entries.Lines))
	}
	if !strings.Contains(entries.Lines[0], "ErrOutOfOrder") {
		t.Errorf("entries line 0 = %q", entries.Lines[0])
	}
	if !strings.Contains(entries.Lines[1], "ErrTooOld") {
		t.Errorf("entries line 1 = %q", entries.Lines[1])
	}
	// Check source refs extracted from multi-line entries
	if len(entries.SourceRefs) != 2 {
		t.Errorf("source refs = %d, want 2", len(entries.SourceRefs))
	}
	if len(warns) > 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

func TestParseLock(t *testing.T) {
	input := `@module test/mod
@lang go
@version 1.0.0
@aid_version 0.1

---

@lock headMu
@kind sync.Mutex
@purpose Serializes all head-append operations
@protects head.minTime, head.maxTime, head.chunks
@acquired_by Appender.Append, Appender.Commit
@ordering headMu -> chunkMu (never reversed)
@deadlock_avoidance Always acquire headMu before chunkMu
@source_file head.go
@source_line 42
`
	f, warns, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warns {
		t.Logf("warning: %s", w)
	}
	if len(f.Annotations) != 1 {
		t.Fatalf("annotations = %d, want 1", len(f.Annotations))
	}
	a := f.Annotations[0]
	if a.Kind != "lock" {
		t.Errorf("kind = %q, want %q", a.Kind, "lock")
	}
	if a.Name != "headMu" {
		t.Errorf("name = %q, want %q", a.Name, "headMu")
	}
	checks := map[string]string{
		"kind":               "sync.Mutex",
		"purpose":            "Serializes all head-append operations",
		"protects":           "head.minTime, head.maxTime, head.chunks",
		"acquired_by":        "Appender.Append, Appender.Commit",
		"ordering":           "headMu -> chunkMu (never reversed)",
		"deadlock_avoidance": "Always acquire headMu before chunkMu",
		"source_file":        "head.go",
		"source_line":        "42",
	}
	for fieldName, want := range checks {
		field, ok := a.Fields[fieldName]
		if !ok {
			t.Errorf("missing field %q", fieldName)
			continue
		}
		if field.InlineValue != want {
			t.Errorf("field %q = %q, want %q", fieldName, field.InlineValue, want)
		}
	}
	if len(warns) > 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}
