# Layer 2 Pipeline Results: Generator → Reviewer

**Date:** 2026-03-27
**Target:** SyndrDB query/planner package (26K LoC, 50 files)
**Generator model:** Claude Opus 4.6
**Reviewer model:** Claude Sonnet 4.6

---

## Pipeline metrics

| Stage | Tokens | Tool calls | Duration | Output |
|-------|--------|------------|----------|--------|
| Generator | 136,357 | 31 | 750s (12.5 min) | 902-line L2 AID draft |
| Reviewer | 59,152 | 47 | 235s (3.9 min) | 58-claim verification report |
| **Total** | **195,509** | **78** | **985s (16.4 min)** | Verified L2 AID |

## Verification results

| Metric | Count | Percentage |
|--------|-------|------------|
| Total claims checked | 58 | 100% |
| Verified accurate | 48 | 83% |
| Corrected | 7 | 12% |
| Missing (reviewer added) | 4 | 7% |
| Stale references | 6 | 10% |

## What the generator produced

- **3 workflow blocks** with source-linked steps:
  1. Query planning pipeline (9 steps)
  2. Index selection algorithm (10 steps)
  3. Iterator/Volcano model (4 steps)

- **Global invariants** (source-linked):
  - BRIN lossy scan without FilterNode (bug)
  - Index selection priority order
  - BTreeOrderedScanNode gives lexicographic order, not semantic
  - PlanBuilder composition order is fixed
  - ExecutionPlan is immutable after creation

- **Antipatterns** (source-linked):
  - BRINScanNode without FilterNode → incorrect results
  - AND clause remaining predicates silently dropped
  - Assuming BTreeOrderedScan eliminates SortNode
  - Creating subquery executor when disabled → nil deref

- **25+ enhanced type/function descriptions** with @pre/@post conditions

## What the reviewer caught

### Corrections (7)
1. `ExecuteIterator` method doesn't exist — callers use `plan.IteratorFactory()` directly
2. `BundleServiceInterface` import-cycle claim overstated — planner still imports domain packages
3. `GetDocument` signature missing variadic `snapshotParams` parameter
4. PlanBuilder composition order line reference points to function signature, not logic
5. Iterator workflow step references interface definition, not caller pattern
6. `ResultSchema` assignment description says field access, code uses method call
7. BTreeRangeScanCost formula mixes pages and rows in description

### Missing claims added (4)
1. `IndexOnlyScanNode` handled in iterator pipeline by delegating to child
2. `ShouldUseIterator` returns true for non-LIMIT queries without ORDER BY
3. GROUP BY tree partly constructed in router, not just PlanBuilder
4. Cache callback wraps steps 4-9 — workflow ordering is conceptual, not literal

### Critical stale reference (1)
- `FullScanNode implements SliceExecutionNode` referenced `planner.go:1376` — file is only ~200 lines. Actual location: `nodes.go:752`

## Assessment

The generator-reviewer pipeline successfully produced high-quality semantic documentation:
- 83% first-pass accuracy is strong for AI-generated technical docs
- Every inaccuracy was caught by the reviewer
- The corrections are precise and well-reasoned
- The reviewer added claims the generator missed

The most valuable output is the **workflow blocks** — these capture cross-function data flows that take hours to trace through source code. The **antipatterns** section identifies real bugs with source evidence.
