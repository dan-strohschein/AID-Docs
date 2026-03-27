package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Parser states per spec §7.3
type parserState int

const (
	stateHeader     parserState = iota
	stateEntry
	stateFieldValue
	stateDone
)

// Entry-starting field names — these start new blocks when seen in stateEntry
var entryKinds = map[string]string{
	"fn":       "fn",
	"type":     "type",
	"trait":    "trait",
	"const":   "const",
	"workflow": "workflow",
}

// Annotation-starting field names (module-level Tier 2.5 blocks)
var annotationKinds = map[string]bool{
	"invariants":  true,
	"antipatterns": true,
	"decision":    true,
	"note":        true,
}

// Manifest marker
const manifestField = "manifest"

// Source reference pattern: [src: file:LINE] or [src: file:START-END]
var srcRefPattern = regexp.MustCompile(`\[src:\s*([^\]]+)\]`)
var lineRefPattern = regexp.MustCompile(`^(.+?):(\d+)(?:-(\d+))?$`)

// ParseFile reads and parses an AID file from disk.
func ParseFile(path string) (*AidFile, []Warning, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	return Parse(f)
}

// ParseString parses an AID document from a string.
func ParseString(content string) (*AidFile, []Warning, error) {
	return Parse(strings.NewReader(content))
}

