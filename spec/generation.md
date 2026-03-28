# AID Generation Design

**How AID files are produced from source code and documentation.**

---

## 1. Generation architecture

AID generation is a three-layer pipeline. Each layer produces progressively higher-quality output. The layers are designed to be independent — a project can stop at any layer and still have useful AID files.

### Layer 1: Mechanical extraction

Parse source code and extract everything that has a direct structural mapping to AID fields. No inference, no guessing. This layer is deterministic and fast.

**What it produces:**
- Module header (`@module`, `@lang`, `@version`)
- Function entries with `@fn`, `@sig`, `@params` (names and types only)
- Type entries with `@type`, `@kind`, `@fields` or `@variants`
- Trait entries with `@trait`, `@requires`
- Constant entries with `@const`, `@type`, `@value`
- `@returns` (type only, no semantic description)
- `@deprecated` (from language-level deprecation markers)
- `@since` (from version control history, if available)

**What it cannot produce:**
- `@purpose` (requires understanding intent)
- `@errors` (requires tracing all raise/throw paths, not just declared exceptions)
- `@pre` / `@post` (requires understanding contracts)
- `@effects` (requires classifying I/O behavior)
- `@invariants` (requires understanding type guarantees)
- `@thread_safety` (requires concurrency analysis)
- `@workflow` (requires understanding multi-step patterns)
- `@antipatterns` (requires experience/tribal knowledge)
- Constraint syntax (`Range`, `Must be`, `Must match`) — types give you the type, not the valid range

**Quality tier: Skeleton.** Useful as a starting point. Missing most of what makes AID valuable.

### Layer 2: AI-assisted inference

An AI agent reads the mechanical skeleton alongside the source code, docstrings, existing documentation, README, and tests. It fills in the semantic fields that require understanding.

**What it adds:**
- `@purpose` for every entry (synthesized from docstrings, naming, context)
- `@errors` (traced from source code `raise`/`throw`/`return error` paths)
- `@params` constraints (inferred from validation code, docstrings, tests)
- `@pre` / `@post` (inferred from assertions, documentation, usage patterns)
- `@effects` (classified from I/O calls in the implementation)
- `@invariants` (inferred from constructors, validation, and usage)
- `@thread_safety` (inferred from locking, shared state, documentation)
- `@complexity` (inferred from algorithm structure)
- `@related` (inferred from call graphs and type relationships)
- `@workflow` blocks (synthesized from README examples, test patterns, and API shape)
- `@antipatterns` (inferred from common mistakes in issues, tests, and documentation)

**What it struggles with:**
- Constraints not enforced in code (API rate limits, external system requirements)
- Error conditions from transitive dependencies
- Thread safety of compositions (individual functions may be safe but combined usage may not be)
- Workflows spanning multiple modules
- Antipatterns only known to experienced users

**Quality tier: Production-ready for most use cases.** Good enough for an AI agent to use the library correctly. May miss edge cases.

### Layer 3: Human review

A human (library author or experienced user) reviews the AI-generated AID file and:
- Corrects any inaccurate inferences
- Adds constraints known only from domain experience
- Adds workflows for advanced usage patterns
- Adds antipatterns learned from production incidents
- Marks the file as reviewed (confidence marker for consumers)

**Quality tier: Authoritative.** The AID file is a trusted contract.

---

## 2. Language-specific extraction mappings

### 2.1 Python

