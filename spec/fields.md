# AID Field Reference

**Complete reference for every field in the AID format.**

---

## Module header fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@module` | Yes | string | Fully qualified module name |
| `@lang` | Yes | string | Source language identifier |
| `@version` | Yes | semver | Library version this AID file describes |
| `@stability` | No | enum | `experimental`, `unstable`, `stable`, `deprecated`. Default: `stable` |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@deps` | No | list | Module dependencies |
| `@source` | No | URL | Link to source code or docs |
| `@aid_version` | No | semver | AID spec version. Default: latest |

---

## Function fields (`@fn`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@fn` | Yes | string | Function or method name. Methods use dot notation: `Type.method` |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@sig` | Yes | signature | Full type signature (see signature syntax) |
| `@params` | Conditional | block | Parameter descriptions. Required if function has params. |
| `@returns` | No | string | Return value description and guarantees |
| `@errors` | Conditional | block | Error types and conditions. Required if function can error. |
| `@pre` | No | string/block | Preconditions. Use `None` to explicitly state there are none. |
| `@post` | No | string/block | Postconditions guaranteed after successful return |
| `@effects` | No | list | Side effect categories: `Net`, `Fs`, `Io`, `Env`, `Time`, `Random`, `Db`, `Process`, `Callback` |
| `@thread_safety` | No | string | Concurrency safety description |
| `@complexity` | No | string | Time and/or space complexity |
| `@since` | No | semver | Version when introduced |
| `@deprecated` | No | string | Deprecation notice with replacement |
| `@related` | No | list | Names of related entries. Bare names for same module, `module/path.Name` for cross-module. |
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
| `Callback` | Effects depend on caller-provided function arguments. Cannot determine purity without inspecting inputs. |

---

## Type fields (`@type`)

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `@type` | Yes | string | Type name |
| `@kind` | Yes | enum | `struct`, `enum`, `union`, `class`, `alias`, `newtype` |
| `@purpose` | Yes | string | One-line description. Max 120 chars. |
| `@fields` | Conditional | block | Field descriptions. Required for struct/class. |
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
| `@purpose` | Yes | string | One-line description |
| `@type` | Yes | string | The constant's type |
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
