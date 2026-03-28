package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"unicode"
)

// ExtractPackage parses a Go package directory and produces an AidFile.
// When includeInternal is true, unexported functions are included with minimal
// info (@fn + @sig only) so call-graph tools like cartograph can trace the
// complete call chain through internal helpers.
func ExtractPackage(dir string, moduleName string, version string, includeInternal bool) (*AidFile, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Pick the non-test package
	var pkg *ast.Package
	for name, p := range pkgs {
		if !strings.HasSuffix(name, "_test") {
			pkg = p
			break
		}
	}
	if pkg == nil {
		return nil, fmt.Errorf("no non-test package found in %s", dir)
	}

	// Use go/doc for organized access
	dpkg := doc.New(pkg, moduleName, doc.AllDecls)

	if moduleName == "" {
		moduleName = dpkg.Name
	}

	purpose := firstSentence(dpkg.Doc)

	header := ModuleHeader{
		Module:     moduleName,
		Lang:       "go",
		Version:    version,
		Purpose:    purpose,
		AidVersion: "0.1",
	}

	var entries []Entry

	// Extract constants
	for _, c := range dpkg.Consts {
		entries = append(entries, extractConsts(c)...)
	}

	// Extract variables (sentinel errors, etc.)
	for _, v := range dpkg.Vars {
		entries = append(entries, extractVars(v)...)
	}

	// Extract types (structs, interfaces, defined types)
	for _, t := range dpkg.Types {
		entries = append(entries, extractType(t, includeInternal)...)
	}

	// Extract package-level functions
	for _, f := range dpkg.Funcs {
		if fn := extractFunc(f, includeInternal); fn != nil {
			entries = append(entries, *fn)
		}
	}

	return &AidFile{
		Header:  header,
		Entries: entries,
	}, nil
}

func extractConsts(value *doc.Value) []Entry {
	var entries []Entry

	// Check for iota enum pattern
	if isIotaEnum(value) {
		entries = append(entries, extractIotaEnum(value))
		return entries
	}

	for _, spec := range value.Decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			if !isExported(name.Name) {
				continue
			}
			typ := ""
			if vs.Type != nil {
				typ = GoTypeToAID(vs.Type)
			} else if i < len(vs.Values) {
				typ = inferConstType(vs.Values[i])
			}
			val := ""
			if i < len(vs.Values) {
				val = exprToString(vs.Values[i])
			}
			entries = append(entries, ConstEntry{
				Name:    name.Name,
				Purpose: firstSentence(value.Doc),
				Type:    typ,
				Value:   val,
			})
		}
	}
	return entries
}

func extractVars(value *doc.Value) []Entry {
	var entries []Entry
	for _, spec := range value.Decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			if !isExported(name.Name) {
				continue
			}
			// Check for sentinel errors: var ErrFoo = errors.New(...)
			typ := ""
			if vs.Type != nil {
				typ = GoTypeToAID(vs.Type)
			} else if i < len(vs.Values) {
				if isSentinelError(vs.Values[i]) {
					typ = "error"
				} else {
					typ = inferConstType(vs.Values[i])
				}
			}
			val := ""
			if i < len(vs.Values) {
				val = exprToString(vs.Values[i])
			}
			entries = append(entries, ConstEntry{
				Name:    name.Name,
				Purpose: firstSentence(value.Doc),
				Type:    typ,
				Value:   val,
			})
		}
	}
	return entries
}

