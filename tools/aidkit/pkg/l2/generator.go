// Package l2 implements the Layer 2 AID generation pipeline.
package l2

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// BuildGeneratorPrompt constructs the prompt for a Layer 2 generator agent.
// The agent reads the L1 AID + source code and produces source-linked semantic docs.
//
// Source file selection is guided by L1 metadata: only files containing documented
// functions/types (via @source_file) and their callees (via @calls) are listed.
// This typically reduces the file set by 40-60% compared to listing all source files.
func BuildGeneratorPrompt(l1Aid *parser.AidFile, sourceDir string, depAids []*parser.AidFile) (string, error) {
	var b strings.Builder

	b.WriteString("You are a Layer 2 AID Generator. Your job is to produce semantic documentation.\n\n")

	// L1 AID content
	b.WriteString("## Layer 1 AID (mechanical extraction)\n\n")
	l1Content, err := readAidAsText(l1Aid)
	if err != nil {
		return "", fmt.Errorf("read L1 AID: %w", err)
	}
	b.WriteString(l1Content)
	b.WriteString("\n\n")

	// Dependency AIDs
	if len(depAids) > 0 {
		b.WriteString("## Related package AIDs (for cross-package context)\n\n")
		for _, dep := range depAids {
			depContent, err := readAidAsText(dep)
			if err != nil {
				continue
			}
			b.WriteString(depContent)
			b.WriteString("\n\n---\n\n")
		}
	}

	// Source file listing — guided by L1 metadata
	b.WriteString("## Source files to read\n\n")
	b.WriteString(fmt.Sprintf("Source directory: %s\n\n", sourceDir))

	files := extractRelevantFiles(l1Aid)
	if len(files) == 0 {
		// Fallback: L1 has no @source_file fields (pre-extraction L1)
		files, _ = listSourceFiles(sourceDir)
		b.WriteString("(All source files listed — L1 has no @source_file metadata)\n\n")
	} else {
		b.WriteString("(Selected by L1 analysis — contains all documented functions and their callees)\n\n")
	}
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}
	b.WriteString("\n")

	// Instructions — conditionally assembled
	b.WriteString(coreInstructions)

	if hasErrorFields(l1Aid) {
		b.WriteString(errorMapInstructions)
	}
	if detectConcurrencyPrimitives(sourceDir, files) {
		b.WriteString(lockInstructions)
	}

	b.WriteString(outputFormatInstructions)

	return b.String(), nil
}

