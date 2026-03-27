# Benchmark 4: Flusher Bug Fix — Different Subsystem Validation

**Date:** 2026-03-27
**Task:** Fix AdaptiveFlusher pending request accumulation bug
**Codebase:** SyndrDB storage/flusher package (separate subsystem from BM2/BM3)
**Model:** Claude Sonnet 4.6 (both agents)

---

## Efficiency

| Metric | No Docs (Agent A) | L1+L2 AID (Agent B) | Difference |
|--------|-------------------|---------------------|------------|
| Total tokens | 26,038 | 43,653 | **+68%** |
| Tool calls | 6 | 8 | **+33%** |
| Duration | 62s | 76s | **+23%** |

**Agent B used more tokens** because it read the manifest + L2 flusher AID + L1 bundlestore AID before touching source code. For this single-file bug fix, the upfront context loading added overhead without the payoff seen in cross-package tasks.

---

## Solution Comparison

Both agents proposed the same core fix: add an `inQueue bool` field to `FlushRequest` to prevent duplicate sends and enable `flushStaleRequests` to skip already-queued entries.

### Architecture

| Aspect | No Docs (A) | With AID (B) |
|--------|-------------|-------------|
| Core mechanism | `queued` bool field on FlushRequest | `inQueue` bool field on FlushRequest |
| RequestFlush guard | Check `req.queued` under lock, skip if true | Check `req.inQueue` under lock, skip if true |
| Worker reset | Worker clears `queued = false` BEFORE calling flushFunc | No explicit reset — relies on `clearPendingFlush` deleting the entry |
| flushStaleRequests | Skips `queued` entries | Skips `inQueue` entries |
| Lock during send | Releases lock, then sends (lock-free send) | Holds lock during non-blocking select (safe — never blocks) |

### Correctness analysis

| Scenario | No Docs | With AID |
|----------|---------|----------|
| Queue full, stale retry | Retries non-queued entries only | Retries non-queued entries only |
| Concurrent writes during flush | Worker clears flag before flushFunc, allowing immediate re-queue | Entry deleted by clearPendingFlush, new write creates fresh entry |
| Duplicate sends | Prevented by `queued` check | Prevented by `inQueue` check |
| Deadlock risk | Lock released before channel send — safe | Lock held during non-blocking select — also safe (never blocks) |
| Shutdown behavior | Not discussed | Explicitly notes AID antipattern about shutdown bypass |

### Quality differences

| Dimension | No Docs | With AID |
|-----------|---------|----------|
| **Worker reset timing** | Resets `queued = false` before `flushFunc` — allows writes during flush to immediately re-queue | No explicit reset — simpler but relies on delete-then-recreate pattern |
| **Return value** | `RequestFlush` returns `bool` (breaking API change) | `RequestFlush` keeps `void` return (non-breaking) |
| **Lock strategy** | Lock → unlock → send (two lock acquisitions if send succeeds) | Lock → send-under-lock → unlock (one acquisition, atomic check-and-mark) |
| **Existing entry check** | Returns early if no entry exists in `pendingFlushes` — but this is wrong; the current code creates entries only in RecordDocuments, so RequestFlush could be called for bundles not in the map | Returns early if no entry exists — correct, with explicit comment explaining why |
| **Awareness of related bugs** | Not mentioned | References AID antipattern: "shutdown flush bypasses queue" — acknowledges it's a separate issue |
| **Growth analysis** | Table showing before/after for 3 scenarios | Identical table with same 3 scenarios |

---

## Key Findings

### Both solutions are correct and equivalent

Unlike BM3, where the no-docs agent built a parallel filtering system while the AID agent reused existing infrastructure, here both agents converged on the same design. This makes sense: it's a single-file bug fix with one clear solution pattern (add a state flag). There's no "existing infrastructure to discover."

### AID's value shows in the CONTEXT, not the code

The AID agent's response demonstrated awareness of:
1. **The shutdown antipattern** — the L2 AID documented that `flushAllPending` bypasses the queue and can cause concurrent flushFunc calls for the same bundle. Agent B explicitly noted this: "The AID documents this as an existing issue (`@antipattern shutdown-flush-bypasses-queue`), which is a separate problem from the one being fixed here."
2. **The lock ordering invariant** — Agent B's comment "Lock ordering is unchanged: there is no code that holds both `flushMu` and any other lock" directly references the L2 AID's lock ordering invariants.
3. **The non-breaking API** — Agent B kept `RequestFlush` as void, while Agent A changed it to return `bool`. The L2 AID's workflow showed that `RecordDocuments` and `RecordBytes` call `RequestFlush` without checking a return value.

### Token overhead for single-file tasks

The L2 flusher AID was 27K chars — almost as many tokens as Agent A's entire run. For a well-scoped, single-file bug fix, this is overhead without proportional benefit. The L2 AID's value scales with task complexity:

| Task type | L2 AID value |
|-----------|-------------|
| Cross-package feature (BM3) | **High** — reveals existing infrastructure, prevents parallel builds |
| Complex state machine bug (BM4) | **Medium** — provides context about related bugs and invariants |
| Simple single-function fix | **Low** — overhead exceeds benefit |

---

## Scoring

| Dimension | No Docs | With AID | Winner |
|-----------|---------|----------|--------|
| Bug detection | Found it | Found it | Tie |
| Fix correctness | Correct | Correct | Tie |
| Fix design | `queued` flag | `inQueue` flag | Tie |
| API compatibility | Changed return type (breaking) | Kept void (non-breaking) | **AID** |
| Lock strategy | Two acquisitions | Atomic check-and-mark | **AID** |
| Awareness of related bugs | Not mentioned | Referenced shutdown antipattern | **AID** |
| Token efficiency | 26K | 43K | **No docs** |
| Speed | 62s | 76s | **No docs** |

---

## Cumulative Benchmark Results

| Benchmark | Task type | AID quality advantage | AID efficiency advantage |
|-----------|-----------|----------------------|-------------------------|
| BM1 (Aria) | Pattern-following | None | **+47% fewer tokens, 6x faster** |
| BM2 (SyndrDB L1) | Cross-package | Moderate (better package boundaries) | Worse (-24% more tokens) |
| BM3 (SyndrDB L1+L2) | Cross-package | **Strong (5x less code, 6 components reused)** | **+10% fewer tokens** |
| BM4 (SyndrDB flusher) | Single-file bug fix | Moderate (better API compat, context awareness) | Worse (-68% more tokens) |

## Conclusion

**AID's value is proportional to task complexity and cross-package scope.**

- For cross-package tasks requiring architectural understanding (BM3): AID produces fundamentally better solutions with fewer tokens.
- For single-file fixes where the solution is obvious (BM4): AID provides useful context (related bugs, invariants) but the token overhead isn't justified.
- The ideal workflow: use the **manifest** to determine whether AID loading is worthwhile for the task at hand. Single-file fixes → skip AID. Cross-package features → load relevant AID via `@depends` chain.
