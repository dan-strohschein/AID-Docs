package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
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
	// Filter out test files — they are handled separately by ExtractTestPackage
	noTests := func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}
	pkgs, err := parser.ParseDir(fset, dir, noTests, parser.ParseComments)
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

	// Extract call relationships and source positions BEFORE doc.New strips bodies.
	// doc.New mutates the AST in place, setting Body to nil on all FuncDecls.
	funcCalls, funcPositions := extractAllCallsAndPositions(pkg, fset, dir)

	// Use go/doc for organized access (note: doc.New strips function bodies)
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
		typeEntries := extractType(t, includeInternal)
		// Add source location and calls to method entries
		for i, e := range typeEntries {
			if fn, ok := e.(FnEntry); ok {
				if c, exists := funcCalls[fn.Name]; exists {
					fn.Calls = c
				}
				if p, exists := funcPositions[fn.Name]; exists {
					fn.SourceFile = p.File
					fn.SourceLine = p.Line
				}
				typeEntries[i] = fn
			}
		}
		entries = append(entries, typeEntries...)
	}

	// Extract package-level functions
	for _, f := range dpkg.Funcs {
		if fn := extractFunc(f, includeInternal); fn != nil {
			if c, exists := funcCalls[f.Name]; exists {
				fn.Calls = c
			}
			if p, exists := funcPositions[f.Name]; exists {
				fn.SourceFile = p.File
				fn.SourceLine = p.Line
			}
			entries = append(entries, *fn)
		}
	}

	return &AidFile{
		Header:  header,
		Entries: entries,
	}, nil
}

// ExtractTestPackage parses a Go test package and produces an AidFile containing
// only mock types, test helper functions, and test-only interfaces — symbols that
// form edges back into production code. Individual TestFoo/BenchmarkFoo functions
// are excluded as they are too numerous and volatile.
func ExtractTestPackage(dir string, moduleName string, version string) (*AidFile, error) {
	fset := token.NewFileSet()

	// Only parse _test.go files
	filter := func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), "_test.go")
	}
	pkgs, err := parser.ParseDir(fset, dir, filter, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no test package found in %s", dir)
	}

	// Pick any package (prefer _test suffix, fall back to the main package's test files)
	var pkg *ast.Package
	for name, p := range pkgs {
		if strings.HasSuffix(name, "_test") {
			pkg = p
			break
		}
		pkg = p // fallback: test files in the same package
	}

	// Extract calls and positions before doc.New strips bodies
	funcCalls, funcPositions := extractAllCallsAndPositions(pkg, fset, dir)

	dpkg := doc.New(pkg, moduleName, doc.AllDecls)

	purpose := firstSentence(dpkg.Doc)
	if purpose == "" {
		purpose = "Test package for " + moduleName
	}

	header := ModuleHeader{
		Module:     moduleName,
		Lang:       "go",
		Version:    version,
		Purpose:    purpose,
		AidVersion: "0.1",
	}

	var entries []Entry

	// Extract types — keep mock types and test-only interfaces, skip plain test helpers structs
	for _, t := range dpkg.Types {
		typeEntries := extractTestType(t, fset, dir, funcCalls, funcPositions)
		entries = append(entries, typeEntries...)
	}

	// Extract package-level functions — keep test helpers, skip TestFoo/BenchmarkFoo/ExampleFoo
	for _, f := range dpkg.Funcs {
		if isTestOrBenchFunc(f.Name) {
			continue
		}
		fn := extractFunc(f, false)
		if fn == nil {
			continue
		}
		if c, exists := funcCalls[f.Name]; exists {
			fn.Calls = c
		}
		if p, exists := funcPositions[f.Name]; exists {
			fn.SourceFile = p.File
			fn.SourceLine = p.Line
		}
		entries = append(entries, *fn)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no test-relevant symbols found in %s", dir)
	}

	return &AidFile{
		Header:  header,
		Entries: entries,
	}, nil
}

// extractTestType extracts a test type and its methods. For mock/stub types
// (identified by name prefix or interface implementation), it includes full
// details. Returns nil for types that don't form useful edges.
func extractTestType(t *doc.Type, fset *token.FileSet, dir string, funcCalls map[string][]string, funcPositions map[string]sourcePos) []Entry {
	var entries []Entry

	for _, spec := range t.Decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		name := ts.Name.Name
		if !isMockOrStubType(name) && !isTestInterface(ts) {
			continue
		}

		purpose := firstSentence(t.Doc)

		switch typ := ts.Type.(type) {
		case *ast.StructType:
			entry := extractStruct(ts, typ, purpose)
			// Find which interfaces this mock implements by looking at embedded fields
			// and the @related tag
			entry.Related = inferMockRelated(name, typ)
			if p, exists := funcPositions[name]; exists {
				entry.SourceFile = p.File
				entry.SourceLine = p.Line
			} else {
				// Types don't appear in funcPositions; get position from the AST
				pos := fset.Position(ts.Pos())
				entry.SourceFile = relPath(pos.Filename, dir)
				entry.SourceLine = pos.Line
			}
			entries = append(entries, entry)

		case *ast.InterfaceType:
			entry := extractInterface(ts, typ, purpose)
			entries = append(entries, entry)

		default:
			entries = append(entries, TypeEntry{
				Name:    name,
				Kind:    "struct",
				Purpose: purpose,
			})
		}
	}

	// Extract methods for mock types
	for _, m := range t.Methods {
		key := t.Name + "." + m.Name
		fn := extractDocFunc(m, t.Name, false)
		if fn == nil {
			// Include unexported methods on mock types too — they often implement interfaces
			fn = extractDocFunc(m, t.Name, true)
		}
		if fn == nil {
			continue
		}
		if c, exists := funcCalls[key]; exists {
			fn.Calls = c
		}
		if p, exists := funcPositions[key]; exists {
			fn.SourceFile = p.File
			fn.SourceLine = p.Line
		}
		entries = append(entries, *fn)
	}

	return entries
}

