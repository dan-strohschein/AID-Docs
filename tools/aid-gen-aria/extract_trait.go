package main

import (
	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// extractTrait converts a TraitDecl to a TraitEntry and its required-method
// signatures. Methods declared inside the trait are emitted as additional
// FnEntry items (namespaced as Trait.method) so they appear in the ordered
// output alongside the trait block.
func extractTrait(d *parser.TraitDecl, filePath string, docs *sourceIndex) (TraitEntry, []FnEntry) {
	t := TraitEntry{
		Name:    d.Name,
		Extends: append([]string{}, d.Supertraits...),
	}
	if c := docs.commentAbove(d.Pos.File, d.Pos.Line); c != "" {
		t.Purpose = firstSentence(c)
	}

	var methods []FnEntry
	for _, m := range d.Methods {
		fe := extractFn(m, d.Name, filePath, docs)
		methods = append(methods, fe)
		t.Requires = append(t.Requires, buildFnSig(m, d.Name))
	}

	return t, methods
}

// extractImpl produces FnEntry items for every method in an impl block,
// namespaced as TypeName.method. Inherent impls (TraitName == "") and trait
// impls are handled identically from AID's perspective — both are methods
// attached to TypeName. The trait-implementation relationship is recorded by
// the caller on the target TypeEntry via .Implements.
func extractImpl(d *parser.ImplDecl, filePath string, docs *sourceIndex) []FnEntry {
	var out []FnEntry
	for _, m := range d.Methods {
		out = append(out, extractFn(m, d.TypeName, filePath, docs))
	}
	return out
}
