package parser

import "strings"

// LineType classifies a line in an AID file.
type LineType int

const (
	LineField        LineType = iota // Starts with @
	LineContinuation                 // Starts with 2+ spaces (no @)
	LineSeparator                    // Exactly "---"
	LineComment                      // Starts with //
	LineBlank                        // Empty or whitespace only
)

// ClassifyLine determines the type of a line and extracts the field name
// and remaining value for Field lines.
// Returns (lineType, fieldName, remainingValue).
// For non-Field lines, fieldName and remainingValue are empty.
func ClassifyLine(line string) (LineType, string, string) {
	trimmed := strings.TrimRight(line, " \t\r\n")

	// Blank
	if trimmed == "" {
		return LineBlank, "", ""
	}

	// Separator
	if trimmed == "---" {
		return LineSeparator, "", ""
	}

	// Comment
	if strings.HasPrefix(trimmed, "//") {
		return LineComment, "", trimmed
	}

	// Field
	if strings.HasPrefix(trimmed, "@") {
		rest := trimmed[1:] // strip @
		spaceIdx := strings.IndexByte(rest, ' ')
		if spaceIdx < 0 {
			// @field with no value
			return LineField, rest, ""
		}
		fieldName := rest[:spaceIdx]
		value := strings.TrimSpace(rest[spaceIdx+1:])
		return LineField, fieldName, value
	}

	// Continuation — starts with 2+ spaces
	if len(line) >= 2 && line[0] == ' ' && line[1] == ' ' {
		// Strip first 2 spaces, preserve rest
		content := line[2:]
		return LineContinuation, "", content
	}

	// Fallback — treat as continuation if it starts with any whitespace
	if line[0] == ' ' || line[0] == '\t' {
		content := strings.TrimLeft(line, " \t")
		return LineContinuation, "", content
	}

	// Unknown — treat as continuation
	return LineContinuation, "", trimmed
}
