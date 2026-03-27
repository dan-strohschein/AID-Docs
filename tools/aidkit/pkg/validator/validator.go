// Package validator checks AID files against spec rules.
// All checks produce warnings, not errors — a file with warnings is still usable.
package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dan-strohschein/aidkit/pkg/parser"
)

// Severity indicates how important a warning is.
type Severity int

const (
	SeverityInfo    Severity = iota // Nice to have
	SeverityWarning                 // Should fix
	SeverityError                   // Must fix for correctness
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARN"
	case SeverityError:
		return "ERROR"
	}
	return "UNKNOWN"
}

// Issue is a single validation finding.
type Issue struct {
	Rule     string
	Severity Severity
	Entry    string // Entry name (empty for header issues)
	Field    string // Field name (empty for entry-level issues)
	Message  string
}

func (i Issue) String() string {
	loc := ""
	if i.Entry != "" {
		loc = i.Entry
		if i.Field != "" {
			loc += "." + i.Field
		}
		loc += ": "
	}
	return fmt.Sprintf("[%s] %s%s (%s)", i.Severity, loc, i.Message, i.Rule)
}

// Rule is the interface for a single validation check.
type Rule interface {
	Name() string
	Check(file *parser.AidFile) []Issue
}

// Validate runs all rules against a parsed AID file.
func Validate(file *parser.AidFile) []Issue {
	rules := AllRules()
	var issues []Issue
	for _, rule := range rules {
		issues = append(issues, rule.Check(file)...)
	}
	return issues
}

// AllRules returns all built-in validation rules.
func AllRules() []Rule {
	return []Rule{
		&HeaderCompleteRule{},
		&RequiredFieldsRule{},
		&MethodBindingRule{},
		&CrossReferencesRule{},
		&DecisionFieldsRule{},
		&ManifestFieldsRule{},
		&SourceRefFormatRule{},
		&StatusValidRule{},
		&CodeVersionFormatRule{},
	}
}

// --- Rule implementations ---

// HeaderCompleteRule checks that required header fields are present.
type HeaderCompleteRule struct{}

func (r *HeaderCompleteRule) Name() string { return "header-complete" }
func (r *HeaderCompleteRule) Check(file *parser.AidFile) []Issue {
	// Manifest files use @manifest + @project instead of @module + @lang
	if file.IsManifest {
		return nil
	}
	var issues []Issue
	if file.Header.Module == "" {
		issues = append(issues, Issue{
			Rule: r.Name(), Severity: SeverityError,
			Message: "@module is required",
		})
	}
	if file.Header.Lang == "" {
		issues = append(issues, Issue{
			Rule: r.Name(), Severity: SeverityError,
			Message: "@lang is required",
		})
	}
	if file.Header.Version == "" {
		issues = append(issues, Issue{
			Rule: r.Name(), Severity: SeverityWarning,
			Message: "@version is missing",
		})
	}
	return issues
}

// RequiredFieldsRule checks entry-level required fields.
type RequiredFieldsRule struct{}

func (r *RequiredFieldsRule) Name() string { return "required-fields" }
func (r *RequiredFieldsRule) Check(file *parser.AidFile) []Issue {
	var issues []Issue
	for _, e := range file.Entries {
		// @purpose required on all entries
		if _, has := e.Fields["purpose"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: e.Name, Message: "@purpose is missing",
			})
		}

		switch e.Kind {
		case "fn":
			if _, has := e.Fields["sig"]; !has {
				issues = append(issues, Issue{
					Rule: r.Name(), Severity: SeverityError,
					Entry: e.Name, Message: "@sig is required for @fn entries",
				})
			}

		case "type":
			kind := ""
			if k, has := e.Fields["kind"]; has {
				kind = k.InlineValue
			} else {
				issues = append(issues, Issue{
					Rule: r.Name(), Severity: SeverityError,
					Entry: e.Name, Message: "@kind is required for @type entries",
				})
			}
			// struct/class need @fields, enum/union need @variants
			switch kind {
			case "struct", "class":
				if _, has := e.Fields["fields"]; !has {
					issues = append(issues, Issue{
						Rule: r.Name(), Severity: SeverityWarning,
						Entry: e.Name, Message: "@fields is expected for struct/class types",
					})
				}
			case "enum", "union":
				if _, has := e.Fields["variants"]; !has {
					issues = append(issues, Issue{
						Rule: r.Name(), Severity: SeverityWarning,
						Entry: e.Name, Message: "@variants is expected for enum/union types",
					})
				}
			}

		case "trait":
			if _, has := e.Fields["requires"]; !has {
				issues = append(issues, Issue{
					Rule: r.Name(), Severity: SeverityWarning,
					Entry: e.Name, Message: "@requires is expected for @trait entries",
				})
			}
		}
	}

	// Workflows need @purpose and @steps
	for _, w := range file.Workflows {
		if _, has := w.Fields["purpose"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: "workflow:" + w.Name, Message: "@purpose is missing",
			})
		}
		if _, has := w.Fields["steps"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: "workflow:" + w.Name, Message: "@steps is required for @workflow entries",
			})
		}
	}
	return issues
}

