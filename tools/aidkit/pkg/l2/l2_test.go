// L2 tests cover prompt construction, source listing, staleness helpers, and CheckStaleness (git).
package l2

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

func TestReadAidAsText(t *testing.T) {
	f := &parser.AidFile{
		Header: parser.Header{
			Module:  "svc/auth",
			Lang:    "go",
			Purpose: "Authentication helpers.",
		},
		Entries: []parser.Entry{
			{
				Kind: "fn",
				Name: "Login",
				Fields: map[string]parser.Field{
					"fn":      {Name: "fn", InlineValue: "Login"},
					"purpose": {Name: "purpose", InlineValue: "Validates credentials."},
				},
			},
		},
	}
	out, err := readAidAsText(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@module svc/auth") {
		t.Errorf("missing module: %q", out)
	}
	if !strings.Contains(out, "@fn Login") {
		t.Errorf("missing fn: %q", out)
	}
}

func TestListSourceFiles(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// Included extensions
	for _, rel := range []string{"a.go", "b.py", filepath.Join("sub", "c.ts"), "d.rs"} {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Skipped: _test.go
	if err := os.WriteFile(filepath.Join(root, "z_test.go"), []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Skipped: wrong extension
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := listSourceFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"a.go", "b.py", "d.rs", filepath.Join("sub", "c.ts")} {
		if !containsFile(files, want) {
			t.Errorf("expected path %q in %v", want, files)
		}
	}
	if containsFile(files, "z_test.go") {
		t.Errorf("should skip _test.go, got %v", files)
	}
	if containsFile(files, "readme.md") {
		t.Errorf("should skip .md, got %v", files)
	}
}

func containsFile(files []string, want string) bool {
	for _, f := range files {
		if f == want {
			return true
		}
	}
	return false
}

func TestBuildIncrementalPrompt(t *testing.T) {
	aid := &parser.AidFile{
		Header: parser.Header{
			Module:      "app",
			CodeVersion: "git:abc1234",
		},
	}
	claims := []StaleClaim{
		{
			Entry:     "fn:Handle",
			Field:     "invariants",
			Ref:       parser.SourceRef{File: "api/handler.go", StartLine: 10, EndLine: 12},
			Reason:    "lines changed",
			ClaimText: "Must validate input",
		},
	}
	prompt := BuildIncrementalPrompt(aid, claims, "/proj/root")
	if !strings.Contains(prompt, "app") {
		t.Error("missing module name")
	}
	if !strings.Contains(prompt, "1 claim(s)") {
		t.Error("missing claim count")
	}
	if !strings.Contains(prompt, "api/handler.go") {
		t.Error("missing referenced file")
	}
	if !strings.Contains(prompt, "Stale claim 1") {
		t.Error("missing stale claim section")
	}
	if !strings.Contains(prompt, "ONLY the stale claims") {
		t.Error("missing incremental instructions")
	}
}

func TestBuildGeneratorPrompt(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	aid := &parser.AidFile{
		Header: parser.Header{Module: "demo", Lang: "go", Purpose: "Demo."},
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Run", Fields: map[string]parser.Field{
				"fn": {Name: "fn", InlineValue: "Run"},
			}},
		},
	}
	prompt, err := BuildGeneratorPrompt(aid, srcDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{
		"Layer 1 AID",
		"Layer 2 AID Generator",
		"@module demo",
		"Source directory:",
		"main.go",
		"Do NOT read CLAUDE.md",
	} {
		if !strings.Contains(prompt, frag) {
			t.Errorf("prompt missing %q", frag)
		}
	}
}

