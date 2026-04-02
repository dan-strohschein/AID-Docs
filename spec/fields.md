# AID Field Reference

**Complete reference for every field in the AID format.**

---

## Module header fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@module` | Yes | string | Fully qualified module name |
| `@lang` | Yes | string | Source language identifier |
| `@version` | Yes | semver | Library version this AID file describes |
| `@stability` | No | enum | `unknown`, `experimental`, `unstable`, `stable`, `deprecated`. Default: `unknown` |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@depends` | No | list | Modules/packages this module depends on (for selective AID loading) |
| `@source` | No | URL | Link to source code or docs |
| `@code_version` | No | string | Git commit hash this AID describes. Format: `git:HASH` |
| `@aid_status` | No | enum | `draft`, `reviewed`, `approved`, `stale`. Default: `draft` |
| `@aid_generated_by` | No | string | Agent role that produced this AID |
| `@aid_reviewed_by` | No | string | Agent role that verified this AID |
| `@aid_version` | No | semver | AID spec version. Default: latest |
| `@test_framework` | No | string | Test framework name (e.g., `go test`, `pytest`, `jest`) |
| `@test_cmd` | No | string | Command to run this module's tests |
| `@test_fixtures` | No | string | Path to test fixtures/data relative to module root |
| `@mock_strategy` | No | string | How dependencies are mocked |
| `@env` | No | block | Environment variables this module reads. Uses constraint syntax. Mark secrets with `Sensitive.` |
| `@services` | No | block | External services. Format: `ServiceName: ENV_VAR — purpose` |
| `@config_files` | No | block | Config files. Format: `filename — description` |
| `@init_order` | No | int | Initialization order. Lower numbers init first. |
| `@init_fn` | No | string | Function that initializes this module |
| `@shutdown_fn` | No | string | Function that shuts down this module |
| `@global_state` | No | block | Module-level mutable state. Format: `name: Type — description` |

---

## Function fields (`@fn`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@fn` | Yes | string | Function or method name. Methods use dot notation: `Type.method` |
| `@visibility` | No | enum | `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@sig` | Yes | signature | Full type signature (see signature syntax) |
| `@params` | Conditional | block | Parameter descriptions. Required if function has params. |
| `@returns` | No | string | Return value description and guarantees |
| `@errors` | Conditional | block | Error types and conditions. Required if function can error. |
| `@pre` | No | string/block | Preconditions. Use `None` to explicitly state there are none. |
| `@post` | No | string/block | Postconditions guaranteed after successful return |
| `@effects` | No | list | Side effect categories: `Net`, `Fs`, `Io`, `Env`, `Time`, `Random`, `Db`, `Process`, `Gpu`, `Config`, `Callback` |
| `@thread_safety` | No | string | Structured keyword first (`safe`, `immutable`, `channel-based`, `requires-sync`, `not-safe`), optional elaboration after. |
| `@complexity` | No | string | Time and/or space complexity |
| `@since` | No | semver | Version when introduced |
| `@deprecated` | No | string | Deprecation notice with replacement |
| `@related` | No | block | Typed block. Each line: `type: name [, name]`. Types: `calls`, `produces`, `consumes`, `sibling`, `wraps`, `inverse`, `replaces`. Untyped flat lists accepted as `sibling:` for backward compat. |
| `@calls` | No | list | Functions this function calls internally. Populated by Layer 1 extractors from AST analysis. |
| `@reads` | No | list | Fields this function reads. Format: `[Type.field, ...]` |
| `@writes` | No | list | Fields this function writes. Format: `[Type.field, ...]` |
| `@tested` | No | bool | Whether this function has test coverage. `true`/`false`. |
| `@test_hint` | No | list | Test function names that exercise this function |
| `@source_file` | No | string | Source file path relative to project root. Layer 1 field. |
| `@source_line` | No | int | Line number in source file. Layer 1 field. |
| `@platform` | No | block | Platform-specific behavior differences |
| `@example` | No | block | Minimal usage example |

### Signature syntax

```
(params) -> ReturnType
(params) -> ReturnType ! ErrorType
(params) -> ReturnType ! ErrorA | ErrorB | ErrorC
async (params) -> ReturnType ! ErrorType
```

Parameter syntax within signatures:
- `name: Type` — required parameter
- `name?: Type` — optional parameter (has default)
- `...name: Type` — variadic/rest parameter
- `name: Type = value` — parameter with explicit default
- `self` — immutable method receiver (not listed in `@params`)
- `mut self` — mutable method receiver (method modifies the instance)
- `async` — before params, marks the function as asynchronous (must be awaited/spawned)

### Overloaded signatures

