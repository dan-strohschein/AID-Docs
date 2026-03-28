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

## 24. Why source-linked claims in Layer 2?

Layer 2 adds semantic information that Layer 1 can't extract mechanically: invariants, constraints, antipatterns, workflows. The risk: an AI generating these could hallucinate constraints that don't exist. Source references (`[src: file:line]`) solve this by making every claim **verifiable**.

A claim like "BRIN is lossy, always filter results" is an assertion. `[src: planner/nodes.go:245-280]` turns it into a checkable fact — a reviewer agent reads those lines and confirms or denies. This is the key insight: **we don't need to trust the generator. We need to make its claims verifiable.**

The generator-reviewer split is intentional. One agent infers semantics (creative, exploratory). A separate agent verifies claims against source (critical, focused). They run in separate contexts so the reviewer can't be influenced by the generator's reasoning. This is the same principle as code review — the reviewer catches what the author misses.

## 25. Why `@code_version` and `@aid_status`?

AID files are living documents. When code changes, the AID must either be confirmed still-valid or updated. Without version tracking, stale AID is worse than no AID — it gives the agent false confidence in claims that are no longer true.

`@code_version git:HASH` pins the AID to a specific code state. Staleness detection becomes mechanical: if the hash doesn't match HEAD, check whether the `[src:]` references point to changed lines. If they do, the specific claims are stale. If they don't, the AID is still valid — just update the hash.

The status lifecycle (`draft → reviewed → approved → stale`) gives both agents and humans visibility into how trustworthy each AID file is. An agent can weight its confidence accordingly: `reviewed` AID is more reliable than `draft`, and `stale` AID should be treated with caution.

## 26. Why `@depends` for selective loading?

Benchmark BM2 showed that loading all 69 AID files for a 219K LoC codebase actually increased token usage by 24%. Most AID files were irrelevant to the task. `@depends` tells the agent which packages a module interacts with, so it can load only the relevant AID files.

For a task on the query planner, `@depends [syndrQL, domain/index, domain/models, storage/buffer]` reduces the AID set from 69 files to 5. This restores the token efficiency advantage seen in BM1 while keeping the architectural awareness advantage.

## 27. Why a multi-agent pipeline instead of a single generator?

Three reasons, validated by benchmarks:

1. **Hallucination detection.** A single agent generating and self-verifying has a confirmation bias — it tends to justify its own claims. A separate reviewer with a fresh context evaluates claims independently.

2. **Source-linking enables parallelism.** The reviewer only reads files referenced in `[src:]` links, not the entire codebase. Multiple reviewers can verify different AID files in parallel.

3. **Incremental updates.** When code changes, only claims with stale `[src:]` references need re-verification. The pipeline doesn't regenerate everything — it surgically updates the changed claims and re-verifies only those.

## 28. Why module-level annotations (invariants, antipatterns, decisions)?

Benchmark BM3 proved that the most valuable L2 output was module-level invariants and antipatterns — not per-entry descriptions. The statement "BRIN is lossy — always wrap in FilterNode" applies to the entire planner module, not to any single function. Without a spec-level home, L2 generators invented ad-hoc block types (global `@invariants`, `@decision`) that the parser couldn't handle.

Module annotations are a new tier because they represent a fundamentally different kind of knowledge: **cross-cutting concerns** that span multiple entries. An invariant like "ExecutionPlan is immutable after creation" constrains every function that touches ExecutionPlan, not just the constructor. Putting it on one entry would hide it from agents reading other entries.

Decision records (`@decision`) are specifically designed to prevent a failure mode we observed: agents "improving" code that was intentionally designed a certain way. When the agent reads `@chosen BTree first, BRIN fallback` with a rationale, it won't waste tokens proposing a cost-based selection that was already considered and rejected.

## 29. Why a manifest file?

Benchmark BM2 showed that loading all 69 AID files for a 219K LoC codebase increased tokens by 24% — most files were irrelevant. The `@depends` field helps with selective loading, but it requires reading the header of every file to find the dependency chain.

