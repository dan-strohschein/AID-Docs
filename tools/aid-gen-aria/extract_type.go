package main

import (
	"fmt"
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// extractTypeDecl converts a struct / sum / newtype declaration to a TypeEntry.
func extractTypeDecl(d *parser.TypeDecl, filePath string, docs *sourceIndex) TypeEntry {
	e := TypeEntry{
		Name:          d.Name,
		GenericParams: renderGenerics(d.GenericParams),
		SourceFile:    filePath,
		SourceLine:    d.Pos.Line,
	}
	if c := docs.commentAbove(d.Pos.File, d.Pos.Line); c != "" {
		e.Purpose = firstSentence(c)
	}
	if len(d.Derives) > 0 {
		e.Implements = append([]string{}, d.Derives...)
	}

	switch d.Kind {
	case parser.StructDecl:
		e.Kind = "struct"
		// All fields of an emitted struct are part of its documented shape.
		// Aria's field-level `pub` controls write access, not visibility in
		// documentation — hiding private fields would mislead consumers about
		// memory layout and pattern-match possibilities.
		for _, f := range d.Fields {
			e.Fields = append(e.Fields, Field{
				Name: f.Name,
				Type: AriaTypeToAID(f.Type),
			})
		}

	case parser.SumTypeDecl:
		e.Kind = "union"
		for _, v := range d.Variants {
			e.Variants = append(e.Variants, Variant{
				Name:    v.Name,
				Payload: renderVariantPayload(v),
			})
		}

	case parser.NewtypeDecl:
		e.Kind = "newtype"
		if d.Underlying != nil {
			// Express the wrapped type as a single synthetic field so readers
			// see what the newtype wraps without needing a separate field.
			e.Fields = append(e.Fields, Field{
				Name: "inner",
				Type: AriaTypeToAID(d.Underlying),
			})
		}
	}

	return e
}

func renderVariantPayload(v *parser.VariantDecl) string {
	if len(v.Fields) > 0 {
		parts := make([]string, len(v.Fields))
		for i, f := range v.Fields {
			parts[i] = fmt.Sprintf("%s: %s", f.Name, AriaTypeToAID(f.Type))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	if len(v.Types) > 0 {
		parts := make([]string, len(v.Types))
		for i, t := range v.Types {
			parts[i] = AriaTypeToAID(t)
		}
		return strings.Join(parts, ", ")
	}
	return ""
}

// extractEnum converts an enum (a set of untagged name-only members).
func extractEnum(d *parser.EnumDecl, filePath string, docs *sourceIndex) TypeEntry {
	e := TypeEntry{
		Name:       d.Name,
		Kind:       "enum",
		SourceFile: filePath,
		SourceLine: d.Pos.Line,
	}
	if c := docs.commentAbove(d.Pos.File, d.Pos.Line); c != "" {
		e.Purpose = firstSentence(c)
	}
	for _, m := range d.Members {
		e.Variants = append(e.Variants, Variant{Name: m})
	}
	return e
}

// extractAlias converts a type alias to a TypeEntry of kind "alias".
func extractAlias(d *parser.AliasDecl, filePath string, docs *sourceIndex) TypeEntry {
	e := TypeEntry{
		Name:       d.Name,
		Kind:       "alias",
		SourceFile: filePath,
		SourceLine: d.Pos.Line,
	}
	if c := docs.commentAbove(d.Pos.File, d.Pos.Line); c != "" {
		e.Purpose = firstSentence(c)
	}
	if d.Target != nil {
		e.Fields = append(e.Fields, Field{
			Name: "target",
			Type: AriaTypeToAID(d.Target),
		})
	}
	return e
}

func renderGenerics(gps []*parser.GenericParam) string {
	if len(gps) == 0 {
		return ""
	}
	parts := make([]string, len(gps))
	for i, g := range gps {
		if len(g.Bounds) > 0 {
			parts[i] = g.Name + ": " + strings.Join(g.Bounds, " + ")
		} else {
			parts[i] = g.Name
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
