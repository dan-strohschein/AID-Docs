# AID Format Specification v0.1

**The complete specification for the Agent Interface Document format.**

---

## Table of Contents

1. [Overview](#1-overview)
2. [File structure](#2-file-structure)
3. [Tier 1: Module header](#3-tier-1-module-header)
4. [Tier 2: Entries](#4-tier-2-entries)
5. [Tier 2.5: Module annotations](#5-tier-25-module-annotations)
6. [Tier 3: Workflows](#6-tier-3-workflows)
7. [Syntax rules](#7-syntax-rules)
8. [Parsing](#8-parsing)
9. [Versioning](#9-versioning)
10. [File organization](#10-file-organization)
11. [Examples](#11-examples)
12. [Security considerations](#12-security-considerations)
13. [Migration](#13-migration)

---

## 1. Overview

An AID file (`.aid`) is a plain-text document that describes the API surface of a software module in a format optimized for consumption by AI coding agents. It is structured, token-efficient, and eliminates all prose in favor of named fields with defined semantics.

AID files are the source of truth for what an AI agent needs to know to use an API correctly: complete type signatures, exhaustive error conditions, parameter constraints, postconditions, invariants, thread safety guarantees, and multi-step workflow patterns.

### 1.1 Goals

- **Token efficiency** — an agent should be able to consume a full module's API in under 2,000 tokens for typical modules
- **Completeness** — every piece of information needed to use an API correctly is present, not implied
- **Unambiguity** — every field has a single defined meaning; no natural language interpretation required
- **Universality** — language-agnostic; works for Python, Go, TypeScript, Rust, Aria, or any other language
- **Parsability** — trivially parseable with line-by-line processing; no complex grammar required

### 1.2 Non-goals

- Replacing human-readable documentation (AID supplements it; humans can read AID but it's not optimized for them)
- Encoding implementation details (AID describes contracts, not internals)
- Full formal verification (constraints are descriptive, not a proof system)

---

## 2. File structure

An AID file consists of three tiers, separated by `---` (horizontal rule) delimiters:

```
[Module Header]
---
[Entry 1]
---
[Entry 2]
---
...
---
[Workflow 1]
---
[Workflow 2]
```

### 2.1 Ordering

1. Module header comes first (exactly one)
2. Entries follow (zero or more), in any order
3. Workflows come last (zero or more)

Entries and workflows are each separated by `---` on its own line.

### 2.2 Encoding

- UTF-8
- Lines terminated by `\n` (LF) or `\r\n` (CRLF)
- No BOM
- No trailing whitespace is significant

---

## 3. Tier 1: Module header

The module header identifies the module and provides top-level metadata.

### 3.1 Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@module` | Yes | Fully qualified module name (e.g., `http/client`, `os.path`, `@org/package`) |
| `@lang` | Yes | Source language (`python`, `go`, `typescript`, `rust`, `aria`, etc.) |
| `@version` | Yes | Semantic version of the library/module this AID file describes |
| `@stability` | No | One of: `experimental`, `unstable`, `stable`, `deprecated`. Default: `stable` |
| `@purpose` | Yes | One-line description of what the module does. Max 120 characters. |
| `@deps` | No | Comma-separated list of modules this module depends on |
| `@depends` | No | Packages this module calls into (for selective AID loading) |
| `@source` | No | URL to source repository or documentation |
| `@code_version` | No | Git commit hash or tag identifying the code version this AID describes. Format: `git:HASH` |
| `@aid_status` | No | Document lifecycle status: `draft`, `reviewed`, `approved`, `stale`. Default: `draft` |
| `@aid_generated_by` | No | Agent role that produced this AID (e.g., `layer1-extractor`, `layer2-generator`) |
| `@aid_reviewed_by` | No | Agent role that verified this AID (e.g., `layer2-reviewer`) |
| `@aid_version` | No | Version of the AID spec this file conforms to. Default: latest |

### 3.2 Example

```
@module http/client
@lang python
@version 2.31.0
@stability stable
@purpose HTTP client library for making web requests and handling responses
@deps [ssl, dns, url, io]
@source https://github.com/psf/requests
@aid_version 0.1
```

---

## 4. Tier 2: Entries

Entries describe individual API elements. There are four entry types: `@fn`, `@type`, `@trait`, and `@const`.

### 4.1 Function entries (`@fn`)

Functions are the most common entry type and carry the most information.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@fn` | Yes | Function or method name. Methods use dot notation: `Type.method` |
| `@purpose` | Yes | One-line description. Max 120 characters. |
| `@sig` | Yes | Full type signature |
| `@params` | No* | Parameter descriptions with constraints. *Required if function has parameters. |
| `@returns` | No | Description of return value and its guarantees |
| `@errors` | No* | Exhaustive list of error types and conditions. *Required if function can error. |
| `@pre` | No | Preconditions that must hold before calling |
| `@post` | No | Postconditions guaranteed after successful return |
| `@effects` | No | Side effects: `[Net]`, `[Fs]`, `[Io]`, `[Env]`, `[Time]`, `[Random]`, `[Callback]`, etc. |
| `@thread_safety` | No | Thread/concurrency safety guarantees |
| `@complexity` | No | Time and/or space complexity |
| `@since` | No | Version when this function was introduced |
| `@deprecated` | No | Deprecation notice with replacement |
| `@related` | No | Comma-separated names of related entries |
| `@platform` | No | Platform-specific behavior notes (see Platform section) |
| `@example` | No | Minimal usage example (as few lines as possible) |

#### Signature syntax

Signatures use a universal notation regardless of source language:

```
(param: Type, param?: OptionalType, ...rest: Type) -> ReturnType
(param: Type) -> ReturnType ! ErrorType
(param: Type) -> ReturnType ! ErrorA | ErrorB
[T](param: T) -> T                          // generic function
[K: Hash + Eq, V](self, key: K) -> V?       // generic method with bounds
async (url: str) -> Response ! HttpError     // async function (must be awaited)
```

- `?` after param name means optional (has default)
- `...` before param name means variadic/rest
- `!` separates return type from error types
- `|` separates multiple error types
- `-> None` for functions that return nothing
- `self` as first parameter means this is a method (immutable receiver)
- `mut self` as first parameter means this method mutates its receiver
- `[T, U]` before params declares generic type parameters for this function/method
- `T: Bound` constrains a type parameter to types implementing a trait/interface
- `T: A + B` applies multiple bounds with `+`
- `async` before params means the function is asynchronous and must be awaited/spawned

#### Overloaded signatures

When a function accepts different argument shapes and returns different types, list multiple `@sig` lines:

```
@fn parse
@purpose Parse a value from a string or bytes
@sig (input: str) -> Value ! ParseError
@sig (input: bytes, encoding?: str) -> Value ! ParseError | EncodingError
```

Each `@sig` is a valid calling convention. The `@params` block should document the union of all parameters across all signatures, noting which parameters apply to which overload if ambiguous.

Overloads are distinct from optional parameters. Use optional parameters (`param?: Type`) when the same return type applies regardless. Use overloads when different input types produce different return types or error sets.

#### Parameter constraint syntax

Parameters are described under `@params` with optional constraints:

```
@params
  name: Description. Constraint. Default value.
  name: Description.
    .subfield: Description. Constraint. Default value.
```

Constraint formats:
- `Range [min, max]` — inclusive range
- `Range (min, max)` — exclusive range
- `Range [min, max)` — half-open range
- `Must be > 0` / `Must be >= 0` / `Must be != null` — simple comparisons
- `Must match ^regex$` — pattern constraint
- `One of: value1, value2, value3` — enumerated values
- `Length [min, max]` — length constraint for strings/collections
- `Required.` — parameter is not optional

#### Error listing syntax

Errors are listed under `@errors`, one per line:

```
@errors
  ErrorType.Variant — condition that triggers this error
  ErrorType — condition (when no variants)
```

Every possible error the function can produce must be listed. This is the exhaustive contract.

#### Full function example

```
@fn get
@purpose Perform an HTTP GET request and return the response
@sig (url: str, opts?: RequestOpts) -> Response ! HttpError | TimeoutError
@params
  url: Full URL including scheme. Must match ^https?://. Required.
  opts: Request configuration.
    .timeout: Duration. Default 30s. Must be > 0.
    .redirects: int. Default 5. Range [0, 20].
    .headers: dict[str, str]. Default empty.
    .verify_ssl: bool. Default true.
@returns Response with status code, headers, and body stream
@errors
  HttpError.DnsFailure — url domain cannot be resolved
  HttpError.ConnectionRefused — server not accepting connections on port
  HttpError.TlsError — certificate validation failed (only when verify_ssl=true)
  HttpError.NetworkError — connection dropped during transfer
  TimeoutError — no response headers received within opts.timeout
@pre None
@post Response.body is open. Caller must call body.close() or read to completion.
@effects [Net]
@thread_safety Safe. Each call is independent. No shared mutable state.
@complexity O(1) local setup. Network-bound.
@since 1.0.0
@related post, put, delete, request, Response, RequestOpts
@example
  resp := http.get("https://api.example.com/users")?
  data := resp.json[[]User]()?
```

### 4.1.1 Method entries

Methods are functions that belong to a type. They follow all the same rules as `@fn` entries with these additional conventions:

#### Naming

Methods use **dot notation**: `@fn Type.method`. The dot is semantically meaningful — `Type` must refer to a `@type` entry in the same module, and `method` must appear in that type's `@methods` list.

#### Receiver (`self`)

Methods declare their receiver as the first parameter in `@sig`:

- `(self)` — immutable receiver. The method reads but does not modify the instance.
- `(mut self)` — mutable receiver. The method modifies the instance in place.

The receiver is not listed under `@params` (it has no constraints to document — its type is already known from the `@fn` name).

```
// Immutable — reading body does not modify Response (it consumes the stream)
@fn Response.is_ok
@sig (self) -> bool

// Mutable — close changes internal state
@fn Connection.close
@sig (mut self) -> None

// Method with additional parameters
@fn Headers.get
@sig (self, name: str) -> str?
@params
  name: Header name. Case-insensitive.
```

#### Relationship to `@type`

A method entry `@fn T.m` and a type entry `@type T` are linked:

- `T.m` must appear in `T`'s `@methods` list
- `T`'s `@methods` list should include every method that has a `@fn T.m` entry
- If a method is listed in `@methods` but has no `@fn` entry, it is acknowledged but undocumented (partial docs are valid)

---

### 4.2 Type entries (`@type`)

Type entries describe structs, classes, enums, unions, or any named data type.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@type` | Yes | Type name |
| `@kind` | Yes | One of: `struct`, `enum`, `union`, `class`, `alias`, `newtype` |
| `@purpose` | Yes | One-line description. Max 120 characters. |
| `@fields` | No* | Field names, types, and constraints. *Required for struct/class. |
| `@variants` | No* | Variant names and payloads. *Required for enum/union. |
| `@invariants` | No | Properties that always hold for valid instances of this type |
| `@constructors` | No | How instances are created. `None` if only produced by other functions. |
| `@methods` | No | Comma-separated list of method names (details in separate @fn entries) |
| `@extends` | No | Parent class/type this type inherits from. Single or comma-separated for multiple inheritance. |
| `@implements` | No | Traits/interfaces this type implements |
| `@generic_params` | No | Type parameters with bounds |
| `@platform` | No | Platform-specific behavior notes (see Platform section) |
| `@since` | No | Version introduced |
| `@deprecated` | No | Deprecation notice |
| `@related` | No | Related types and functions |
| `@example` | No | Construction and basic usage |

#### Fields syntax

```
@fields
  name: Type — description. Constraint.
  name: Type — description. Default value.
```

#### Variants syntax (for enums/unions)

```
@variants
  | VariantName — description
  | VariantName(PayloadType) — description
  | VariantName { field: Type, field: Type } — description
```

#### Full type example

```
@type Response
@kind struct
@purpose HTTP response with status code, headers, and body stream
@fields
  status: int — HTTP status code. Range [100, 599].
  headers: Headers — Response headers. Always present, may be empty.
  body: BodyStream — Readable stream. Must be closed when done.
  url: str — Final URL after all redirects.
  elapsed: Duration — Time from request sent to headers received.
@invariants
  - status is always in valid HTTP range [100, 599]
  - headers is never null/None
  - body is open on construction; closed after .close() or full read
  - url always contains scheme and host
@constructors None — produced by http.get(), http.post(), and other request functions
@methods json, text, bytes, close, raise_for_status
@implements [Display, Debug]
@related Headers, BodyStream, http.get, http.post
```

#### Enum/union example

```
@type HttpError
@kind enum
@purpose Errors that can occur during HTTP operations
@variants
  | DnsFailure { domain: str } — could not resolve domain name
  | ConnectionRefused { host: str, port: int } — server rejected connection
  | TlsError { reason: str } — TLS/SSL handshake or verification failed
  | NetworkError { message: str } — connection lost during transfer
  | InvalidUrl { url: str } — URL is malformed or unsupported scheme
@implements [Error, Display, Debug]
@related TimeoutError, Response
```

### 4.3 Trait/interface entries (`@trait`)

Trait entries describe interfaces, protocols, or abstract contracts.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@trait` | Yes | Trait/interface name |
| `@purpose` | Yes | One-line description |
| `@requires` | Yes | Method signatures that implementors must provide |
| `@provided` | No | Methods with default implementations |
| `@implementors` | No | Known types that implement this trait |
| `@extends` | No | Parent traits this trait extends |
| `@related` | No | Related traits and types |

#### Example

```
@trait Serializable
@purpose Type can be converted to and from a wire format
@requires
  fn serialize(self) -> bytes ! SerializeError
  fn deserialize(data: bytes) -> Self ! DeserializeError
@provided
  fn to_json(self) -> str ! SerializeError
  fn from_json(data: str) -> Self ! DeserializeError
@implementors [str, int, float, bool, List, Map, DateTime]
@related SerializeError, DeserializeError
```

### 4.4 Constant entries (`@const`)

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@const` | Yes | Constant name |
| `@purpose` | Yes | One-line description |
| `@type` | Yes | The constant's type |
| `@value` | No | The constant's value (if publicly known/useful) |
| `@since` | No | Version introduced |

#### Example

```
@const MAX_REDIRECTS
@purpose Maximum number of HTTP redirects before aborting
@type int
@value 30
```

---

## 5. Tier 2.5: Module annotations

Module annotations are semantic blocks that apply to the module as a whole — not to any individual function, type, or workflow. They capture cross-cutting concerns: invariants that span multiple entries, architectural decisions, common mistakes, and free-form notes.

Module annotations are separated by `---` like entries. They appear between entries and workflows in the file.

### 5.1 Invariants block (`@invariants`)

Module-level constraints that hold across the entire module:

```
@invariants
  - BRIN indexes are lossy: results include false positives from matching page ranges.
    Any query using a BRIN index MUST have a downstream FilterNode. [src: planner/nodes.go:1215-1238]
  - Index selection priority: hash → btree → brin → full scan [src: planner/query_router.go:835-1083]
  - ExecutionPlan is immutable after creation [src: planner/planner.go:111-112]
```

Each invariant is a bulleted line (prefixed with `- `) under `@invariants`. Source references (`[src:]`) are strongly recommended — they make the claim verifiable.

### 5.2 Antipatterns block (`@antipatterns`)

Module-level mistakes to avoid:

```
@antipatterns
  - Returning BRINScanNode without FilterNode wrapping produces incorrect results.
    [src: planner/query_router.go:1003-1022]
  - Assuming BTreeOrderedScanNode eliminates the need for SortNode. B-tree keys use
    ASCII encoding where "10" < "9". [src: planner/plan_builder.go:160-165]
```

### 5.3 Decision records (`@decision`)

Architectural decision records explain WHY the code is structured a certain way. These prevent agents from "improving" code that was designed a specific way for a reason.

```
@decision index_selection_order
@purpose Why BTree is checked before BRIN in the planner
@context Both index types can serve range queries on the same field
@chosen BTree first, BRIN as fallback when no BTree exists
@rejected Cost-based selection between both; BRIN first (cheaper I/O)
@rationale BTree gives exact results (no false positives) and avoids the FilterNode
  overhead required by BRIN. For the common case where a BTree exists, this is always
  faster. BRIN is only worth considering when no BTree covers the field.
  [src: src/internal/query/planner/query_router.go:973-1022]
```

| Field | Required | Description |
|-------|----------|-------------|
| `@decision` | Yes | Decision name (snake_case) |
| `@purpose` | Yes | What question this answers |
| `@context` | No | Constraints that existed when the decision was made |
| `@chosen` | Yes | What was chosen |
| `@rejected` | No | What was considered and rejected |
| `@rationale` | Yes | Why, with `[src:]` references |

### 5.4 Notes (`@note`)

Free-form annotations for deprecation notices, migration notes, TODOs, and other module-level information:

```
@note adapter-deprecation
@purpose ExpressionAdapter is a migration bridge — new code should use Expression AST directly
  [src: syndrQL/expression_adapter.go:29-31]

@note future-helpers
@purpose Planned additions to expression_helpers.go: ExtractLIKEPattern, ExtractINList
  [src: syndrQL/expression_helpers.go:214-216]
```

| Field | Required | Description |
|-------|----------|-------------|
| `@note` | Yes | Note name (descriptive identifier) |
| `@purpose` | Yes | What this note communicates |

---

## 6. Tier 3: Workflows

Workflows describe how multiple entries work together to accomplish a task. This is the tier that has no equivalent in any existing documentation format.

### 5.1 Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@workflow` | Yes | Workflow name (snake_case) |
| `@purpose` | Yes | What this workflow accomplishes |
| `@steps` | Yes | Numbered sequence of operations |
| `@errors_at` | No | Which errors can occur at which steps |
| `@antipatterns` | No | Common mistakes to avoid |
| `@variants` | No | Alternative paths through the workflow |
| `@example` | No | Complete example showing the full workflow |

### 5.2 Steps syntax

```
@steps
  1. Label: function_or_operation — description
  2. Label: function_or_operation — description
  3. Label: function_or_operation — description
```

Steps are numbered sequentially. Each step has a short label, the function/operation involved, and a description of what happens.

### 5.3 Errors-at syntax

```
@errors_at
  step 2: ErrorType — condition
  step 3: ErrorType — condition
```

Maps errors to specific workflow steps so the agent knows exactly where error handling is needed.

### 5.4 Variants syntax

```
@variants
  - streaming: Replace step 3 with resp.stream() for chunked processing
  - async: Wrap steps 2-4 in spawn for concurrent execution
```

### 5.5 Full workflow example

```
@workflow http_request_lifecycle
@purpose Make an HTTP request, process the response, and clean up resources
@steps
  1. Configure: RequestOpts{} — set timeout, headers, redirects, auth
  2. Execute: http.get(url, opts) — sends request, returns Response
  3. Validate: resp.raise_for_status() — throws if status >= 400
  4. Consume: resp.json[T]() or resp.text() or resp.bytes() — parse body
  5. Cleanup: resp.body.close() — release connection (automatic if step 4 reads fully)
@errors_at
  step 2: HttpError | TimeoutError — network or server failure
  step 3: HttpStatusError — server returned error status code
  step 4: ParseError — body doesn't match expected format
@antipatterns
  - Don't read body twice. The stream is consumed on first read.
  - Don't skip close(). It leaks a connection from the pool.
  - Don't ignore raise_for_status(). A 404 response is not an error by default.
@variants
  - streaming: Replace step 4 with resp.stream() -> Iterator[bytes] for large responses
  - retry: Wrap steps 2-4 in retry loop with exponential backoff for transient errors
@example
  opts := RequestOpts{timeout: 10s, headers: {"Authorization": "Bearer " + token}}
  resp := http.get("https://api.example.com/data", opts)?
  resp.raise_for_status()?
  data := resp.json[MyData]()?
  // body auto-closed after full read
```

---

## 7. Syntax rules

### 6.1 Field syntax

All fields start with `@` at the beginning of a line:

```
@fieldname value
```

Or for multi-line fields:

```
@fieldname
  indented content line 1
  indented content line 2
```

Multi-line field content is indented by 2 spaces. The field ends when the next `@field`, `---`, or end-of-file is encountered.

### 6.2 Comments

Lines starting with `//` are comments and are ignored by parsers:

```
// This is a comment
@fn get
// TODO: verify error list is exhaustive
@errors
  ...
```

### 6.3 Entry separators

Entries are separated by `---` on its own line (no leading/trailing whitespace).

### 6.4 Inline descriptions

Within field values, `—` (em dash) separates a name from its description:

```
  HttpError.DnsFailure — domain cannot be resolved
  status: int — HTTP status code. Range [100, 599].
```

### 6.5 Source references

Layer 2 (AI-generated) semantic claims must be linked to the source code that supports them using `[src:]` references:

```
@invariants
  - BRIN is a lossy index. Results must be filtered. [src: planner/nodes.go:245-280]
  - Indexes are checked in order: hash, btree, brin, full scan. [src: planner/query_router.go:950-1080]

@antipatterns
  - Don't return BRINScanNode without FilterNode. [src: planner/nodes.go:250]
```

Source reference syntax:
- `[src: file:LINE]` — single line
- `[src: file:START-END]` — line range
- `[src: file:LINE, other_file:LINE]` — multiple locations

Paths are relative to the project root. Line numbers reference the code version in `@code_version`. Source references enable **mechanical verification** — a reviewer agent reads the referenced code and confirms the claim.

### 6.6 Lists

Lists within fields use comma-separated values in brackets:

```
@deps [ssl, dns, url]
@effects [Net, Fs]
@implements [Display, Debug, Clone]
```

### 6.7 Cross-module references

Any field that references another entry (`@related`, `@deps`, `@implements`, `@extends`, `@implementors`, `@constructors`) supports both bare and qualified names.

**Bare names** resolve to entries within the current module:

```
@related post, Response, RequestOpts
```

**Qualified names** use the module path followed by a dot and the entry name, for referencing entries in other modules:

```
@related crypto/tls.TlsConfig, http/middleware.Middleware
```

The module path uses the same format as `@module` (slash-separated). The dot after the module path separates the module from the entry name. This does not conflict with method dot notation because method names always appear as `Type.method` (no slashes), while cross-module references always contain a `/`.

Qualified names are only required for cross-module references. Bare names always resolve to the current module.

```
// Same module — bare names
@related get, post, Response

// Cross-module — qualified names
@related http/types.Headers, crypto/tls.Certificate

// Mixed
@related get, post, http/types.Headers
```

### 6.8 Sub-fields

Nested properties within parameters use `.` prefix with additional indentation:

```
@params
  opts: Request configuration.
    .timeout: Duration. Default 30s. Must be > 0.
    .redirects: int. Default 5. Range [0, 20].
```

### 6.9 Type notation

AID uses a universal type notation that maps to any source language:

| AID notation | Meaning |
|---|---|
| `str` | String |
| `int` | Integer (platform-width) |
| `i32`, `i64` | Sized integers |
| `float`, `f32`, `f64` | Floating point |
| `bool` | Boolean |
| `bytes` | Byte sequence |
| `None` | No value / void / unit |
| `[T]` | List/array of T |
| `dict[K, V]` | Map/dictionary |
| `set[T]` | Set |
| `T?` | Optional (may be absent) |
| `T ! E` | Result: T on success, E on error |
| `fn(A, B) -> C` | Function type |
| `(A, B)` | Tuple |

These are AID-universal types. The `@lang` field in the header tells tooling how to map them to language-specific types.

### 6.10 Inheritance (`@extends`)

Types that inherit from a parent class use `@extends` to declare the relationship:

```
@type HttpError
@kind class
@extends Exception
```

`@extends` means the type inherits all fields, methods, and behavior from the parent. This is distinct from `@implements`, which means the type satisfies a trait/interface contract without inheriting implementation.

For multiple inheritance (Python, C++):

```
@extends BaseA, BaseB
```

For languages without class inheritance (Go, Rust), `@extends` is not used. Use `@implements` for interface/trait satisfaction and composition for embedding.

### 6.11 Platform-specific behavior (`@platform`)

When a function or type behaves differently across operating systems or platforms, use `@platform` to document the differences:

```
@platform
  windows: Uses backslash path separators. Max path 260 chars unless long path enabled.
  linux: Uses forward slash. Max path 4096 chars.
  macos: Uses forward slash. Case-insensitive filesystem by default.
```

Each line under `@platform` names a platform and describes the divergent behavior. Only document what differs — shared behavior belongs in the regular fields.

Platform names use lowercase: `windows`, `linux`, `macos`, `bsd`, `wasm`, `android`, `ios`.

If a function is only available on certain platforms:

```
@platform
  windows: Not available. Use win_specific_fn instead.
  linux: Available.
  macos: Available.
```

### 6.12 Well-known protocols

The `@implements` field accepts both language-specific names and AID-universal protocol names. Universal protocol names describe behavioral contracts that exist across languages under different names:

| AID protocol | Meaning | Python | Go | TypeScript | Rust |
|-------------|---------|--------|----|------------|------|
| `Closeable` | Has cleanup method. Supports resource management syntax. | `__enter__`/`__exit__` (context manager) | `io.Closer` / `defer` | `Disposable` / `using` | `Drop` |
| `Iterable` | Can produce an iterator over elements. | `__iter__` | `range` pattern | `Symbol.iterator` | `IntoIterator` |
| `Iterator` | Stateful cursor that yields elements one at a time. | `__next__` | N/A (use channels) | `next()` protocol | `Iterator` |
| `Comparable` | Supports ordering (`<`, `>`, `<=`, `>=`). | `__lt__` etc. | `sort.Interface` | N/A | `Ord` |
| `Hashable` | Can be used as a hash map key. | `__hash__` | comparable | N/A | `Hash` |
| `Display` | Has a human-readable string representation. | `__str__` | `fmt.Stringer` | `toString()` | `Display` |
| `Debug` | Has a debug/developer string representation. | `__repr__` | `fmt.GoStringer` | `inspect()` | `Debug` |
| `Serializable` | Can convert to/from wire format. | various | `json.Marshaler` | various | `Serialize` |
| `Cloneable` | Can produce an independent copy. | `__copy__`/`__deepcopy__` | N/A (value types) | `structuredClone` | `Clone` |
| `Callable` | Can be invoked as a function. | `__call__` | N/A | call signature | `Fn`/`FnMut` |

When a type implements `Closeable`, an agent knows to use the language-appropriate resource management syntax (`with` in Python, `defer` in Go, `using` in TypeScript/C#). This is the structured way to express "must clean up after use."

```
@type Connection
@kind class
@implements [Closeable, Debug]
// Agent knows: use `with Connection(...) as conn:` in Python
```

---

## 8. Parsing

An AID parser is a line-by-line state machine. No lookahead, no backtracking, no context-dependent rules. Each line is classified by its prefix, and the parser transitions between states accordingly.

### 8.1 Line classification

Every line in an AID file is exactly one of these types:

| Line type | Rule | Example |
|-----------|------|---------|
| **Field** | Starts with `@` | `@fn get` |
| **Continuation** | Starts with 2+ spaces (and no `@`) | `  url: Full URL. Required.` |
| **Separator** | Is exactly `---` | `---` |
| **Comment** | Starts with `//` | `// TODO: verify errors` |
| **Blank** | Empty or whitespace-only | |

No line can be ambiguous — the first character(s) determine its type.

### 8.2 Parsing rules

1. **Read line by line.** Trim trailing whitespace. Classify each line by its prefix.
2. **Skip comments and blanks.** They carry no semantic content.
3. **On a field line (`@`):**
   - Extract the field name (characters between `@` and the first space).
   - Everything after the first space is the field's inline value (may be empty for multi-line fields).
   - The first field line determines the entry type: `@module` starts the header, `@fn`/`@type`/`@trait`/`@const` starts an entry, `@workflow` starts a workflow.
   - Set the current field to this field name.
4. **On a continuation line (indented):**
   - Append to the current field's value. Preserve relative indentation (strip the first 2 spaces only).
   - Sub-fields (`.name`) are continuation lines with 4 spaces of indentation — they belong to the current field, not a new field.
5. **On a separator (`---`):**
   - Close the current entry/block. The next field line begins a new entry.
   - The first separator ends the module header. Subsequent separators separate entries and workflows.
6. **On end-of-file:**
   - Close the current entry. Parsing is complete.

### 8.3 State machine

```
States: HEADER, ENTRY, FIELD_VALUE, DONE
```

| Current state | Line type | Action | Next state |
|---------------|-----------|--------|------------|
| HEADER | Field (`@`) | Store field on module header | HEADER |
| HEADER | Continuation | Append to current header field | HEADER |
| HEADER | Separator | Finalize header | ENTRY |
| ENTRY | Field (`@fn/type/trait/const/workflow`) | Start new entry, store field | FIELD_VALUE |
| ENTRY | Field (`@`) | Store field on current entry | FIELD_VALUE |
| ENTRY | Comment/Blank | Skip | ENTRY |
| FIELD_VALUE | Field (`@`) | Close current field, store new field | FIELD_VALUE |
| FIELD_VALUE | Continuation | Append to current field | FIELD_VALUE |
| FIELD_VALUE | Separator | Finalize current entry | ENTRY |
| FIELD_VALUE | Comment/Blank | Skip | FIELD_VALUE |
| Any | EOF | Finalize current entry/header | DONE |

### 8.4 Output structure

A parsed AID file produces:

```
AidFile {
  header: {
    module: str,
    lang: str,
    version: str,
    ...remaining header fields
  },
  entries: [
    {
      kind: "fn" | "type" | "trait" | "const",
      name: str,
      fields: { field_name: str | [str] }
    },
    ...
  ],
  workflows: [
    {
      name: str,
      fields: { field_name: str | [str] }
    },
    ...
  ]
}
```

Entries are distinguished from workflows by their opening field: `@fn`, `@type`, `@trait`, `@const` produce entries; `@workflow` produces workflows.

### 8.5 Error handling

Parsers should be lenient:
- **Unknown fields:** Ignore them. Forward compatibility requires this.
- **Missing required fields:** Warn but don't reject. Partial AID files are valid.
- **Malformed lines:** Skip with a warning. One bad line should not invalidate the file.
- **Duplicate fields:** Last value wins. Warn on duplicates.

---

## 9. Versioning

### 9.1 AID spec versioning

The AID format itself is versioned using semantic versioning. The `@aid_version` field in the module header declares which spec version the file conforms to.

### 9.2 Library versioning

The `@version` field tracks which version of the documented library the AID file describes. When a library updates its API:

- New functions/types: add new entries, update `@since`
- Changed signatures: update the entry, add `@since` to note the change
- Removed APIs: mark with `@deprecated` before removal, then remove in next major version

### 9.3 Backwards compatibility

New fields may be added to the AID spec in minor versions. Parsers must ignore unknown fields. Fields will not be removed or have their semantics changed except in major versions.

---

## 10. File organization

### 10.1 Naming convention

```
module-name.aid
```

Lowercase, hyphenated. Matches the module name with `/` replaced by `-`.

Examples:
- `http-client.aid` for `http/client`
- `os-path.aid` for `os.path`
- `std-collections.aid` for `std/collections`

### 10.2 Directory structure

For a library with multiple modules:

```
.aidocs/
├── http-client.aid
├── http-server.aid
├── http-types.aid
└── http-middleware.aid
```

AID files live in a `.aidocs/` directory at the project root, or in a central registry for third-party libraries.

### 10.3 One file per module

Each `.aid` file documents exactly one module. This keeps files at a manageable size (typically under 2,000 tokens) and allows agents to load only what they need.

### 10.4 Manifest file

Large projects (20+ packages) should include a `.aidocs/manifest.aid` file that indexes all AID files. The manifest lets agents identify relevant packages from a task description without opening every AID file.

```
@manifest
@project SyndrDB
@aid_version 0.1

---

@package query/planner
@aid_file planner.aid
@aid_status reviewed
@depends [syndrQL, domain/index, domain/models]
@purpose Query planning and optimization — converts parsed queries into execution plans
@layer l2

---

@package domain/index/brinindex
@aid_file brinindex.aid
@aid_status draft
@depends [domain/models]
@purpose Block Range INdex — lossy page-range filtering for range queries
@layer l2
```

| Field | Required | Description |
|-------|----------|-------------|
| `@manifest` | Yes | Marker — identifies this as a manifest file |
| `@project` | Yes | Project name |
| `@package` | Yes | Full package path |
| `@aid_file` | Yes | Filename in `.aidocs/` |
| `@aid_status` | No | draft, reviewed, approved, stale |
| `@depends` | No | Packages this one calls into |
| `@purpose` | Yes | One-line description for relevance filtering |
| `@layer` | No | `l1` or `l2` — tells agent what depth of info to expect |

**Agent workflow:** Read manifest first. Identify relevant packages by matching the task description against `@purpose` fields. Load only those AID files plus their `@depends` chain. This prevents the token bloat seen in benchmarks when all AID files are loaded indiscriminately.

### 10.5 Discovery protocol

When an agent or tool needs to find AID files, it follows this discovery chain:

1. Check for `.aidocs/` in the current working directory
2. Walk up parent directories until `.aidocs/` is found or the filesystem root is reached
3. If `.aidocs/manifest.aid` exists, use it for package-to-file mapping
4. If no manifest exists, discover files by naming convention: `{package-name}.aid`
5. For cross-project dependencies, check:
   - `.aidocs/vendor/` — vendored AID from third-party dependencies
   - `~/.aidocs/` — user-level central registry (configurable)

The first `.aidocs/` directory found wins. Tools should not search multiple `.aidocs/` directories simultaneously — this keeps the resolution deterministic.

---

## 11. Examples

### 11.1 Example blocks

The `@example` field on entries contains minimal usage examples. Rules:

- Examples use the language specified by the module's `@lang` field
- Multi-line examples are indented continuation lines (standard AID syntax)
- Examples should show the ONE thing the entry does — not a full program
- Examples are patterns for an agent to follow, not executable tests

```
@example
  resp := http.get("https://api.example.com/users")?
  data := resp.json[[]User]()?
```

### 11.2 When to include examples

Layer 1 extractors should only include examples from existing docstrings. Layer 2 generators may synthesize examples when the usage pattern is non-obvious — especially for workflows and entries with complex constraints.

---

## 12. Security considerations

AID files are documentation artifacts. They carry the same trust level as the source code they describe.

- **`[src:]` references are relative paths.** Tools must validate that resolved paths stay within the project root. A malicious AID file could reference `../../etc/passwd` — path traversal must be prevented.
- **Generated AID should be reviewed before committing.** The same way generated code is reviewed. The L2 generator→reviewer pipeline provides automated review, but human review is appropriate for critical systems.
- **Don't generate AID from untrusted source code without review.** If the source contains prompt injection patterns (in comments, docstrings, or string literals), the L2 generator could be influenced to produce misleading documentation.
- **AID files should be committed to version control.** They are project artifacts, not ephemeral outputs. Committing them provides auditability and enables staleness detection via `@code_version`.

---

## 13. Migration

### 13.1 Spec version compatibility

The `@aid_version` field declares which AID spec version the file targets. Compatibility rules:

- **Parsers must handle older spec versions gracefully.** Unknown fields are ignored (forward compatibility). Missing new fields use defaults.
- **Minor version changes are additive only.** New fields may be added; existing fields retain their semantics. AID 0.1 files are valid AID 0.2 files.
- **Breaking changes require a major version bump.** Field semantics may change or fields may be removed only in major versions (0.x → 1.0 allows breaking changes, since pre-1.0 is unstable).

### 13.2 Updating AID files

When the spec changes, existing AID files are updated by re-running the generation pipeline:

1. Layer 1: re-run the extractor — captures any new fields the extractor now produces
2. Layer 2: re-run the generator and reviewer — applies new semantic field requirements
3. Manual edits: only needed if field semantics changed (major version)

The `aid-gen-l2 stale` command detects files that need re-generation by comparing `@code_version` against the current git HEAD.