| Source construct | AID field | Notes |
|-----------------|-----------|-------|
| `def name(...)` | `@fn name` | |
| `def name(...) -> Type` | `@sig` return type | |
| Parameter with type hint | `@sig` param type | `param: str` → `param: str` |
| Parameter with default | `@sig` optional marker | `param: str = "x"` → `param?: str` |
| `*args: Type` | `@sig` variadic | `...args: Type` |
| `**kwargs: Type` | `@params` sub-fields | Expand known keys if documented |
| `class Name:` | `@type Name` / `@kind class` | |
| `@dataclass` | `@type Name` / `@kind struct` | Fields from class attributes |
| `class Name(Enum):` | `@type Name` / `@kind enum` | Variants from members |
| `class Name(TypedDict):` | `@type Name` / `@kind struct` | Fields from annotations |
| `class Name(Protocol):` | `@trait Name` | `@requires` from methods |
| `ABC` / `abstractmethod` | `@trait Name` | `@requires` from abstract methods |
| `NAME = value` (module-level) | `@const NAME` | |
| Docstring (first line) | `@purpose` candidate | Layer 2 refines |
| `raise ExceptionType` | `@errors` candidate | Layer 2 traces all paths |
| `@deprecated` decorator | `@deprecated` | |
| `typing.Optional[T]` | `T?` in AID | |
| `typing.Union[A, B]` | Depends on context | Error union if in return type with exception, otherwise note in description |
| `self` parameter | `self` in `@sig` | Determine `mut self` by Layer 2 analysis of whether method mutates instance state |
| `async def name(...)` | `async` in `@sig` | `@sig async (params) -> ReturnType` |
| `@overload` decorator | Multiple `@sig` lines | Each overload becomes a separate `@sig` |
| `class Name(Base):` | `@extends Base` | Class inheritance — mechanical extraction |
| `class Name(Base1, Base2):` | `@extends Base1, Base2` | Multiple inheritance |
| `__enter__`/`__exit__` methods | `@implements [Closeable]` | Context manager protocol |
| `__iter__` method | `@implements [Iterable]` | Iterator protocol |
| `__hash__` method | `@implements [Hashable]` | Hashable protocol |
| `sys.platform` checks | `@platform` candidate | Layer 2 traces platform-conditional code |

#### Python-specific challenges

- **`**kwargs`**: Python's kwargs are unstructured. If the docstring or type stub enumerates known keys, map them to `@params` sub-fields. Otherwise, note `**kwargs: dict[str, any]` and flag for human review.
- **Exception hierarchy**: Python exceptions use inheritance. `@errors` should list the most specific exception, not the base class. Layer 2 must trace `raise` statements through the call graph.
- **Dynamic types**: Functions without type hints produce incomplete `@sig` entries. Layer 2 can infer types from docstrings, usage, and tests. Missing types should use `any` rather than being omitted.
- **Decorators**: `@property`, `@staticmethod`, `@classmethod` affect the signature. Properties should map to `@fields` on the parent type, not `@fn` entries. Static methods have no `self`. Class methods receive `cls` which maps to a constructor-like pattern.
- **Mutability inference**: Python has no `const` or `mut` keyword. Layer 2 must analyze method bodies — if the method writes to `self.x`, it's `mut self`. If it only reads, it's `self`. When uncertain, default to `mut self` (conservative).

### 2.2 Go

| Source construct | AID field | Notes |
|-----------------|-----------|-------|
| `func Name(...)` | `@fn Name` | |
| `func (t Type) Name(...)` | `@fn Type.Name` | Value receiver → `self`. Pointer receiver → `mut self`. |
| `(result Type, err error)` | `@sig` → `-> Type ! Error` | Map Go's multi-return error pattern to AID error syntax |
| `type Name struct` | `@type Name` / `@kind struct` | |
| `type Name interface` | `@trait Name` | |
| `type Name int` + `iota` | `@type Name` / `@kind enum` | Variants from const block |
| `type Name = Other` | `@type Name` / `@kind alias` | |
| Exported (`Name`) vs unexported (`name`) | Only export capitalized names | Unexported = internal, skip by default. Use `--internal` flag to include unexported functions with minimal info (`@fn` + `@sig` only) for call-graph tools like cartograph. |
| Godoc comment | `@purpose` candidate | First sentence |
| `const Name = value` | `@const Name` | |
| `io.Closer` implementation | `@implements [Closeable]` | |
| `sort.Interface` implementation | `@implements [Comparable]` | |
| `fmt.Stringer` implementation | `@implements [Display]` | |
| `runtime.GOOS` checks | `@platform` candidate | Layer 2 traces platform-conditional code |
| Embedded structs | `@extends` candidate | Go embedding is composition, not inheritance — Layer 2 decides |

