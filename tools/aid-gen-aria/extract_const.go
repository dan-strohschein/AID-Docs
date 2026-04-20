package main

import (
	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// extractConst converts a ConstDecl to a ConstEntry. The @value field is
// intentionally left empty — rendering arbitrary Aria initialiser expressions
// as strings is out of scope for the mechanical L1 extractor; a future L2
// pass can fill this in from source.
func extractConst(d *parser.ConstDecl, filePath string, docs *sourceIndex) ConstEntry {
	e := ConstEntry{Name: d.Name}
	if d.Type != nil {
		e.Type = AriaTypeToAID(d.Type)
	}
	if c := docs.commentAbove(d.Pos.File, d.Pos.Line); c != "" {
		e.Purpose = firstSentence(c)
	}
	_ = filePath // @source_file/@source_line live on fn/type entries; consts omit
	return e
}