func extractType(t *doc.Type, includeInternal bool) []Entry {
	var entries []Entry

	for _, spec := range t.Decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok || !isExported(ts.Name.Name) {
			continue
		}

		purpose := firstSentence(t.Doc)

		switch typ := ts.Type.(type) {
		case *ast.StructType:
			entry := extractStruct(ts, typ, purpose)
			entries = append(entries, entry)

		case *ast.InterfaceType:
			entry := extractInterface(ts, typ, purpose)
			entries = append(entries, entry)

		case *ast.Ident, *ast.SelectorExpr, *ast.ArrayType, *ast.MapType, *ast.FuncType, *ast.StarExpr:
			// Defined type (type Name OtherType)
			if ts.Assign.IsValid() {
				// Type alias: type Name = Other
				entries = append(entries, TypeEntry{
					Name:    ts.Name.Name,
					Kind:    "alias",
					Purpose: purpose,
				})
			} else {
				// New type: type Name Other
				underlying := GoTypeToAID(ts.Type)
				entries = append(entries, TypeEntry{
					Name:    ts.Name.Name,
					Kind:    "newtype",
					Purpose: purpose,
					Fields: []Field{
						{Name: "(inner)", Type: underlying, Desc: "The wrapped " + underlying + " value"},
					},
				})
			}

		default:
			// Generics or complex types — extract with best effort
			genParams := extractGenericParams(ts)
			entries = append(entries, TypeEntry{
				Name:          ts.Name.Name,
				Kind:          "struct",
				Purpose:       purpose,
				GenericParams: genParams,
			})
		}
	}

	// Extract methods
	methods := map[string][]string{} // typeName -> method names
	for _, m := range t.Methods {
		fn := extractDocFunc(m, t.Name, includeInternal)
		if fn != nil {
			entries = append(entries, *fn)
			methods[t.Name] = append(methods[t.Name], m.Name)
		}
	}

	// Backfill methods list on type entries
	if len(methods[t.Name]) > 0 {
		for i, e := range entries {
			if te, ok := e.(TypeEntry); ok && te.Name == t.Name {
				te.Methods = methods[t.Name]
				entries[i] = te
				break
			}
		}
	}

	// Extract associated constants and constructors
	for _, c := range t.Consts {
		entries = append(entries, extractConsts(c)...)
	}
	for _, f := range t.Funcs {
		if fn := extractFunc(f, includeInternal); fn != nil {
			entries = append(entries, *fn)
		}
	}

	return entries
}

func extractStruct(ts *ast.TypeSpec, st *ast.StructType, purpose string) TypeEntry {
	var fields []Field
	var embedded []string

	if st.Fields != nil {
		for _, field := range st.Fields.List {
			if len(field.Names) == 0 {
				// Embedded field
				typeName := GoTypeToAID(field.Type)
				embedded = append(embedded, typeName)
				continue
			}
			for _, name := range field.Names {
				if !isExported(name.Name) {
					continue
				}
				desc := strings.TrimSpace(field.Doc.Text())
				if desc == "" {
					desc = strings.TrimSpace(field.Comment.Text())
				}
				fields = append(fields, Field{
					Name: name.Name,
					Type: GoTypeToAID(field.Type),
					Desc: desc,
				})
			}
		}
	}

	genParams := extractGenericParams(ts)

	return TypeEntry{
		Name:          ts.Name.Name,
		Kind:          "struct",
		Purpose:       purpose,
		Fields:        fields,
		Extends:       embedded,
		GenericParams: genParams,
	}
}

func extractInterface(ts *ast.TypeSpec, iface *ast.InterfaceType, purpose string) TraitEntry {
	var requires []string
	var extends []string

	if iface.Methods != nil {
		for _, m := range iface.Methods.List {
			if len(m.Names) > 0 {
				// Method
				name := m.Names[0].Name
				if ft, ok := m.Type.(*ast.FuncType); ok {
					sig := buildMethodSig(name, ft, false, false)
					requires = append(requires, "fn "+sig)
				}
			} else {
				// Embedded interface
				extends = append(extends, GoTypeToAID(m.Type))
			}
		}
	}

	return TraitEntry{
		Name:     ts.Name.Name,
		Purpose:  purpose,
		Requires: requires,
		Extends:  extends,
	}
}

func extractFunc(f *doc.Func, includeInternal bool) *FnEntry {
	if !isExported(f.Name) && !includeInternal {
		return nil
	}
	sig := buildFuncSig(f.Decl)

	// Unexported functions get minimal entries (just @fn + @sig) for call-graph tools
	if !isExported(f.Name) {
		return &FnEntry{
			Name: f.Name,
			Sigs: []string{sig},
		}
	}

	purpose := firstSentence(f.Doc)
	params := extractParams(f.Decl.Type)
	returns := extractReturnType(f.Decl.Type)

	return &FnEntry{
		Name:    f.Name,
		Purpose: purpose,
		Sigs:    []string{sig},
		Params:  params,
		Returns: returns,
	}
}

