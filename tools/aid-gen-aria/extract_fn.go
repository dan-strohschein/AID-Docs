package main

import (
	"fmt"
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// extractFn builds an FnEntry from an Aria FnDecl. qualifier is the owning
// type name for methods (e.g. "Checker") or "" for free functions.
// docs is the source index used to pull leading-// @purpose comments.
func extractFn(fn *parser.FnDecl, qualifier, filePath string, docs *sourceIndex) FnEntry {
	name := fn.Name
	if qualifier != "" {
		name = qualifier + "." + fn.Name
	}

	e := FnEntry{
		Name:       name,
		Sigs:       []string{buildFnSig(fn, qualifier)},
		Params:     buildFnParams(fn),
		Returns:    aidReturn(fn.ReturnType),
		Errors:     aidErrors(fn.ErrorTypes),
		Effects:    append([]string{}, fn.Effects...),
		Calls:      extractCalls(fn, qualifier),
		SourceFile: filePath,
		SourceLine: fn.Pos.Line,
	}
	if comment := docs.commentAbove(fn.Pos.File, fn.Pos.Line); comment != "" {
		e.Purpose = firstSentence(comment)
	}
	return e
}

// buildFnSig renders the callable shape in a form that matches Aria source
// syntax so a reader can cross-reference directly:
//
//	(p: T, q: U) -> R ! E with [Io]
//
// Returns, errors, and effects are intentionally duplicated into their own
// AID fields (@returns, @errors, @effects); keeping them in @sig aids
// one-line scanning without round-tripping to the sidecar fields.
func buildFnSig(fn *parser.FnDecl, qualifier string) string {
	var b strings.Builder

	if len(fn.GenericParams) > 0 {
		gs := make([]string, len(fn.GenericParams))
		for i, g := range fn.GenericParams {
			if len(g.Bounds) > 0 {
				gs[i] = g.Name + ": " + strings.Join(g.Bounds, " + ")
			} else {
				gs[i] = g.Name
			}
		}
		fmt.Fprintf(&b, "[%s]", strings.Join(gs, ", "))
	}

	b.WriteByte('(')
	parts := make([]string, 0, len(fn.Params))
	// A method with an implicit self/mut-self first param is represented in
	// Aria with explicit `self` / `mut self` tokens; the parser surfaces them
	// as normal Params, so we preserve them verbatim.
	for _, p := range fn.Params {
		parts = append(parts, renderParam(p))
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteByte(')')

	if fn.ReturnType != nil {
		fmt.Fprintf(&b, " -> %s", AriaTypeToAID(fn.ReturnType))
	}

	if len(fn.ErrorTypes) > 0 {
		errs := make([]string, len(fn.ErrorTypes))
		for i, et := range fn.ErrorTypes {
			errs[i] = AriaTypeToAID(et)
		}
		fmt.Fprintf(&b, " ! %s", strings.Join(errs, " | "))
	}

	if len(fn.Effects) > 0 {
		fmt.Fprintf(&b, " with [%s]", strings.Join(fn.Effects, ", "))
	}

	_ = qualifier // reserved for future method-receiver prefixing in @sig
	return b.String()
}

func renderParam(p *parser.Param) string {
	name := p.Name
	if p.Mutable {
		name = "mut " + name
	}
	if p.Type == nil {
		return name
	}
	return fmt.Sprintf("%s: %s", name, AriaTypeToAID(p.Type))
}

func buildFnParams(fn *parser.FnDecl) []Param {
	out := make([]Param, 0, len(fn.Params))
	for _, p := range fn.Params {
		out = append(out, Param{
			Name: p.Name,
			Type: AriaTypeToAID(p.Type),
		})
	}
	return out
}

func aidReturn(t parser.TypeExpr) string {
	if t == nil {
		return ""
	}
	return AriaTypeToAID(t)
}

// extractFnMinimal produces a stripped FnEntry with only the fields needed
// for call-graph analysis: name, sig, calls, source position. Used for
// transitive-closure backfill of private callees when -internal is false.
func extractFnMinimal(fn *parser.FnDecl, qualifier, filePath string) FnEntry {
	name := fn.Name
	if qualifier != "" {
		name = qualifier + "." + fn.Name
	}
	return FnEntry{
		Name:       name,
		Sigs:       []string{buildFnSig(fn, qualifier)},
		Calls:      extractCalls(fn, qualifier),
		SourceFile: filePath,
		SourceLine: fn.Pos.Line,
	}
}

func aidErrors(ts []parser.TypeExpr) []string {
	if len(ts) == 0 {
		return nil
	}
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = AriaTypeToAID(t)
	}
	return out
}
