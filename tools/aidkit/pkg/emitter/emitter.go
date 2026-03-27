// Package emitter converts parsed AID data structures back to .aid text format.
package emitter

import (
	"fmt"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// Emit converts an AidFile to .aid formatted text.
func Emit(f *parser.AidFile) string {
	var b strings.Builder

	for _, c := range f.Comments {
		b.WriteString(c + "\n")
	}
	if len(f.Comments) > 0 {
		b.WriteString("\n")
	}

	emitHeader(&b, &f.Header, f.IsManifest)

	for _, e := range f.Entries {
		b.WriteString("\n---\n\n")
		emitEntry(&b, &e)
	}

	for _, a := range f.Annotations {
		b.WriteString("\n---\n\n")
		emitAnnotation(&b, &a)
	}

	for _, w := range f.Workflows {
		b.WriteString("\n---\n\n")
		emitWorkflow(&b, &w)
	}

	return b.String()
}

func emitHeader(b *strings.Builder, h *parser.Header, isManifest bool) {
	if isManifest {
		b.WriteString("@manifest\n")
	}
	if h.Module != "" {
		fmt.Fprintf(b, "@module %s\n", h.Module)
	}
	if h.Lang != "" {
		fmt.Fprintf(b, "@lang %s\n", h.Lang)
	}
	if h.Version != "" {
		fmt.Fprintf(b, "@version %s\n", h.Version)
	}
	if h.Stability != "" {
		fmt.Fprintf(b, "@stability %s\n", h.Stability)
	}
	if h.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", h.Purpose)
	}
	if len(h.Deps) > 0 {
		fmt.Fprintf(b, "@deps [%s]\n", strings.Join(h.Deps, ", "))
	}
	if len(h.Depends) > 0 {
		fmt.Fprintf(b, "@depends [%s]\n", strings.Join(h.Depends, ", "))
	}
	if h.Source != "" {
		fmt.Fprintf(b, "@source %s\n", h.Source)
	}
	if h.CodeVersion != "" {
		fmt.Fprintf(b, "@code_version %s\n", h.CodeVersion)
	}
	if h.AidStatus != "" {
		fmt.Fprintf(b, "@aid_status %s\n", h.AidStatus)
	}
	if h.AidGeneratedBy != "" {
		fmt.Fprintf(b, "@aid_generated_by %s\n", h.AidGeneratedBy)
	}
	if h.AidReviewedBy != "" {
		fmt.Fprintf(b, "@aid_reviewed_by %s\n", h.AidReviewedBy)
	}
	if h.AidVersion != "" {
		fmt.Fprintf(b, "@aid_version %s\n", h.AidVersion)
	}
	for k, v := range h.Extra {
		fmt.Fprintf(b, "@%s %s\n", k, v)
	}
}

func emitEntry(b *strings.Builder, e *parser.Entry) {
	fmt.Fprintf(b, "@%s %s\n", e.Kind, e.Name)
	emitFields(b, e.Fields, e.Kind)
}

func emitAnnotation(b *strings.Builder, a *parser.Annotation) {
	if a.Name != "" {
		fmt.Fprintf(b, "@%s %s\n", a.Kind, a.Name)
	} else {
		fmt.Fprintf(b, "@%s\n", a.Kind)
	}

	// For block-style annotations (invariants, antipatterns), the content
	// is stored under a field with the same key as the kind. Emit those
	// continuation lines first, then any other fields.
	if blockField, has := a.Fields[a.Kind]; has {
		for _, line := range blockField.Lines {
			fmt.Fprintf(b, "  %s\n", line)
		}
	}

	// Emit remaining fields (purpose, chosen, rationale, etc.)
	emitFields(b, a.Fields, a.Kind)
}

func emitWorkflow(b *strings.Builder, w *parser.Workflow) {
	fmt.Fprintf(b, "@workflow %s\n", w.Name)
	emitFields(b, w.Fields, "workflow")
}

func emitFields(b *strings.Builder, fields map[string]parser.Field, skipKey string) {
	// Emit fields in a stable order: known important fields first, then alphabetical
	order := fieldOrder(fields, skipKey)
	for _, name := range order {
		field := fields[name]
		if field.InlineValue != "" {
			fmt.Fprintf(b, "@%s %s\n", name, field.InlineValue)
		} else if len(field.Lines) > 0 {
			fmt.Fprintf(b, "@%s\n", name)
		}
		for _, line := range field.Lines {
			fmt.Fprintf(b, "  %s\n", line)
		}
	}
}

// fieldOrder returns field names in a sensible emit order.
func fieldOrder(fields map[string]parser.Field, skipKey string) []string {
	// Priority fields in preferred order
	priority := []string{
		"purpose", "kind", "generic_params", "extends",
		"sig", "params", "returns", "errors",
		"fields", "variants", "invariants",
		"constructors", "methods", "implements",
		"requires", "provided", "implementors",
		"pre", "post", "effects", "thread_safety", "complexity",
		"steps", "errors_at", "antipatterns",
		"context", "chosen", "rejected", "rationale", "tradeoff",
		"since", "deprecated", "related", "platform",
		"aid_file", "aid_status", "depends", "layer", "key_risks",
		"type", "value",
		"example",
	}

	seen := map[string]bool{skipKey: true}
	var result []string

	for _, name := range priority {
		if _, has := fields[name]; has && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}

	// Remaining fields alphabetically
	for name := range fields {
		if !seen[name] {
			result = append(result, name)
		}
	}

	return result
}