A manifest solves this with a single file read. The agent reads `.aidocs/manifest.aid`, scans the `@purpose` fields to identify relevant packages, follows the `@depends` chain, and loads only those AID files. For the BRIN integration task, this would have reduced the loaded AID from 69 files to 5.

The `@layer` field (l1 or l2) tells the agent what quality of documentation to expect. L2 files with workflows and invariants are worth reading carefully; L1 files with only type signatures are useful for API reference but won't provide architectural insight.

## 30. Why a formal discovery protocol?

Without a standard discovery chain, every tool and agent integration must hardcode assumptions about where AID files live. The discovery protocol (`.aidocs/` in current directory → walk up → check manifest → fallback to naming convention) is modeled after how `.git/`, `.github/`, and `.vscode/` directories are discovered.

The vendor directory (`.aidocs/vendor/`) handles the common case where a project depends on third-party libraries that have their own AID files. Rather than requiring a central registry, vendoring keeps AID files alongside the code they describe — the same philosophy as Go modules and npm.

## 31. Why security considerations in the spec?

`[src:]` references are paths that tools resolve and read. A malicious AID file could reference `../../.env` or `/etc/passwd`, causing a reviewer agent to read sensitive files and potentially expose them in its output. Path traversal prevention must be explicitly called out because the natural assumption — "it's just documentation" — understates the risk.

This is especially important because AID files can be AI-generated. If the L2 generator processes source code containing prompt injection (crafted comments or string literals), it could produce AID files with misleading claims or malicious source references. The spec must establish that AID files are trusted artifacts, not untrusted input.

## 32. Why project snapshots in the manifest?

The manifest (rationale #29) solves selective loading — reading the right AID files for a given task. But agents have a second, equally expensive problem: **re-orientation across conversations.**

Every new conversation or context window starts cold. The agent has no memory of the project's shape. It must re-read files to rebuild its understanding, even if nothing changed. For a 72-module project, this costs ~379K tokens (the full AID corpus) before the agent can do any useful work.

Project snapshots solve this in two ways:

**Shape fields** (`@shape`, `@entry_points`, `@key_types`) give the agent a compressed orientation in ~200 tokens instead of loading all AID files. An agent reading the manifest for the first time gets: what the project does, how data flows through it, which types are central, and where to start navigating. This replaces the "read everything, then figure out what matters" pattern.

**Delta fields** (`@snapshot_version`, `@delta`) enable incremental updates. An agent that previously read the project can compare its last-seen version against `@snapshot_version`. If only 2 of 72 packages changed, the agent loads only those 2 instead of all 72. For multi-conversation workflows (e.g., an agent working on a project over days), this eliminates redundant re-reading entirely.

The design keeps snapshots in the manifest rather than a separate file because the manifest is already the agent's first read. Adding snapshot fields to the manifest header means the agent gets orientation + change tracking + package index in a single file read — no additional I/O.

## 33. Why free-form `@shape` instead of structured fields?

Different projects have different architectural concepts worth surfacing. A microservices project needs to describe service boundaries and communication patterns. A CLI tool needs to describe command structure and flag parsing. A library needs to describe its public API surface and extension points.

A structured schema (e.g., `@data_flow`, `@service_boundaries`, `@api_surface`) would either be too generic to be useful or too specific to be universal. The `@shape` block uses free-form continuation lines — each line is one concept, greppable and scannable, but the concepts themselves are project-defined.

The structured fields (`@entry_points`, `@key_types`) extract the two concepts that are universal across all projects: where to start and what matters most. These are lists, not prose, because tooling can act on them programmatically (Cartograph can prioritize these nodes, IDE integrations can highlight them).

## 34. Why `@snapshot_version` uses git hashes?

The delta needs a reference point that is unambiguous, universal, and automatable. Git commit hashes satisfy all three: they uniquely identify a code state, every project uses git, and tooling can compute them without configuration.

Semver (`@version`) was considered but rejected because library versions change infrequently while code changes constantly. An agent needs to know "has anything changed since the last time I read this?" — that's a commit-level question, not a release-level question.

The format `git:SHORT_HASH` is explicit about the version control system, leaving room for future alternatives (e.g., `svn:REV`, `hg:NODE`) without ambiguity.
