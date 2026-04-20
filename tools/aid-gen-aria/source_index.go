package main

import (
	"os"
	"path/filepath"
	"strings"
)

// sourceIndex is a per-file map of line number → trailing leading comment
// block. Aria's lexer strips comments, so @purpose extraction is done by
// reading the raw source and pairing each declaration's start line with the
// contiguous run of `//` lines immediately above it.
type sourceIndex struct {
	// leadingComment[absPath][declLine] = joined comment text
	leadingComment map[string]map[int]string
}

func newSourceIndex() *sourceIndex {
	return &sourceIndex{leadingComment: map[string]map[int]string{}}
}

// loadFile reads an Aria source file and precomputes, for every line that
// has a contiguous `//` run above it, the joined comment text.
func (s *sourceIndex) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	lines := strings.Split(string(data), "\n")
	m := map[int]string{}

	// lineComment[i] = the // text on line i+1 (trimmed of "// ") or "" if none.
	// Blank lines also produce "" which breaks the block.
	commentAt := make([]string, len(lines))
	for i, raw := range lines {
		trim := strings.TrimSpace(raw)
		if strings.HasPrefix(trim, "//") {
			commentAt[i] = strings.TrimSpace(strings.TrimPrefix(trim, "//"))
		} else if trim == "" {
			commentAt[i] = "" // blank — block break
		} else {
			commentAt[i] = "\x00" // non-comment, non-blank → no comment
		}
	}

	// For each non-comment/non-blank line, walk upward collecting contiguous
	// // lines (blank lines break the block).
	for i, raw := range lines {
		if commentAt[i] != "\x00" {
			continue // only index lines that hold real code
		}
		// Skip leading whitespace-only mapping: declaration lines we care about
		// are ones that contain code.
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var block []string
		for j := i - 1; j >= 0; j-- {
			c := commentAt[j]
			if c == "\x00" || c == "" {
				break
			}
			block = append([]string{c}, block...)
		}
		if len(block) > 0 {
			// Aria Pos is 1-based.
			m[i+1] = strings.Join(block, " ")
		}
	}

	s.leadingComment[abs] = m
	return nil
}

// commentAbove returns the joined leading-comment block immediately above the
// given source position, or "" if none exists / the file wasn't indexed.
func (s *sourceIndex) commentAbove(file string, line int) string {
	abs, err := filepath.Abs(file)
	if err != nil {
		abs = file
	}
	if m, ok := s.leadingComment[abs]; ok {
		return m[line]
	}
	return ""
}

// firstSentence returns the first sentence of a doc block, truncated at 120
// chars. Matches aid-gen-go's behaviour so both generators produce comparable
// @purpose fields.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, ". "); i >= 0 {
		s = s[:i+1]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}
