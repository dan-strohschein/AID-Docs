package main

import (
	"fmt"
	"go/ast"
	"strings"
)

// GoTypeToAID converts a Go AST type expression to AID universal type notation.
func GoTypeToAID(expr ast.Expr) string {
	if expr == nil {
		return "any"
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return goIdentToAID(t.Name)

	case *ast.StarExpr:
		inner := GoTypeToAID(t.X)
		return inner + "?"

	case *ast.ArrayType:
		elem := GoTypeToAID(t.Elt)
		if t.Len == nil {
			// Slice
			if elem == "u8" {
				return "bytes"
			}
			return "[" + elem + "]"
		}
		// Fixed-size array — treat as list
		return "[" + elem + "]"

	case *ast.MapType:
		key := GoTypeToAID(t.Key)
		val := GoTypeToAID(t.Value)
		return fmt.Sprintf("dict[%s, %s]", key, val)

	case *ast.ChanType:
		elem := GoTypeToAID(t.Value)
		return fmt.Sprintf("chan[%s]", elem)

	case *ast.FuncType:
		return goFuncTypeToAID(t)

	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "any"
		}
		return "any" // named interfaces are handled elsewhere

	case *ast.StructType:
		return "struct{}"

	case *ast.SelectorExpr:
		// pkg.Type — e.g., context.Context, time.Duration
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name

	case *ast.Ellipsis:
		// ...T in variadic params
		return GoTypeToAID(t.Elt)

	case *ast.IndexExpr:
		// Generic type with single param: Type[T]
		base := GoTypeToAID(t.X)
		param := GoTypeToAID(t.Index)
		return fmt.Sprintf("%s[%s]", base, param)

	case *ast.IndexListExpr:
		// Generic type with multiple params: Type[K, V]
		base := GoTypeToAID(t.X)
		params := make([]string, len(t.Indices))
		for i, idx := range t.Indices {
			params[i] = GoTypeToAID(idx)
		}
		return fmt.Sprintf("%s[%s]", base, strings.Join(params, ", "))

	case *ast.ParenExpr:
		return GoTypeToAID(t.X)

	case *ast.UnaryExpr:
		return GoTypeToAID(t.X)

	default:
		return "any"
	}
}

func goIdentToAID(name string) string {
	switch name {
	case "string":
		return "str"
	case "int":
		return "int"
	case "int8":
		return "i8"
	case "int16":
		return "i16"
	case "int32":
		return "i32"
	case "int64":
		return "i64"
	case "uint":
		return "int"
	case "uint8":
		return "u8"
	case "uint16":
		return "u16"
	case "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "bool":
		return "bool"
	case "byte":
		return "u8"
	case "rune":
		return "i32"
	case "error":
		return "error"
	case "any":
		return "any"
	case "uintptr":
		return "int"
	default:
		return name
	}
}

func goFuncTypeToAID(ft *ast.FuncType) string {
	params := []string{}
	if ft.Params != nil {
		for _, field := range ft.Params.List {
			t := GoTypeToAID(field.Type)
			if len(field.Names) == 0 {
				params = append(params, t)
			} else {
				for range field.Names {
					params = append(params, t)
				}
			}
		}
	}

	ret := "None"
	if ft.Results != nil && len(ft.Results.List) > 0 {
		results := []string{}
		for _, field := range ft.Results.List {
			results = append(results, GoTypeToAID(field.Type))
		}
		if len(results) == 1 {
			ret = results[0]
		} else {
			ret = "(" + strings.Join(results, ", ") + ")"
		}
	}

	return fmt.Sprintf("fn(%s) -> %s", strings.Join(params, ", "), ret)
}
