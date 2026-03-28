package main

import (
	"fmt"
	"strings"
)

// Emit converts an AidFile to .aid formatted text.
func Emit(f *AidFile) string {
	var b strings.Builder

	b.WriteString("// [generated] Layer 1 mechanical extraction — not yet reviewed\n\n")
	emitHeader(&b, &f.Header)

	ordered := orderEntries(f.Entries)
	for _, e := range ordered {
		b.WriteString("\n---\n\n")
		switch v := e.(type) {
		case FnEntry:
			emitFn(&b, &v)
		case TypeEntry:
			emitType(&b, &v)
		case TraitEntry:
			emitTrait(&b, &v)
		case ConstEntry:
			emitConst(&b, &v)
		}
	}

	return b.String()
}

func emitHeader(b *strings.Builder, h *ModuleHeader) {
	fmt.Fprintf(b, "@module %s\n", h.Module)
	fmt.Fprintf(b, "@lang %s\n", h.Lang)
	fmt.Fprintf(b, "@version %s\n", h.Version)
	if h.Stability != "" {
		fmt.Fprintf(b, "@stability %s\n", h.Stability)
	}
	if h.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", h.Purpose)
	}
	if len(h.Deps) > 0 {
		fmt.Fprintf(b, "@deps [%s]\n", strings.Join(h.Deps, ", "))
	}
	if h.Source != "" {
		fmt.Fprintf(b, "@source %s\n", h.Source)
	}
	fmt.Fprintf(b, "@aid_version %s\n", h.AidVersion)
}

func emitFn(b *strings.Builder, e *FnEntry) {
	fmt.Fprintf(b, "@fn %s\n", e.Name)
	if e.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", e.Purpose)
	}
	for _, sig := range e.Sigs {
		fmt.Fprintf(b, "@sig %s\n", sig)
	}
	if len(e.Params) > 0 {
		b.WriteString("@params\n")
		for _, p := range e.Params {
			emitParam(b, &p, 2)
		}
	}
	if e.Returns != "" && !isRedundantReturns(e) {
		fmt.Fprintf(b, "@returns %s\n", e.Returns)
	}
	if len(e.Errors) > 0 {
		b.WriteString("@errors\n")
		for _, err := range e.Errors {
			fmt.Fprintf(b, "  %s\n", err)
		}
	}
	if e.Pre != "" {
		fmt.Fprintf(b, "@pre %s\n", e.Pre)
	}
	if e.Post != "" {
		fmt.Fprintf(b, "@post %s\n", e.Post)
	}
	if len(e.Calls) > 0 {
		fmt.Fprintf(b, "@calls [%s]\n", strings.Join(e.Calls, ", "))
	}
	if e.SourceFile != "" {
		fmt.Fprintf(b, "@source_file %s\n", e.SourceFile)
	}
	if e.SourceLine > 0 {
		fmt.Fprintf(b, "@source_line %d\n", e.SourceLine)
	}
	if len(e.Effects) > 0 {
		fmt.Fprintf(b, "@effects [%s]\n", strings.Join(e.Effects, ", "))
	}
	if e.ThreadSafe != "" {
		fmt.Fprintf(b, "@thread_safety %s\n", e.ThreadSafe)
	}
	if e.Deprecated != "" {
		fmt.Fprintf(b, "@deprecated %s\n", e.Deprecated)
	}
	if len(e.Related) > 0 {
		fmt.Fprintf(b, "@related %s\n", strings.Join(e.Related, ", "))
	}
}

func emitParam(b *strings.Builder, p *Param, indent int) {
	prefix := strings.Repeat(" ", indent)
	parts := []string{}
	if p.Type != "" {
		parts = append(parts, p.Type)
	}
	if p.Desc != "" {
		parts = append(parts, p.Desc)
	}
	if p.Default != "" {
		parts = append(parts, "Default "+p.Default+".")
	}

	name := p.Name
	if p.Variadic {
		name = "..." + name
	}

	detail := strings.Join(parts, " — ")
	if detail != "" {
		fmt.Fprintf(b, "%s%s: %s\n", prefix, name, detail)
	} else {
		fmt.Fprintf(b, "%s%s:\n", prefix, name)
	}
}