func extractDocFunc(f *doc.Func, typeName string, includeInternal bool) *FnEntry {
	if !isExported(f.Name) && !includeInternal {
		return nil
	}

	isPointer := false
	if f.Decl.Recv != nil && len(f.Decl.Recv.List) > 0 {
		_, isPointer = f.Decl.Recv.List[0].Type.(*ast.StarExpr)
	}

	sig := buildMethodSigFromDecl(f.Decl, isPointer)
	name := typeName + "." + f.Name

	// Unexported methods get minimal entries for call-graph tools
	if !isExported(f.Name) {
		return &FnEntry{
			Name: name,
			Sigs: []string{sig},
		}
	}

	purpose := firstSentence(f.Doc)
	params := extractParams(f.Decl.Type)
	returns := extractReturnType(f.Decl.Type)

	return &FnEntry{
		Name:    name,
		Purpose: purpose,
		Sigs:    []string{sig},
		Params:  params,
		Returns: returns,
	}
}

func buildFuncSig(decl *ast.FuncDecl) string {
	ft := decl.Type
	params := buildParamList(ft.Params)
	retType, errType := buildReturnSig(ft.Results)

	sig := "(" + strings.Join(params, ", ") + ") -> " + retType
	if errType != "" {
		sig += " ! " + errType
	}
	return sig
}

func buildMethodSigFromDecl(decl *ast.FuncDecl, isPointer bool) string {
	ft := decl.Type
	selfParam := "self"
	if isPointer {
		selfParam = "mut self"
	}

	params := []string{selfParam}
	params = append(params, buildParamList(ft.Params)...)
	retType, errType := buildReturnSig(ft.Results)

	sig := "(" + strings.Join(params, ", ") + ") -> " + retType
	if errType != "" {
		sig += " ! " + errType
	}
	return sig
}

func buildMethodSig(name string, ft *ast.FuncType, hasSelf bool, isPointer bool) string {
	params := buildParamList(ft.Params)
	retType, errType := buildReturnSig(ft.Results)

	sig := name + "(" + strings.Join(params, ", ") + ") -> " + retType
	if errType != "" {
		sig += " ! " + errType
	}
	return sig
}

func buildParamList(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	var params []string
	for _, field := range fields.List {
		t := GoTypeToAID(field.Type)
		_, isVariadic := field.Type.(*ast.Ellipsis)
		if len(field.Names) == 0 {
			if isVariadic {
				params = append(params, "...args: "+t)
			} else {
				params = append(params, t)
			}
		} else {
			for _, name := range field.Names {
				if isVariadic {
					params = append(params, "..."+name.Name+": "+t)
				} else {
					params = append(params, name.Name+": "+t)
				}
			}
		}
	}
	return params
}

func buildReturnSig(results *ast.FieldList) (retType string, errType string) {
	if results == nil || len(results.List) == 0 {
		return "None", ""
	}

	types := []string{}
	for _, field := range results.List {
		n := 1
		if len(field.Names) > 0 {
			n = len(field.Names)
		}
		for i := 0; i < n; i++ {
			types = append(types, GoTypeToAID(field.Type))
		}
	}

	// Check for (T, error) pattern
	if len(types) >= 2 && types[len(types)-1] == "error" {
		errType = "error"
		types = types[:len(types)-1]
	}

	if len(types) == 0 {
		retType = "None"
	} else if len(types) == 1 {
		retType = types[0]
	} else {
		retType = "(" + strings.Join(types, ", ") + ")"
	}

	return retType, errType
}

