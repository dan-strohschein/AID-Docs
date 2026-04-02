package l2

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// L1Diff categorizes entries by comparing an old L1 AID against a new L1 AID.
// This enables incremental L2 generation: only NEW and MODIFIED entries need
// L2 inference. UNCHANGED entries can carry forward their existing L2 annotations.
type L1Diff struct {
	New       []parser.Entry // Entries only in newL1
	Modified  []EntryPair    // Entries in both but with changed signatures/calls
	Unchanged []parser.Entry // Entries in both with identical key fields
	Removed   []parser.Entry // Entries only in oldL1
}

// EntryPair holds the old and new versions of a modified entry.
type EntryPair struct {
	Old parser.Entry
	New parser.Entry
}

// DiffL1Aids compares old and new L1 AID files and categorizes each entry.
// Comparison key: Kind + ":" + Name (e.g., "fn:Handler.ServeHTTP").
// Fields compared: @sig, @calls, @params, @source_line.
//
// After initial classification, reverse-call propagation marks entries
// that @calls a MODIFIED entry as also MODIFIED (one level deep).
func DiffL1Aids(oldL1, newL1 *parser.AidFile) *L1Diff {
	diff := &L1Diff{}

	// Index old entries by key
	oldByKey := map[string]parser.Entry{}
	for _, e := range oldL1.Entries {
		key := entryKey(e)
		oldByKey[key] = e
	}

	// Index new entries by key
	newByKey := map[string]parser.Entry{}
	for _, e := range newL1.Entries {
		newByKey[entryKey(e)] = e
	}

	// Classify new entries
	modifiedKeys := map[string]bool{}
	for _, newEntry := range newL1.Entries {
		key := entryKey(newEntry)
		if oldEntry, exists := oldByKey[key]; exists {
			if entriesMatch(oldEntry, newEntry) {
				diff.Unchanged = append(diff.Unchanged, newEntry)
			} else {
				diff.Modified = append(diff.Modified, EntryPair{Old: oldEntry, New: newEntry})
				modifiedKeys[key] = true
			}
		} else {
			diff.New = append(diff.New, newEntry)
		}
	}

	// Find removed entries
	for _, oldEntry := range oldL1.Entries {
		key := entryKey(oldEntry)
		if _, exists := newByKey[key]; !exists {
			diff.Removed = append(diff.Removed, oldEntry)
		}
	}

	// Reverse-call propagation: if entry A calls entry B and B is MODIFIED,
	// mark A as MODIFIED too (one level). This catches cascading semantic changes.
	diff.propagateCallers(newL1, modifiedKeys)

	return diff
}

// propagateCallers promotes UNCHANGED entries to MODIFIED if they call a MODIFIED entry.
func (d *L1Diff) propagateCallers(newL1 *parser.AidFile, modifiedKeys map[string]bool) {
	// Build a set of modified entry names (just the Name, not Kind:Name)
	modifiedNames := map[string]bool{}
	for key := range modifiedKeys {
		// key is "kind:Name", extract Name
		if idx := strings.Index(key, ":"); idx >= 0 {
			modifiedNames[key[idx+1:]] = true
		}
	}

	// Check each UNCHANGED entry to see if it calls a modified function
	var stillUnchanged []parser.Entry
	for _, entry := range d.Unchanged {
		if callsModified(entry, modifiedNames) {
			// Promote to MODIFIED — use same entry as both old and new since
			// the entry itself didn't change, but its semantics may have
			d.Modified = append(d.Modified, EntryPair{Old: entry, New: entry})
		} else {
			stillUnchanged = append(stillUnchanged, entry)
		}
	}
	d.Unchanged = stillUnchanged
}

// callsModified checks if an entry's @calls list includes any modified function name.
func callsModified(entry parser.Entry, modifiedNames map[string]bool) bool {
	calls, ok := entry.Fields["calls"]
	if !ok {
		return false
	}
	for _, callee := range parseCallsList(calls.InlineValue) {
		if modifiedNames[callee] {
			return true
		}
		// Also check short name (method without type prefix)
		if idx := strings.LastIndex(callee, "."); idx >= 0 {
			if modifiedNames[callee[idx+1:]] {
				return true
			}
		}
	}
	return false
}

// entryKey returns a unique key for an entry: "kind:Name".
func entryKey(e parser.Entry) string {
	return e.Kind + ":" + e.Name
}

// entriesMatch returns true if two entries have identical key fields.
func entriesMatch(a, b parser.Entry) bool {
	return fieldValue(a, "sig") == fieldValue(b, "sig") &&
		fieldValue(a, "calls") == fieldValue(b, "calls") &&
		fieldValue(a, "params") == fieldValue(b, "params") &&
		fieldValue(a, "source_line") == fieldValue(b, "source_line")
}

// fieldValue extracts the full value of a named field from an entry.
func fieldValue(e parser.Entry, name string) string {
	if f, ok := e.Fields[name]; ok {
		return f.Value()
	}
	return ""
}