A function may have multiple `@sig` lines when it accepts different argument shapes:

```
@sig (input: str) -> Value ! ParseError
@sig (input: bytes, encoding?: str) -> Value ! ParseError | EncodingError
```

Each line is a valid calling convention. Use overloads when different input types produce different return types or error sets. Use optional parameters (`param?: Type`) when the return type is the same regardless.

### Generic type parameters in signatures

Functions and methods can declare generic type parameters in brackets before the parameter list:

```
[T](items: [T]) -> T?
[K: Hash + Eq, V](self, key: K) -> V?
[T: Serializable](self) -> T ! ParseError
```

- `[T]` — unconstrained type parameter
- `[T: Bound]` — type parameter constrained by a trait/interface
- `[T: A + B]` — type parameter with multiple bounds (all must be satisfied)
- Multiple parameters separated by commas: `[K: Hash + Eq, V]`

Generic parameters on a method are independent of generic parameters on the parent type. The method may reference the type's parameters, introduce its own, or both.

### Bounds syntax

Bounds constrain generic type parameters to types that implement specific traits or interfaces:

```
K: Hash              // K must implement Hash
K: Hash + Eq         // K must implement both Hash and Eq
V                    // unconstrained — any type
```

Bounds appear in two places:
- In `@generic_params` on `@type` entries: `K: Hash + Eq, V`
- In `[...]` on `@sig` for generic functions/methods: `[K: Hash + Eq, V](self, key: K) -> V?`

### Parameter constraint keywords

| Constraint | Syntax | Example |
|-----------|--------|---------|
| Inclusive range | `Range [min, max]` | `Range [0, 100]` |
| Exclusive range | `Range (min, max)` | `Range (0, 1)` |
| Half-open range | `Range [min, max)` | `Range [0, 256)` |
| Comparison | `Must be > N` | `Must be > 0` |
| Pattern | `Must match ^regex$` | `Must match ^https?://` |
| Enumeration | `One of: a, b, c` | `One of: GET, POST, PUT` |
| Length | `Length [min, max]` | `Length [1, 255]` |
| Not null | `Required.` | `Required.` |
| Default | `Default VALUE.` | `Default 30s.` |

### Effect categories

| Effect | Meaning |
|--------|---------|
| `Net` | Network I/O (HTTP, TCP, UDP, DNS) |
| `Fs` | Filesystem read/write |
| `Io` | General I/O (stdin/stdout, devices) |
| `Env` | Environment variable access |
| `Time` | System clock access |
| `Random` | Non-deterministic random generation |
| `Db` | Database operations |
| `Process` | Process spawning, signals |
| `Gpu` | GPU computation |
| `Config` | Reads configuration (env vars, config files) at runtime. Distinct from `Env` which covers process environment mutation. |
| `Callback` | Effects depend on caller-provided function arguments. Cannot determine purity without inspecting inputs. |

---

## Type fields (`@type`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@type` | Yes | string | Type name |
| `@visibility` | No | enum | `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@kind` | Yes | enum | `struct`, `enum`, `union`, `class`, `alias`, `newtype` |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@fields` | Conditional | block | Field descriptions. Required for struct/class. |
| `@fields_visibility` | No | enum | `full` or `partial`. Signals if `@fields` lists all fields. Default: `full`. |
| `@variants` | Conditional | block | Variant descriptions. Required for enum/union. |
| `@invariants` | No | block | Properties that always hold for valid instances |
| `@constructors` | No | string/block | How to create instances. `None` if factory-produced only. |
| `@methods` | No | list | Method names (details in separate @fn entries) |
| `@extends` | No | list | Parent class(es) this type inherits from |
| `@implements` | No | list | Traits/interfaces/protocols implemented (see well-known protocols) |
| `@generic_params` | No | string | Type parameters with bounds |
| `@platform` | No | block | Platform-specific behavior differences |
| `@since` | No | semver | Version introduced |
| `@deprecated` | No | string | Deprecation notice |
| `@related` | No | list | Related types and functions |
| `@example` | No | block | Construction and usage |

### Kind values

| Kind | Use for |
|------|---------|
| `struct` | Product types, records, plain data objects |
| `enum` | Sum types, tagged unions, algebraic data types |
| `union` | Untagged unions (TypeScript-style) |
| `class` | Class types (Python, Java, TypeScript) |
| `alias` | Type aliases (interchangeable with the aliased type) |
| `newtype` | Distinct wrapper types (not interchangeable) |

---

## Trait/interface fields (`@trait`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@trait` | Yes | string | Trait/interface name |
| `@visibility` | No | enum | `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@purpose` | Yes | string | One-line description |
| `@requires` | Yes | block | Method signatures implementors must provide |
| `@provided` | No | block | Methods with default implementations |
| `@implementors` | No | list | Known implementing types |
| `@extends` | No | list | Parent traits |
| `@related` | No | list | Related traits and types |