// isTestOrBenchFunc returns true for TestXxx, BenchmarkXxx, ExampleXxx, FuzzXxx functions.
func isTestOrBenchFunc(name string) bool {
	for _, prefix := range []string{"Test", "Benchmark", "Example", "Fuzz"} {
		if strings.HasPrefix(name, prefix) {
			rest := strings.TrimPrefix(name, prefix)
			// Must be followed by uppercase letter or be exactly the prefix
			if rest == "" || unicode.IsUpper(rune(rest[0])) || rest[0] == '_' {
				return true
			}
		}
	}
	return false
}

// isMockOrStubType returns true if the type name suggests it's a mock, stub, fake, or spy.
func isMockOrStubType(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"mock", "stub", "fake", "spy"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	// Also match suffixed patterns like FooMock, FooStub
	for _, suffix := range []string{"mock", "stub", "fake", "spy"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

// isTestInterface checks if a type spec is an interface (test-only interfaces
// are useful for documenting test contracts).
func isTestInterface(ts *ast.TypeSpec) bool {
	_, ok := ts.Type.(*ast.InterfaceType)
	return ok
}

// inferMockRelated tries to guess which interface a mock type implements
// based on naming conventions (e.g., mockFooService → FooService).
func inferMockRelated(name string, st *ast.StructType) []string {
	lower := strings.ToLower(name)
	var related []string

	// Strip mock/stub/fake/spy prefix (case-insensitive)
	for _, prefix := range []string{"mock", "stub", "fake", "spy"} {
		if strings.HasPrefix(lower, prefix) {
			rest := name[len(prefix):]
			if rest != "" {
				related = append(related, rest)
			}
			return related
		}
	}
	return related
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

// sourcePos holds a function's source file and line.
type sourcePos struct {
	File string
	Line int
}

// extractAllCallsAndPositions walks the raw AST BEFORE doc.New strips bodies.
// Returns two maps keyed by function name ("funcName" or "TypeName.MethodName"):
// - calls: function name → list of callee names
// - positions: function name → source file and line
func extractAllCallsAndPositions(pkg *ast.Package, fset *token.FileSet, dir string) (map[string][]string, map[string]sourcePos) {
	calls := map[string][]string{}
	positions := map[string]sourcePos{}

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			var key string
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				typeName := receiverTypeName(fn.Recv.List[0].Type)
				if typeName != "" {
					key = typeName + "." + fn.Name.Name
				}
			} else {
				key = fn.Name.Name
			}

			if key == "" {
				continue
			}

			pos := fset.Position(fn.Pos())
			positions[key] = sourcePos{
				File: relPath(pos.Filename, dir),
				Line: pos.Line,
			}

			if fn.Body != nil {
				calls[key] = extractCalls(fn)
			}
		}
	}
	return calls, positions
}

// receiverTypeName extracts the type name from a method receiver.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.IndexExpr:
		// Generic type: T[P]
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		// Generic type with multiple params: T[P, Q]
		return receiverTypeName(t.X)
	default:
		return ""
	}
}

// extractCalls walks a function body's AST to find all function/method calls.
// Returns a deduplicated, sorted list of callee names (e.g., ["Foo", "Type.Method"]).
// receiverVar is the receiver variable name (e.g., "c" for "(c *Checker)"),
// receiverType is the type name (e.g., "Checker"). When a call like c.foo()
// is found, it's mapped to Checker.foo in the output.
func extractCalls(decl *ast.FuncDecl) []string {
	// Determine receiver variable name and type for self-call mapping
	var receiverVar, receiverType string
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		if len(recv.Names) > 0 {
			receiverVar = recv.Names[0].Name
		}
		receiverType = receiverTypeName(recv.Type)
	}

	return extractCallsWithReceiver(decl, receiverVar, receiverType)
}

func extractCallsWithReceiver(decl *ast.FuncDecl, receiverVar, receiverType string) []string {
	if decl.Body == nil {
		return nil
	}

	seen := map[string]bool{}
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		var name string
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			// Simple function call: Foo()
			name = fn.Name
		case *ast.SelectorExpr:
			// Method or qualified call: obj.Method() or pkg.Func()
			name = selectorName(fn)
			// Map receiver variable to type name: c.checkExpr → Checker.checkExpr
			if receiverVar != "" && receiverType != "" && strings.HasPrefix(name, receiverVar+".") {
				name = receiverType + name[len(receiverVar):]
			}
		}

		if name != "" && !seen[name] {
			seen[name] = true
		}
		return true
	})

	if len(seen) == 0 {
		return nil
	}

	calls := make([]string, 0, len(seen))
	for name := range seen {
		calls = append(calls, name)
	}
	// Sort for deterministic output
	sortStrings(calls)
	return calls
}

// selectorName extracts "Receiver.Method" or "pkg.Func" from a SelectorExpr.
func selectorName(sel *ast.SelectorExpr) string {
	method := sel.Sel.Name
	switch x := sel.X.(type) {
	case *ast.Ident:
		// Could be: obj.Method(), pkg.Func(), or Type.StaticMethod()
		// We use the identifier name + method name
		return x.Name + "." + method
	case *ast.SelectorExpr:
		// Chained: obj.field.Method() — use the deepest selector
		return selectorName(x) + "." + method
	case *ast.CallExpr:
		// Function call result: foo().Method() — just use the method name
		return method
	default:
		// Type assertion, index, etc. — just use the method name
		return method
	}
}

// relPath returns a path relative to base, or the original if relativing fails.
func relPath(path, base string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