#### Go-specific challenges

- **Error values vs. types**: Go uses both sentinel errors (`var ErrNotFound = errors.New(...)`) and error types (`type NotFoundError struct{...}`). Both map to `@errors` variants, but extraction differs.
- **Receiver mutability**: Pointer receivers (`*T`) are `mut self`. Value receivers (`T`) are `self`. This is a clean mechanical mapping.
- **Interface satisfaction**: Go interfaces are implicit. Layer 2 must analyze which types satisfy which interfaces for `@implements`.
- **Context parameter**: `ctx context.Context` as first parameter is idiomatic Go but not an API-level concern. Omit from `@sig` or include with a note that it's for cancellation/deadline propagation. Recommend: include in `@sig` as `ctx: Context` and note cancellation behavior in `@pre` or `@post`.

### 2.3 TypeScript

| Source construct | AID field | Notes |
|-----------------|-----------|-------|
| `function name(...)` | `@fn name` | |
| `class Name { method(...) }` | `@fn Name.method` | |
| `interface Name` | `@trait Name` | |
| `type Name = { ... }` | `@type Name` / `@kind struct` | |
| `type Name = A \| B` | `@type Name` / `@kind union` | |
| `enum Name` | `@type Name` / `@kind enum` | |
| `type Name = Other` | `@type Name` / `@kind alias` | |
| `readonly field` | `self` (not `mut self`) for accessor methods | |
| `Promise<T>` | `async` in `@sig` | `@sig async (params) -> T ! Error` — show resolved type, not Promise wrapper |
| `.d.ts` file | Primary extraction source | Types without implementation |
| JSDoc `@throws` | `@errors` candidate | |
| JSDoc `@deprecated` | `@deprecated` | |
| `as const` assertions | `@const` | |
| `class A extends B` | `@extends B` | Class inheritance |
| `class A implements I` | `@implements [I]` | Interface implementation |
| Function overloads | Multiple `@sig` lines | Each overload declaration becomes a `@sig` |
| `Symbol.dispose` / `Disposable` | `@implements [Closeable]` | TC39 explicit resource management |
| `Symbol.iterator` | `@implements [Iterable]` | |
| `process.platform` checks | `@platform` candidate | Layer 2 traces |
| `T extends Constraint` | `T: Constraint` in bounds | Generics with constraints |

#### TypeScript-specific challenges

- **Union types**: TypeScript unions (`string | number`) don't map cleanly to AID. Simple unions can use AID notation. Complex unions (discriminated unions) should map to `@type` with `@kind union` and `@variants`.
- **Overloads**: TypeScript allows multiple signatures for one function. Map each overload as a separate `@sig` line.
- **`Promise<T>`**: Async functions return promises. The AID `@sig` should use `async` keyword and show the resolved type: `async (params) -> T`, not `(params) -> Promise<T>`.
- **Generics**: TypeScript generics map directly to AID generics. Constraints (`T extends Foo`) map to bounds (`T: Foo`).

---

## 3. Generation pipeline

```
Source code + docs + tests
        │
        ▼
┌──────────────────┐
│  Layer 1: Extract │  ← Parser per language (AST-based)
│  Mechanical       │  ← Deterministic, fast
│  skeleton.aid     │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Layer 2: Infer   │  ← AI agent reads skeleton + source + docs
│  AI-enhanced      │  ← Fills @purpose, @errors, @pre/@post,
│  enhanced.aid     │  ← @effects, @workflows, @antipatterns
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Layer 3: Review  │  ← Human reviews and corrects
│  Human-approved   │  ← Adds domain knowledge
│  final.aid        │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Validate         │  ← Check AID file against spec
│                   │  ← Required fields present?
│                   │  ← Cross-references resolve?
│                   │  ← Constraints parseable?
└──────────────────┘
```

### 3.1 Layer 1 implementation strategy