func emitType(b *strings.Builder, e *TypeEntry) {
	fmt.Fprintf(b, "@type %s\n", e.Name)
	fmt.Fprintf(b, "@kind %s\n", e.Kind)
	if e.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", e.Purpose)
	}
	if e.GenericParams != "" {
		fmt.Fprintf(b, "@generic_params %s\n", e.GenericParams)
	}
	if len(e.Extends) > 0 {
		fmt.Fprintf(b, "@extends %s\n", strings.Join(e.Extends, ", "))
	}
	if len(e.Fields) > 0 {
		b.WriteString("@fields\n")
		for _, f := range e.Fields {
			if f.Desc != "" {
				fmt.Fprintf(b, "  %s: %s — %s\n", f.Name, f.Type, f.Desc)
			} else {
				fmt.Fprintf(b, "  %s: %s\n", f.Name, f.Type)
			}
		}
	}
	if len(e.Variants) > 0 {
		b.WriteString("@variants\n")
		for _, v := range e.Variants {
			payload := ""
			if v.Payload != "" {
				payload = "(" + v.Payload + ")"
			}
			desc := ""
			if v.Desc != "" {
				desc = " — " + v.Desc
			}
			fmt.Fprintf(b, "  | %s%s%s\n", v.Name, payload, desc)
		}
	}
	if e.Constructors != "" {
		fmt.Fprintf(b, "@constructors %s\n", e.Constructors)
	}
	if len(e.Methods) > 0 {
		fmt.Fprintf(b, "@methods %s\n", strings.Join(e.Methods, ", "))
	}
	if len(e.Implements) > 0 {
		fmt.Fprintf(b, "@implements [%s]\n", strings.Join(e.Implements, ", "))
	}
	if len(e.Related) > 0 {
		fmt.Fprintf(b, "@related %s\n", strings.Join(e.Related, ", "))
	}
}

func emitTrait(b *strings.Builder, e *TraitEntry) {
	fmt.Fprintf(b, "@trait %s\n", e.Name)
	if e.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", e.Purpose)
	}
	if len(e.Extends) > 0 {
		fmt.Fprintf(b, "@extends %s\n", strings.Join(e.Extends, ", "))
	}
	if len(e.Requires) > 0 {
		b.WriteString("@requires\n")
		for _, r := range e.Requires {
			fmt.Fprintf(b, "  %s\n", r)
		}
	}
	if len(e.Related) > 0 {
		fmt.Fprintf(b, "@related %s\n", strings.Join(e.Related, ", "))
	}
}

func emitConst(b *strings.Builder, e *ConstEntry) {
	fmt.Fprintf(b, "@const %s\n", e.Name)
	if e.Purpose != "" {
		fmt.Fprintf(b, "@purpose %s\n", e.Purpose)
	}
	if e.Type != "" {
		fmt.Fprintf(b, "@type %s\n", e.Type)
	}
	if e.Value != "" {
		fmt.Fprintf(b, "@value %s\n", e.Value)
	}
}

func isRedundantReturns(e *FnEntry) bool {
	if e.Returns == "" || len(e.Sigs) == 0 {
		return false
	}
	ret := strings.TrimSpace(e.Returns)
	if strings.Contains(ret, " ") {
		return false
	}
	for _, sig := range e.Sigs {
		if strings.Contains(sig, "-> "+ret) {
			return true
		}
	}
	return false
}

func orderEntries(entries []Entry) []Entry {
	typeNames := map[string]bool{}
	for _, e := range entries {
		if t, ok := e.(TypeEntry); ok {
			typeNames[t.Name] = true
		}
		if t, ok := e.(TraitEntry); ok {
			typeNames[t.Name] = true
		}
	}

	methodMap := map[string][]Entry{}
	for _, e := range entries {
		if fn, ok := e.(FnEntry); ok {
			if idx := strings.Index(fn.Name, "."); idx > 0 {
				parent := fn.Name[:idx]
				if typeNames[parent] {
					methodMap[parent] = append(methodMap[parent], e)
				}
			}
		}
	}

	var consts, typesWithMethods, traits, fns []Entry
	for _, e := range entries {
		switch v := e.(type) {
		case ConstEntry:
			consts = append(consts, e)
		case TypeEntry:
			typesWithMethods = append(typesWithMethods, e)
			for _, m := range methodMap[v.Name] {
				typesWithMethods = append(typesWithMethods, m)
			}
		case TraitEntry:
			traits = append(traits, e)
		case FnEntry:
			if idx := strings.Index(v.Name, "."); idx > 0 {
				parent := v.Name[:idx]
				if typeNames[parent] {
					continue // already grouped
				}
			}
			fns = append(fns, e)
		}
	}

	result := make([]Entry, 0, len(entries))
	result = append(result, consts...)
	result = append(result, typesWithMethods...)
	result = append(result, traits...)
	result = append(result, fns...)
	return result
}
