// Tests AriaTypeToAID: Aria TypeExpr AST nodes mapped to AID universal type strings.
package main

import (
	"testing"

	parser "github.com/aria-lang/aria/pkg/ariaparser"
)

// --- small AST builders (avoid the full lex/parse path so these tests are
// unit tests of the mapper, not integration tests of the Aria parser) ---

func named(path ...string) *parser.NamedTypeExpr {
	return &parser.NamedTypeExpr{Path: path}
}

func namedG(path string, args ...parser.TypeExpr) *parser.NamedTypeExpr {
	return &parser.NamedTypeExpr{Path: []string{path}, TypeArgs: args}
}

// ---

func TestAriaTypeToAID_Nil(t *testing.T) {
	if got := AriaTypeToAID(nil); got != "any" {
		t.Errorf("AriaTypeToAID(nil) = %q, want any", got)
	}
}

func TestAriaTypeToAID_PrimitivesAndIdents(t *testing.T) {
	tests := []struct {
		name string
		in   parser.TypeExpr
		want string
	}{
		{"i8", named("i8"), "i8"},
		{"i64", named("i64"), "i64"},
		{"u32", named("u32"), "u32"},
		{"f64", named("f64"), "f64"},
		{"bool", named("bool"), "bool"},
		{"str", named("str"), "str"},
		{"dur", named("dur"), "dur"},
		{"byte_alias", named("byte"), "u8"},
		{"any", named("any"), "any"},
		{"user_type", named("MyType"), "MyType"},
		{"qualified", named("std", "io", "Reader"), "std.io.Reader"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AriaTypeToAID(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAriaTypeToAID_Composite(t *testing.T) {
	tests := []struct {
		name string
		in   parser.TypeExpr
		want string
	}{
		{
			"array_int",
			&parser.ArrayTypeExpr{Element: named("i64")},
			"[i64]",
		},
		{
			"array_u8_to_bytes",
			&parser.ArrayTypeExpr{Element: named("u8")},
			"bytes",
		},
		{
			"array_byte_alias_to_bytes",
			&parser.ArrayTypeExpr{Element: named("byte")},
			"bytes",
		},
		{
			"optional",
			&parser.OptionalTypeExpr{Inner: named("str")},
			"str?",
		},
		{
			"map",
			&parser.MapTypeExpr{Key: named("str"), Value: named("i64")},
			"dict[str, i64]",
		},
		{
			"map_nested",
			&parser.MapTypeExpr{Key: named("str"), Value: &parser.ArrayTypeExpr{Element: named("u8")}},
			"dict[str, bytes]",
		},
		{
			"set",
			&parser.SetTypeExpr{Element: named("str")},
			"set[str]",
		},
		{
			"tuple_pair",
			&parser.TupleTypeExpr{Elements: []parser.TypeExpr{named("i64"), named("str")}},
			"(i64, str)",
		},
		{
			"result",
			&parser.ResultTypeExpr{Ok: named("Data"), Err: named("IoError")},
			"Result[Data, IoError]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AriaTypeToAID(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAriaTypeToAID_Generics(t *testing.T) {
	tests := []struct {
		name string
		in   parser.TypeExpr
		want string
	}{
		{
			"pair_one_arg",
			namedG("Pair", named("i64")),
			"Pair[i64]",
		},
		{
			"map_two_args",
			namedG("Map", named("str"), named("i64")),
			"Map[str, i64]",
		},
		{
			"nested_generic",
			namedG("Result", named("Data"),
				namedG("List", named("Err"))),
			"Result[Data, List[Err]]",
		},
		{
			"generic_with_byte_alias_arg",
			namedG("Box", named("byte")),
			"Box[u8]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AriaTypeToAID(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAriaTypeToAID_FunctionType(t *testing.T) {
	// fn(str, i64) -> Data
	ft := &parser.FunctionTypeExpr{
		Params: []parser.TypeExpr{named("str"), named("i64")},
		Return: named("Data"),
	}
	if got, want := AriaTypeToAID(ft), "fn(str, i64) -> Data"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// fn() -> None (nil return)
	empty := &parser.FunctionTypeExpr{}
	if got, want := AriaTypeToAID(empty), "fn() -> None"; got != want {
		t.Errorf("empty fn: got %q, want %q", got, want)
	}

	// higher-order: fn(fn(i64) -> i64) -> i64
	inner := &parser.FunctionTypeExpr{Params: []parser.TypeExpr{named("i64")}, Return: named("i64")}
	outer := &parser.FunctionTypeExpr{Params: []parser.TypeExpr{inner}, Return: named("i64")}
	if got, want := AriaTypeToAID(outer), "fn(fn(i64) -> i64) -> i64"; got != want {
		t.Errorf("HOF: got %q, want %q", got, want)
	}
}

func TestAriaTypeToAID_EmptyPath(t *testing.T) {
	if got := AriaTypeToAID(&parser.NamedTypeExpr{}); got != "any" {
		t.Errorf("empty NamedTypeExpr = %q, want any", got)
	}
}

// TestAriaTypeToAID_FromSource round-trips a few Aria type syntaxes through
// the real lexer+parser to guarantee the mapper stays in sync with the grammar.
func TestAriaTypeToAID_FromSource(t *testing.T) {
	tests := []struct {
		// Wrap each type in a fn signature so we can parse it in context.
		src  string
		want string
	}{
		{"fn f(x: i64) -> i64 { x }", "i64"},
		{"fn f(x: str?) -> str { x.unwrap() }", "str?"},
		{"fn f(x: [u8]) -> i64 { 0 }", "bytes"},
		{"fn f(x: {str: i64}) -> i64 { 0 }", "dict[str, i64]"},
		{"fn f(x: (i64, str)) -> i64 { 0 }", "(i64, str)"},
		{"fn f(x: Result[Data, IoError]) -> i64 { 0 }", "Result[Data, IoError]"},
		{"fn f(x: Map[str, [u8]]) -> i64 { 0 }", "Map[str, bytes]"},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			prog := parser.Parse("test.aria", tt.src)
			if prog == nil || len(prog.Decls) == 0 {
				t.Fatalf("failed to parse: %s", tt.src)
			}
			fn, ok := prog.Decls[0].(*parser.FnDecl)
			if !ok {
				t.Fatalf("expected FnDecl, got %T", prog.Decls[0])
			}
			if len(fn.Params) == 0 {
				t.Fatalf("no params parsed in %s", tt.src)
			}
			got := AriaTypeToAID(fn.Params[0].Type)
			if got != tt.want {
				t.Errorf("got %q, want %q (src: %s)", got, tt.want, tt.src)
			}
		})
	}
}
