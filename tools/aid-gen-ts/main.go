// Command aid-gen-ts generates AID files from TypeScript source or .d.ts declaration files.
// It uses a Node.js helper script (extract.js) to parse TypeScript via the TS compiler API,
// then converts the structured JSON output to .aid format.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	outputDir  = flag.String("output", ".aidocs", "Output directory for .aid files")
	stdout     = flag.Bool("stdout", false, "Print output to stdout")
	moduleName = flag.String("module", "", "Override module name")
	version    = flag.String("version", "0.0.0", "Library version")
	verbose    = flag.Bool("v", false, "Verbose output")
)

// JSON structures matching extract.js output
type ExtractResult struct {
	File       string       `json:"file"`
	Module     string       `json:"module"`
	Functions  []TSFunction `json:"functions"`
	Classes    []TSClass    `json:"classes"`
	Interfaces []TSInterface `json:"interfaces"`
	Types      []TSTypeAlias `json:"types"`
	Enums      []TSEnum     `json:"enums"`
	Constants  []TSConstant `json:"constants"`
}

type TSFunction struct {
	Name       string      `json:"name"`
	Async      bool        `json:"async"`
	TypeParams []TypeParam `json:"typeParams"`
	Params     []TSParam   `json:"params"`
	ReturnType string      `json:"returnType"`
	JSDoc      string      `json:"jsdoc"`
}

type TSClass struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	Extends    string      `json:"extends"`
	Implements []string    `json:"implements"`
	Members    []TSMember  `json:"members"`
	JSDoc      string      `json:"jsdoc"`
}

type TSInterface struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	Extends    []string    `json:"extends"`
	Members    []TSMember  `json:"members"`
	JSDoc      string      `json:"jsdoc"`
}

type TSTypeAlias struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	Type       string      `json:"type"`
	JSDoc      string      `json:"jsdoc"`
}

type TSEnum struct {
	Name    string         `json:"name"`
	Members []TSEnumMember `json:"members"`
	JSDoc   string         `json:"jsdoc"`
}

type TSEnumMember struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
}

type TSConstant struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	JSDoc string `json:"jsdoc"`
}

type TSMember struct {
	Kind       string      `json:"kind"` // method, property, constructor, call, index
	Name       string      `json:"name"`
	Static     bool        `json:"static"`
	Async      bool        `json:"async"`
	TypeParams []TypeParam `json:"typeParams"`
	Params     []TSParam   `json:"params"`
	ReturnType string      `json:"returnType"`
	Type       string      `json:"type"` // for properties
	Readonly   bool        `json:"readonly"`
	Optional   bool        `json:"optional"`
	JSDoc      string      `json:"jsdoc"`
}

type TSParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
	Rest     bool   `json:"rest"`
}