func extractParams(ft *ast.FuncType) []Param {
	if ft.Params == nil {
		return nil
	}
	var params []Param
	for _, field := range ft.Params.List {
		t := GoTypeToAID(field.Type)
		_, isVariadic := field.Type.(*ast.Ellipsis)
		if len(field.Names) == 0 {
			params = append(params, Param{Name: "_", Type: t, Variadic: isVariadic})
		} else {
			for _, name := range field.Names {
				params = append(params, Param{Name: name.Name, Type: t, Variadic: isVariadic})
			}
		}
	}
	return params
}

func extractReturnType(ft *ast.FuncType) string {
	if ft.Results == nil || len(ft.Results.List) == 0 {
		return ""
	}
	types := []string{}
	for _, field := range ft.Results.List {
		types = append(types, GoTypeToAID(field.Type))
	}
	// Filter out error for return description
	filtered := []string{}
	for _, t := range types {
		if t != "error" {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return "(" + strings.Join(filtered, ", ") + ")"
}

func extractGenericParams(ts *ast.TypeSpec) string {
	if ts.TypeParams == nil || len(ts.TypeParams.List) == 0 {
		return ""
	}
	var params []string
	for _, field := range ts.TypeParams.List {
		constraint := GoTypeToAID(field.Type)
		for _, name := range field.Names {
			if constraint != "" && constraint != "any" {
				params = append(params, name.Name+": "+constraint)
			} else {
				params = append(params, name.Name)
			}
		}
	}
	return strings.Join(params, ", ")
}

// --- Helpers ---

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

func firstSentence(doc string) string {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return ""
	}
	// Take first line, or up to first period
	lines := strings.SplitN(doc, "\n", 2)
	first := strings.TrimSpace(lines[0])

	if idx := strings.Index(first, ". "); idx > 0 {
		first = first[:idx+1]
	}

	// Truncate to 120 chars
	if len(first) > 120 {
		first = first[:117] + "..."
	}
	return first
}

func isIotaEnum(value *doc.Value) bool {
	if len(value.Decl.Specs) < 2 {
		return false
	}
	first, ok := value.Decl.Specs[0].(*ast.ValueSpec)
	if !ok || len(first.Values) == 0 {
		return false
	}
	// Check if the first value is iota
	ident, ok := first.Values[0].(*ast.Ident)
	if !ok || ident.Name != "iota" {
		return false
	}
	// Check if there's a shared type
	return first.Type != nil
}

func extractIotaEnum(value *doc.Value) TypeEntry {
	var typeName string
	var variants []Variant

	for _, spec := range value.Decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if vs.Type != nil {
			typeName = GoTypeToAID(vs.Type)
		}
		for _, name := range vs.Names {
			if !isExported(name.Name) {
				continue
			}
			variants = append(variants, Variant{Name: name.Name})
		}
	}

	return TypeEntry{
		Name:     typeName,
		Kind:     "enum",
		Purpose:  firstSentence(value.Doc),
		Variants: variants,
	}
}

func isSentinelError(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	// errors.New(...) or fmt.Errorf(...)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return (ident.Name == "errors" && sel.Sel.Name == "New") ||
		(ident.Name == "fmt" && sel.Sel.Name == "Errorf")
}

func inferConstType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "f64"
		case token.STRING:
			return "str"
		case token.CHAR:
			return "i32"
		}
	case *ast.Ident:
		if e.Name == "true" || e.Name == "false" {
			return "bool"
		}
		if e.Name == "iota" {
			return "int"
		}
	case *ast.CallExpr:
		if isSentinelError(expr) {
			return "error"
		}
	}
	return "any"
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.CallExpr:
		// e.g., errors.New("not found")
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				args := []string{}
				for _, arg := range e.Args {
					args = append(args, exprToString(arg))
				}
				return ident.Name + "." + sel.Sel.Name + "(" + strings.Join(args, ", ") + ")"
			}
		}
		return ""
	case *ast.BinaryExpr:
		return exprToString(e.X) + " " + e.Op.String() + " " + exprToString(e.Y)
	default:
		return ""
	}
}

// PackageDirFromImportPath resolves a Go import path to a directory.
// For local paths like ./pkg, it just cleans the path.
func PackageDirFromImportPath(path string) string {
	if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
		return filepath.Clean(path)
	}
	return path
}