---

## Constant fields (`@const`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@const` | Yes | string | Constant name |
| `@visibility` | No | enum | `public`, `internal`, `protected`, `private`. Default: `public`. |
| `@purpose` | Yes | string | One-line description |
| `@value_type` | Yes | string | The constant's type. Named `@value_type` to avoid ambiguity with the `@type` entry keyword. |
| `@value` | No | string | The constant's value |
| `@since` | No | semver | Version introduced |

---

## Workflow fields (`@workflow`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@workflow` | Yes | string | Workflow name (snake_case) |
| `@purpose` | Yes | string | What this workflow accomplishes |
| `@steps` | Yes | block | Numbered sequence of operations |
| `@errors_at` | No | block | Errors mapped to specific steps |
| `@antipatterns` | No | block | Common mistakes to avoid |
| `@variants` | No | block | Alternative paths through the workflow |
| `@example` | No | block | Complete worked example |

---

## Module annotation fields

### Invariants block

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@invariants` | — | block | Module-level constraints. Each line prefixed with `- `. Include `[src:]` references. |

### Antipatterns block

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@antipatterns` | — | block | Module-level mistakes to avoid. Each line prefixed with `- `. Include `[src:]` references. |

### Decision record (`@decision`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@decision` | Yes | string | Decision name (snake_case) |
| `@purpose` | Yes | string | What question this decision answers |
| `@context` | No | string/block | Constraints that existed when the decision was made |
| `@chosen` | Yes | string | What was chosen |
| `@rejected` | No | string/block | What was considered and rejected |
| `@rationale` | Yes | string/block | Why, with `[src:]` references |

### Note (`@note`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@note` | Yes | string | Note name (descriptive identifier) |
| `@purpose` | Yes | string | What this note communicates |

---

## Project file fields (`project.aid`)

### Project header fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@project` | Yes | string | Project name (same as manifest `@project`) |
| `@lang` | Yes | string | Primary language |
| `@aid_version` | No | semver | AID spec version |
| `@layers` | No | block | Named architectural layers, ordered outermost to innermost. One layer per line: `name — description` |
| `@boundaries` | No | block | Dependency rules. Format: `source -> target: ALLOWED\|FORBIDDEN [reason]` |
| `@patterns` | No | block | Project-wide design patterns. Format: `pattern_name: description` |
| `@owners` | No | block | Module ownership. Format: `module_glob: team/person` |

### Cross-cutting concern fields (`@cross_cutting`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@cross_cutting` | Yes | string | Concern name (snake_case) |
| `@purpose` | Yes | string | One-line description |
| `@modules` | Yes | list | Modules involved in this concern |
| `@flow` | No | block | Numbered steps showing how the concern flows across modules |
| `@errors` | No | block | Errors specific to this concern, with module-qualified names |
| `@patterns` | No | block | Patterns specific to this concern |
| `@antipatterns` | No | block | Mistakes to avoid |
| `@config` | No | block | Configuration this concern depends on |

### Convention fields (`@convention`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@convention` | Yes | string | Convention name (snake_case) |
| `@purpose` | Yes | string | One-line description |
| `@rule` | Yes | string (repeatable) | Convention rules. Multiple `@rule` lines accumulate into a list. |
| `@example` | No | block | Code example showing correct usage |
| `@antipatterns` | No | block | Common violations to avoid |

### Lifecycle fields (`@lifecycle`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@lifecycle` | Yes | string | Lifecycle name (`startup`, `shutdown`, `migration`, or custom) |
| `@purpose` | Yes | string | One-line description |
| `@steps` | Yes | block | Numbered sequence (same syntax as `@workflow` steps) |
| `@shutdown_order` | No | string | `reverse` or custom ordering description |
| `@timeout` | No | string | Timeout for this lifecycle phase |

---

## Manifest fields (`manifest.aid`)

### Core manifest fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@manifest` | Yes | — | Marker identifying this as a manifest file |
| `@project` | Yes | string | Project name |
| `@package` | Yes | string | Full package path (one per entry in manifest) |
| `@aid_file` | Yes | string | Filename in `.aidocs/` |
| `@aid_status` | No | enum | `draft`, `reviewed`, `approved`, `stale` |
| `@depends` | No | list | Packages this one calls into |
| `@purpose` | Yes | string | One-line description for relevance filtering |
| `@layer` | No | enum | `l1` or `l2` — documentation depth available |
| `@key_risks` | No | string | 1-2 critical things about this package |

