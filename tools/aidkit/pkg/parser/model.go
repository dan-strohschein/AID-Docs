// Package parser implements the AID file parser as specified in format.md §7.
package parser

import "fmt"

// AidFile is the parsed representation of a complete .aid document.
type AidFile struct {
	Header    Header
	Entries   []Entry
	Workflows []Workflow
	Comments  []string // Top-level comments (provenance markers, etc.)
}

// Header contains module-level metadata from the first section.
type Header struct {
	Module         string
	Lang           string
	Version        string
	Stability      string
	Purpose        string
	Deps           []string
	Depends        []string // @depends — packages for selective AID loading
	Source         string
	CodeVersion    string // @code_version git:HASH
	AidStatus      string // draft | reviewed | approved | stale
	AidGeneratedBy string
	AidReviewedBy  string
	AidVersion     string
	Extra          map[string]string // Unknown fields (forward compatibility)
}

// Entry represents a single API entry: @fn, @type, @trait, or @const.
type Entry struct {
	Kind   string           // "fn", "type", "trait", "const"
	Name   string           // Entry name from the opening field value
	Fields map[string]Field // All fields on this entry, keyed by field name
}

// Field holds a single @field and its value(s).
type Field struct {
	Name        string
	InlineValue string      // Value on the same line as @field
	Lines       []string    // Multi-line continuation values
	SourceRefs  []SourceRef // Extracted [src:] references
}

// Value returns the full field value — inline value plus any continuation lines.
func (f Field) Value() string {
	if len(f.Lines) == 0 {
		return f.InlineValue
	}
	result := f.InlineValue
	for _, line := range f.Lines {
		if result != "" {
			result += "\n"
		}
		result += line
	}
	return result
}

// SourceRef is a parsed [src: file:line] reference linking a claim to code.
type SourceRef struct {
	File      string
	StartLine int
	EndLine   int // Same as StartLine for single-line refs
}

func (s SourceRef) String() string {
	if s.StartLine == s.EndLine {
		return fmt.Sprintf("[src: %s:%d]", s.File, s.StartLine)
	}
	return fmt.Sprintf("[src: %s:%d-%d]", s.File, s.StartLine, s.EndLine)
}

// Workflow represents a @workflow block.
type Workflow struct {
	Name   string
	Fields map[string]Field
}

// Warning represents a non-fatal issue found during parsing or validation.
type Warning struct {
	Line    int
	Message string
}

func (w Warning) String() string {
	if w.Line > 0 {
		return fmt.Sprintf("line %d: %s", w.Line, w.Message)
	}
	return w.Message
}
