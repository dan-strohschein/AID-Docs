# AID Spec Gaps — Discovered by Cartograph Design

**Date:** 2026-03-27
**Updated:** 2026-04-01 (v0.2 spec addresses all three gaps)
**Context:** While designing the Cartograph semantic index (a tool that builds a queryable graph from AID files), we discovered three categories of relationships that AID cannot currently express. These gaps prevent full graph construction from AID alone.

---

## Gap 1: ~~No `@calls` field~~ — RESOLVED in v0.2

**Status:** Resolved. `@calls` added to spec as an optional field on `@fn` entries (format.md Section 4.1). Already implemented in `aid-gen-go` via AST analysis. Formalized in v0.2 spec.

### The original problem

AID documents what a function accepts (`@params`), returns (`@returns`), and can error with (`@errors`). It does NOT document which other functions a function calls internally.

```
@fn ProcessOrder
@sig (order: Order) -> Receipt ! PaymentError
@params
  order: The order to process
```

From this AID entry, we know `ProcessOrder` takes an `Order` and can produce a `PaymentError`. We do NOT know that it calls `ValidateOrder`, `ChargePayment`, `SendReceipt`, and `UpdateInventory`.

### Why it matters

The `CallStack` and `SideEffects` queries in Cartograph depend on knowing the call graph. Without `@calls`, the only call relationships that exist in AID come from:
- `@workflow` steps (partial — only documented happy paths)
- `@related` references (loose — doesn't distinguish "calls" from "is similar to")

### Proposed solution

Add `@calls` as an optional field on `@fn` entries:

```
@fn ProcessOrder
@sig (order: Order) -> Receipt ! PaymentError
@calls ValidateOrder, ChargePayment, SendReceipt, UpdateInventory
```

**Layer 1:** Extractors could populate `@calls` by analyzing the function body's AST for function call expressions. Go's `go/ast` and Python's `ast` module both support this. This is mechanical extraction — no AI needed.

**Layer 2:** AI generators could add `@calls` for functions where the L1 extractor missed calls through interfaces or dynamic dispatch.

**Tradeoff:** Adding `@calls` to L1 extraction increases AID file size. For a function that calls 10 other functions, that's one line. For a codebase with 1000 functions, it's ~1000 extra lines across all AID files. The token cost is modest and the value for graph construction is high.

---

## Gap 2: ~~No field access tracking~~ — RESOLVED in v0.2

**Status:** Resolved. `@reads` and `@writes` added to spec as optional fields on `@fn` entries (format.md Section 4.1). Recommended for functions touching fields of types with `@invariants`. Partially extractable by Layer 1.

### The original problem

AID documents what fields a type has (`@fields`) and what types a function accepts (`@params`). It does NOT document which specific fields a function reads or writes.

```
@type User
@fields
  email: str
  name: str
  role: str

@fn UpdateEmail
@sig (user: User, newEmail: str) -> None
```

From this, we know `UpdateEmail` takes a `User`. We do NOT know it reads `user.role` (for permission check) and writes `user.email`. The query "what touches User.email?" can't be answered from AID alone.

### Why it matters

The `FieldTouchers` query is one of the most valuable for understanding data flow. "What code can modify user.email?" is a security-critical question that currently requires grepping source code.

### Proposed solution

Add `@reads` and `@writes` as optional fields on `@fn` entries:

```
@fn UpdateEmail
@sig (user: User, newEmail: str) -> None
@reads User.role
@writes User.email
```

**Layer 1:** Extractors could partially populate these by analyzing assignment targets and field access expressions in the AST. Go's `go/ast` can identify `user.Email = x` (write) and `if user.Role == "admin"` (read). This is mechanical but imperfect — field access through interfaces or reflection would be missed.

**Layer 2:** AI generators could fill gaps by analyzing function logic semantically.

**Tradeoff:** Field-level tracking is verbose. A function that touches 5 fields gets 5 extra entries. This may not be worth it for every function — perhaps only for functions that touch fields of types with `@invariants`.

---

## Gap 3: ~~No error propagation tracking~~ — RESOLVED in v0.2

**Status:** Resolved via inline error annotations instead of a separate `@propagates` field. `@errors` entries now support `[origin]`, `[from: FnName]`, and `[caught: description]` annotations (format.md Section 4.1). More token-efficient than a separate field.

### The original problem

AID documents what errors a function can produce (`@errors`). It does NOT document whether a caller catches, handles, or propagates those errors.

```
@fn GetUser
@errors
  NotFoundError — user doesn't exist
  DbError — database connection failed

@fn HandleRequest
@errors
  NotFoundError — from GetUser (propagated)
  # DbError is caught and logged internally — NOT propagated
```

From AID alone, if `HandleRequest` calls `GetUser`, we know both can produce `NotFoundError`. But we don't know whether `HandleRequest` propagates `DbError` to its callers or catches it internally. The error trace query can't distinguish these cases.

### Why it matters

The `ErrorProducers` query needs to trace error propagation chains: "ConnectionError originates in `HttpClient.Send`, propagates through `ApiClient.Get`, and surfaces in `Handler.Process`." Without knowing which errors are propagated vs caught, the trace includes false positives (errors that are handled internally but show up as propagated).

### Proposed solution

Add `@propagates` as an optional field on `@fn` entries:

```
@fn HandleRequest
@errors
  NotFoundError — user not found
@propagates NotFoundError from GetUser
@catches DbError from GetUser
```

Or more concisely, annotate `@errors` entries with their source:

```
@errors
  NotFoundError — user not found [from: GetUser]
```

**Layer 1:** Extractors could detect propagation in Go by analyzing whether an `error` return from a called function is returned directly or wrapped. In Python, `except` blocks that re-raise (`raise`) propagate; those that don't, catch.

**Layer 2:** AI generators could identify propagation patterns from the function's error handling logic.

**Tradeoff:** Error propagation is the most complex of the three gaps to extract mechanically. L1 extraction would catch the obvious cases (direct return of error, re-raise). L2 would handle the nuanced cases (error wrapping, conditional propagation).

---

## Impact on Cartograph

| Gap | Cartograph query affected | Workaround without spec change |
|-----|--------------------------|-------------------------------|
| No `@calls` | CallStack, SideEffects | Use `@workflow` steps + `@related` for partial call graph |
| No `@reads`/`@writes` | FieldTouchers | Match `@params` type against `@fields` type (coarse — shows which functions COULD touch the field, not which DO) |
| No `@propagates` | ErrorProducers | Assume all callers propagate all errors from callees (overly conservative — shows false positives) |

## Resolution

All three gaps addressed in AID spec v0.2:
1. **`@calls`** — added as L1-extractable field on `@fn` entries
2. **Error provenance** — solved via `@errors` inline annotations (`[origin]`, `[from:]`, `[caught:]`) instead of a separate `@propagates` field
3. **`@reads`/`@writes`** — added as optional fields, recommended for invariant-bearing types

Cartograph can now build complete call graphs, field access graphs, and error propagation chains from AID alone.