### Project snapshot fields (manifest header)

These fields appear in the manifest header (before the first `---`), alongside `@manifest` and `@project`.

#### Shape fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@shape` | No | multi-line | Free-form structural summary: entry points, data flow, key types, boundaries, scale. One concept per continuation line. |
| `@entry_points` | No | list | Packages that serve as entry points (CLIs, servers, library roots) |
| `@key_types` | No | list | The 5-10 most important types in the project — types referenced across many modules |

#### Delta fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@snapshot_version` | No | string | Git reference point for the delta. Format: `git:SHORT_HASH` |
| `@snapshot_timestamp` | No | string | ISO 8601 timestamp of snapshot generation |
| `@delta` | No | multi-line | Changes since previous snapshot. Lines use `key: [list]` or `key: count` format |

Delta line keys:

| Key | Type | Description |
|-----|------|-------------|
| `modified` | list | Packages whose AID files changed since `@snapshot_version` |
| `added` | list | New packages not in the previous snapshot |
| `removed` | list | Packages that no longer exist |
| `unchanged` | int or list | Packages with no changes (count for brevity, list for precision) |

---

## Universal type notation

These type names are used in signatures regardless of source language. Tooling uses `@lang` to map to language-specific types.

| AID type | Description | Python | Go | TypeScript |
|----------|-------------|--------|----|------------|
| `str` | String | `str` | `string` | `string` |
| `int` | Integer | `int` | `int` | `number` |
| `i8`..`i64` | Sized signed int | — | `int8`..`int64` | `number` |
| `u8`..`u64` | Sized unsigned int | — | `uint8`..`uint64` | `number` |
| `float` | Float | `float` | `float64` | `number` |
| `f32`, `f64` | Sized float | — | `float32`, `float64` | `number` |
| `bool` | Boolean | `bool` | `bool` | `boolean` |
| `bytes` | Byte sequence | `bytes` | `[]byte` | `Uint8Array` |
| `None` | No value | `None` | — | `void` |
| `[T]` | List/array | `list[T]` | `[]T` | `T[]` |
| `dict[K, V]` | Map | `dict[K, V]` | `map[K]V` | `Record<K, V>` |
| `set[T]` | Set | `set[T]` | — | `Set<T>` |
| `T?` | Optional | `Optional[T]` | `*T` | `T \| undefined` |
| `T ! E` | Result | raises `E` | `(T, error)` | `throws E` |
| `fn(A) -> B` | Function | `Callable[[A], B]` | `func(A) B` | `(a: A) => B` |
| `(A, B)` | Tuple | `tuple[A, B]` | — | `[A, B]` |
| `any` | Any type | `Any` | `any` | `any` |

---

## Well-known protocols

AID-universal protocol names for `@implements`. These map to language-specific constructs:

| Protocol | Meaning | Python | Go | TypeScript | Rust |
|----------|---------|--------|----|------------|------|
| `Closeable` | Resource cleanup. Use language resource syntax. | `with` / `__enter__`+`__exit__` | `io.Closer` / `defer` | `Disposable` / `using` | `Drop` |
| `Iterable` | Produces an iterator. | `__iter__` | `range` | `Symbol.iterator` | `IntoIterator` |
| `Iterator` | Stateful cursor yielding elements. | `__next__` | N/A | `next()` | `Iterator` |
| `Comparable` | Supports ordering. | `__lt__` etc. | `sort.Interface` | N/A | `Ord` |
| `Hashable` | Usable as hash map key. | `__hash__` | comparable | N/A | `Hash` |
| `Display` | Human-readable string form. | `__str__` | `fmt.Stringer` | `toString()` | `Display` |
| `Debug` | Developer-oriented string form. | `__repr__` | `fmt.GoStringer` | `inspect()` | `Debug` |
| `Serializable` | Wire format conversion. | various | `json.Marshaler` | various | `Serialize` |
| `Cloneable` | Independent copy. | `__copy__`/`__deepcopy__` | value types | `structuredClone` | `Clone` |
| `Callable` | Invocable as function. | `__call__` | N/A | call signature | `Fn`/`FnMut` |

When an agent sees `@implements [Closeable]`, it should use the appropriate resource management syntax for the target language (e.g., `with` in Python, `defer x.Close()` in Go).

---

## Platform names

Standard platform identifiers for `@platform` fields:

| Platform | Meaning |
|----------|---------|
| `windows` | Microsoft Windows |
| `linux` | Linux |
| `macos` | macOS / Darwin |
| `bsd` | FreeBSD, OpenBSD, NetBSD |
| `wasm` | WebAssembly |
| `android` | Android |
| `ios` | iOS / iPadOS |
