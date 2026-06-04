# AID Format Specification v0.3

**The complete specification for the Agent Interface Document format.**

---

## Table of Contents

1. [Overview](#1-overview)
2. [File structure](#2-file-structure)
3. [Tier 1: Module header](#3-tier-1-module-header)
4. [Tier 2: Entries](#4-tier-2-entries)
5. [Tier 2.5: Module annotations](#5-tier-25-module-annotations)
6. [Tier 3: Workflows](#6-tier-3-workflows)
7. [Tier 4: Agentic and dataflow constructs](#7-tier-4-agentic-and-dataflow-constructs)
8. [Syntax rules](#8-syntax-rules)
9. [Parsing](#9-parsing)
10. [Versioning](#10-versioning)
11. [File organization](#11-file-organization)
12. [Project file (`project.aid`)](#12-project-file-projectaid)
13. [Examples](#13-examples)
14. [Security considerations](#14-security-considerations)
15. [Migration](#15-migration)

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
| `@stability` | No | One of: `unknown`, `experimental`, `unstable`, `stable`, `deprecated`. Default: `unknown` |
| `@purpose` | Yes | One-line description of what the module does. Max 120 characters. |
| `@depends` | No | Modules/packages this module depends on (for selective AID loading). Comma-separated or bracketed list. |
| `@source` | No | URL to source repository or documentation |
| `@code_version` | No | Git commit hash or tag identifying the code version this AID describes. Format: `git:HASH` |
| `@aid_status` | No | Document lifecycle status: `draft`, `reviewed`, `approved`, `stale`. Default: `draft` |
| `@aid_generated_by` | No | Agent role that produced this AID (e.g., `layer1-extractor`, `layer2-generator`) |
| `@aid_reviewed_by` | No | Agent role that verified this AID (e.g., `layer2-reviewer`) |
| `@aid_version` | No | Version of the AID spec this file conforms to. Default: latest |
| `@test_framework` | No | Test framework name (e.g., `go test`, `pytest`, `jest`) |
| `@test_cmd` | No | Command to run this module's tests |
| `@test_fixtures` | No | Path to test fixtures/data relative to module root |
| `@mock_strategy` | No | How dependencies are mocked (e.g., `interfaces`, `monkey-patching`, framework name) |
| `@env` | No | Environment variables this module reads. Block format with constraint syntax. |
| `@services` | No | External services this module connects to |
| `@config_files` | No | Config files read by this module |
| `@init_order` | No | Numeric initialization order. Lower numbers init first. |
| `@init_fn` | No | Function that initializes this module |
| `@shutdown_fn` | No | Function that shuts down this module |
| `@global_state` | No | Module-level mutable state |
| `@engine` | No | Dataflow framework this module targets (Tier 4): `langgraph`, `pipecat`, `lcel`, `crewai`, `autogen`, `custom`. The framework analog of `@lang`. |
| `@checkpointer` | No | Persistence backend for agent/graph state (Tier 4). Format: `name — backend description`. |

### 3.2 Test information

Modules can declare their test infrastructure to help agents write tests that match existing patterns:

```
@test_framework pytest
@test_cmd pytest tests/unit/test_http_client.py
@test_fixtures tests/fixtures/
@mock_strategy interfaces — mock via interface implementation, no mocking framework
```

### 3.3 Configuration and environment

Modules that read configuration should declare what they need:

```
@env
  DATABASE_URL: str — PostgreSQL connection string. Required.
  REDIS_URL: str — Redis connection string. Default "localhost:6379".
  JWT_SECRET: str — HMAC signing key. Required. Sensitive.
@services
  PostgreSQL: DATABASE_URL — primary data store
  Stripe API: STRIPE_API_KEY — payment processing
@config_files
  config.yaml — application config, loaded at startup
```

The `@env` block reuses parameter constraint syntax (`Required.`, `Default VALUE.`, `One of: ...`). Mark secrets with `Sensitive.` — agents must not log, print, or hardcode these values.

### 3.4 Module lifecycle

Modules with initialization or shutdown requirements:

```
@init_order 2
@init_fn Initialize
@shutdown_fn Shutdown
@global_state
  pool: *pgxpool.Pool — connection pool. Created in Initialize, closed in Shutdown.
  logger: *slog.Logger — set once in Initialize, read-only after.
```

### 3.5 Example

```
@module http/client
@lang python
@version 2.31.0
@stability stable
@purpose HTTP client library for making web requests and handling responses
@depends [ssl, dns, url, io]
@source https://github.com/psf/requests
@test_framework pytest
@test_cmd pytest tests/unit/test_http_client.py
@aid_version 0.3
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
| `@visibility` | No | Access level: `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@purpose` | Yes | One-line description. Max 120 characters. |
| `@sig` | Yes | Full type signature |
| `@params` | No* | Parameter descriptions with constraints. *Required if function has parameters. |
| `@returns` | No | Description of return value and its guarantees |
| `@errors` | No* | Exhaustive list of error types and conditions. *Required if function can error. |
| `@pre` | No | Preconditions that must hold before calling |
| `@post` | No | Postconditions guaranteed after successful return |
| `@effects` | No | Side effects: `[Net]`, `[Fs]`, `[Io]`, `[Env]`, `[Time]`, `[Random]`, `[Db]`, `[Process]`, `[Gpu]`, `[Callback]`, and the Tier 4 effects `[Llm]`, `[Tool]`, `[Embed]`, `[Stream]`, etc. |
| `@calls` | No | Comma-separated list of functions this function calls internally |
| `@thread_safety` | No | Concurrency safety. Structured keyword first, optional elaboration after. See Thread safety vocabulary. |
| `@complexity` | No | Time and/or space complexity |
| `@since` | No | Version when this function was introduced |
| `@deprecated` | No | Deprecation notice with replacement |
| `@related` | No | Typed relationship block (see Typed relationships). Untyped flat lists accepted as `sibling:` for backward compat. |
| `@reads` | No | Fields this function reads. Format: `[Type.field, ...]` |
| `@writes` | No | Fields this function writes. Format: `[Type.field, ...]` |
| `@tested` | No | Whether this function has test coverage. `true` or `false`. |
| `@test_hint` | No | Test function names that exercise this function |
| `@source_file` | No | Source file path relative to project root. Populated by Layer 1 extractors. |
| `@source_line` | No | Line number in source file. Populated by Layer 1 extractors. |
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

**Error hierarchy:** When using dot notation (`ErrorType.Variant`), the error name must reference a declared `@type` entry with `@kind enum` whose `@variants` include the named variant. For cross-module errors, use qualified names: `module/path.ErrorType.Variant`. This makes the relationship between `@errors` and `@type` mechanically verifiable — a validator can confirm that every error variant listed actually exists as a declared type.

#### Call relationships (`@calls`)

The `@calls` field lists functions this function calls internally:

```
@fn ProcessOrder
@sig (order: Order) -> Receipt ! PaymentError
@calls ValidateOrder, ChargePayment, SendReceipt, UpdateInventory
```

`@calls` is populated by Layer 1 extractors from AST analysis. It enables call graph queries without reading source code. Bare names refer to the same module; qualified names (`module/path.Name`) refer to other modules.

#### Typed relationships (`@related`)

`@related` uses typed block form to express how entries are related:

```
@related
  calls: request
  produces: Response
  sibling: post, put, delete
  wraps: urllib3.request
  inverse: Response.close
```

| Relationship | Meaning |
|-------------|---------|
| `calls` | This entry calls that entry |
| `produces` | This entry creates/returns instances of that type |
| `consumes` | This entry takes that type as input |
| `sibling` | Same conceptual family, interchangeable alternatives |
| `wraps` | This entry is a convenience wrapper around that entry |
| `inverse` | Opposite/counterpart operation (open/close, lock/unlock) |
| `replaces` | This entry supersedes a deprecated entry |

Every entry in `@related` must have a relationship type prefix. Untyped flat lists (`@related post, put, delete`) are accepted for backward compatibility and treated as `sibling:` entries, but new AID files should always use typed relationships.

#### Error provenance annotations

Error entries can include inline annotations indicating where errors originate or how they're handled:

```
@errors
  NotFoundError — user not found [from: GetUser]
  ValidationError — invalid email format [origin]
  DbError — database connection failed [from: storage.Query, caught: logged and retried]
```

| Annotation | Meaning |
|-----------|---------|
| `[origin]` | This function originates this error (not propagated) |
| `[from: FnName]` | Error propagated from the named function |
| `[caught: description]` | Error is caught internally, not propagated (informational) |

Annotations appear at the end of the error line, inside brackets. They enable error trace queries without reading source code.

#### Field access tracking (`@reads`, `@writes`)

Functions that access specific fields of complex types can declare what they read and write:

```
@fn UpdateEmail
@sig (user: User, newEmail: str) -> None
@reads [User.role]
@writes [User.email]
```

Use `@reads` and `@writes` on functions that touch fields of types with `@invariants`. This enables "what touches X.field?" queries.

#### Thread safety vocabulary

`@thread_safety` uses a structured keyword followed by optional elaboration:

| Keyword | Meaning |
|---------|---------|
| `safe` | No external synchronization needed. Multiple goroutines/threads can call concurrently. |
| `immutable` | All state is read-only after construction. Inherently safe. |
| `channel-based` | Uses channels for inter-goroutine/thread communication. |
| `requires-sync` | Caller must provide external synchronization (mutex, lock, etc.). |
| `not-safe` | Not safe for concurrent use. Must be confined to a single goroutine/thread. |

```
@thread_safety safe. Each call is independent. No shared mutable state.
@thread_safety not-safe. Call only from the main thread.
@thread_safety requires-sync. Wrap calls in a mutex if sharing across goroutines.
```

The keyword comes first, enabling machine parsing. The elaboration after `.` or `—` provides human-readable context.

#### Test coverage (`@tested`, `@test_hint`)

```
@fn ProcessOrder
@tested true
@test_hint TestProcessOrder_ValidOrder, TestProcessOrder_PaymentFails
```

`@tested` indicates whether the function has test coverage. `@test_hint` lists test function names for agents to reference when writing new tests for related functions.

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
| `@visibility` | No | Access level: `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@kind` | Yes | One of: `struct`, `enum`, `union`, `class`, `alias`, `newtype` |
| `@purpose` | Yes | One-line description. Max 120 characters. |
| `@fields` | No* | Field names, types, and constraints. *Required for struct/class. |
| `@fields_visibility` | No | `full` or `partial`. Signals whether `@fields` lists all fields. Default: `full`. |
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

#### `@fields_visibility`

When a type has internal fields that are not part of the public contract, use `@fields_visibility partial` to signal that `@fields` is an incomplete listing:

```
@type HashMap
@kind struct
@fields_visibility partial
@fields
  count: int — number of elements
  capacity: int — current bucket allocation
```

| Value | Meaning |
|-------|---------|
| `full` | All fields are listed (default if omitted) |
| `partial` | Type has additional undocumented internal fields |

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
| `@visibility` | No | Access level: `public`, `internal`, `protected`, `private`. Default: `public`. |
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
| `@visibility` | No | Access level: `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@purpose` | Yes | One-line description |
| `@value_type` | Yes | The constant's type |
| `@value` | No | The constant's value (if publicly known/useful) |
| `@since` | No | Version introduced |

Note: The field is `@value_type`, not `@type`, to avoid ambiguity with the `@type` entry keyword.

#### Example

```
@const MAX_REDIRECTS
@purpose Maximum number of HTTP redirects before aborting
@value_type int
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

### 6.1 Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@workflow` | Yes | Workflow name (snake_case) |
| `@purpose` | Yes | What this workflow accomplishes |
| `@steps` | Yes | Numbered sequence of operations |
| `@errors_at` | No | Which errors can occur at which steps |
| `@antipatterns` | No | Common mistakes to avoid |
| `@variants` | No | Alternative paths through the workflow |
| `@example` | No | Complete example showing the full workflow |

### 6.2 Steps syntax

```
@steps
  1. Label: function_or_operation — description
  2. Label: function_or_operation — description
  3. Label: function_or_operation — description
```

Steps are numbered sequentially. Each step has a short label, the function/operation involved, and a description of what happens.

### 6.3 Errors-at syntax

```
@errors_at
  step 2: ErrorType — condition
  step 3: ErrorType — condition
```

Maps errors to specific workflow steps so the agent knows exactly where error handling is needed.

### 6.4 Variants syntax

```
@variants
  - streaming: Replace step 3 with resp.stream() for chunked processing
  - async: Wrap steps 2-4 in spawn for concurrent execution
```

### 6.5 Full workflow example

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

## 7. Tier 4: Agentic and dataflow constructs

Tiers 1–3 describe an API surface and the linear workflows that use it. Modern AI systems — agent frameworks like LangGraph, real-time pipelines like Pipecat, composition libraries like LangChain — are not linear API calls. They are **dataflow graphs**: nodes connected by edges, with conditional routing, **cycles**, shared mutable state that merges on update, model-driven tool selection, and non-deterministic behavior. None of these can be expressed with `@fn`/`@type` entries or with the strictly-sequential `@steps` of a `@workflow`.

Tier 4 adds the constructs needed to make such a system fully comprehensible from its AID files alone: `@graph` (topology), state channels with reducers (shared state), `@tool` (model-invocable functions), `@agent` (autonomous actors), `@prompt` and `@model` (invocation contracts), the `@determinism` marker, and agentic effect tags.

Every Tier 4 entry is a normal AID entry: it opens with a marker field and parses with the same line-by-line state machine (Section 9.3). No new parser states are required — only new entry keywords (Section 9.4).

### 7.1 The `@engine` field

Just as `@lang` makes type notation language-agnostic, `@engine` makes dataflow notation **framework-agnostic**. The Tier 4 primitives are universal; `@engine` tells tooling which framework's concepts they map to.

```
@engine langgraph | pipecat | lcel | langchain | crewai | autogen | custom
```

`@engine` appears on the module header (when a module is dedicated to one framework) and/or on individual `@graph`/`@agent` entries (when a module mixes engines). It is informational: an agent that understands the named framework can map nodes/edges/state to concrete APIs; an agent that doesn't still reads the universal topology. Unknown engine values are preserved, not rejected.

### 7.2 Graph entries (`@graph`)

A `@graph` describes a dataflow topology. Unlike a `@workflow` — which is a numbered sequence — a `@graph` has named nodes, directed edges, conditional routing, and may contain **cycles**.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@graph` | Yes | Graph name (snake_case) |
| `@purpose` | Yes | One-line description. Max 120 characters. |
| `@engine` | No | Framework this graph targets (see 7.1) |
| `@state` | No | Name of the `@type` entry holding shared state (see 7.3) |
| `@entry` | No | Name of the entry node(s) where execution begins. Reserved name: `START`. |
| `@nodes` | Yes | The graph's nodes. One per line: `name: fn_or_runnable — description [Effects]` |
| `@edges` | No | Unconditional directed edges. One per line: `src -> dst`. A back-edge forms a cycle. |
| `@conditional_edges` | No | Routed edges. One per line: `src: router_fn -> targetA \| targetB \| END — condition` |
| `@cycles` | No | Explicit call-out of loops and their bound (recursion limit / max iterations) |
| `@interrupts` | No | Human-in-the-loop pause points. One per line: `before\|after node — reason` |
| `@composition` | No | For Runnable/LCEL graphs: `sequence`, `parallel`, `branch`, `fallback`, `map` (see 7.2.3) |
| `@frames` | No | For push-based engines (Pipecat): the frame vocabulary flowing through the graph (see 7.2.4) |
| `@effects` | No | Aggregate side effects of running the graph |
| `@antipatterns` | No | Common mistakes when modifying or using this graph |
| `@example` | No | Minimal construction/compilation example |

#### 7.2.1 Nodes, edges, and routing

`END` is the reserved terminal sink — an edge to `END` ends that path. `START` is the reserved source; `@entry` names the node(s) reached from `START`.

```
@nodes
  plan: planner_node — LLM selects the next action [Llm]
  act: tool_executor — runs the selected tool [Tool]
  observe: format_observation — folds the tool result into state
@edges
  act -> observe
  observe -> plan
@conditional_edges
  plan: should_continue -> act | END — act if the last message has a tool call, else END
```

Each node line is `name: fn — description [Effects]`. The `fn` should match an entry in the file (`@fn`, `@tool`, or `@agent`); bare names resolve to the current module, qualified names (`module/path.Name`) to others (Section 8.7). `[Effects]` reuses the effect vocabulary (Section 7.9).

Each `@edges` line is one directed edge `src -> dst`. A back-edge (an edge whose target appears earlier in the flow) creates a cycle — perfectly legal, and the thing a `@workflow` cannot express.

Each `@conditional_edges` line names the source node, the **router function** that decides where to go, the `|`-separated set of possible targets, and the condition. This is how branching and loop-termination are expressed.

#### 7.2.2 Cycles

Cycles are the defining feature of agent graphs (an agent loops "reason → act → observe → reason" until done). Because an unbounded cycle is a hazard, `@cycles` states the loop and its bound explicitly:

```
@cycles
  plan -> act -> observe -> plan — bounded by recursion_limit=25; raises GraphRecursionError if exceeded
```

An agent reading this knows the loop exists, what advances it toward termination (the `should_continue` router), and what happens at the bound. Omitting `@cycles` when a back-edge exists means "this graph has a cycle but its bound is undocumented" — treat with caution (Section 8.0).

#### 7.2.3 Composition (LCEL / Runnable graphs)

When a graph is a composition of Runnables rather than an explicitly-wired node graph (LangChain Expression Language), `@composition` names the operator and `@nodes`/`@edges` capture the linear or fan-out structure:

| Operator | Meaning | LCEL |
|----------|---------|------|
| `sequence` | Output of each stage feeds the next | `a \| b \| c` |
| `parallel` | Run branches concurrently, collect into a dict | `RunnableParallel` |
| `branch` | Route to one branch by predicate | `RunnableBranch` |
| `fallback` | Try primary, fall back on error | `.with_fallbacks([...])` |
| `map` | Apply to each element of an input list | `.map()` |

```
@graph rag_chain
@purpose Retrieval-augmented generation: fetch context, prompt, generate, parse
@engine lcel
@composition sequence
@nodes
  retrieve: vector_search — fetch top-k context chunks [Embed, Net]
  prompt: rag_prompt — fill the prompt template
  generate: chat_model — call the LLM [Llm, Stream]
  parse: output_parser — coerce to structured output
@edges
  retrieve -> prompt
  prompt -> generate
  generate -> parse
```

#### 7.2.4 Push-based graphs and frames (Pipecat)

Real-time pipelines push typed **frames** between processors, bidirectionally. For `@engine pipecat`, edges carry a frame type and a direction, and `@frames` declares the frame vocabulary:

```
@graph voice_loop
@purpose Real-time voice assistant: audio in → transcribe → LLM → speak
@engine pipecat
@frames
  AudioRawFrame — raw PCM audio chunks. downstream.
  TranscriptionFrame — recognized text from STT. downstream.
  TextFrame — LLM output text. downstream.
  StartInterruptionFrame — user barge-in signal. upstream.
@nodes
  transport_in: WebRTCInput — receives mic audio [Net, Stream]
  stt: DeepgramSTT — speech-to-text [Llm, Net, Stream]
  llm: OpenAILLM — generates response [Llm, Net, Stream]
  tts: CartesiaTTS — text-to-speech [Net, Stream]
  transport_out: WebRTCOutput — plays audio [Net, Stream]
@edges
  transport_in -> stt : AudioRawFrame [downstream]
  stt -> llm : TranscriptionFrame [downstream]
  llm -> tts : TextFrame [downstream]
  tts -> transport_out : AudioRawFrame [downstream]
  transport_out -> llm : StartInterruptionFrame [upstream]
```

Each `@frames` line is `FrameType — description. direction.` where direction is `downstream`, `upstream`, or `bidirectional`. Edge annotations `: FrameType [direction]` say which frame travels which way along that connection.

### 7.3 State channels and reducers

LangGraph (and similar) state is a data object whose fields **merge** on each node update rather than being overwritten. A node returning `{"messages": [m]}` *appends* to `messages` if its reducer is append-based. AID expresses this by marking a `@type` with `@channels` and annotating each field with a `reducer:` constraint.

```
@type AgentState
@kind struct
@channels
@purpose Shared state threaded through every node of agent_loop
@fields
  messages: [Message] — conversation history. reducer: append.
  step_count: int — loop counter. reducer: increment. Default 0.
  next: str — routing decision written by the planner. reducer: last-wins.
```

| Reducer | Merge behavior |
|---------|----------------|
| `last-wins` | New value replaces old (default if a `@channels` field omits `reducer:`) |
| `append` | New items appended to the existing list |
| `extend` | New list concatenated onto the existing list |
| `add` | Numeric/aggregate addition (e.g. `operator.add`) |
| `merge` | Dict/set union |
| `custom:fn_name` | A named reducer function defines the merge |

`@channels` is an opt-in marker: a `@type` without it uses ordinary assignment semantics. `reducer:` is a constraint keyword in the `@fields` block, reusing the existing field-constraint syntax (Section 8) — no new block type. Knowing a field's reducer is essential for correct code: a node that returns the whole `messages` list when the reducer is `append` will duplicate history.

### 7.4 Tool entries (`@tool`)

A `@tool` is a function exposed to a model, which decides when to call it. Structurally it is an `@fn` plus tool-call metadata; it is a distinct entry type because "the model may invoke this autonomously" is a load-bearing fact an agent must know.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@tool` | Yes | Tool name (the name the model sees) |
| `@purpose` | Yes | One-line description. This is what the model reads to decide when to call. Max 120 chars. |
| `@sig` | Yes | Full type signature (same syntax as `@fn`) |
| `@params` | No* | Parameter descriptions. *Required if the tool has parameters. |
| `@returns` | No | Return value description |
| `@errors` | No* | Error conditions. *Required if the tool can error. |
| `@schema` | No | Path to the JSON-Schema the model sees (a pointer, never inlined) |
| `@invoked_by` | No | `llm`, `agent`, or `code` — who calls this tool |
| `@effects` | No | Side effects (reuse vocabulary, including `Tool`, `Net`, `Db`) |
| `@determinism` | No | `deterministic`, `nondeterministic`, `seeded` (see 7.8) |
| `@idempotent` | No | `true`/`false` — whether repeated calls with the same args are safe |
| `@determinism`, `@pre`, `@post`, `@related`, `@example` | No | As on `@fn` |

```
@tool web_search
@purpose Search the web and return ranked results. Use for current events or unknown facts.
@sig (query: str, k?: int) -> [SearchResult] ! SearchError
@params
  query: Search query. Required.
  k: Number of results. Default 5. Range [1, 20].
@returns Ranked list of results with title, url, snippet
@errors
  SearchError.RateLimited — provider quota exceeded [origin]
  SearchError.Network — upstream request failed [origin]
@schema tools/web_search.schema.json
@invoked_by llm
@effects [Net, Tool]
@determinism nondeterministic — results vary with index state
@idempotent true
```

### 7.5 Agent entries (`@agent`)

An `@agent` is an autonomous actor: a model plus a system prompt, a set of tools it may call, guardrails on its behavior, and optionally the ability to hand off to other agents.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@agent` | Yes | Agent name |
| `@purpose` | Yes | One-line description of the agent's role. Max 120 chars. |
| `@model` | No | Name of a `@model` entry, or an inline model id |
| `@system_prompt` | No | Path to the system prompt file (a pointer — never inline prose) |
| `@tools` | No | Tools this agent may call. List of `@tool` names. |
| `@handoffs` | No | Other agents this agent can transfer control to (multi-agent / swarm) |
| `@autonomy` | No | `autonomous`, `supervised`, or `human-gated` |
| `@guardrails` | No | Behavioral constraints. Bulleted, with `[src:]` references. |
| `@memory` | No | `none`, `thread`, or `persistent` (see 7.10) |
| `@output` | No | Structured output type the agent produces |
| `@determinism` | No | As in 7.8 |
| `@effects`, `@related`, `@example` | No | As on other entries |

```
@agent researcher
@purpose Gathers and summarizes web sources on a topic
@model planner_llm
@system_prompt prompts/researcher.md
@tools [web_search, fetch_url]
@handoffs [writer, critic]
@autonomy supervised
@guardrails
  - Never fabricate citations; every claim links to a fetched source [src: agents/researcher.py:40-55]
  - At most 10 tool calls per task [src: agents/researcher.py:22]
@memory thread
@output ResearchReport
@effects [Llm, Tool, Net]
```

### 7.6 Prompt entries (`@prompt`)

A `@prompt` is a templated model invocation treated as a contract: declared inputs, an expected output shape, the model it targets, and its known failure modes. The prompt text itself lives in a file referenced by `@template` — it is never inlined (prompts are prose, and Tier 4 keeps prose out of AID).

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@prompt` | Yes | Prompt name |
| `@purpose` | Yes | One-line description. Max 120 chars. |
| `@inputs` | No* | Template input variables, with constraints. *Required if the template has variables. |
| `@output` | No | Expected output shape (type, or `str` for free text) |
| `@model` | No | Target model (`@model` name or inline id) |
| `@template` | No | Path to the prompt template file (pointer, never inlined) |
| `@determinism` | No | As in 7.8 |
| `@failure_modes` | No | Known ways the model output goes wrong, with mitigations. Bulleted. |
| `@example` | No | Minimal usage example |

```
@prompt extract_entities
@purpose Extract named entities from text as a JSON array
@inputs
  text: str — source document. Required. Length [1, 20000].
@output [Entity] — validated against the Entity type
@model gpt-4o-mini
@template prompts/extract_entities.j2
@determinism nondeterministic. temperature=0 reduces variance but does not eliminate it.
@failure_modes
  - May emit malformed JSON — wrap the parse in a retry with repair
  - May hallucinate entities absent from the text — validate every entity against the source span
```

### 7.7 Model entries (`@model`)

A `@model` declares a configured language model: provider, model id, parameters, and whether it produces structured output. Agents and prompts reference it by name.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@model` | Yes | Model handle (the name agents/prompts reference) |
| `@purpose` | Yes | One-line description. Max 120 chars. |
| `@provider` | No | `anthropic`, `openai`, `google`, `local`, etc. |
| `@model_id` | No | Concrete model identifier (e.g. `claude-opus-4-8`) |
| `@params` | No | Inference parameters: `temperature=…, max_tokens=…`, etc. |
| `@structured_output` | No | Type the model is constrained to emit (forces schema/tool-format output) |
| `@determinism` | No | As in 7.8 |
| `@cost` | No | Approximate token cost, e.g. `input≈$3/Mtok, output≈$15/Mtok` |
| `@effects` | No | Typically `[Llm, Net]` |

```
@model planner_llm
@purpose LLM used for the planning step of agent_loop
@provider anthropic
@model_id claude-opus-4-8
@params temperature=0.0, max_tokens=4096
@structured_output AgentDecision
@determinism nondeterministic
@cost input≈$3/Mtok, output≈$15/Mtok
@effects [Llm, Net]
```

### 7.8 Determinism (`@determinism`)

Agentic units are frequently **non-deterministic**: the same input can produce different output across runs. This is first-class information — it tells an agent that tests must assert on *properties* (schema-valid, non-empty, contains a citation) rather than exact strings, and that retries/guardrails are warranted.

`@determinism` may appear on `@fn`, `@tool`, `@prompt`, `@model`, and `@agent`:

| Value | Meaning |
|-------|---------|
| `deterministic` | Same input always yields the same output |
| `nondeterministic` | Output may vary across runs (model sampling, tool index state, time) |
| `seeded` | Deterministic given a fixed seed/temperature; varies otherwise |

Omission means determinism is unknown (Section 8.0) — an agent should not assume reproducibility.

### 7.9 Agentic effect tags

Tier 4 adds effect categories to the vocabulary used by `@effects` and node annotations (the full table is in `fields.md`):

| Effect | Meaning |
|--------|---------|
| `Llm` | Invokes a language model — implies cost, latency, and (usually) non-determinism |
| `Tool` | Invokes model-selected tools; concrete effects depend on which tools are chosen |
| `Embed` | Generates vector embeddings |
| `Stream` | Emits incremental output (tokens, frames, streamed events) rather than a single value |

These compose with existing effects (`Net`, `Db`, `Async`, `Callback`). A streaming LLM node is `[Llm, Net, Stream]`.

### 7.10 Runtime semantics: memory, checkpointing, interrupts, streaming

**Memory and checkpointing.** Agents and graphs that persist state across turns declare it with `@memory` (`none` / `thread` / `persistent`). The persistence backend is declared once on the module header with `@checkpointer`:

```
@checkpointer postgres — LangGraph PostgresSaver; threads keyed by thread_id
```

`thread` memory means state is scoped to a `thread_id` and survives across invocations within that thread. `persistent` means it outlives the process (durable store). `none` means each invocation starts fresh.

**Interrupts (human-in-the-loop).** Pause points are declared on the graph via `@interrupts` (Section 7.2). An interrupt suspends execution before/after a node so a human can inspect or approve state, then resume. This is how approval gates for destructive tools are made visible.

**Streaming.** A unit that yields partial results carries the `[Stream]` effect (Section 7.9). For push-based engines, frame direction (Section 7.2.4) describes the streaming topology. An agent generating a consumer of a streaming unit knows to iterate/await chunks rather than expect a single return value.

---

## 8. Syntax rules

### 8.0 Omission vs explicit None

For any optional field, there is a semantic distinction between omission and an explicit value of `None`:

- **Field omitted:** The information is unknown or undocumented. Agents should treat this conservatively — assume constraints, errors, or effects may exist but aren't documented.
- **`@field None`:** The author explicitly asserts there are none. Agents can rely on this as an affirmative claim.

This distinction applies to: `@pre`, `@post`, `@errors`, `@effects`, `@thread_safety`, `@constructors`, `@invariants`.

Example: `@pre None` means "this function has no preconditions — verified." A missing `@pre` means "preconditions are unknown — be cautious."

### 8.1 Field syntax

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

### 8.2 Comments

Lines starting with `//` are comments and are ignored by parsers:

```
// This is a comment
@fn get
// TODO: verify error list is exhaustive
@errors
  ...
```

### 8.3 Entry separators

Entries are separated by `---` on its own line (no leading/trailing whitespace).

### 8.4 Inline descriptions

Within field values, `—` (em dash) separates a name from its description:

```
  HttpError.DnsFailure — domain cannot be resolved
  status: int — HTTP status code. Range [100, 599].
```

### 8.5 Source references

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

### 8.6 Lists

Lists within fields use comma-separated values in brackets:

```
@depends [ssl, dns, url]
@effects [Net, Fs]
@implements [Display, Debug, Clone]
```

### 8.7 Cross-module references

Any field that references another entry (`@related`, `@depends`, `@implements`, `@extends`, `@implementors`, `@constructors`) supports both bare and qualified names.

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

### 8.8 Sub-fields

Nested properties within parameters use `.` prefix with additional indentation:

```
@params
  opts: Request configuration.
    .timeout: Duration. Default 30s. Must be > 0.
    .redirects: int. Default 5. Range [0, 20].
```

### 8.9 Type notation

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

### 8.10 Inheritance (`@extends`)

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

### 8.11 Platform-specific behavior (`@platform`)

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

### 8.12 Well-known protocols

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

## 9. Parsing

An AID parser is a line-by-line state machine. No lookahead, no backtracking, no context-dependent rules. Each line is classified by its prefix, and the parser transitions between states accordingly.

### 9.1 Line classification

Every line in an AID file is exactly one of these types:

| Line type | Rule | Example |
|-----------|------|---------|
| **Field** | Starts with `@` | `@fn get` |
| **Continuation** | Starts with 2+ spaces (and no `@`) | `  url: Full URL. Required.` |
| **Separator** | Is exactly `---` | `---` |
| **Comment** | Starts with `//` | `// TODO: verify errors` |
| **Blank** | Empty or whitespace-only | |

No line can be ambiguous — the first character(s) determine its type.

### 9.2 Parsing rules

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

### 9.3 State machine

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

### 9.4 Output structure

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
      kind: "fn" | "type" | "trait" | "const"
          | "graph" | "tool" | "agent" | "prompt" | "model",
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

Entries are distinguished from workflows by their opening field: `@fn`, `@type`, `@trait`, `@const`, and the Tier 4 keywords `@graph`, `@tool`, `@agent`, `@prompt`, `@model` produce entries; `@workflow` produces workflows. In `project.aid` files, `@cross_cutting`, `@convention`, `@lifecycle`, and `@decision` also start entries. A parser treats any unrecognized entry-opening keyword as a generic entry (kind = the keyword) so that forward-compatible files still parse.

### 9.5 Error handling

Parsers should be lenient:
- **Unknown fields:** Ignore them. Forward compatibility requires this.
- **Missing required fields:** Warn but don't reject. Partial AID files are valid.
- **Malformed lines:** Skip with a warning. One bad line should not invalidate the file.
- **Duplicate fields:** Last value wins. Warn on duplicates. **Exception:** accumulating fields (see 9.6).

### 9.6 Accumulating fields

Most fields follow the "last value wins" rule — if `@purpose` appears twice, the second value overwrites the first. However, some fields are **accumulating**: multiple occurrences are collected into a list rather than overwriting.

Accumulating fields:
- **`@sig`** — Multiple signatures represent overloaded calling conventions. Each `@sig` line is a valid way to call the function. Also accumulates on `@tool` entries (tools can be overloaded like functions).
- **`@rule`** — Multiple rules on a `@convention` entry are collected into an ordered list.

Parsers must implement accumulation for these fields. All other fields use last-value-wins. If a future spec version adds accumulating fields, they will be explicitly listed here.

Note on Tier 4: `@nodes`, `@edges`, `@conditional_edges`, `@guardrails`, `@failure_modes`, and `@frames` are **multi-line block fields**, not accumulating fields — their content is the set of indented continuation lines under a single field occurrence, parsed exactly like `@params` or `@steps`.

---

## 10. Versioning

### 10.1 AID spec versioning

The AID format itself is versioned using semantic versioning. The `@aid_version` field in the module header declares which spec version the file conforms to.

### 10.2 Library versioning

The `@version` field tracks which version of the documented library the AID file describes. When a library updates its API:

- New functions/types: add new entries, update `@since`
- Changed signatures: update the entry, add `@since` to note the change
- Removed APIs: mark with `@deprecated` before removal, then remove in next major version

### 10.3 Backwards compatibility

New fields may be added to the AID spec in minor versions. Parsers must ignore unknown fields. Fields will not be removed or have their semantics changed except in major versions.

---

## 11. File organization

### 11.1 Naming convention

```
module-name.aid
```

Lowercase, hyphenated. Matches the module name with `/` replaced by `-`.

Examples:
- `http-client.aid` for `http/client`
- `os-path.aid` for `os.path`
- `std-collections.aid` for `std/collections`

### 11.2 Directory structure

For a library with multiple modules:

```
.aidocs/
├── http-client.aid
├── http-server.aid
├── http-types.aid
└── http-middleware.aid
```

AID files live in a `.aidocs/` directory at the project root, or in a central registry for third-party libraries.

### 11.3 One file per module

Each `.aid` file documents exactly one module. This keeps files at a manageable size (typically under 2,000 tokens) and allows agents to load only what they need.

### 11.4 Manifest file

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
| `@key_risks` | No | 1-2 most critical things to know about this package, in a few words. Enables quick context without loading the full AID file. |

**Agent workflow:** Read manifest first. Identify relevant packages by matching the task description against `@purpose` fields. Load only those AID files plus their `@depends` chain. This prevents the token bloat seen in benchmarks when all AID files are loaded indiscriminately.

### 11.4.1 Project snapshot

The manifest header may include a **project snapshot** — a compressed, canonical representation of the project's shape and recent changes. The snapshot gives an agent full project orientation in a single read, without loading any individual AID files.

The snapshot has two parts: **shape** (what the project is) and **delta** (what changed).

#### Shape fields

Shape fields appear in the manifest header (before the first `---` separator), alongside `@manifest` and `@project`.

```
@manifest
@project SyndrDB
@aid_version 0.1
@lang go
@shape
  Entry points: cmd/syndrdb (CLI), pkg/client (library), internal/graphQL (API)
  Data flow: query → parser → planner → executor → storage → bundlestore
  Key types: Bundle, Document, FieldValue, BundleFieldSchema, QueryPlan, ExecutionNode
  Boundaries: domain (pure logic), storage (disk I/O), query (planning + execution), server (network)
  Scale: 496 source files, 214K lines, 72 modules
@entry_points [cmd/syndrdb, pkg/client, internal/graphQL/server]
@key_types [Bundle, Document, FieldValue, BundleFieldSchema, QueryPlan, ExecutionNode]
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@shape` | No | multi-line | Free-form structural summary. Entry points, data flow, key types, architectural boundaries, scale. One concept per line. |
| `@entry_points` | No | list | Packages that serve as entry points (CLIs, servers, library roots). Agents start navigation here. |
| `@key_types` | No | list | The 5-10 most important types in the project. Types that appear in many signatures across many modules. |

The `@shape` block is intentionally free-form. Different projects have different architectural concepts worth surfacing. The only rule is one concept per continuation line, keeping each line greppable and parseable.

`@entry_points` and `@key_types` are structured lists that enable programmatic use — Cartograph-style tools can prioritize these nodes in the graph.

#### Delta fields

Delta fields track what changed since a reference point, enabling agents to skip re-reading unchanged modules across conversations.

```
@manifest
@project SyndrDB
@aid_version 0.1
@snapshot_version git:abc1234
@snapshot_timestamp 2026-03-27T18:00:00Z
@delta
  modified: [query/planner, storage/flusher]
  added: [monitoring/metrics]
  removed: [legacy/importer]
  unchanged: 69 packages
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@snapshot_version` | No | string | Reference point for the delta. Format: `git:SHORT_HASH`. Agents compare this against their last-seen version. |
| `@snapshot_timestamp` | No | string | ISO 8601 timestamp of when the snapshot was generated. Agents use this to judge staleness. |
| `@delta` | No | multi-line | Changes since the previous snapshot version. Lines use `key: [list]` format. |

Delta line keys:

| Key | Meaning |
|-----|---------|
| `modified` | Packages whose AID files changed (agent should re-read these) |
| `added` | New packages not present in the previous snapshot |
| `removed` | Packages that no longer exist |
| `unchanged` | Count or list of packages with no changes (agent can skip these) |

**Agent workflow with snapshots:**

1. Read manifest header — get `@shape` for project orientation, `@snapshot_version` for delta tracking
2. Compare `@snapshot_version` against the agent's last-seen version (stored in conversation memory or session state)
3. If versions match: the agent's cached understanding is current. No AID files need to be loaded.
4. If versions differ: read `@delta` to identify what changed. Load only `modified` and `added` packages.
5. If no previous version exists (first contact): use `@shape` + `@entry_points` + `@key_types` for orientation, then selectively load packages as needed.

This workflow is additive — agents that don't track versions simply ignore the delta fields and use the manifest as before.

#### Full manifest example with snapshot

```
@manifest
@project SyndrDB
@aid_version 0.1
@lang go
@shape
  Entry points: cmd/syndrdb (CLI), pkg/client (library), internal/graphQL (API)
  Data flow: query → parser → planner → executor → storage → bundlestore
  Key types: Bundle, Document, FieldValue, BundleFieldSchema, QueryPlan, ExecutionNode
  Boundaries: domain (pure logic), storage (disk I/O), query (planning + execution), server (network)
  Scale: 496 source files, 214K lines, 72 modules
@entry_points [cmd/syndrdb, pkg/client, internal/graphQL/server]
@key_types [Bundle, Document, FieldValue, BundleFieldSchema, QueryPlan, ExecutionNode]
@snapshot_version git:f56db52
@snapshot_timestamp 2026-03-27T18:00:00Z
@delta
  modified: [query/planner, storage/flusher]
  added: [monitoring/metrics]
  removed: [legacy/importer]
  unchanged: 69 packages

---

@package query/planner
@aid_file planner.aid
@aid_status reviewed
@depends [syndrQL, domain/index, domain/models]
@purpose Query planning and optimization — converts parsed queries into execution plans
@layer l2
@key_risks BRIN is lossy (needs FilterNode), index selection order matters, plan immutability

---

@package monitoring/metrics
@aid_file metrics.aid
@aid_status draft
@depends [domain/models]
@purpose Prometheus metrics collection and export
@layer l1
```

#### Snapshot generation

Snapshots should be auto-generated by tooling, not manually maintained. The generation workflow:

1. Run `aid-manifest-gen` after AID files are generated or updated
2. Tool computes `@snapshot_version` from the current git HEAD
3. Tool diffs against the previous manifest to produce `@delta`
4. Tool extracts `@shape`, `@entry_points`, and `@key_types` from the AID files (entry points from packages with CLI/server entry kinds, key types from types that appear in the most cross-module `@sig` references)

The `@shape` block may be seeded automatically and refined by a human or L2 agent. The structured fields (`@entry_points`, `@key_types`, `@snapshot_version`, `@delta`) are fully automatable.

### 11.5 Discovery protocol

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

## 12. Project file (`project.aid`)

Large projects benefit from project-level documentation that spans all modules. While `manifest.aid` indexes individual modules, `project.aid` captures architectural context, cross-cutting concerns, conventions, and lifecycle information that no single module owns.

`project.aid` lives alongside `manifest.aid` in `.aidocs/` and uses the same parsing model: a header followed by entries separated by `---`.

**Co-evolution with `manifest.aid`:** When both files exist, they must stay consistent. Modules referenced in `project.aid` `@modules` lists should exist in `manifest.aid`. When `manifest.aid` adds or removes packages, `project.aid` `@layers`, `@boundaries`, and `@cross_cutting` entries may need updating. The `aid-validate` tool should warn on references to modules that don't exist in the manifest.

### 12.1 Project header

| Field | Required | Description |
|-------|----------|-------------|
| `@project` | Yes | Project name (same as manifest `@project`) |
| `@lang` | Yes | Primary language |
| `@aid_version` | No | AID spec version. Default: latest |
| `@layers` | No | Named architectural layers, ordered outermost to innermost |
| `@boundaries` | No | Dependency rules between layers/modules |
| `@patterns` | No | Project-wide design patterns |
| `@owners` | No | Module ownership assignments |

#### `@layers`

Named architectural layers with one-line descriptions. Order is significant: first listed = outermost (entry points), last listed = innermost (core domain).

```
@layers
  cmd — CLI entry points and flag parsing
  server — HTTP/gRPC handlers, middleware, routing
  query — Query planning, optimization, execution
  storage — Disk I/O, file formats, caching
  domain — Pure business logic, types, validation
```

#### `@boundaries`

Dependency rules between layers or modules. Format: `source -> target: ALLOWED|FORBIDDEN [reason]`.

```
@boundaries
  domain -> storage: FORBIDDEN
  domain -> server: FORBIDDEN
  storage -> query: FORBIDDEN
  server -> storage: FORBIDDEN. Must go through query layer.
```

Agents check boundary rules before proposing imports. A `FORBIDDEN` boundary means code in the source layer must not import from the target layer.

#### `@patterns`

Project-wide design patterns. Format: `pattern_name: description`.

```
@patterns
  error_handling: Wrap with context at layer boundaries using fmt.Errorf
  dependency_injection: Constructor injection, no global singletons
  concurrency: Channel-based communication between goroutines
  testing: Table-driven tests, interfaces for mocking
```

#### `@owners`

Module ownership. Format: `module_glob: team/person`.

```
@owners
  query/*: @query-team
  storage/*: @storage-team
  server/*: @platform-team
```

### 12.2 Cross-cutting concern entries (`@cross_cutting`)

Cross-cutting concerns document patterns that span multiple modules — authentication flows, error handling strategy, observability, middleware chains. They are the project-level equivalent of `@workflow`.

**Interaction with `@workflow`:** When a module-level `@workflow` overlaps with a `@cross_cutting` entry (e.g., both describe auth), they serve complementary roles. `@cross_cutting` provides the authoritative cross-module flow — how the concern spans the entire system. Module-level `@workflow` provides internal detail for that module's part of the flow. Agents should read `@cross_cutting` for the big picture and module `@workflow` for implementation detail within a single module.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@cross_cutting` | Yes | Concern name (snake_case) |
| `@purpose` | Yes | One-line description |
| `@modules` | Yes | List of modules involved in this concern |
| `@flow` | No | Numbered sequence showing how the concern flows across modules |
| `@errors` | No | Errors specific to this concern, with module-qualified names |
| `@patterns` | No | Patterns specific to this concern |
| `@antipatterns` | No | Mistakes to avoid |
| `@config` | No | Configuration this concern depends on (env vars, config keys) |

#### Example

```
@cross_cutting auth_flow
@purpose JWT-based authentication for all HTTP endpoints
@modules [server/middleware, server/auth, domain/user]
@flow
  1. Request arrives at server/middleware.AuthMiddleware
  2. Extract JWT from Authorization header
  3. Validate token via server/auth.ValidateToken
  4. Attach domain/user.User to request context
  5. Downstream handlers access user via ctx.Value(UserKey)
@errors
  server/auth.InvalidTokenError — malformed or expired JWT
  server/auth.UnauthorizedError — valid JWT but insufficient permissions
@antipatterns
  - Don't validate tokens in individual handlers. Always use AuthMiddleware.
  - Don't pass user as a function parameter. Use context.
```

### 12.3 Convention entries (`@convention`)

Conventions document project-wide coding standards that new code must follow. These capture the knowledge agents need to write code that fits the existing codebase.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@convention` | Yes | Convention name (snake_case) |
| `@purpose` | Yes | One-line description |
| `@rule` | Yes | Individual convention rules. Repeatable — multiple `@rule` lines accumulate (like `@sig`). |
| `@example` | No | Code example showing correct usage |
| `@antipatterns` | No | Common violations to avoid |

#### Example

```
@convention error_wrapping
@purpose How to wrap errors at layer boundaries
@rule Wrap with fmt.Errorf("[layer]: [context]: %w", err) at every boundary
@rule Never wrap within the same layer — return the error directly
@rule Use errors.Is() for matching, never type assertion
@example
  result, err := storage.Get(id)
  if err != nil {
    return fmt.Errorf("query: fetch document: %w", err)
  }
```

Note: `@rule` is an accumulating field — multiple `@rule` lines on the same entry are collected into a list, not overwritten. This follows the same pattern as `@sig` on overloaded functions.

### 12.4 Lifecycle entries (`@lifecycle`)

Lifecycle entries document initialization, shutdown, and other lifecycle sequences.

#### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `@lifecycle` | Yes | Lifecycle name (`startup`, `shutdown`, `migration`, or custom) |
| `@purpose` | Yes | One-line description |
| `@steps` | Yes | Numbered sequence of operations (same syntax as `@workflow` steps) |
| `@shutdown_order` | No | `reverse` or custom ordering description |
| `@timeout` | No | Timeout for this lifecycle phase |

#### Example

```
@lifecycle startup
@purpose Application initialization sequence
@steps
  1. config.Load() — parse config.yaml + env vars
  2. logging.Init(config.LogLevel) — structured logging setup
  3. storage.Initialize(config.DatabaseURL) — create connection pool
  4. query.Initialize(storage.Pool) — start query engine
  5. server.Start(config.Port) — begin accepting connections
@shutdown_order reverse
@timeout 30s
```

### 12.5 Decision entries

`@decision` entries may also appear in `project.aid` for project-level architectural decisions (same syntax as module-level decisions in Section 5.3).

### 12.6 Full `project.aid` example

```
@project SyndrDB
@lang go
@aid_version 0.2

@layers
  cmd — CLI entry points and flag parsing
  server — HTTP/gRPC handlers, middleware, routing
  query — Query planning, optimization, execution
  storage — Disk I/O, file formats, caching
  domain — Pure business logic, types, validation

@boundaries
  domain -> storage: FORBIDDEN
  domain -> server: FORBIDDEN
  storage -> query: FORBIDDEN
  server -> storage: FORBIDDEN. Must go through query layer.

@patterns
  error_handling: Wrap with context at layer boundaries using fmt.Errorf
  dependency_injection: Constructor injection, no global singletons
  concurrency: Channel-based communication between goroutines
  testing: Table-driven tests, interfaces for mocking

---

@cross_cutting error_propagation
@purpose How errors flow from storage through query to server responses
@modules [domain, storage, query, server]
@flow
  1. Domain defines error types (NotFoundError, ValidationError)
  2. Storage wraps I/O errors: fmt.Errorf("storage: %w", err)
  3. Query propagates domain errors, wraps storage errors
  4. Server maps error types to HTTP status codes via error_mapper middleware
@patterns
  wrap_at_boundary: fmt.Errorf("[layer]: [context]: %w", err)
  never_log_and_return: Either log OR return, not both

---

@cross_cutting observability
@purpose Distributed tracing and metrics across all layers
@modules [server/middleware, query, storage]
@flow
  1. server/middleware.TracingMiddleware creates span from request
  2. Span propagated via context.Context through all layers
  3. query and storage create child spans for their operations
  4. Metrics exported via /metrics endpoint (Prometheus format)
@config
  OTEL_ENDPOINT: str — OpenTelemetry collector URL. Default "localhost:4317".
  METRICS_PORT: int — Prometheus metrics port. Default 9090.

---

@convention error_wrapping
@purpose Error wrapping at layer boundaries
@rule Wrap with fmt.Errorf("[layer]: [context]: %w", err) at every boundary
@rule Never wrap within the same layer
@rule Use errors.Is() for matching, never type assertion

---

@convention naming
@purpose Project naming conventions
@rule Types: PascalCase
@rule Exported functions: PascalCase
@rule Unexported functions: camelCase
@rule Files: snake_case.go
@rule Packages: single lowercase word, no underscores

---

@lifecycle startup
@purpose Application initialization sequence
@steps
  1. config.Load() — parse config.yaml + env vars
  2. logging.Init(config.LogLevel) — structured logging setup
  3. storage.Initialize(config.DatabaseURL) — connection pool
  4. query.Initialize(storage.Pool) — query engine
  5. server.Start(config.Port) — begin accepting connections
@shutdown_order reverse

---

@decision layered_architecture
@purpose Why strict layer boundaries exist
@context Go circular imports are compile errors. Team of 5 devs.
@chosen Strict layered architecture with one-way deps
@rejected Hexagonal/ports-and-adapters (too much boilerplate for team size)
@rationale Layer violations cause circular imports in Go. Strict layers keep build times
  low and make dependency direction predictable for agents.
```

---

## 13. Examples

### 13.1 Example blocks

The `@example` field on entries contains minimal usage examples. Rules:

- In real AID files, examples **must** use the language specified by the module's `@lang` field
- Multi-line examples are indented continuation lines (standard AID syntax)
- Examples should show the ONE thing the entry does — not a full program
- Examples are patterns for an agent to follow, not executable tests

**Note on spec examples:** The examples in this specification document and in `examples/*.aid` use a language-neutral pseudocode (`:=` for assignment, `?` for error propagation, `{}` for struct literals) for illustrative purposes. This pseudocode is intentional — the spec must be readable regardless of the reader's language background. Real-world AID files generated for a Go project should use Go syntax, for Python should use Python syntax, etc.

```
// Spec pseudocode (illustrative):
@example
  resp := http.get("https://api.example.com/users")?
  data := resp.json[[]User]()?

// Real Go AID file:
@example
  resp, err := http.Get("https://api.example.com/users")
  if err != nil { return err }
  var data []User
  err = resp.JSON(&data)
```

### 13.2 When to include examples

Layer 1 extractors should only include examples from existing docstrings. Layer 2 generators may synthesize examples when the usage pattern is non-obvious — especially for workflows and entries with complex constraints.

---

## 14. Security considerations

AID files are documentation artifacts. They carry the same trust level as the source code they describe.

- **`[src:]` references are relative paths.** Tools must validate that resolved paths stay within the project root. A malicious AID file could reference `../../etc/passwd` — path traversal must be prevented.
- **Generated AID should be reviewed before committing.** The same way generated code is reviewed. The L2 generator→reviewer pipeline provides automated review, but human review is appropriate for critical systems.
- **Don't generate AID from untrusted source code without review.** If the source contains prompt injection patterns (in comments, docstrings, or string literals), the L2 generator could be influenced to produce misleading documentation.
- **AID files should be committed to version control.** They are project artifacts, not ephemeral outputs. Committing them provides auditability and enables staleness detection via `@code_version`.

---

## 15. Migration

### 15.1 Spec version compatibility

The `@aid_version` field declares which AID spec version the file targets. Compatibility rules:

- **Parsers must handle older spec versions gracefully.** Unknown fields are ignored (forward compatibility). Missing new fields use defaults.
- **Minor version changes are additive only.** New fields may be added; existing fields retain their semantics. AID 0.1 files are valid AID 0.2 files, and AID 0.2 files are valid AID 0.3 files. The v0.3 Tier 4 additions (`@graph`, `@tool`, `@agent`, `@prompt`, `@model`, state channels/reducers, the `@engine`/`@determinism` fields, and the `Llm`/`Tool`/`Embed`/`Stream` effects) introduce no breaking changes — a 0.2 parser ignores the new entry keywords and fields, and a 0.3 parser reads 0.2 files unchanged.
- **Breaking changes require a major version bump.** Field semantics may change or fields may be removed only in major versions (0.x → 1.0 allows breaking changes, since pre-1.0 is unstable).

### 15.2 Updating AID files

When the spec changes, existing AID files are updated by re-running the generation pipeline:

1. Layer 1: re-run the extractor — captures any new fields the extractor now produces
2. Layer 2: re-run the generator and reviewer — applies new semantic field requirements
3. Manual edits: only needed if field semantics changed (major version)

The `aid-gen-l2 stale` command detects files that need re-generation by comparing `@code_version` against the current git HEAD.
