# AID Design Rationale

**Why every design decision was made.**

---

## 1. Why plain text, not JSON/YAML/XML?

| Format | Tokens for same content | Parse complexity | Human readable |
|--------|------------------------|-----------------|----------------|
| JSON | ~1.8x baseline | Low | Poor (deeply nested) |
| YAML | ~1.3x baseline | Medium (indent-sensitive) | Good |
| XML | ~2.5x baseline | Medium | Poor (tag overhead) |
| AID | 1.0x baseline | Trivial (line-by-line) | Good |

JSON wastes tokens on structural characters: `{`, `}`, `"`, `,`, `:`. For a format whose primary consumer is paying per-token, this matters. YAML is closer but has notorious parsing edge cases (the Norway problem, implicit type coercion). XML is right out.

AID uses `@field value` syntax because:
- Zero structural overhead per field (one `@` character)
- No quoting, no braces, no commas
- Parseable with a trivial line-by-line state machine
- Every line is self-describing (the `@` prefix tells you it's a field)

## 2. Why `@` prefix for fields?

Alternatives considered:
- `field:` (YAML-style) — conflicts with parameter constraint syntax like `url: str`
- `#field` — conflicts with comments in many languages
- `[field]` (TOML-style) — implies sections, not fields
- Bare `FIELD` — hard to distinguish from content

`@` was chosen because:
- Visually distinct from content
- Universally recognized as a metadata/annotation marker
- No conflict with any common programming syntax within descriptions
- Single character — minimal token cost

## 3. Why one file per module?

**Too granular (one file per function):** An agent loading `http.get` would also need `http.post`, `Response`, `HttpError`, and the workflow. One-per-function means 10+ file reads for a single task. Token and latency cost is too high.

**Too coarse (one file per library):** A library like `pandas` would produce an AID file of 50,000+ tokens. Agents would waste most of their context loading APIs they don't need.

**Module-level is the sweet spot.** A typical module has 5-20 functions, 3-10 types, and 1-3 workflows. That fits in 1,000-3,000 tokens — small enough to load entirely, large enough to give full context for a task.

## 4. Why are workflows a separate tier?

Workflows could theoretically be inferred by an agent reading individual function entries. But in practice:

- **Inference costs tokens.** The agent has to read every function, figure out which ones relate, determine the correct order, and identify the error handling strategy. This takes 500-1,000 extra tokens per task.
- **Inference is error-prone.** Agents frequently miss ordering constraints ("must call init before use") and cleanup requirements ("must close after reading").
- **Workflows encode tribal knowledge.** "Don't read the body twice" isn't in any individual function's contract. It's an emergent property of how the stream works. Workflows capture this.

The workflow tier makes multi-step API usage a first-class concept. An agent can read a single workflow block and generate correct multi-step code without ever reading the individual function entries for simple cases.

## 5. Why require exhaustive error listings?

Current documentation typically says "raises HttpError on failure." This forces the agent to either:
1. Read the source code to find all error variants (expensive, 1,000+ tokens)
2. Guess and handle a generic error (produces vague, unhelpful error handling)
3. Generate code that handles some errors but misses others (bugs)

By requiring every error variant and its trigger condition, AID eliminates all three failure modes. The agent generates complete error handling on the first try.

The cost is that AID files are slightly larger. The savings in reduced debugging cycles and eliminated re-reads vastly outweigh this.

## 6. Why explicit constraints instead of prose descriptions?

Compare:

**Prose (current docs):**
> The timeout parameter should be a positive number representing seconds. Values that are too high may cause the connection to hang.

**AID:**
```
.timeout: Duration. Default 30s. Must be > 0. Range (0, 300].
```

The prose version is 23 tokens. The AID version is 12 tokens. But more importantly:
- The AID version is machine-parseable — a validator can check generated code against it
- The AID version is unambiguous — "too high" is subjective; `Range (0, 300]` is not
- The AID version separates the default from the constraint — the agent knows both

Constraint keywords (`Range`, `Must be`, `Must match`, `One of`, `Length`) form a small, closed vocabulary that any parser can handle.

## 7. Why `@invariants` on types?

Invariants answer the question: "What can I assume about this value?"

Without invariants, agents defensively check everything:
```
if response.headers != null {
    // use headers
}
```

With `@invariants: headers is never null`:
```
// use headers directly
```

This eliminates unnecessary null checks, defensive coding, and validation that the type system already guarantees. It also prevents agents from writing code that violates invariants (trying to construct a Response with status=0, for example).

## 8. Why `@constructors None`?

Many types in real APIs cannot be directly instantiated. `Response` is produced by `http.get()`, not by calling `Response(...)`. Database cursors are produced by queries. File handles are produced by `open()`.

Without this information, agents waste tokens trying to figure out how to construct a type, often generating invalid code like `resp = Response(status=200)`. The `@constructors` field — especially `None — produced by X` — immediately tells the agent where instances come from.

## 9. Why is `@purpose` the only universally required field?

Pragmatism. If generating AID files from existing code, some information may not be extractable:
- Type signatures might be incomplete (dynamic languages)
- Error types might not be explicit
- Thread safety might be unknown

Requiring everything would make AID files impossible to generate for many libraries. By requiring only `@purpose` (plus `@sig` for functions and `@fields`/`@variants` for types), we allow partial AID files that still provide value. The generator fills in what it can; humans and AI refine the rest.

## 10. Why not extend an existing format?

Formats considered:

| Format | Why not |
|--------|---------|
| OpenAPI/Swagger | HTTP-specific. Can't describe a general programming library. |
| JSDoc/docstring | Embedded in source code. Not language-agnostic. No workflow concept. |
| TypeScript `.d.ts` | TypeScript-specific. Types only, no constraints or workflows. |
| Protocol Buffers | Binary, requires compilation, no prose fields. |
| man pages | Human-oriented, no structure for constraints or types. |

No existing format captures the combination of: type signatures + constraints + exhaustive errors + postconditions + invariants + workflows, in a language-agnostic, token-efficient package. AID isn't a tweak to an existing format — it's a new category of document.

## 11. Why `.aidocs/` directory convention?

Following established patterns:
- `.github/` for GitHub config
- `.vscode/` for VS Code settings
- `.husky/` for git hooks

A `.aidocs/` directory:
- Is discoverable by convention (agents know where to look without configuration)
- Doesn't pollute the source tree
- Can be `.gitignore`d if generated, or committed if curated
- Works for any project regardless of language or build system

## 12. Why language-agnostic type notation?

If AID used Python types for Python libraries and Go types for Go libraries, an agent working across languages would need to understand every type system. The universal notation (`str`, `int`, `[T]`, `T?`, `T ! E`) provides a single vocabulary that maps to any language via the `@lang` field.

This also enables cross-language workflows: an agent building a Python client for a Go API can read both AID files using the same mental model.

## 13. Why dot notation for methods?

Three approaches were considered for binding methods to their parent type:

1. **Dot notation**: `@fn Response.json` — the type-method relationship is encoded in the name
2. **Explicit field**: `@fn json` with `@on Response` — adds a separate field for the binding
3. **Nesting**: methods indented under their `@type` block — creates hierarchy

Dot notation was chosen because:
- It's the most token-efficient (no extra field needed)
- It keeps entries flat (no nesting)
- The relationship is immediately visible in the entry's first line
- It matches how most languages already refer to methods (`Type.method`)
- The dot is semantically meaningful, not just a convention — `Type` must correspond to a `@type` entry

Option 2 adds a field (`@on`) that carries no information the dot doesn't already convey. Option 3 breaks the flat structure principle and makes parsing context-dependent.

## 14. Why `self` and `mut self` in method signatures?

Methods need to declare their receiver. Two questions: should the receiver be explicit, and should mutability be distinguished?

**Explicit vs. implicit receiver:** If signatures omitted `self`, the only way to know a function is a method would be the dot in `@fn Type.method`. But the signature is meant to be a complete contract — it should show all inputs including the receiver. Making `self` explicit keeps signatures self-contained.

**Mutability:** Knowing whether a method mutates its receiver is critical for generating correct code. Without it, an agent must either:
- Assume mutation is possible (overly conservative — unnecessary cloning/copying)
- Assume no mutation (optimistic — produces bugs with methods like `close()` or `sort()`)

`mut self` resolves this at near-zero token cost (one extra keyword). It maps cleanly to Rust's `&self` / `&mut self`, Python's conventions around mutating methods, Go's pointer vs. value receivers, and similar patterns across languages.

The receiver is not listed under `@params` because its type is already known from the `@fn` name, and it has no constraints to document beyond mutability.

## 15. Why slash-dot syntax for cross-module references?

When `@related` or other reference fields point to entries in other modules, a qualified name syntax is needed. Three separators were considered:

1. `module/Type` (slash + dot): `http/types.Headers`
2. `module.Type` (dot only): `http.types.Headers`
3. `module::Type` (double colon): `http/types::Headers`

Slash-dot was chosen because:
- The slash matches `@module` naming (`http/client`), so the module part of a qualified name is visually identical to how the module identifies itself
- The dot after the module path is unambiguous because module paths always contain a `/` — this distinguishes `http/types.Headers` (cross-module reference) from `Response.json` (method on a type)
- Single character separators keep token cost minimal

Bare names resolve to the current module. Qualified names are only required for cross-module references. This keeps the common case (referencing entries in the same file) free of overhead — most `@related` lists will never need qualified names.

## 16. Why include a formal parser spec?

AID's fifth design goal is parsability: "an intern should be able to write a parser in an afternoon." Including a state machine description in the format spec makes this testable rather than aspirational.

The parser has only three real states (HEADER, ENTRY, FIELD_VALUE) and five line types. Every line is classifiable by its first character(s) with no lookahead. This is deliberately simpler than YAML (which requires tracking indentation depth as state) or JSON (which requires recursive descent for nesting).

The lenient error handling rules (skip unknown fields, warn on missing required fields, don't reject on malformed lines) serve two purposes:
- **Forward compatibility:** AID spec 0.2 can add fields without breaking 0.1 parsers
- **Partial document support:** A half-generated AID file is still useful, not an error

## 17. Why inline generics in signatures?

Generic type parameters could live in a separate field (`@generic_params` on the function) or inline in the signature (`[K, V](self, key: K) -> V?`). Inline was chosen because a signature should be self-contained — it is the single most important line in any entry, and an agent reading it should not need to cross-reference another field to understand the types.

The `@generic_params` field on `@type` entries serves a different purpose: it declares the type-level parameters that all methods share. A method's `[...]` in its `@sig` can reference the parent type's parameters, introduce new ones, or both. This mirrors how generics work in every language — the type is parameterized, and individual methods can add their own parameters on top.

Bounds (`K: Hash + Eq`) are inline with the type parameter for the same reason. Separating "K is a type parameter" from "K must implement Hash" forces cross-referencing. Keeping `K: Hash + Eq` together makes the constraint immediately visible.

## 18. Why a `[Callback]` effect tag?

The existing effect categories (`[Net]`, `[Fs]`, etc.) describe what a function does. But some functions — event emitters, higher-order functions, middleware chains — have effects determined entirely by their arguments. A `sort(items, comparator)` is pure. An `emit(event, payload)` might do network I/O, filesystem writes, or nothing, depending on what listeners are registered.

Without `[Callback]`, the only options are:
1. List no effects (misleading — the function may trigger I/O)
2. List all possible effects (overly conservative — most invocations may be pure)
3. Use prose (not machine-parseable)

`[Callback]` solves this by saying: "this function's effects are determined by caller-provided functions, not by the function itself." An agent seeing `[Callback]` knows it must inspect the arguments to reason about purity. This is a fundamentally different kind of information from "this function does network I/O" and deserves its own tag rather than being lumped into an existing category.

## 19. Why `async` in the signature, not an effect tag?

Async is a **calling convention**, not a side effect. It changes how you invoke the function — you must `await` it, or `spawn` it, or handle a `Promise`/`Future`. An async function that reads from a database has effects `[Db]` and is also async. These are orthogonal.

Putting `async` in the signature keeps it visible in the single most-scanned line of any entry. An agent deciding how to call a function looks at `@sig` first. If async were buried in `@effects [Async, Net]`, the agent would have to read a second field to know the calling convention. Worse, `[Async]` in effects conflates "this function does something asynchronously" with "the caller must use async syntax to invoke it" — these aren't the same thing (a function can spawn background work without being async itself).

The `async` keyword appears before the parameter list, mirroring its position in most languages: `async def f()` (Python), `async fn f()` (Rust), `async function f()` (JavaScript).

## 20. Why support overloaded signatures?

Many languages allow a single function name to accept different argument shapes:
- Python: `@overload` decorator
- TypeScript: multiple call signatures
- C++/Java: method overloading

A single `@sig` with optional parameters handles some cases, but not when the return type or error set changes based on input types. For example, `json.loads(str) -> dict` vs. `json.loads(bytes, encoding) -> dict ! EncodingError` — the error set depends on the input type.

Multiple `@sig` lines on a single `@fn` entry capture this cleanly without duplicating the entire entry. The `@params` block documents the union of all parameters, noting which apply to which overload when ambiguous.

The alternative — separate `@fn` entries for each overload — would duplicate `@purpose`, `@effects`, `@thread_safety`, and every other shared field. Overloads are not separate functions; they're one function with multiple calling conventions.

## 21. Why `@extends` on types?

`@implements` means "this type satisfies a behavioral contract (trait/interface)" — it says what the type can do. `@extends` means "this type inherits structure and behavior from a parent" — it says what the type *is*.

This distinction matters for code generation:
- `@implements [Serializable]` → the type has `serialize()`/`deserialize()` methods
- `@extends Exception` → the type IS an Exception, can be caught as one, has all Exception fields and methods

Without `@extends`, an agent seeing `HttpError` has no way to know it can be caught with `except Exception`. It would have to guess from the name or read source code — exactly what AID exists to prevent.

For languages without class inheritance (Go, Rust), `@extends` simply isn't used. For languages with it (Python, Java, TypeScript, C++), it's essential information.

## 22. Why well-known protocols in `@implements`?

Every language has its own names for the same behavioral contracts: Python's `__enter__`/`__exit__` is Go's `io.Closer` is Rust's `Drop` is C#'s `IDisposable`. They all mean "this thing needs cleanup."

AID-universal protocol names (`Closeable`, `Iterable`, `Comparable`, etc.) let an agent pattern-match on behavior regardless of language:

- See `Closeable` → generate `with` block (Python), `defer x.Close()` (Go), `using` (C#)
- See `Iterable` → generate `for x in collection` (Python), `for _, x := range collection` (Go)
- See `Comparable` → safe to use in sorting, min/max operations

Without this, an agent must know that `__enter__` means "context manager" in Python and `io.Closer` means the same thing in Go. Universal protocol names eliminate this mapping from the agent's job and push it into tooling.

`Closeable` is the single most impactful protocol name — it's the structured answer to "must be closed after use," replacing prose in `@post` with a machine-actionable signal.

## 23. Why `@platform` for platform-specific behavior?

Some APIs behave differently on different operating systems. Path separators, file permissions, maximum path lengths, signal handling, process management — these differences cause subtle bugs that are invisible in documentation that doesn't mention them.

An agent generating cross-platform code needs to know:
- Does this function exist on all platforms?
- Does it behave the same on all platforms?
- What are the platform-specific constraints?

`@platform` provides this as structured data rather than burying it in prose descriptions. Each line names a platform and describes only the divergent behavior, keeping the common-case documentation clean.

Platform differences are particularly dangerous because they produce code that works on the developer's machine but fails in production (or vice versa). An agent that knows `@platform windows: Max path 260 chars` can proactively add path length validation that a platform-unaware agent would miss.
