package main

import (
	"sort"
	"strings"
	"unicode"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// Aria tests live inline as `test "name" { ... }` TestBlock decls rather than
// in separate *_test.aria files (no such convention exists). This file
// implements the --test surface:
//
//   - TestBlock decls: always filtered from the main AID (handled by falling
//     through the switch in aria_extractor.go). They only appear in *_test.aid.
//   - Test-scaffolding types (Mock*, Stub*, Fake*, Spy*): surface in *_test.aid
//     so consumers can see the shape of the mock contract.
//   - Test-helper functions (setup_, teardown_, helper_, mock_, new_mock_):
//     surface in *_test.aid.
//   - Each TestBlock becomes a synthetic FnEntry with name `test_<slug>` and
//     @calls populated by walking the block body. This gives cartograph a
//     test→production call edge without inventing a new entry kind.

func extractTestsFromPrograms(progs []*parser.Program, files []string, dir, modName, version string) *AidFile {
	aid := &AidFile{
		Header: ModuleHeader{
			Module:     modName,
			Lang:       "aria",
			Version:    version,
			AidVersion: "0.2",
		},
	}

	// Gather deps (tests often import more than production code).
	seenDep := map[string]bool{}
	for _, p := range progs {
		if p == nil {
			continue
		}
		for _, imp := range p.Imports {
			path := strings.Join(imp.Path, ".")
			if path != "" && !seenDep[path] {
				seenDep[path] = true
				aid.Header.Deps = append(aid.Header.Deps, path)
			}
		}
	}
	sort.Strings(aid.Header.Deps)

	docs := newSourceIndex()
	for _, f := range files {
		_ = docs.loadFile(f) // best-effort; missing comments are fine in tests
	}

	for i, prog := range progs {
		if prog == nil {
			continue
		}
		rel := relPath(dir, files[i])

		for _, d := range prog.Decls {
			switch dd := d.(type) {
			case *parser.TestBlock:
				aid.Entries = append(aid.Entries, testBlockToEntry(dd, rel))

			case *parser.TypeDecl:
				if isTestScaffoldName(dd.Name) {
					aid.Entries = append(aid.Entries, extractTypeDecl(dd, rel, docs))
				}
			case *parser.EnumDecl:
				if isTestScaffoldName(dd.Name) {
					aid.Entries = append(aid.Entries, extractEnum(dd, rel, docs))
				}
			case *parser.AliasDecl:
				if isTestScaffoldName(dd.Name) {
					aid.Entries = append(aid.Entries, extractAlias(dd, rel, docs))
				}
			case *parser.TraitDecl:
				if isTestScaffoldName(dd.Name) {
					trait, methods := extractTrait(dd, rel, docs)
					aid.Entries = append(aid.Entries, trait)
					for _, m := range methods {
						aid.Entries = append(aid.Entries, m)
					}
				}
			case *parser.FnDecl:
				if isTestHelperFnName(dd.Name) {
					aid.Entries = append(aid.Entries, extractFn(dd, "", rel, docs))
				}
			case *parser.ImplDecl:
				if isTestScaffoldName(dd.TypeName) {
					for _, m := range dd.Methods {
						aid.Entries = append(aid.Entries, extractFn(m, dd.TypeName, rel, docs))
					}
				}
			}
		}
	}

	sort.SliceStable(aid.Entries, func(i, j int) bool {
		return entryName(aid.Entries[i]) < entryName(aid.Entries[j])
	})
	return aid
}

// testBlockToEntry renders an Aria `test "name" { body }` as a synthetic
// FnEntry so the entry format stays single-kind. The @purpose preserves the
// original quoted name; @calls exposes which production fns the test exercises.
func testBlockToEntry(tb *parser.TestBlock, filePath string) FnEntry {
	fnName := "test_" + slugify(tb.Name)
	w := &callWalker{
		paramTypes: map[string]string{},
		localVars:  map[string]bool{},
		seen:       map[string]bool{},
	}
	if tb.Body != nil {
		w.collectLocalBindings(tb.Body)
		w.walkExpr(tb.Body)
	}
	calls := make([]string, 0, len(w.seen))
	for n := range w.seen {
		calls = append(calls, n)
	}
	sort.Strings(calls)

	return FnEntry{
		Name:       fnName,
		Purpose:    "Test: " + tb.Name,
		Sigs:       []string{"()"},
		Calls:      calls,
		SourceFile: filePath,
		SourceLine: tb.Pos.Line,
	}
}

// isTestScaffoldName matches Mock* / Stub* / Fake* / Spy* — the conventional
// prefixes for test-scaffolding types across languages.
func isTestScaffoldName(name string) bool {
	for _, prefix := range []string{"Mock", "Stub", "Fake", "Spy"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// isTestHelperFnName matches conventional test-helper prefixes. Snake-case
// and camelCase both accepted so the rule works whether a project favours
// `setup_fixture` or `setupFixture`.
func isTestHelperFnName(name string) bool {
	lc := strings.ToLower(name)
	for _, prefix := range []string{"setup_", "setup", "teardown_", "teardown", "helper_", "helper", "mock_", "new_mock_", "new_stub_", "new_fake_", "new_spy_"} {
		if strings.HasPrefix(lc, prefix) {
			return true
		}
	}
	return false
}

// slugify turns an arbitrary test-block name into a safe identifier:
// non-alphanumerics collapse to single underscores, leading/trailing stripped,
// lowercased. "my feature, yay!" → "my_feature_yay".
func slugify(s string) string {
	var b strings.Builder
	lastUnderscore := true
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "anon"
	}
	return out
}