Layer 1 is a set of per-language extractors that operate on the AST (abstract syntax tree) of the source code.

**Input:** Source files, type stubs (`.pyi`, `.d.ts`), package metadata
**Output:** Partial `.aid` file with all mechanically extractable fields

Each extractor:
1. Parses the source into an AST
2. Walks exported/public declarations
3. Maps each declaration to the appropriate AID entry type
4. Extracts type information, defaults, and structural relationships
5. Outputs the `.aid` skeleton

Layer 1 extractors should be conservative — only emit fields they are certain about. An empty `@errors` is better than a wrong one. Missing fields are expected; the file is a skeleton.

### 3.2 Layer 2 implementation strategy

Layer 2 is an AI agent that receives the skeleton and all available context.

**Input:** Skeleton `.aid` file + source code + docstrings + README + tests + existing docs
**Output:** Complete `.aid` file with all fields populated

The agent's prompt should:
1. Present the AID spec (so it knows the format and field semantics)
2. Present the skeleton (so it knows what's already extracted)
3. Present the source code for the module
4. Present any existing documentation (docstrings, README sections, wiki)
5. Present test files (tests reveal usage patterns, error cases, and edge cases)
6. Ask the agent to fill in every missing field, generate workflows, and flag uncertainties

**Key instructions for the inference agent:**
- When uncertain about an error condition, include it with a `// inferred` comment
- When a constraint is likely but not confirmed, use the constraint with a `// unverified` comment
- Generate at least one workflow per module (the basic usage lifecycle)
- Flag any function that takes `self` where mutability could not be determined
- For `@thread_safety`, default to "Unknown" rather than guessing

### 3.3 Validation

After generation, a validator checks:

1. **Structural validity** — does the file parse according to the AID grammar?
2. **Required fields** — are `@purpose` and `@sig`/`@fields`/`@variants` present?
3. **Cross-reference integrity** — do all `@related` names resolve to entries in this file or known modules?
4. **Constraint syntax** — are all constraints using recognized keywords (`Range`, `Must be`, etc.)?
5. **Method binding** — does every `@fn Type.method` have a corresponding `@type Type` with the method in `@methods`?
6. **Signature consistency** — do `@params` entries match the parameters in `@sig`?
7. **Error consistency** — do `@errors` entries match the error types in `@sig`?

Validation produces warnings, not errors. A file with warnings is still usable — it just signals where human review is most needed.

---

## 4. Incremental generation

AID files should be regenerable as libraries evolve. The generation pipeline must support incremental updates:

1. **Detect changes** — compare the library's current version against the version in `@version`
2. **Re-extract** — run Layer 1 on the updated source
3. **Diff** — identify new, changed, and removed API surface
4. **Merge** — for new entries, run Layer 2 inference. For changed entries, update the mechanical fields and flag semantic fields for re-review. For removed entries, delete them.
5. **Preserve human edits** — fields that were human-reviewed should not be overwritten by Layer 2 unless the underlying API changed

This requires tracking provenance: which fields came from Layer 1, which from Layer 2, which from Layer 3. A simple approach is a comment convention:

```
// [generated] — field was produced by Layer 1 or 2, not yet reviewed
// [reviewed] — field was human-reviewed and approved
```

These comments are optional and ignored by AID consumers. They exist only for the generation pipeline.

---

## 5. Quality metrics

How do we know if an AID file is good?

| Metric | Measurement | Target |
|--------|------------|--------|
| **Completeness** | % of applicable fields populated | > 90% after Layer 2 |
| **Accuracy** | Manual review sample of constraints and errors | > 95% after Layer 2 |
| **Token efficiency** | Tokens in AID file vs. equivalent human-readable docs | < 30% of human doc token count |
| **First-try correctness** | Agent generates correct code from AID without re-reads | > 85% |
| **Error coverage** | % of actual runtime errors documented in `@errors` | > 90% after Layer 2 |

The ultimate metric is **first-try correctness**: when an agent reads the AID file and generates code, how often does it work without debugging? Everything else is a proxy for this.