// BuildIncrementalGeneratorPrompt constructs a prompt that generates L2 annotations
// for only the NEW and MODIFIED entries identified by DiffL1Aids.
// Existing L2 annotations for MODIFIED entries are included for context.
func BuildIncrementalGeneratorPrompt(
	newL1 *parser.AidFile,
	existingL2 *parser.AidFile,
	diff *L1Diff,
	sourceDir string,
	depAids []*parser.AidFile,
) (string, error) {
	var b strings.Builder

	b.WriteString("You are a Layer 2 AID Generator performing an INCREMENTAL update.\n")
	b.WriteString("Only generate L2 annotations for the new/changed entries listed below.\n\n")

	// Summary of changes
	b.WriteString("## Change summary\n\n")
	b.WriteString(fmt.Sprintf("- New entries: %d\n", len(diff.New)))
	b.WriteString(fmt.Sprintf("- Modified entries: %d\n", len(diff.Modified)))
	b.WriteString(fmt.Sprintf("- Unchanged entries: %d (L2 annotations preserved, not shown)\n", len(diff.Unchanged)))
	b.WriteString(fmt.Sprintf("- Removed entries: %d\n\n", len(diff.Removed)))

	// New entries — full L1 content
	if len(diff.New) > 0 {
		b.WriteString("## New entries (generate full L2 annotations)\n\n")
		for _, entry := range diff.New {
			writeEntry(&b, entry)
		}
		b.WriteString("\n")
	}

	// Modified entries — new L1 content + existing L2 for context
	if len(diff.Modified) > 0 {
		b.WriteString("## Modified entries (update L2 annotations)\n\n")
		for _, pair := range diff.Modified {
			b.WriteString(fmt.Sprintf("### %s %s\n\n", pair.New.Kind, pair.New.Name))
			b.WriteString("**Updated L1:**\n")
			writeEntry(&b, pair.New)

			// Include existing L2 annotations from the current L2 file
			if existingL2 != nil {
				if existingL2Entry := findEntry(existingL2, pair.New.Kind, pair.New.Name); existingL2Entry != nil {
					b.WriteString("\n**Existing L2 annotations (may need updating):**\n")
					writeL2Fields(&b, *existingL2Entry)
				}
			}
			b.WriteString("\n")
		}
	}

	// Source files — only those needed for new/modified entries
	b.WriteString("## Source files to read\n\n")
	b.WriteString(fmt.Sprintf("Source directory: %s\n\n", sourceDir))
	files := collectDiffSourceFiles(diff)
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %s\n", f))
	}
	b.WriteString("\n")

	// Dependency AIDs
	if len(depAids) > 0 {
		b.WriteString("## Related package AIDs\n\n")
		for _, dep := range depAids {
			depContent, err := readAidAsText(dep)
			if err != nil {
				continue
			}
			b.WriteString(depContent)
			b.WriteString("\n\n---\n\n")
		}
	}

	// Instructions — always regenerate module-level blocks if anything changed
	b.WriteString(coreInstructions)
	b.WriteString(`### Incremental generation rules

1. Generate L2 annotations (@purpose enhancement, @pre, @post, @errors, @thread_safety) for each NEW entry.
2. Update L2 annotations for each MODIFIED entry — the existing L2 is shown for context.
3. Regenerate @workflow, @invariants, and @antipatterns blocks for the whole module (they cross-reference multiple functions).
4. Output ONLY the new/updated entries and module-level blocks. Do NOT output unchanged entries.
5. Preserve @fn name, @sig, @params, @calls, @source_file, @source_line exactly from the L1 above.

`)
	b.WriteString(outputFormatInstructions)

	return b.String(), nil
}

// collectDiffSourceFiles gathers source files for NEW and MODIFIED entries.
func collectDiffSourceFiles(diff *L1Diff) []string {
	fileSet := map[string]bool{}
	for _, entry := range diff.New {
		if sf, ok := entry.Fields["source_file"]; ok && sf.InlineValue != "" {
			fileSet[sf.InlineValue] = true
		}
	}
	for _, pair := range diff.Modified {
		if sf, ok := pair.New.Fields["source_file"]; ok && sf.InlineValue != "" {
			fileSet[sf.InlineValue] = true
		}
	}
	var files []string
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// findEntry looks up an entry by kind and name in an AID file.
func findEntry(aid *parser.AidFile, kind, name string) *parser.Entry {
	for i := range aid.Entries {
		if aid.Entries[i].Kind == kind && aid.Entries[i].Name == name {
			return &aid.Entries[i]
		}
	}
	return nil
}

// writeEntry writes an entry's L1 fields to the builder.
func writeEntry(b *strings.Builder, e parser.Entry) {
	b.WriteString(fmt.Sprintf("@%s %s\n", e.Kind, e.Name))
	for name, field := range e.Fields {
		if name == e.Kind {
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

// writeL2Fields writes only L2-specific fields (purpose, pre, post, errors, etc.)
// from an existing entry, for context in incremental generation.
func writeL2Fields(b *strings.Builder, e parser.Entry) {
	l2Fields := []string{"purpose", "pre", "post", "errors", "effects", "thread_safety", "complexity"}
	for _, name := range l2Fields {
		if field, ok := e.Fields[name]; ok {
			if field.InlineValue != "" {
				b.WriteString(fmt.Sprintf("@%s %s\n", name, field.InlineValue))
			} else if len(field.Lines) > 0 {
				b.WriteString(fmt.Sprintf("@%s\n", name))
				for _, line := range field.Lines {
					b.WriteString(fmt.Sprintf("  %s\n", line))
				}
			}
		}
	}
}
