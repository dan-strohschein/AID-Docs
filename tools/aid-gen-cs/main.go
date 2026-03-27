// Command aid-gen-cs generates AID files from C# source files.
// It uses a .NET/Roslyn helper (CSharpExtractor) to parse C# via the Roslyn compiler API,
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

// JSON structures matching CSharpExtractor output
type ExtractResult struct {
	File       string        `json:"file"`
	Module     string        `json:"module"`
	Namespace  string        `json:"namespace"`
	Classes    []CSClass     `json:"classes"`
	Interfaces []CSInterface `json:"interfaces"`
	Structs    []CSStruct    `json:"structs"`
	Enums      []CSEnum      `json:"enums"`
	Delegates  []CSDelegate  `json:"delegates"`
}

type CSClass struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	BaseTypes  []string    `json:"baseTypes"`
	IsAbstract bool        `json:"isAbstract"`
	IsStatic   bool        `json:"isStatic"`
	IsSealed   bool        `json:"isSealed"`
	Members    []CSMember  `json:"members"`
	Doc        string      `json:"doc"`
}

type CSInterface struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	BaseTypes  []string    `json:"baseTypes"`
	Members    []CSMember  `json:"members"`
	Doc        string      `json:"doc"`
}

type CSStruct struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	BaseTypes  []string    `json:"baseTypes"`
	Members    []CSMember  `json:"members"`
	Doc        string      `json:"doc"`
}

type CSEnum struct {
	Name    string         `json:"name"`
	Members []CSEnumMember `json:"members"`
	Doc     string         `json:"doc"`
}

type CSEnumMember struct {
	Name  string  `json:"name"`
	Value *string `json:"value"`
	Doc   string  `json:"doc"`
}

type CSDelegate struct {
	Name       string      `json:"name"`
	TypeParams []TypeParam `json:"typeParams"`
	Params     []CSParam   `json:"params"`
	ReturnType string      `json:"returnType"`
	Doc        string      `json:"doc"`
}

type CSMember struct {
	Kind       string      `json:"kind"` // method, property, field, constructor, event, indexer
	Name       string      `json:"name"`
	IsStatic   bool        `json:"isStatic"`
	IsAsync    bool        `json:"isAsync"`
	IsAbstract bool        `json:"isAbstract"`
	IsVirtual  bool        `json:"isVirtual"`
	IsReadonly bool        `json:"isReadonly"`
	IsConst    bool        `json:"isConst"`
	TypeParams []TypeParam `json:"typeParams"`
	Params     []CSParam   `json:"params"`
	ReturnType string      `json:"returnType"`
	Type       string      `json:"type"` // for properties/fields
	HasGetter  bool        `json:"hasGetter"`
	HasSetter  bool        `json:"hasSetter"`
	Value      *string     `json:"value"`
	Doc        string      `json:"doc"`
}

type CSParam struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	IsOptional bool    `json:"isOptional"`
	IsParams   bool    `json:"isParams"`
	IsRef      bool    `json:"isRef"`
	IsOut      bool    `json:"isOut"`
	Default    *string `json:"default"`
}