// MethodBindingRule checks that @fn Type.method has a corresponding @type Type.
type MethodBindingRule struct{}

func (r *MethodBindingRule) Name() string { return "method-binding" }
func (r *MethodBindingRule) Check(file *parser.AidFile) []Issue {
	// Collect type names
	typeNames := map[string]bool{}
	typeMethods := map[string]map[string]bool{} // type → set of methods listed in @methods

	for _, e := range file.Entries {
		if e.Kind == "type" || e.Kind == "trait" {
			typeNames[e.Name] = true
			if methods, has := e.Fields["methods"]; has {
				methodSet := map[string]bool{}
				for _, m := range strings.Split(methods.InlineValue, ",") {
					m = strings.TrimSpace(m)
					if m != "" {
						methodSet[m] = true
					}
				}
				typeMethods[e.Name] = methodSet
			}
		}
	}

	var issues []Issue
	for _, e := range file.Entries {
		if e.Kind != "fn" {
			continue
		}
		dotIdx := strings.Index(e.Name, ".")
		if dotIdx < 0 {
			continue // Not a method
		}
		typeName := e.Name[:dotIdx]
		methodName := e.Name[dotIdx+1:]

		// Check type exists
		if !typeNames[typeName] {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: e.Name,
				Message: fmt.Sprintf("method on type %q but no @type %s entry exists", typeName, typeName),
			})
			continue
		}

		// Check method is listed in @methods
		if methods, has := typeMethods[typeName]; has {
			if !methods[methodName] {
				issues = append(issues, Issue{
					Rule: r.Name(), Severity: SeverityInfo,
					Entry: e.Name,
					Message: fmt.Sprintf("method %q not listed in @type %s @methods", methodName, typeName),
				})
			}
		}
	}
	return issues
}

// CrossReferencesRule checks that @related names resolve to entries in this file.
type CrossReferencesRule struct{}

func (r *CrossReferencesRule) Name() string { return "cross-references" }
func (r *CrossReferencesRule) Check(file *parser.AidFile) []Issue {
	// Collect all entry names
	names := map[string]bool{}
	for _, e := range file.Entries {
		names[e.Name] = true
	}
	for _, w := range file.Workflows {
		names[w.Name] = true
	}

	var issues []Issue
	for _, e := range file.Entries {
		related, has := e.Fields["related"]
		if !has {
			continue
		}
		refs := strings.Split(related.InlineValue, ",")
		for _, ref := range refs {
			ref = strings.TrimSpace(ref)
			if ref == "" {
				continue
			}
			// Cross-module refs (contain /) are not checked here
			if strings.Contains(ref, "/") {
				continue
			}
			if !names[ref] {
				issues = append(issues, Issue{
					Rule: r.Name(), Severity: SeverityInfo,
					Entry: e.Name, Field: "related",
					Message: fmt.Sprintf("@related reference %q not found in this file", ref),
				})
			}
		}
	}
	return issues
}

// SourceRefFormatRule checks that [src:] references have valid format.
type SourceRefFormatRule struct{}

var validSrcRefPattern = regexp.MustCompile(`^[a-zA-Z0-9_./-]+:\d+(-\d+)?$`)

