// Package l2 implements the Layer 2 AID generation pipeline.
package l2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// BuildGeneratorPrompt constructs the prompt for a Layer 2 generator agent.
// The agent reads the L1 AID + source code and produces source-linked semantic docs.
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

	// Source file listing
	b.WriteString("## Source files to read\n\n")
	b.WriteString(fmt.Sprintf("Source directory: %s\n\n", sourceDir))
	files, _ := listSourceFiles(sourceDir)
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}
	b.WriteString("\n")

	// Instructions
	b.WriteString(generatorInstructions)

	return b.String(), nil
}

const generatorInstructions = `## Instructions

Read the L1 AID to understand the API surface, then read the KEY source files. Produce a Layer 2 AID file that adds:

1. **@workflow blocks** — document major data flows with numbered steps
2. **Enhanced @purpose** — explain WHY, not just WHAT
3. **@invariants with [src:] references** — constraints that always hold
4. **@antipatterns with [src:] references** — common mistakes to avoid
5. **@pre/@post with [src:] references** — preconditions and postconditions
6. **@error_map blocks** — if the module defines error sentinel values or has complex error handling, document the error taxonomy (see format below)
7. **@lock blocks** — if the module uses mutexes, RWMutexes, channels as semaphores, or atomic operations for concurrency control, document each lock (see format below)

For EVERY semantic claim, include a [src: relative/path:LINE] or [src: relative/path:START-END] reference.

### @error_map format

Use when a module defines error sentinel values (e.g., ErrOutOfOrder, ErrNotFound) and callers handle them differently. Each entry documents one error path.

` + "```" + `
@error_map <name>
@purpose <what this error group covers>
@entries
  <ErrorName> — <when it occurs> | <classification> | <metric> | <caller_behavior> [src: file:LINE]
  <ErrorName> — <when it occurs> | <classification> | <metric> | <caller_behavior> [src: file:LINE]
` + "```" + `

Classification values: retriable, fatal, fatal_for_batch, silent_drop, logged_only
If no metric is associated, use "none". If caller behavior varies, describe the most common path.

### @lock format

Use when a module contains sync.Mutex, sync.RWMutex, channel semaphores, or atomic-based locks.

` + "```" + `
@lock <LockName>
@kind <sync.Mutex | sync.RWMutex | chan struct{} | atomic | sync.Cond>
@purpose <what data/invariant this lock protects>
@protects <specific fields or state guarded>
@acquired_by [<Function1>, <Function2>]
@ordering <lock ordering constraints, e.g., "acquire AFTER BundleOperationLock, BEFORE rotationLocks">
@deadlock_avoidance <strategy used, e.g., "released before disk I/O", "sorted acquisition order">
@source_file <relative/path>
@source_line <line number>
` + "```" + `

Only document locks that are architecturally significant — skip trivial internal mutexes on small helper structs.

### Output format

Start the output with:
` + "```" + `
@module <module-name>
@lang <language>
@code_version git:<current-commit-hash>
@aid_status draft
@aid_generated_by layer2-generator
@depends [<dependency-packages>]
` + "```" + `

Focus on the MOST IMPORTANT architectural knowledge — the stuff that would take hours to figure out from reading code. Don't document trivial getters.

DO NOT read CLAUDE.md or README.md.
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