// extractRelevantFiles analyzes L1 AID entries to determine which source files
// the generator actually needs to read. Uses @source_file for direct references
// and @calls to include callee files.
func extractRelevantFiles(l1Aid *parser.AidFile) []string {
	fileSet := map[string]bool{}

	// Build a name → source_file lookup for resolving callees
	nameToFile := map[string]string{}
	for _, e := range l1Aid.Entries {
		if sf, ok := e.Fields["source_file"]; ok && sf.InlineValue != "" {
			fileSet[sf.InlineValue] = true
			nameToFile[e.Name] = sf.InlineValue
			// Also index by short name (without type prefix) for method resolution
			if idx := strings.LastIndex(e.Name, "."); idx >= 0 {
				nameToFile[e.Name[idx+1:]] = sf.InlineValue
			}
		}
	}

	// Resolve callees to their source files
	for _, e := range l1Aid.Entries {
		if calls, ok := e.Fields["calls"]; ok {
			callList := parseCallsList(calls.InlineValue)
			for _, callee := range callList {
				if f, ok := nameToFile[callee]; ok {
					fileSet[f] = true
				}
			}
		}
	}

	// Convert to sorted slice for deterministic output
	var files []string
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// parseCallsList splits a @calls value like "Validate, store.Push, json.Marshal"
// into individual function names.
func parseCallsList(callsValue string) []string {
	callsValue = strings.TrimPrefix(callsValue, "[")
	callsValue = strings.TrimSuffix(callsValue, "]")
	parts := strings.Split(callsValue, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// hasErrorFields checks if any L1 entry has an @errors field.
func hasErrorFields(l1Aid *parser.AidFile) bool {
	for _, e := range l1Aid.Entries {
		if _, ok := e.Fields["errors"]; ok {
			return true
		}
	}
	return false
}

// detectConcurrencyPrimitives scans the relevant source files for sync primitives.
func detectConcurrencyPrimitives(sourceDir string, files []string) bool {
	patterns := []string{"sync.Mutex", "sync.RWMutex", "chan struct{}", "atomic."}
	for _, f := range files {
		fullPath := filepath.Join(sourceDir, f)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		for _, pat := range patterns {
			if strings.Contains(content, pat) {
				return true
			}
		}
	}
	return false
}

// coreInstructions is always included in the generator prompt.
const coreInstructions = `## Instructions

Read ONLY the source files listed above. For each @fn entry in the L1 AID, its source is at the @source_file and @source_line indicated. Focus on: (1) function bodies, (2) type definitions, (3) error sentinels. Do NOT read CLAUDE.md, README.md, or test files. If you discover a function calls into a file not listed, you may read it.

Produce an enriched AID file that **preserves ALL L1 content** and adds L2 semantic annotations.

### CRITICAL: Preserve L1 Content

Your output MUST include every @fn, @type, @trait, and @const entry from the L1 AID above — with their @sig, @params, @returns, @calls, @source_file, and @source_line fields intact. Do NOT drop or rewrite L1 entries.

For each L1 entry, you MAY enhance:
- The @purpose field (explain WHY, not just WHAT)
- Add @pre/@post conditions
- Add @errors details
- Add @thread_safety notes

But you MUST keep: @fn name, @sig, @params, @calls, @source_file, @source_line unchanged.

### Add L2 Blocks

After the preserved L1 entries, add:

1. **@workflow blocks** — major data flows with numbered steps
2. **@invariants with [src:] references** — constraints that always hold
3. **@antipatterns with [src:] references** — common mistakes to avoid

For EVERY semantic claim, include a [src: relative/path:LINE] or [src: relative/path:START-END] reference.

`

// errorMapInstructions is included only when L1 entries have @errors fields.
const errorMapInstructions = `### @error_map format

If the module defines error sentinel values (e.g., ErrOutOfOrder, ErrNotFound), add @error_map blocks:

` + "```" + `
@error_map <name>
@purpose <what this error group covers>
@entries
  <ErrorName> — <when it occurs> | <classification> | <metric> | <caller_behavior> [src: file:LINE]
` + "```" + `

Classification values: retriable, fatal, fatal_for_batch, silent_drop, logged_only

`

// lockInstructions is included only when source files contain sync primitives.
const lockInstructions = `### @lock format

Document architecturally significant locks (skip trivial internal mutexes):

` + "```" + `
@lock <LockName>
@kind <sync.Mutex | sync.RWMutex | chan struct{} | atomic | sync.Cond>
@purpose <what data/invariant this lock protects>
@protects <specific fields or state guarded>
@acquired_by [<Function1>, <Function2>]
@ordering <lock ordering constraints>
@source_file <relative/path>
@source_line <line number>
` + "```" + `

`

// outputFormatInstructions defines the output structure. Always included.
const outputFormatInstructions = `### Output format

` + "```" + `
@module <module-name>
@lang <language>
@code_version git:<current-commit-hash>
@aid_status draft
@aid_generated_by layer2-generator
@depends [<dependency-packages>]
---
[ALL L1 entries preserved, with L2 enhancements on existing entries]
---
[NEW @workflow, @invariants, @antipatterns blocks]
` + "```" + `

Focus on the MOST IMPORTANT architectural knowledge — the stuff that would take hours to figure out from reading code. Don't document trivial getters. Preserve every L1 entry even if you have nothing to add.
`

func readAidAsText(f *parser.AidFile) (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("@module %s\n", f.Header.Module))
	if f.Header.Lang != "" {
		b.WriteString(fmt.Sprintf("@lang %s\n", f.Header.Lang))
	}
	if f.Header.Purpose != "" {
		b.WriteString(fmt.Sprintf("@purpose %s\n", f.Header.Purpose))
	}

	for _, e := range f.Entries {
		b.WriteString(fmt.Sprintf("\n@%s %s\n", e.Kind, e.Name))
		for name, field := range e.Fields {
			if name == e.Kind { // skip the entry-defining field
				continue
			}
			if field.InlineValue != "" {
				b.WriteString(fmt.Sprintf("@%s %s\n", name, field.InlineValue))
			} else {
				b.WriteString(fmt.Sprintf("@%s\n", name))
			}
			for _, line := range field.Lines {
				b.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
	}
	return b.String(), nil
}

func listSourceFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".py" || ext == ".ts" || ext == ".rs" {
			if !strings.HasSuffix(path, "_test.go") {
				rel, _ := filepath.Rel(dir, path)
				files = append(files, rel)
			}
		}
		return nil
	})
	return files, err
}
