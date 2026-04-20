# aid-gen-aria

Layer-1 AID generator for the Aria programming language. Parses `.aria`
source and emits mechanical `.aid` skeletons compatible with the AID v0.2
specification. Behaves identically to `aid-gen-go` from a CLI and output
perspective — the only thing that differs is the input language.

## Build

```sh
cd tools/aid-gen-aria
go build -o aid-gen-aria .
```

The tool imports the Aria compiler's parser via a public shim package at
`github.com/aria-lang/aria/pkg/ariaparser`. A `replace` directive in
`go.mod` points at the local Aria compiler checkout — update the path if
you clone the compiler elsewhere.

## Usage

```
aid-gen-aria [flags] <package-dir> [package-dir...]
```

### Flags

| Flag | Description |
|------|-------------|
| `-output <dir>` | Output directory for `.aid` files (default `.aidocs`) |
| `-stdout` | Print to stdout instead of writing files |
| `-module <name>` | Override the module name (default: derived from dir) |
| `-version <v>` | Library version for the AID `@version` header (default `0.0.0`) |
| `-v` | Verbose progress to stderr |
| `-internal` | Include unexported fns as minimal entries (name + sig only) |
| `-all` | Emit every declaration with full detail regardless of `pub`. Use for code that predates `pub` annotations, e.g. the current Aria stdlib. |
| `-test` | Emit a separate `<module>_test.aid` with Mock/Stub/Fake/Spy types, test helpers, and one synthetic entry per `test "…" { }` block (carrying its call graph). |
| `-per-file` | Emit one `.aid` per `.aria` file instead of one per directory. Use for dirs containing multiple top-level modules, e.g. `/aria/lib/`. |

Supports `./...` glob for recursive directory discovery.

## Output conventions

- One `.aid` per module, written as `<modname>.aid` in `-output`.
- When `-test` is set, test scaffolding goes to `<modname>_test.aid`.
- Module names containing `/` are written with `-` substitution.
- Header always emits `@aid_version 0.2`.

## Aria-specific AID extensions

Two spec extensions were added in `spec/fields.md` for this generator:

1. **`Async` and `Ffi` added to `@effects` vocabulary** (spec/fields.md § "Effect categories"). Aria's `with [Async, Ffi]` clauses now round-trip through AID cleanly.
2. **`@error_categories`** on `@type` entries. Populated by detecting `impl Transient|Permanent|UserFault|SystemFault|Retryable for <ErrorType>` in the source.

Both are purely additive; older AID consumers see them as optional fields and ignore them.

## Parity with aid-gen-go

All six Go fixtures (`basic`, `callchain`, `errors`, `generics`, `interfaces`, `testpkg`) have Aria-equivalent fixtures under `testdata/`. Structural parity — not byte-exact output — is asserted in `fixtures_test.go`: same entry counts, same fn/type names, same call-graph shape, same trait/impl edges. The two generators produce legitimately different AID (Aria emits `@effects` and `@error_categories`; Go doesn't), so byte-exact comparison would be misleading.

## Known upstream Aria-parser issues

Discovered while running over the real Aria compiler + stdlib:

1. **`expect()` doesn't advance the cursor on unexpected tokens**, which sends the parser into an infinite loop on malformed input (e.g. `pub` on an impl-block method). Worked around in `aid-gen-aria` by requiring callers that run over the full compiler to use a timeout. (Filed upstream in `BUG_parser_expect_infinite_loop.md`.)
2. **`mod test` conflicts with the `test` keyword** — the parser mis-parses `/aria/lib/test.aria` as a single `TestBlock` instead of a `ModDecl` followed by four `FnDecl`s. The generated `test.aid` is therefore empty. Fix needed upstream in the parser.
3. **`Spawn`/`Scope`/`Select` tokens are lexer-only** — the parser doesn't yet produce AST nodes for structured-concurrency expressions. `Async` effect inference from body walks is deferred until those nodes land; the explicit `with [Async]` clause still works.

None of these are generator bugs; the tool handles each gracefully and produces valid (if sometimes empty) output.

## Reproducing the Aria compiler + stdlib AID

```sh
AID_OUT=/path/to/aria/aria/.aidocs

# Stdlib — 11 modules, one file each
aid-gen-aria -all -per-file -v -output "$AID_OUT" -version 0.1.0 /path/to/aria/aria/lib

# Compiler packages — one dir per module
for pkg in lexer parser checker codegen resolver diagnostic; do
  aid-gen-aria -all -test -v -module "$pkg" -output "$AID_OUT" -version 0.1.0 /path/to/aria/aria/src/$pkg
done

# main.aria (standalone)
aid-gen-aria -all -per-file -v -output "$AID_OUT" -version 0.1.0 /path/to/aria/aria/src
```

Produces 24 `.aid` files (11 stdlib + 7 compiler packages + 6 compiler `_test.aid`) totalling ~12k lines.