// Parse reads and parses an AID document from a reader.
func Parse(r io.Reader) (*AidFile, []Warning, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

	result := &AidFile{
		Header: Header{
			Extra: make(map[string]string),
		},
	}
	var warnings []Warning
	var state parserState = stateHeader

	var currentEntry *Entry
	var currentWorkflow *Workflow
	var currentAnnotation *Annotation
	var currentFieldName string
	lineNum := 0

	finishBlock := func() {
		if currentEntry != nil {
			result.Entries = append(result.Entries, *currentEntry)
			currentEntry = nil
		}
		if currentWorkflow != nil {
			result.Workflows = append(result.Workflows, *currentWorkflow)
			currentWorkflow = nil
		}
		if currentAnnotation != nil {
			result.Annotations = append(result.Annotations, *currentAnnotation)
			currentAnnotation = nil
		}
		currentFieldName = ""
	}

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()
		lineType, fieldName, value := ClassifyLine(rawLine)

		switch state {
		case stateHeader:
			switch lineType {
			case LineField:
				if fieldName == manifestField {
					result.IsManifest = true
				}
				setHeaderField(&result.Header, fieldName, value)
				currentFieldName = fieldName

			case LineContinuation:
				if currentFieldName != "" {
					appendHeaderField(&result.Header, currentFieldName, value)
				}

			case LineSeparator:
				state = stateEntry
				currentFieldName = ""

			case LineComment:
				result.Comments = append(result.Comments, value)

			case LineBlank:
				// skip
			}

		case stateEntry:
			switch lineType {
			case LineField:
				kind, isEntry := entryKinds[fieldName]
				isAnnotation := annotationKinds[fieldName]

				// Manifest entries use @package as the entry-starting field
				if result.IsManifest && fieldName == "package" {
					currentEntry = &Entry{
						Kind:   "package",
						Name:   value,
						Fields: make(map[string]Field),
					}
					currentFieldName = fieldName
					state = stateFieldValue
				} else if isAnnotation {
					// Module-level annotation block
					currentAnnotation = &Annotation{
						Kind:   fieldName,
						Name:   value,
						Fields: make(map[string]Field),
					}
					// For invariants/antipatterns, the content is continuation lines
					// For decision/note, the name is the value
					currentFieldName = fieldName
					state = stateFieldValue
				} else if isEntry {
					if kind == "workflow" {
						currentWorkflow = &Workflow{
							Name:   value,
							Fields: make(map[string]Field),
						}
						currentFieldName = fieldName
					} else {
						currentEntry = &Entry{
							Kind:   kind,
							Name:   value,
							Fields: make(map[string]Field),
						}
						currentFieldName = fieldName
					}
					state = stateFieldValue
				} else {
					// Field before entry declaration — warning
					warnings = append(warnings, Warning{
						Line:    lineNum,
						Message: fmt.Sprintf("field @%s before entry declaration", fieldName),
					})
				}

			case LineComment, LineBlank:
				// skip

			case LineSeparator:
				// Extra separator — skip

			case LineContinuation:
				warnings = append(warnings, Warning{
					Line:    lineNum,
					Message: "continuation line outside an entry",
				})
			}

		case stateFieldValue:
			switch lineType {
			case LineField:
				// In stateFieldValue, ALL @field lines are fields on the current
				// entry — not new entries. New entries only start after a separator
				// (which transitions to stateEntry first).
				currentFieldName = fieldName
				field := Field{
					Name:        fieldName,
					InlineValue: value,
					SourceRefs:  extractSourceRefs(value),
				}
				if currentEntry != nil {
					currentEntry.Fields[fieldName] = field
				} else if currentWorkflow != nil {
					currentWorkflow.Fields[fieldName] = field
				} else if currentAnnotation != nil {
					currentAnnotation.Fields[fieldName] = field
				}

			case LineContinuation:
				if currentFieldName != "" {
					appendBlockField(currentEntry, currentWorkflow, currentAnnotation, currentFieldName, value)
				}

			case LineSeparator:
				finishBlock()
				state = stateEntry

			case LineComment, LineBlank:
				// skip
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, warnings, fmt.Errorf("scan error: %w", err)
	}

	// Finalize any open entry
	finishBlock()

	return result, warnings, nil
}

func setHeaderField(h *Header, name, value string) {
	switch name {
	case "module":
		h.Module = value
	case "lang":
		h.Lang = value
	case "version":
		h.Version = value
	case "stability":
		h.Stability = value
	case "purpose":
		h.Purpose = value
	case "deps":
		h.Deps = parseList(value)
	case "depends":
		h.Depends = parseList(value)
	case "source":
		h.Source = value
	case "code_version":
		h.CodeVersion = value
	case "aid_status":
		h.AidStatus = value
	case "aid_generated_by":
		h.AidGeneratedBy = value
	case "aid_reviewed_by":
		h.AidReviewedBy = value
	case "aid_version":
		h.AidVersion = value
	default:
		h.Extra[name] = value
	}
}

func appendHeaderField(h *Header, name, value string) {
	switch name {
	case "purpose":
		h.Purpose += " " + value
	default:
		if existing, ok := h.Extra[name]; ok {
			h.Extra[name] = existing + "\n" + value
		}
	}
}

func appendBlockField(entry *Entry, workflow *Workflow, annotation *Annotation, fieldName, value string) {
	var fields map[string]Field
	if entry != nil {
		fields = entry.Fields
	} else if workflow != nil {
		fields = workflow.Fields
	} else if annotation != nil {
		fields = annotation.Fields
	}
	if fields == nil {
		return
	}

	field, exists := fields[fieldName]
	if !exists {
		field = Field{Name: fieldName}
	}
	field.Lines = append(field.Lines, value)

	// Extract source refs from continuation lines too
	refs := extractSourceRefs(value)
	field.SourceRefs = append(field.SourceRefs, refs...)

	fields[fieldName] = field
}

// parseList parses "[a, b, c]" or "a, b, c" into a string slice.
func parseList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// extractSourceRefs finds all [src: file:line] references in a string.
func extractSourceRefs(text string) []SourceRef {
	matches := srcRefPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var refs []SourceRef
	for _, match := range matches {
		// match[1] is the content inside [src: ...]
		// Could be "file:line" or "file:start-end" or "file:line, file2:line"
		parts := strings.Split(match[1], ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			sub := lineRefPattern.FindStringSubmatch(part)
			if sub == nil {
				continue
			}
			file := strings.TrimSpace(sub[1])
			startLine, _ := strconv.Atoi(sub[2])
			endLine := startLine
			if sub[3] != "" {
				endLine, _ = strconv.Atoi(sub[3])
			}
			refs = append(refs, SourceRef{
				File:      file,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
	}
	return refs
}
