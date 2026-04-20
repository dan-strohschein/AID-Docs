package main

import (
	"fmt"
	"strings"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// AriaTypeToAID converts an Aria parser.TypeExpr to AID universal type notation.
//
// Aria types map to AID largely 1:1 because Aria already uses universal-style
// notation (square-bracket generics, postfix ?, fn(...) -> ..., Result[T,E]).
// The main normalisations:
//
//   - byte → u8 (byte is spec-defined as an alias)
//   - [u8] → bytes (matches aid-gen-go's []byte → bytes rule so both languages
//     emit the same AID type string for byte slices)
//   - {K: V} map literal → dict[K, V] (AID dict convention, matches Go tool)
//   - {T} set literal → set[T]
//   - nil / unknown → any
func AriaTypeToAID(expr parser.TypeExpr) string {
	if expr == nil {
		return "any"
	}

	switch t := expr.(type) {
	case *parser.NamedTypeExpr:
		return namedTypeToAID(t)

	case *parser.OptionalTypeExpr:
		return AriaTypeToAID(t.Inner) + "?"

	case *parser.ArrayTypeExpr:
		elem := AriaTypeToAID(t.Element)
		if elem == "u8" {
			return "bytes"
		}
		return "[" + elem + "]"

	case *parser.MapTypeExpr:
		return fmt.Sprintf("dict[%s, %s]", AriaTypeToAID(t.Key), AriaTypeToAID(t.Value))

	case *parser.SetTypeExpr:
		return fmt.Sprintf("set[%s]", AriaTypeToAID(t.Element))

	case *parser.TupleTypeExpr:
		elems := make([]string, len(t.Elements))
		for i, e := range t.Elements {
			elems[i] = AriaTypeToAID(e)
		}
		return "(" + strings.Join(elems, ", ") + ")"

	case *parser.FunctionTypeExpr:
		return ariaFuncTypeToAID(t)

	case *parser.ResultTypeExpr:
		return fmt.Sprintf("Result[%s, %s]", AriaTypeToAID(t.Ok), AriaTypeToAID(t.Err))

	default:
		return "any"
	}
}

func namedTypeToAID(t *parser.NamedTypeExpr) string {
	if len(t.Path) == 0 {
		return "any"
	}

	name := strings.Join(t.Path, ".")

	// Normalise primitive aliases.
	if len(t.Path) == 1 && len(t.TypeArgs) == 0 {
		if n := ariaPrimitiveAlias(t.Path[0]); n != "" {
			return n
		}
	}

	if len(t.TypeArgs) == 0 {
		return name
	}

	args := make([]string, len(t.TypeArgs))
	for i, a := range t.TypeArgs {
		args[i] = AriaTypeToAID(a)
	}
	return fmt.Sprintf("%s[%s]", name, strings.Join(args, ", "))
}

// ariaPrimitiveAlias returns the normalised AID type for Aria primitives that
// need mapping. Returns "" if the name is not a primitive alias (caller keeps
// the original name).
func ariaPrimitiveAlias(name string) string {
	switch name {
	case "byte":
		return "u8"
	// Identity primitives — listed explicitly so we can audit the set the spec
	// guarantees. Anything unknown falls through to the identifier itself.
	case "i8", "i16", "i32", "i64",
		"u8", "u16", "u32", "u64",
		"f32", "f64",
		"bool", "str", "dur",
		"any":
		return name
	default:
		return ""
	}
}

func ariaFuncTypeToAID(ft *parser.FunctionTypeExpr) string {
	params := make([]string, len(ft.Params))
	for i, p := range ft.Params {
		params[i] = AriaTypeToAID(p)
	}
	ret := "None"
	if ft.Return != nil {
		ret = AriaTypeToAID(ft.Return)
	}
	return fmt.Sprintf("fn(%s) -> %s", strings.Join(params, ", "), ret)
}
