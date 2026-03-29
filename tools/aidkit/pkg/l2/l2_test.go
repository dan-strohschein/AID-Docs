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
		"DO NOT read CLAUDE.md",
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