func (r *SourceRefFormatRule) Name() string { return "source-ref-format" }
func (r *SourceRefFormatRule) Check(file *parser.AidFile) []Issue {
	var issues []Issue

	checkFields := func(entryName string, fields map[string]parser.Field) {
		for _, field := range fields {
			// Check all [src:] references in field text
			fullText := field.Value()
			matches := regexp.MustCompile(`\[src:\s*([^\]]+)\]`).FindAllStringSubmatch(fullText, -1)
			for _, match := range matches {
				parts := strings.Split(match[1], ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if !validSrcRefPattern.MatchString(part) {
						issues = append(issues, Issue{
							Rule: r.Name(), Severity: SeverityWarning,
							Entry: entryName, Field: field.Name,
							Message: fmt.Sprintf("malformed source reference: %q", part),
						})
					}
				}
			}
		}
	}

	for _, e := range file.Entries {
		checkFields(e.Name, e.Fields)
	}
	for _, w := range file.Workflows {
		checkFields("workflow:"+w.Name, w.Fields)
	}
	return issues
}

// StatusValidRule checks @aid_status is a recognized value.
type StatusValidRule struct{}

var validStatuses = map[string]bool{
	"draft": true, "reviewed": true, "approved": true, "stale": true,
}

func (r *StatusValidRule) Name() string { return "status-valid" }
func (r *StatusValidRule) Check(file *parser.AidFile) []Issue {
	if file.Header.AidStatus == "" {
		return nil // Optional field
	}
	if !validStatuses[file.Header.AidStatus] {
		return []Issue{{
			Rule: r.Name(), Severity: SeverityWarning,
			Message: fmt.Sprintf("@aid_status %q is not one of: draft, reviewed, approved, stale", file.Header.AidStatus),
		}}
	}
	return nil
}

// CodeVersionFormatRule checks @code_version format.
type CodeVersionFormatRule struct{}

var codeVersionPattern = regexp.MustCompile(`^git:[a-f0-9]{7,40}$`)

func (r *CodeVersionFormatRule) Name() string { return "code-version-format" }
func (r *CodeVersionFormatRule) Check(file *parser.AidFile) []Issue {
	if file.Header.CodeVersion == "" {
		return nil // Optional field
	}
	if !codeVersionPattern.MatchString(file.Header.CodeVersion) {
		return []Issue{{
			Rule: r.Name(), Severity: SeverityWarning,
			Message: fmt.Sprintf("@code_version %q doesn't match format git:HASH", file.Header.CodeVersion),
		}}
	}
	return nil
}

// DecisionFieldsRule checks @decision blocks have required fields.
type DecisionFieldsRule struct{}

func (r *DecisionFieldsRule) Name() string { return "decision-fields" }
func (r *DecisionFieldsRule) Check(file *parser.AidFile) []Issue {
	var issues []Issue
	for _, a := range file.Annotations {
		if a.Kind != "decision" {
			continue
		}
		name := "decision:" + a.Name
		if _, has := a.Fields["purpose"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: name, Message: "@purpose is required for @decision blocks",
			})
		}
		if _, has := a.Fields["chosen"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: name, Message: "@chosen is required for @decision blocks",
			})
		}
		if _, has := a.Fields["rationale"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: name, Message: "@rationale is required for @decision blocks",
			})
		}
	}
	return issues
}

// ManifestFieldsRule checks manifest entries have required fields.
type ManifestFieldsRule struct{}

func (r *ManifestFieldsRule) Name() string { return "manifest-fields" }
func (r *ManifestFieldsRule) Check(file *parser.AidFile) []Issue {
	if !file.IsManifest {
		return nil
	}
	var issues []Issue
	for _, e := range file.Entries {
		if e.Kind != "package" {
			continue
		}
		name := "package:" + e.Name
		if _, has := e.Fields["aid_file"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityError,
				Entry: name, Message: "@aid_file is required in manifest entries",
			})
		}
		if _, has := e.Fields["purpose"]; !has {
			issues = append(issues, Issue{
				Rule: r.Name(), Severity: SeverityWarning,
				Entry: name, Message: "@purpose is recommended in manifest entries",
			})
		}
	}
	return issues
}