type TypeParam struct {
	Name       string `json:"name"`
	Constraint string `json:"constraint,omitempty"`
	Default    string `json:"default,omitempty"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-ts [flags] <file.d.ts> [file2.ts ...]\n\n")
		fmt.Fprintf(os.Stderr, "Generate AID files from TypeScript declarations.\n")
		fmt.Fprintf(os.Stderr, "Requires Node.js and the 'typescript' npm package.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Find extract.js relative to the binary
	extractScript := findExtractScript()

	// Call Node.js to parse the files
	args := append([]string{extractScript}, flag.Args()...)
	cmd := exec.Command("node", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running extract.js: %v\n", err)
		os.Exit(1)
	}

	var results []ExtractResult
	if err := json.Unmarshal(out, &results); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing extract.js output: %v\n", err)
		os.Exit(1)
	}

	for _, r := range results {
		aid := convertToAID(&r)

		if *stdout {
			fmt.Print(aid)
		} else {
			if err := os.MkdirAll(*outputDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
				os.Exit(1)
			}
			modName := r.Module
			if *moduleName != "" {
				modName = *moduleName
			}
			filename := strings.ReplaceAll(modName, "/", "-") + ".aid"
			outPath := filepath.Join(*outputDir, filename)
			if err := os.WriteFile(outPath, []byte(aid), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outPath, err)
				os.Exit(1)
			}
			if *verbose {
				fmt.Fprintf(os.Stderr, "  → %s\n", outPath)
			}
		}
	}
}

func findExtractScript() string {
	// Check next to the binary
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	candidate := filepath.Join(dir, "extract.js")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	// Check current working directory
	if _, err := os.Stat("extract.js"); err == nil {
		return "extract.js"
	}
	// Check tools/aid-gen-ts/
	candidate = filepath.Join("tools", "aid-gen-ts", "extract.js")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	fmt.Fprintf(os.Stderr, "Cannot find extract.js. Make sure it's next to the binary or in the current directory.\n")
	os.Exit(1)
	return ""
}

func convertToAID(r *ExtractResult) string {
	var b strings.Builder

	b.WriteString("// [generated] Layer 1 mechanical extraction — not yet reviewed\n\n")

	modName := r.Module
	if *moduleName != "" {
		modName = *moduleName
	}

	// Header
	fmt.Fprintf(&b, "@module %s\n", modName)
	fmt.Fprintf(&b, "@lang typescript\n")
	fmt.Fprintf(&b, "@version %s\n", *version)
	fmt.Fprintf(&b, "@aid_version 0.1\n")

	// Constants
	for _, c := range r.Constants {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "@const %s\n", c.Name)
		if c.JSDoc != "" {
			fmt.Fprintf(&b, "@purpose %s\n", firstSentence(c.JSDoc))
		}
		fmt.Fprintf(&b, "@type %s\n", tsTypeToAID(c.Type))
	}

	// Enums
	for _, e := range r.Enums {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "@type %s\n", e.Name)
		fmt.Fprintf(&b, "@kind enum\n")
		if e.JSDoc != "" {
			fmt.Fprintf(&b, "@purpose %s\n", firstSentence(e.JSDoc))
		}
		if len(e.Members) > 0 {
			b.WriteString("@variants\n")
			for _, m := range e.Members {
				if m.Value != nil {
					fmt.Fprintf(&b, "  | %s(%s)\n", m.Name, *m.Value)
				} else {
					fmt.Fprintf(&b, "  | %s\n", m.Name)
				}
			}
		}
	}

	// Type aliases
	for _, t := range r.Types {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "@type %s\n", t.Name)
		fmt.Fprintf(&b, "@kind alias\n")
		if t.JSDoc != "" {
			fmt.Fprintf(&b, "@purpose %s\n", firstSentence(t.JSDoc))
		}
		if len(t.TypeParams) > 0 {
			fmt.Fprintf(&b, "@generic_params %s\n", formatTypeParams(t.TypeParams))
		}
	}

	// Interfaces
	for _, iface := range r.Interfaces {
		b.WriteString("\n---\n\n")
		emitInterface(&b, &iface)
	}

	// Classes
	for _, cls := range r.Classes {
		b.WriteString("\n---\n\n")
		emitClass(&b, &cls)
	}

	// Standalone functions
	for _, fn := range r.Functions {
		b.WriteString("\n---\n\n")
		emitFunction(&b, &fn, "")
	}

	return b.String()
}

func emitInterface(b *strings.Builder, iface *TSInterface) {
	fmt.Fprintf(b, "@trait %s\n", iface.Name)
	if iface.JSDoc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(iface.JSDoc))
	}
	if len(iface.TypeParams) > 0 {
		fmt.Fprintf(b, "@generic_params %s\n", formatTypeParams(iface.TypeParams))
	}
	if len(iface.Extends) > 0 {
		fmt.Fprintf(b, "@extends %s\n", strings.Join(iface.Extends, ", "))
	}

	var requires []string
	for _, m := range iface.Members {
		if m.Kind == "method" {
			sig := buildMethodSig(m.Name, &m)
			requires = append(requires, "fn "+sig)
		}
	}
	if len(requires) > 0 {
		b.WriteString("@requires\n")
		for _, r := range requires {
			fmt.Fprintf(b, "  %s\n", r)
		}
	}

	// Properties as fields
	var props []TSMember
	for _, m := range iface.Members {
		if m.Kind == "property" {
			props = append(props, m)
		}
	}
	if len(props) > 0 {
		b.WriteString("@fields\n")
		for _, p := range props {
			opt := ""
			if p.Optional {
				opt = "?"
			}
			fmt.Fprintf(b, "  %s: %s%s\n", p.Name, tsTypeToAID(p.Type), opt)
		}
	}
}

func emitClass(b *strings.Builder, cls *TSClass) {
	fmt.Fprintf(b, "@type %s\n", cls.Name)
	fmt.Fprintf(b, "@kind class\n")
	if cls.JSDoc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(cls.JSDoc))
	}
	if len(cls.TypeParams) > 0 {
		fmt.Fprintf(b, "@generic_params %s\n", formatTypeParams(cls.TypeParams))
	}
	if cls.Extends != "" {
		fmt.Fprintf(b, "@extends %s\n", cls.Extends)
	}
	if len(cls.Implements) > 0 {
		fmt.Fprintf(b, "@implements [%s]\n", strings.Join(cls.Implements, ", "))
	}

	// Properties as fields
	var props []TSMember
	var methods []string
	for _, m := range cls.Members {
		switch m.Kind {
		case "property":
			props = append(props, m)
		case "method":
			if !m.Static {
				methods = append(methods, m.Name)
			}
		}
	}

	if len(props) > 0 {
		b.WriteString("@fields\n")
		for _, p := range props {
			desc := ""
			if p.Readonly {
				desc = " — readonly"
			}
			opt := ""
			if p.Optional {
				opt = "?"
			}
			fmt.Fprintf(b, "  %s: %s%s%s\n", p.Name, tsTypeToAID(p.Type), opt, desc)
		}
	}

	if len(methods) > 0 {
		fmt.Fprintf(b, "@methods %s\n", strings.Join(methods, ", "))
	}

	// Emit method entries
	for _, m := range cls.Members {
		if m.Kind == "method" {
			b.WriteString("\n---\n\n")
			emitMethod(b, cls.Name, &m)
		}
	}
}

func emitFunction(b *strings.Builder, fn *TSFunction, className string) {
	name := fn.Name
	if className != "" {
		name = className + "." + fn.Name
	}
	fmt.Fprintf(b, "@fn %s\n", name)
	if fn.JSDoc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(fn.JSDoc))
	}

	sig := buildFnSig(fn)
	fmt.Fprintf(b, "@sig %s\n", sig)

	if len(fn.Params) > 0 {
		b.WriteString("@params\n")
		for _, p := range fn.Params {
			fmt.Fprintf(b, "  %s: %s\n", paramName(p), tsTypeToAID(p.Type))
		}
	}

	retType := tsTypeToAID(fn.ReturnType)
	if retType != "None" && !strings.Contains(retType, " ") {
		// Don't emit redundant @returns
	} else if retType != "None" {
		fmt.Fprintf(b, "@returns %s\n", retType)
	}
}

func emitMethod(b *strings.Builder, className string, m *TSMember) {
	name := className + "." + m.Name
	fmt.Fprintf(b, "@fn %s\n", name)
	if m.JSDoc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(m.JSDoc))
	}

	sig := buildMethodSig(m.Name, m)
	if m.Static {
		sig = strings.Replace(sig, "(", "(", 1) // static: no self
	} else {
		// Add self for instance methods
		if strings.HasPrefix(sig, m.Name+"(") {
			sig = strings.Replace(sig, m.Name+"(", m.Name+"(self, ", 1)
			sig = strings.Replace(sig, "(self, )", "(self)", 1)
		}
	}
	fmt.Fprintf(b, "@sig %s\n", sig)

	// Params (skip self)
	if len(m.Params) > 0 {
		b.WriteString("@params\n")
		for _, p := range m.Params {
			fmt.Fprintf(b, "  %s: %s\n", paramName(p), tsTypeToAID(p.Type))
		}
	}
}

func buildFnSig(fn *TSFunction) string {
	var parts []string

	prefix := ""
	if fn.Async {
		prefix = "async "
	}

	if len(fn.TypeParams) > 0 {
		prefix += "[" + formatTypeParams(fn.TypeParams) + "]"
	}

	for _, p := range fn.Params {
		parts = append(parts, paramSig(p))
	}

	retType := tsTypeToAID(fn.ReturnType)

	return fmt.Sprintf("%s(%s) -> %s", prefix, strings.Join(parts, ", "), retType)
}

func buildMethodSig(name string, m *TSMember) string {
	var parts []string

	for _, p := range m.Params {
		parts = append(parts, paramSig(p))
	}

	retType := tsTypeToAID(m.ReturnType)
	asyncPrefix := ""
	if m.Async {
		asyncPrefix = "async "
	}

	return fmt.Sprintf("%s%s(%s) -> %s", asyncPrefix, name, strings.Join(parts, ", "), retType)
}

func paramSig(p TSParam) string {
	name := p.Name
	if p.Rest {
		name = "..." + name
	}
	t := tsTypeToAID(p.Type)
	if p.Optional {
		return fmt.Sprintf("%s?: %s", name, t)
	}
	return fmt.Sprintf("%s: %s", name, t)
}

func paramName(p TSParam) string {
	if p.Rest {
		return "..." + p.Name
	}
	return p.Name
}

func formatTypeParams(params []TypeParam) string {
	parts := make([]string, len(params))
	for i, p := range params {
		s := p.Name
		if p.Constraint != "" {
			s += ": " + tsTypeToAID(p.Constraint)
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

// tsTypeToAID converts TypeScript type notation to AID universal types.
func tsTypeToAID(t string) string {
	t = strings.TrimSpace(t)

	switch t {
	case "string":
		return "str"
	case "number":
		return "int"
	case "boolean":
		return "bool"
	case "void", "undefined":
		return "None"
	case "null":
		return "None"
	case "any", "unknown":
		return "any"
	case "never":
		return "None"
	case "Uint8Array", "Buffer":
		return "bytes"
	}

	// Promise<T> → T (async already marked in sig)
	if strings.HasPrefix(t, "Promise<") && strings.HasSuffix(t, ">") {
		inner := t[8 : len(t)-1]
		return tsTypeToAID(inner)
	}

	// Array<T> → [T]
	if strings.HasPrefix(t, "Array<") && strings.HasSuffix(t, ">") {
		inner := t[6 : len(t)-1]
		return "[" + tsTypeToAID(inner) + "]"
	}

	// T[] → [T]
	if strings.HasSuffix(t, "[]") {
		inner := t[:len(t)-2]
		return "[" + tsTypeToAID(inner) + "]"
	}

	// Map<K, V> → dict[K, V]
	if strings.HasPrefix(t, "Map<") {
		return "dict" + t[3:]
	}

	// Record<K, V> → dict[K, V]
	if strings.HasPrefix(t, "Record<") {
		inner := t[7 : len(t)-1]
		return "dict[" + inner + "]"
	}

	// Set<T> → set[T]
	if strings.HasPrefix(t, "Set<") {
		inner := t[4 : len(t)-1]
		return "set[" + tsTypeToAID(inner) + "]"
	}

	// T | undefined → T?
	if strings.HasSuffix(t, " | undefined") {
		inner := strings.TrimSuffix(t, " | undefined")
		return tsTypeToAID(inner) + "?"
	}
	if strings.HasSuffix(t, " | null") {
		inner := strings.TrimSuffix(t, " | null")
		return tsTypeToAID(inner) + "?"
	}

	return t
}

func firstSentence(doc string) string {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return ""
	}
	lines := strings.SplitN(doc, "\n", 2)
	first := strings.TrimSpace(lines[0])
	if idx := strings.Index(first, ". "); idx > 0 {
		first = first[:idx+1]
	}
	if len(first) > 120 {
		first = first[:117] + "..."
	}
	return first
}