type TypeParam struct {
	Name string `json:"name"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aid-gen-cs [flags] <file.cs> [file2.cs ...]\n\n")
		fmt.Fprintf(os.Stderr, "Generate AID files from C# source code.\n")
		fmt.Fprintf(os.Stderr, "Requires .NET SDK with Roslyn.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	extractorDir := findExtractorProject()

	// Call dotnet run to parse the files
	args := append([]string{"run", "--project", extractorDir, "--"}, flag.Args()...)
	cmd := exec.Command("dotnet", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running CSharpExtractor: %v\n", err)
		os.Exit(1)
	}

	var results []ExtractResult
	if err := json.Unmarshal(out, &results); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing extractor output: %v\n", err)
		os.Exit(1)
	}

	for _, r := range results {
		aid := convertToAID(&r)

		if *stdout {
			fmt.Print(aid)
		} else {
			if err := os.MkdirAll(*outputDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			modName := r.Namespace
			if *moduleName != "" {
				modName = *moduleName
			}
			if modName == "" {
				modName = r.Module
			}
			filename := strings.ReplaceAll(modName, ".", "-") + ".aid"
			outPath := filepath.Join(*outputDir, filename)
			if err := os.WriteFile(outPath, []byte(aid), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if *verbose {
				fmt.Fprintf(os.Stderr, "  → %s\n", outPath)
			}
		}
	}
}

func findExtractorProject() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	candidates := []string{
		filepath.Join(dir, "extract", "CSharpExtractor"),
		filepath.Join("extract", "CSharpExtractor"),
		filepath.Join("tools", "aid-gen-cs", "extract", "CSharpExtractor"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "CSharpExtractor.csproj")); err == nil {
			return c
		}
	}
	fmt.Fprintf(os.Stderr, "Cannot find CSharpExtractor project. Run from the aid-gen-cs directory.\n")
	os.Exit(1)
	return ""
}

func convertToAID(r *ExtractResult) string {
	var b strings.Builder

	b.WriteString("// [generated] Layer 1 mechanical extraction — not yet reviewed\n\n")

	modName := r.Namespace
	if *moduleName != "" {
		modName = *moduleName
	}
	if modName == "" {
		modName = r.Module
	}

	fmt.Fprintf(&b, "@module %s\n", modName)
	fmt.Fprintf(&b, "@lang csharp\n")
	fmt.Fprintf(&b, "@version %s\n", *version)
	fmt.Fprintf(&b, "@aid_version 0.1\n")

	// Enums
	for _, e := range r.Enums {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "@type %s\n", e.Name)
		fmt.Fprintf(&b, "@kind enum\n")
		if e.Doc != "" {
			fmt.Fprintf(&b, "@purpose %s\n", firstSentence(e.Doc))
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

	// Delegates → function types
	for _, d := range r.Delegates {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "@type %s\n", d.Name)
		fmt.Fprintf(&b, "@kind alias\n")
		if d.Doc != "" {
			fmt.Fprintf(&b, "@purpose %s\n", firstSentence(d.Doc))
		}
		sig := buildDelegateSig(&d)
		fmt.Fprintf(&b, "// Delegate signature: %s\n", sig)
	}

	// Structs
	for _, s := range r.Structs {
		b.WriteString("\n---\n\n")
		emitTypeDecl(&b, s.Name, "struct", s.Doc, s.TypeParams, s.BaseTypes, s.Members)
	}

	// Interfaces → traits
	for _, iface := range r.Interfaces {
		b.WriteString("\n---\n\n")
		emitInterface(&b, &iface)
	}

	// Classes
	for _, cls := range r.Classes {
		b.WriteString("\n---\n\n")
		emitClass(&b, &cls)
	}

	return b.String()
}

func emitInterface(b *strings.Builder, iface *CSInterface) {
	fmt.Fprintf(b, "@trait %s\n", iface.Name)
	if iface.Doc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(iface.Doc))
	}
	if len(iface.TypeParams) > 0 {
		fmt.Fprintf(b, "@generic_params %s\n", formatTypeParams(iface.TypeParams))
	}
	if len(iface.BaseTypes) > 0 {
		fmt.Fprintf(b, "@extends %s\n", strings.Join(iface.BaseTypes, ", "))
	}

	var requires []string
	for _, m := range iface.Members {
		if m.Kind == "method" {
			sig := buildMethodSig(&m)
			requires = append(requires, "fn "+sig)
		}
	}
	if len(requires) > 0 {
		b.WriteString("@requires\n")
		for _, r := range requires {
			fmt.Fprintf(b, "  %s\n", r)
		}
	}

	// Properties
	var props []CSMember
	for _, m := range iface.Members {
		if m.Kind == "property" {
			props = append(props, m)
		}
	}
	if len(props) > 0 {
		b.WriteString("@fields\n")
		for _, p := range props {
			desc := ""
			if !p.HasSetter {
				desc = " — readonly"
			}
			fmt.Fprintf(b, "  %s: %s%s\n", p.Name, csTypeToAID(p.Type), desc)
		}
	}
}

func emitClass(b *strings.Builder, cls *CSClass) {
	fmt.Fprintf(b, "@type %s\n", cls.Name)
	if cls.IsAbstract {
		fmt.Fprintf(b, "@kind class\n")
	} else {
		fmt.Fprintf(b, "@kind class\n")
	}
	if cls.Doc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(cls.Doc))
	}
	if len(cls.TypeParams) > 0 {
		fmt.Fprintf(b, "@generic_params %s\n", formatTypeParams(cls.TypeParams))
	}

	// Separate base class from interfaces
	var extends []string
	var implements []string
	for _, bt := range cls.BaseTypes {
		// Convention: interfaces start with I in C#
		if len(bt) > 1 && bt[0] == 'I' && bt[1] >= 'A' && bt[1] <= 'Z' {
			implements = append(implements, bt)
		} else {
			extends = append(extends, bt)
		}
	}
	if len(extends) > 0 {
		fmt.Fprintf(b, "@extends %s\n", strings.Join(extends, ", "))
	}
	if len(implements) > 0 {
		fmt.Fprintf(b, "@implements [%s]\n", strings.Join(implements, ", "))
	}

	// Properties and fields
	var fields []CSMember
	var methods []string
	var constructors []CSMember
	var consts []CSMember

	for _, m := range cls.Members {
		switch m.Kind {
		case "property":
			fields = append(fields, m)
		case "field":
			if m.IsConst {
				consts = append(consts, m)
			} else {
				fields = append(fields, m)
			}
		case "method":
			if !m.IsStatic {
				methods = append(methods, m.Name)
			}
		case "constructor":
			constructors = append(constructors, m)
		}
	}

	if len(fields) > 0 {
		b.WriteString("@fields\n")
		for _, f := range fields {
			t := f.Type
			if t == "" {
				t = f.ReturnType
			}
			desc := ""
			if f.Kind == "property" && !f.HasSetter {
				desc = " — readonly"
			}
			if f.IsReadonly {
				desc = " — readonly"
			}
			if f.Doc != "" {
				desc = " — " + firstSentence(f.Doc)
			}
			fmt.Fprintf(b, "  %s: %s%s\n", f.Name, csTypeToAID(t), desc)
		}
	}

	if len(constructors) > 0 {
		ctor := constructors[0] // Primary constructor
		params := make([]string, 0)
		for _, p := range ctor.Params {
			params = append(params, fmt.Sprintf("%s: %s", p.Name, csTypeToAID(p.Type)))
		}
		fmt.Fprintf(b, "@constructors %s(%s)\n", cls.Name, strings.Join(params, ", "))
	}

	if len(methods) > 0 {
		fmt.Fprintf(b, "@methods %s\n", strings.Join(methods, ", "))
	}

	// Emit constants
	for _, c := range consts {
		b.WriteString("\n---\n\n")
		fmt.Fprintf(b, "@const %s.%s\n", cls.Name, c.Name)
		if c.Doc != "" {
			fmt.Fprintf(b, "@purpose %s\n", firstSentence(c.Doc))
		}
		fmt.Fprintf(b, "@type %s\n", csTypeToAID(c.Type))
		if c.Value != nil {
			fmt.Fprintf(b, "@value %s\n", *c.Value)
		}
	}

	// Emit method entries
	for _, m := range cls.Members {
		if m.Kind == "method" {
			b.WriteString("\n---\n\n")
			emitMethod(b, cls.Name, &m)
		}
	}
}

func emitTypeDecl(b *strings.Builder, name, kind, doc string, typeParams []TypeParam, baseTypes []string, members []CSMember) {
	fmt.Fprintf(b, "@type %s\n", name)
	fmt.Fprintf(b, "@kind %s\n", kind)
	if doc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(doc))
	}
	if len(typeParams) > 0 {
		fmt.Fprintf(b, "@generic_params %s\n", formatTypeParams(typeParams))
	}
	if len(baseTypes) > 0 {
		fmt.Fprintf(b, "@implements [%s]\n", strings.Join(baseTypes, ", "))
	}

	var fields []CSMember
	for _, m := range members {
		if m.Kind == "property" || m.Kind == "field" {
			fields = append(fields, m)
		}
	}
	if len(fields) > 0 {
		b.WriteString("@fields\n")
		for _, f := range fields {
			t := f.Type
			desc := ""
			if f.Doc != "" {
				desc = " — " + firstSentence(f.Doc)
			}
			fmt.Fprintf(b, "  %s: %s%s\n", f.Name, csTypeToAID(t), desc)
		}
	}
}

func emitMethod(b *strings.Builder, className string, m *CSMember) {
	name := className + "." + m.Name
	fmt.Fprintf(b, "@fn %s\n", name)
	if m.Doc != "" {
		fmt.Fprintf(b, "@purpose %s\n", firstSentence(m.Doc))
	}

	self := "self"
	if m.IsStatic {
		self = ""
	}

	sig := buildMethodSigWithSelf(m, self)
	fmt.Fprintf(b, "@sig %s\n", sig)

	if len(m.Params) > 0 {
		b.WriteString("@params\n")
		for _, p := range m.Params {
			fmt.Fprintf(b, "  %s: %s\n", p.Name, csTypeToAID(p.Type))
		}
	}
}

func buildMethodSig(m *CSMember) string {
	parts := make([]string, 0)
	for _, p := range m.Params {
		parts = append(parts, paramSig(p))
	}
	ret := csTypeToAID(m.ReturnType)
	asyncPrefix := ""
	if m.IsAsync {
		asyncPrefix = "async "
	}
	generic := ""
	if len(m.TypeParams) > 0 {
		generic = "[" + formatTypeParams(m.TypeParams) + "]"
	}
	return fmt.Sprintf("%s%s%s(%s) -> %s", asyncPrefix, generic, m.Name, strings.Join(parts, ", "), ret)
}

func buildMethodSigWithSelf(m *CSMember, self string) string {
	parts := make([]string, 0)
	if self != "" {
		parts = append(parts, self)
	}
	for _, p := range m.Params {
		parts = append(parts, paramSig(p))
	}
	ret := csTypeToAID(m.ReturnType)
	asyncPrefix := ""
	if m.IsAsync {
		asyncPrefix = "async "
	}
	generic := ""
	if len(m.TypeParams) > 0 {
		generic = "[" + formatTypeParams(m.TypeParams) + "]"
	}
	return fmt.Sprintf("%s%s(%s) -> %s", asyncPrefix, generic, strings.Join(parts, ", "), ret)
}

func buildDelegateSig(d *CSDelegate) string {
	parts := make([]string, 0)
	for _, p := range d.Params {
		parts = append(parts, paramSig(p))
	}
	return fmt.Sprintf("fn(%s) -> %s", strings.Join(parts, ", "), csTypeToAID(d.ReturnType))
}

func paramSig(p CSParam) string {
	t := csTypeToAID(p.Type)
	prefix := ""
	if p.IsRef {
		prefix = "ref "
	} else if p.IsOut {
		prefix = "out "
	} else if p.IsParams {
		return fmt.Sprintf("...%s: %s", p.Name, t)
	}
	if p.IsOptional {
		return fmt.Sprintf("%s%s?: %s", prefix, p.Name, t)
	}
	return fmt.Sprintf("%s%s: %s", prefix, p.Name, t)
}

func formatTypeParams(params []TypeParam) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Name
	}
	return strings.Join(parts, ", ")
}

// csTypeToAID converts C# type notation to AID universal types.
func csTypeToAID(t string) string {
	t = strings.TrimSpace(t)

	switch t {
	case "string", "String":
		return "str"
	case "int", "Int32":
		return "int"
	case "long", "Int64":
		return "i64"
	case "short", "Int16":
		return "i16"
	case "byte", "Byte":
		return "u8"
	case "uint", "UInt32":
		return "u32"
	case "ulong", "UInt64":
		return "u64"
	case "float", "Single":
		return "f32"
	case "double", "Double":
		return "f64"
	case "decimal", "Decimal":
		return "f64"
	case "bool", "Boolean":
		return "bool"
	case "void":
		return "None"
	case "object", "Object":
		return "any"
	case "byte[]":
		return "bytes"
	case "char", "Char":
		return "i32"
	}

	// Task<T> → T (async already marked)
	if strings.HasPrefix(t, "Task<") && strings.HasSuffix(t, ">") {
		inner := t[5 : len(t)-1]
		return csTypeToAID(inner)
	}
	if t == "Task" {
		return "None"
	}

	// Nullable: T? → T?
	if strings.HasSuffix(t, "?") {
		inner := t[:len(t)-1]
		return csTypeToAID(inner) + "?"
	}

	// List<T> → [T]
	if strings.HasPrefix(t, "List<") || strings.HasPrefix(t, "IList<") || strings.HasPrefix(t, "IEnumerable<") || strings.HasPrefix(t, "IReadOnlyList<") {
		idx := strings.Index(t, "<")
		inner := t[idx+1 : len(t)-1]
		return "[" + csTypeToAID(inner) + "]"
	}

	// T[] → [T]
	if strings.HasSuffix(t, "[]") {
		inner := t[:len(t)-2]
		return "[" + csTypeToAID(inner) + "]"
	}

	// Dictionary<K,V> → dict[K, V]
	if strings.HasPrefix(t, "Dictionary<") || strings.HasPrefix(t, "IDictionary<") || strings.HasPrefix(t, "IReadOnlyDictionary<") {
		idx := strings.Index(t, "<")
		inner := t[idx+1 : len(t)-1]
		return "dict[" + inner + "]"
	}

	// HashSet<T> → set[T]
	if strings.HasPrefix(t, "HashSet<") || strings.HasPrefix(t, "ISet<") {
		idx := strings.Index(t, "<")
		inner := t[idx+1 : len(t)-1]
		return "set[" + csTypeToAID(inner) + "]"
	}

	// CancellationToken → Context (analogous to Go's context.Context)
	if t == "CancellationToken" {
		return "CancellationToken"
	}

	// TimeSpan → Duration
	if t == "TimeSpan" {
		return "Duration"
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