func TestBuildReviewerPrompt(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "lib.go")
	if err := os.WriteFile(src, []byte("package p\nfunc X() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := `@module rev
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn X
@purpose Does work [src: lib.go:2]
`
	draft, _, err := parser.ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := BuildReviewerPrompt(draft, root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Layer 2 AID Reviewer") {
		t.Error("missing role")
	}
	if !strings.Contains(prompt, "lib.go") {
		t.Error("missing source file hint")
	}
	if !strings.Contains(prompt, "Verification Report") {
		t.Error("missing output format instructions")
	}
}

func TestCollectAllSourceRefs(t *testing.T) {
	input := `@module m
@lang go
@version 1.0.0
@aid_version 0.1

---

@fn A
@purpose One [src: a.go:1]

---

@workflow main
@steps
  Step [src: b.go:5-10]
`
	f, _, err := parser.ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	refs := collectAllSourceRefs(f)
	if len(refs) != 2 {
		t.Fatalf("refs = %d, want 2: %#v", len(refs), refs)
	}
}

func TestParseHunkRange(t *testing.T) {
	tests := []struct {
		line       string
		wantStart  int
		wantEndMin int // inclusive end >= this (exact depends on count)
	}{
		{"@@ -10,7 +10,8 @@", 10, 17}, // 10 + 8 - 1 = 17
		{"@@ -0,0 +1,3 @@", 1, 3},
	}
	for _, tt := range tests {
		start, end := parseHunkRange(tt.line)
		if start != tt.wantStart {
			t.Errorf("parseHunkRange(%q) start = %d, want %d", tt.line, start, tt.wantStart)
		}
		if end < tt.wantEndMin {
			t.Errorf("parseHunkRange(%q) end = %d, want >= %d", tt.line, end, tt.wantEndMin)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 100); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	long := strings.Repeat("a", 50)
	got := truncate(long, 20)
	if len(got) > 20 {
		t.Errorf("truncate len = %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected ellipsis suffix: %q", got)
	}
	s := "a\nb\nc"
	if truncate(s, 100) != "a b c" {
		t.Errorf("newlines: got %q want %q", truncate(s, 100), "a b c")
	}
}

func TestCheckStaleness_Errors(t *testing.T) {
	_, err := CheckStaleness(&parser.AidFile{Header: parser.Header{}}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "@code_version") {
		t.Errorf("want code_version error, got %v", err)
	}
	_, err = CheckStaleness(&parser.AidFile{Header: parser.Header{CodeVersion: "v1.0.0"}}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "git:") {
		t.Errorf("want git: prefix error, got %v", err)
	}
}

func TestCheckStaleness_SameCommitNoStale(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	if !tryGitInit(t, dir) {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "f.go")
	git(t, dir, "commit", "-m", "init")
	short := strings.TrimSpace(gitOut(t, dir, "rev-parse", "--short", "HEAD"))

	aid := &parser.AidFile{
		Header: parser.Header{CodeVersion: "git:" + short, Module: "m"},
	}
	stale, err := CheckStaleness(aid, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("expected no stale claims, got %+v", stale)
	}
}

func TestCheckStaleness_StaleWhenFileChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	if !tryGitInit(t, dir) {
		return
	}
	p := filepath.Join(dir, "tracked.go")
	if err := os.WriteFile(p, []byte("package p\n\nfunc Alpha() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "tracked.go")
	git(t, dir, "commit", "-m", "c1")
	oldShort := strings.TrimSpace(gitOut(t, dir, "rev-parse", "--short", "HEAD"))

	if err := os.WriteFile(p, []byte("package p\n\nfunc Beta() int { return 2 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "tracked.go")
	git(t, dir, "commit", "-m", "c2")

	aid := &parser.AidFile{
		Header: parser.Header{CodeVersion: "git:" + oldShort, Module: "m"},
		Entries: []parser.Entry{
			{
				Kind: "fn",
				Name: "Alpha",
				Fields: map[string]parser.Field{
					"purpose": {
						Name:        "purpose",
						InlineValue: "Does alpha",
						SourceRefs: []parser.SourceRef{
							{File: "tracked.go", StartLine: 3, EndLine: 3},
						},
					},
				},
			},
		},
	}
	stale, err := CheckStaleness(aid, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) == 0 {
		t.Fatal("expected at least one stale claim")
	}
	if stale[0].Reason != "lines changed" {
		t.Errorf("reason = %q", stale[0].Reason)
	}
	if stale[0].Ref.File != "tracked.go" {
		t.Errorf("file = %q", stale[0].Ref.File)
	}
}

// tryGitInit runs git init in dir. If init fails (e.g. sandbox blocks .git/hooks), the test is skipped.
func tryGitInit(t *testing.T, dir string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "init")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git init not available in this environment: %v\n%s", err, out)
		return false
	}
	git(t, dir, "config", "user.email", "aid-test@local")
	git(t, dir, "config", "user.name", "aid-test")
	return true
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// --- Tests for L2 optimizations ---

func TestExtractRelevantFiles(t *testing.T) {
	aid := &parser.AidFile{
		Header: parser.Header{Module: "pkg", Lang: "go"},
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Handle", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "Handle"},
				"source_file": {Name: "source_file", InlineValue: "handler.go"},
				"calls":       {Name: "calls", InlineValue: "Validate, Store.Push"},
			}},
			{Kind: "fn", Name: "Validate", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "Validate"},
				"source_file": {Name: "source_file", InlineValue: "validate.go"},
			}},
			{Kind: "fn", Name: "Store.Push", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "Store.Push"},
				"source_file": {Name: "source_file", InlineValue: "store.go"},
			}},
			{Kind: "type", Name: "Request", Fields: map[string]parser.Field{
				"type":        {Name: "type", InlineValue: "Request"},
				"source_file": {Name: "source_file", InlineValue: "model.go"},
			}},
		},
	}

	files := extractRelevantFiles(aid)

	// Should include all 4 source files
	want := []string{"handler.go", "model.go", "store.go", "validate.go"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i, w := range want {
		if files[i] != w {
			t.Errorf("files[%d] = %q, want %q", i, files[i], w)
		}
	}
}

func TestExtractRelevantFiles_Empty(t *testing.T) {
	aid := &parser.AidFile{
		Header: parser.Header{Module: "pkg", Lang: "go"},
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Run", Fields: map[string]parser.Field{
				"fn": {Name: "fn", InlineValue: "Run"},
				// No @source_file field
			}},
		},
	}

	files := extractRelevantFiles(aid)
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestExtractRelevantFiles_CalleeResolution(t *testing.T) {
	aid := &parser.AidFile{
		Header: parser.Header{Module: "pkg", Lang: "go"},
		Entries: []parser.Entry{
			{Kind: "fn", Name: "A", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "A"},
				"source_file": {Name: "source_file", InlineValue: "a.go"},
				"calls":       {Name: "calls", InlineValue: "[B, C]"},
			}},
			{Kind: "fn", Name: "B", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "B"},
				"source_file": {Name: "source_file", InlineValue: "b.go"},
			}},
			// C is not in the L1 AID (external call) — should not add a file
		},
	}

	files := extractRelevantFiles(aid)
	if !containsFile(files, "a.go") {
		t.Error("missing a.go")
	}
	if !containsFile(files, "b.go") {
		t.Error("missing b.go (callee of A)")
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %v", files)
	}
}

func TestParseCallsList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"Validate, Store.Push", []string{"Validate", "Store.Push"}},
		{"[A, B, C]", []string{"A", "B", "C"}},
		{"Single", []string{"Single"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseCallsList(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseCallsList(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i, w := range tt.want {
			if got[i] != w {
				t.Errorf("parseCallsList(%q)[%d] = %q, want %q", tt.input, i, got[i], w)
			}
		}
	}
}

func TestHasErrorFields(t *testing.T) {
	withErrors := &parser.AidFile{
		Entries: []parser.Entry{{Kind: "fn", Name: "X", Fields: map[string]parser.Field{
			"errors": {Name: "errors", Lines: []string{"SomeError — bad"}},
		}}},
	}
	without := &parser.AidFile{
		Entries: []parser.Entry{{Kind: "fn", Name: "X", Fields: map[string]parser.Field{
			"fn": {Name: "fn", InlineValue: "X"},
		}}},
	}
	if !hasErrorFields(withErrors) {
		t.Error("expected true for aid with @errors")
	}
	if hasErrorFields(without) {
		t.Error("expected false for aid without @errors")
	}
}

func TestDetectConcurrencyPrimitives(t *testing.T) {
	dir := t.TempDir()

	// File with mutex
	if err := os.WriteFile(filepath.Join(dir, "lock.go"), []byte("var mu sync.Mutex\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// File without
	if err := os.WriteFile(filepath.Join(dir, "plain.go"), []byte("func X() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !detectConcurrencyPrimitives(dir, []string{"lock.go"}) {
		t.Error("expected true for file with sync.Mutex")
	}
	if detectConcurrencyPrimitives(dir, []string{"plain.go"}) {
		t.Error("expected false for file without primitives")
	}
}

func TestDiffL1Aids_AllCategories(t *testing.T) {
	oldL1 := &parser.AidFile{
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Unchanged", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "() -> None"},
			}},
			{Kind: "fn", Name: "Modified", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "(a: int) -> int"},
			}},
			{Kind: "fn", Name: "Removed", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "() -> str"},
			}},
		},
	}
	newL1 := &parser.AidFile{
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Unchanged", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "() -> None"},
			}},
			{Kind: "fn", Name: "Modified", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "(a: int, b: int) -> int"}, // changed
			}},
			{Kind: "fn", Name: "Added", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "() -> bool"},
			}},
		},
	}

	diff := DiffL1Aids(oldL1, newL1)

	if len(diff.Unchanged) != 1 || diff.Unchanged[0].Name != "Unchanged" {
		t.Errorf("Unchanged = %v", diff.Unchanged)
	}
	if len(diff.Modified) != 1 || diff.Modified[0].New.Name != "Modified" {
		t.Errorf("Modified = %v", diff.Modified)
	}
	if len(diff.New) != 1 || diff.New[0].Name != "Added" {
		t.Errorf("New = %v", diff.New)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Name != "Removed" {
		t.Errorf("Removed = %v", diff.Removed)
	}
}

func TestDiffL1Aids_ReverseCallPropagation(t *testing.T) {
	oldL1 := &parser.AidFile{
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Caller", Fields: map[string]parser.Field{
				"sig":   {Name: "sig", InlineValue: "() -> None"},
				"calls": {Name: "calls", InlineValue: "Modified"},
			}},
			{Kind: "fn", Name: "Modified", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "(a: int) -> int"},
			}},
		},
	}
	newL1 := &parser.AidFile{
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Caller", Fields: map[string]parser.Field{
				"sig":   {Name: "sig", InlineValue: "() -> None"}, // same sig
				"calls": {Name: "calls", InlineValue: "Modified"},  // same calls
			}},
			{Kind: "fn", Name: "Modified", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "(a: int, b: int) -> int"}, // changed
			}},
		},
	}

	diff := DiffL1Aids(oldL1, newL1)

	// Caller should be promoted to Modified because it calls Modified
	if len(diff.Modified) != 2 {
		t.Fatalf("expected 2 modified (Modified + Caller via propagation), got %d", len(diff.Modified))
	}
	if len(diff.Unchanged) != 0 {
		t.Errorf("expected 0 unchanged after propagation, got %d", len(diff.Unchanged))
	}
}

func TestDiffL1Aids_EmptyDiff(t *testing.T) {
	aid := &parser.AidFile{
		Entries: []parser.Entry{
			{Kind: "fn", Name: "Same", Fields: map[string]parser.Field{
				"sig": {Name: "sig", InlineValue: "() -> None"},
			}},
		},
	}

	diff := DiffL1Aids(aid, aid)

	if len(diff.New) != 0 || len(diff.Modified) != 0 || len(diff.Removed) != 0 {
		t.Errorf("expected empty diff, got new=%d mod=%d rem=%d",
			len(diff.New), len(diff.Modified), len(diff.Removed))
	}
	if len(diff.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(diff.Unchanged))
	}
}

func TestBuildIncrementalGeneratorPrompt(t *testing.T) {
	newL1 := &parser.AidFile{
		Header: parser.Header{Module: "pkg", Lang: "go"},
		Entries: []parser.Entry{
			{Kind: "fn", Name: "New", Fields: map[string]parser.Field{
				"fn":          {Name: "fn", InlineValue: "New"},
				"sig":         {Name: "sig", InlineValue: "() -> bool"},
				"source_file": {Name: "source_file", InlineValue: "new.go"},
			}},
		},
	}
	existingL2 := &parser.AidFile{
		Header: parser.Header{Module: "pkg"},
	}
	diff := &L1Diff{
		New:       newL1.Entries,
		Modified:  nil,
		Unchanged: nil,
		Removed:   nil,
	}

	srcDir := t.TempDir()

	prompt, err := BuildIncrementalGeneratorPrompt(newL1, existingL2, diff, srcDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, frag := range []string{
		"INCREMENTAL update",
		"New entries: 1",
		"new.go",
		"@fn New",
	} {
		if !strings.Contains(prompt, frag) {
			t.Errorf("prompt missing %q", frag)
		}
	}
}
