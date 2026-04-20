package main

import (
	"sort"
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// extractCalls walks a function body and returns the sorted, deduplicated set
// of call targets found inside. qualifier is the owning type name for methods
// (empty for free functions); it enables resolving `self.foo()` to `T.foo`.
//
// Resolution strategy (no checker dependency — pragmatic heuristics that
// mirror aid-gen-go's behaviour):
//   - CallExpr with IdentExpr callee      → "name"
//   - CallExpr with PathExpr callee       → "pkg.name"
//   - CallExpr with FieldAccessExpr callee → receiver + "." + field, with
//     receiver-name-to-type resolution against param types and `self`.
//   - MethodCallExpr → same receiver resolution over .Object + .Method.
//
// Unresolvable targets (closures, complex expressions) are dropped rather
// than emitted with a bogus name; false positives in a call graph are worse
// than false negatives.
func extractCalls(fn *parser.FnDecl, qualifier string) []string {
	w := &callWalker{
		qualifier:  qualifier,
		paramTypes: buildParamTypes(fn, qualifier),
		localVars:  map[string]bool{},
		seen:       map[string]bool{},
	}
	w.collectLocalBindings(fn.Body)
	w.walkExpr(fn.Body)

	out := make([]string, 0, len(w.seen))
	for name := range w.seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// buildParamTypes maps each parameter name to the concrete type name it was
// annotated with (when resolvable to a named type). `self` and `mut self`
// resolve to the qualifier so receiver calls map to Type.method.
func buildParamTypes(fn *parser.FnDecl, qualifier string) map[string]string {
	m := map[string]string{}
	for _, p := range fn.Params {
		if p.Name == "self" && qualifier != "" {
			m["self"] = qualifier
			continue
		}
		if nt, ok := p.Type.(*parser.NamedTypeExpr); ok && len(nt.Path) > 0 {
			// Use the last path segment — matches how local calls reference types.
			m[p.Name] = nt.Path[len(nt.Path)-1]
		}
	}
	return m
}

type callWalker struct {
	qualifier  string
	paramTypes map[string]string // param name → concrete type name (when known)
	localVars  map[string]bool   // names bound inside the fn body (:= / for-loop)
	seen       map[string]bool
}

// collectLocalBindings pre-scans the body for VarDeclStmt and for-loop index
// names so we can later distinguish "local variable with unknown type" from
// "module / global root". Local-rooted field accesses are dropped from the
// call graph (we can't resolve them without a type checker); module-rooted
// ones are flattened into dotted names.
func (w *callWalker) collectLocalBindings(e parser.Expr) {
	if e == nil {
		return
	}
	switch v := e.(type) {
	case *parser.BlockExpr:
		for _, s := range v.Stmts {
			switch st := s.(type) {
			case *parser.VarDeclStmt:
				if st.Name != "" {
					w.localVars[st.Name] = true
				}
				w.collectLocalBindings(st.Value)
			case *parser.ForStmt:
				// Pattern-bound loop variables not tracked — would require
				// walking Pattern variants. Accept rare false positives if
				// for-loop index names appear as call receivers.
				w.collectLocalBindings(st.Iter)
				w.collectLocalBindings(st.Body)
			case *parser.WhileStmt:
				w.collectLocalBindings(st.Body)
			case *parser.LoopStmt:
				w.collectLocalBindings(st.Body)
			case *parser.ExprStmt:
				w.collectLocalBindings(st.Expr)
			case *parser.AssignStmt:
				w.collectLocalBindings(st.Value)
			case *parser.ReturnStmt:
				w.collectLocalBindings(st.Value)
			}
		}
		w.collectLocalBindings(v.Expr)
	case *parser.IfExpr:
		w.collectLocalBindings(v.Then)
		w.collectLocalBindings(v.Else)
	case *parser.MatchExpr:
		for _, arm := range v.Arms {
			w.collectLocalBindings(arm.Body)
		}
	case *parser.CatchExpr:
		w.collectLocalBindings(v.Expr)
		w.collectLocalBindings(v.Body)
	}
}

func (w *callWalker) record(name string) {
	if name == "" {
		return
	}
	w.seen[name] = true
}

// calleeName resolves a call target expression to a dotted name string.
// Returns "" if the target can't be resolved cleanly.
func (w *callWalker) calleeName(e parser.Expr) string {
	switch v := e.(type) {
	case *parser.IdentExpr:
		return v.Name
	case *parser.PathExpr:
		return strings.Join(v.Parts, ".")
	case *parser.FieldAccessExpr:
		recv := w.receiverName(v.Object)
		if recv == "" {
			return ""
		}
		return recv + "." + v.Field
	}
	return ""
}

// receiverName resolves the object side of an obj.method expression to a
// dotted name suitable as a qualifier. Handles:
//   - IdentExpr in paramTypes → mapped type (receiver variables, params typed
//     as a named type)
//   - IdentExpr not in paramTypes → the bare name itself (module alias / free
//     identifier; Aria's module namespacing often surfaces this way, e.g.
//     `std.fs.read` parses as chained FieldAccess rooted at IdentExpr("std"))
//   - PathExpr → joined path
//   - Nested FieldAccessExpr → recurse and append .field
func (w *callWalker) receiverName(obj parser.Expr) string {
	switch v := obj.(type) {
	case *parser.IdentExpr:
		if t, ok := w.paramTypes[v.Name]; ok {
			return t
		}
		if w.localVars[v.Name] {
			return "" // known local of unknown type; drop to avoid false positive
		}
		return v.Name
	case *parser.PathExpr:
		return strings.Join(v.Parts, ".")
	case *parser.FieldAccessExpr:
		inner := w.receiverName(v.Object)
		if inner == "" {
			return ""
		}
		return inner + "." + v.Field
	}
	return ""
}

// walkExpr recursively visits expression nodes, recording any call targets.
// Unknown expression kinds are ignored (they produce no calls by themselves).
func (w *callWalker) walkExpr(e parser.Expr) {
	if e == nil {
		return
	}
	switch v := e.(type) {
	case *parser.CallExpr:
		w.record(w.calleeName(v.Func))
		// Still walk the callee and args — higher-order calls may contain
		// further calls (f(g(), h)) that we want to capture.
		w.walkExpr(v.Func)
		for _, a := range v.Args {
			w.walkExpr(a.Value)
		}

	case *parser.MethodCallExpr:
		recv := w.receiverName(v.Object)
		if recv != "" {
			w.record(recv + "." + v.Method)
		}
		w.walkExpr(v.Object)
		for _, a := range v.Args {
			w.walkExpr(a.Value)
		}

	case *parser.BinaryExpr:
		w.walkExpr(v.Left)
		w.walkExpr(v.Right)
	case *parser.UnaryExpr:
		w.walkExpr(v.Operand)
	case *parser.PostfixExpr:
		w.walkExpr(v.Operand)
	case *parser.FieldAccessExpr:
		w.walkExpr(v.Object)
	case *parser.OptionalChainExpr:
		w.walkExpr(v.Object)
	case *parser.IndexExpr:
		w.walkExpr(v.Object)
		w.walkExpr(v.Index)
	case *parser.PipelineExpr:
		w.walkExpr(v.Left)
		w.walkExpr(v.Right)
	case *parser.RangeExpr:
		w.walkExpr(v.Start)
		w.walkExpr(v.End)
	case *parser.BlockExpr:
		for _, s := range v.Stmts {
			w.walkStmt(s)
		}
		w.walkExpr(v.Expr)
	case *parser.IfExpr:
		w.walkExpr(v.Cond)
		w.walkExpr(v.Then)
		w.walkExpr(v.Else)
	case *parser.MatchExpr:
		w.walkExpr(v.Subject)
		for _, arm := range v.Arms {
			w.walkExpr(arm.Body)
		}
	case *parser.ClosureExpr:
		w.walkExpr(v.Body)
	case *parser.StructExpr:
		for _, f := range v.Fields {
			w.walkExpr(f.Value)
		}
	case *parser.ArrayExpr:
		for _, el := range v.Elements {
			w.walkExpr(el)
		}
	case *parser.TupleExpr:
		for _, el := range v.Elements {
			w.walkExpr(el)
		}
	case *parser.MapExpr:
		for _, e := range v.Entries {
			w.walkExpr(e.Key)
			w.walkExpr(e.Value)
		}
	case *parser.GroupExpr:
		w.walkExpr(v.Inner)
	case *parser.CatchExpr:
		w.walkExpr(v.Expr)
		w.walkExpr(v.Body)
	case *parser.InterpolatedStringExpr:
		for _, p := range v.Parts {
			w.walkExpr(p)
		}
	case *parser.RecordUpdateExpr:
		w.walkExpr(v.Object)
		for _, f := range v.Fields {
			w.walkExpr(f.Value)
		}
	}
}

func (w *callWalker) walkStmt(s parser.Stmt) {
	if s == nil {
		return
	}
	switch v := s.(type) {
	case *parser.VarDeclStmt:
		w.walkExpr(v.Value)
	case *parser.AssignStmt:
		w.walkExpr(v.Target)
		w.walkExpr(v.Value)
	case *parser.ExprStmt:
		w.walkExpr(v.Expr)
	case *parser.ForStmt:
		w.walkExpr(v.Iter)
		w.walkExpr(v.Body)
	case *parser.WhileStmt:
		w.walkExpr(v.Cond)
		w.walkExpr(v.Body)
	case *parser.LoopStmt:
		w.walkExpr(v.Body)
	case *parser.ReturnStmt:
		w.walkExpr(v.Value)
	case *parser.DeferStmt:
		w.walkExpr(v.Expr)
	}
}
