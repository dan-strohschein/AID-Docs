// Tests GoTypeToAID: Go AST type expressions mapped to AID universal type strings.
package main

import (
	"go/ast"
	"go/parser"
	"testing"
)

// parseTypeExpr parses a Go type expression for use in GoTypeToAID tests.
func parseTypeExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("ParseExpr(%q): %v", src, err)
	}
	return expr
}

func TestGoTypeToAID_Nil(t *testing.T) {
	if got := GoTypeToAID(nil); got != "any" {
		t.Errorf("GoTypeToAID(nil) = %q, want any", got)
	}
}

func TestGoTypeToAID_PrimitivesAndIdents(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"int", "int"},
		{"string", "str"},
		{"bool", "bool"},
		{"byte", "u8"},
		{"rune", "i32"},
		{"error", "error"},
		{"any", "any"},
		{"float64", "f64"},
		{"uint32", "u32"},
		{"uintptr", "int"},
		{"MyCustomType", "MyCustomType"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got := GoTypeToAID(parseTypeExpr(t, tt.src))
			if got != tt.want {
				t.Errorf("GoTypeToAID(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestGoTypeToAID_Composite(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"*int", "int?"},
		{"[]int", "[int]"},
		{"[]byte", "bytes"},
		{"[4]int", "[int]"},
		{"map[string]int", "dict[str, int]"},
		{"map[string][]byte", "dict[str, bytes]"},
		{"chan int", "chan[int]"},
		{"context.Context", "context.Context"},
		{"time.Duration", "time.Duration"},
		{"struct{}", "struct{}"},
		{"interface{}", "any"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got := GoTypeToAID(parseTypeExpr(t, tt.src))
			if got != tt.want {
				t.Errorf("GoTypeToAID(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestGoTypeToAID_Generics(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"Pair[int]", "Pair[int]"},
		{"Pair[int, string]", "Pair[int, str]"},
		{"Map[string, []T]", "Map[str, [T]]"},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			got := GoTypeToAID(parseTypeExpr(t, tt.src))
			if got != tt.want {
				t.Errorf("GoTypeToAID(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestGoTypeToAID_FuncType(t *testing.T) {
	src := "func(context.Context, string) (int, error)"
	got := GoTypeToAID(parseTypeExpr(t, src))
	want := "fn(context.Context, str) -> (int, error)"
	if got != want {
		t.Errorf("GoTypeToAID = %q, want %q", got, want)
	}
}

func TestGoTypeToAID_ParenAndEllipsis(t *testing.T) {
	if got := GoTypeToAID(parseTypeExpr(t, "(int)")); got != "int" {
		t.Errorf("paren int = %q, want int", got)
	}
	// Ellipsis is not valid as a top-level ParseExpr; build the AST directly.
	// Current GoTypeToAID maps Ellipsis to the element type only (variadic marker omitted in AID).
	ell := &ast.Ellipsis{Elt: ast.NewIdent("string")}
	if got := GoTypeToAID(ell); got != "str" {
		t.Errorf("ellipsis Elt string = %q, want str", got)
	}
}

func TestGoTypeToAID_InterfaceWithMethods(t *testing.T) {
	// Non-empty interface: implementation maps to "any" by design.
	src := `interface{ Read([]byte) (int, error) }`
	got := GoTypeToAID(parseTypeExpr(t, src))
	if got != "any" {
		t.Errorf("GoTypeToAID(interface{{...}}) = %q, want any", got)
	}
}

func TestGoTypeToAID_UnknownASTKind(t *testing.T) {
	// Types that are not handled by the switch fall through to "any".
	expr := &ast.BadExpr{}
	if got := GoTypeToAID(expr); got != "any" {
		t.Errorf("GoTypeToAID(BadExpr) = %q, want any", got)
	}
}
