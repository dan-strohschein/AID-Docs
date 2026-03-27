# AID Layer 2: AI-Generated Semantic Documentation

**How semantic AID fields are generated, verified, and maintained by AI agents.**

---

## 1. The problem Layer 2 solves

Layer 1 (mechanical extraction) captures the API surface — types, signatures, fields. But the most valuable AID information is **semantic**: workflows, invariants, constraints, antipatterns. This information lives in the code but isn't expressed in the structure — it's in the logic, the ordering, the error paths, the comments.

Benchmarks show that Layer 1 alone:
- Reduces token consumption by 47% (BM1)
- Improves architectural awareness (BM2)
- Does NOT improve correctness on hard tasks — both agents found the same bugs

Layer 2 fills the gap by producing **source-linked semantic annotations** that are machine-verifiable.

---

## 2. Source-linked claims

Every semantic assertion in Layer 2 must be linked to the code that creates or enforces it. A new syntax element: **source references** in square brackets.

```
@invariants
  - BRIN is a lossy index. Results must always be filtered post-scan. [src: planner/nodes.go:245-280]
  - tryIndexOptimization returns raw index nodes, not filtered nodes. [src: planner/query_router.go:554-560]

@antipatterns
  - Don't return BRINScanNode without a FilterNode wrapper. It produces false positives. [src: planner/nodes.go:250]
```

The `[src: file:line]` reference is a **verifiable claim**. A reviewer agent can read the referenced code and confirm or deny the assertion. If the code changes, the reference becomes stale and triggers re-review.

### Source reference syntax

```
[src: relative/path/to/file.go:LINE]
[src: relative/path/to/file.go:START-END]
[src: relative/path/to/file.go:LINE, other/file.go:LINE]
```

Paths are relative to the project root. Line numbers reference the code version in `@code_version`.

---

## 3. Document versioning and status

### New header fields

```
@module query/planner
@lang go
@version 2.1.0
@code_version git:d470c0f              # git commit hash this AID describes
@aid_status reviewed                    # draft | reviewed | approved | stale
@aid_generated_by layer2-generator     # agent role that produced this
@aid_reviewed_by layer2-reviewer       # agent role that verified this
@aid_version 0.1
```

### Status lifecycle

```
draft → reviewed → approved → stale
  ↑        ↑                    │
  │        └── reviewer fixes ──┘
  └── generator creates
```

- **draft**: Generator agent produced this. Source refs present but unverified.
- **reviewed**: Reviewer agent verified all source refs against the code. Inaccuracies corrected.
- **approved**: Human reviewed (optional — only for critical systems).
- **stale**: Code has changed since `@code_version`. Needs re-generation or re-review.

### Staleness detection

When the code changes (new git commit), any AID file whose `@code_version` doesn't match HEAD is **stale**. The pipeline can:
1. Diff the changed files against the AID's source references
2. If referenced lines changed → flag that specific claim for re-review
3. If no referenced lines changed → AID is still valid, update `@code_version`

---

## 4. Multi-agent pipeline

### Roles

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  Layer 1          │     │  Layer 2          │     │  Layer 2          │
│  Extractor        │────▶│  Generator        │────▶│  Reviewer         │
│  (mechanical)     │     │  (AI inference)   │     │  (AI verification)│
└──────────────────┘     └──────────────────┘     └──────────────────┘
        │                         │                         │
   Deterministic            Reads source +            Reads AID claims +
   AST parsing              L1 AID, infers            referenced source,
                            semantics                  verifies accuracy
```

### Role 1: Generator Agent

**Input:** Layer 1 AID file + full source code for the package + source of dependent packages

**Prompt directives:**
- Fill in all empty semantic fields: `@purpose` (enhanced), `@pre`, `@post`, `@invariants`, `@effects`, `@thread_safety`, `@complexity`, `@antipatterns`
- Generate `@workflow` blocks for multi-step operations
- Every claim MUST include a `[src: file:line]` reference to the code that supports it
- When uncertain, prefix with `// uncertain:` and include the reference anyway
- Do NOT invent constraints that aren't in the code — only document what exists
- Set `@aid_status draft`

**Output:** Complete Layer 2 AID file with all semantic fields populated and source-linked.

### Role 2: Reviewer Agent

**Input:** Layer 2 AID file (draft) + only the source files referenced in `[src:]` links

**Prompt directives:**
- For each `[src: file:line]` reference, read the referenced code
- Verify the claim matches what the code actually does
- If accurate: leave unchanged
- If inaccurate: correct the claim and update the source reference
- If the reference is wrong (code not at that line): find the correct location and update
- If a claim has no supporting code: remove it and add `// removed: no supporting code`
- Look for constraints the generator MISSED — add them with source references
- Set `@aid_status reviewed`
- Report: list of changes made, with rationale

**Key constraint:** The reviewer ONLY reads files referenced in `[src:]` links (plus a small search radius). It does NOT re-read the entire codebase. This keeps it fast and focused.

### Pipeline execution

```python
# Pseudocode for the Layer 2 pipeline
for package in project.packages:
    # Step 1: Layer 1 (already done)
    l1_aid = aid_gen(package)

    # Step 2: Generator agent
    l2_draft = spawn_agent(
        role="layer2-generator",
        input=[l1_aid, package.source_files, package.dependency_sources],
        output_format="aid",
    )
    l2_draft.aid_status = "draft"

    # Step 3: Reviewer agent (separate context, fresh perspective)
    l2_reviewed = spawn_agent(
        role="layer2-reviewer",
        input=[l2_draft, referenced_source_files(l2_draft)],
        output_format="aid",
    )
    l2_reviewed.aid_status = "reviewed"

    # Step 4: Write final AID file
    write_aid(package, l2_reviewed)
```

### Why two agents, not one?

1. **Separation of concerns.** The generator is creative (inferring semantics from code). The reviewer is critical (verifying claims against evidence). These are different cognitive modes.

2. **Fresh context.** The reviewer doesn't have the generator's reasoning in its context window. It evaluates claims independently, catching hallucinations the generator might justify to itself.

3. **Source-linking enables verification.** Because every claim has a `[src:]` reference, the reviewer doesn't need to understand the entire codebase — it only needs to read the specific referenced code and confirm the claim.

4. **Scalability.** Generators and reviewers can run in parallel across packages. A review failure doesn't require re-generating the whole file — just the failed claims.

---

## 5. What Layer 2 adds to each AID field

### Fields that Layer 1 produces (mechanical)
- `@fn`, `@type`, `@trait`, `@const` — names and kinds
- `@sig` — type signatures
- `@params` — parameter names and types
- `@fields` — struct field names and types
- `@variants` — enum variants
- `@methods` — method lists
- `@extends`, `@implements` — inheritance and protocol detection

### Fields that Layer 2 adds (semantic)

| Field | What Layer 2 adds | Source-linked? |
|-------|-------------------|----------------|
| `@purpose` | Enhanced descriptions beyond docstrings — explains WHY, not just WHAT | Yes — links to key implementation logic |
| `@pre` | Preconditions discovered from assertions, nil checks, state guards | Yes — links to the guard/check code |
| `@post` | Postconditions discovered from state mutations, return guarantees | Yes — links to the mutation/guarantee code |
| `@invariants` | Properties that always hold, discovered from constructors and validations | Yes — links to enforcement code |
| `@errors` | Exhaustive error conditions traced through call paths | Yes — links to each error return/raise |
| `@effects` | Side effects classified from I/O calls, state mutations | Yes — links to the effectful code |
| `@thread_safety` | Concurrency analysis from mutex usage, channel patterns, atomic ops | Yes — links to synchronization code |
| `@complexity` | Time/space complexity inferred from algorithm structure | Yes — links to the loop/recursion |
| `@antipatterns` | Common mistakes discovered from error handling, defensive code, comments | Yes — links to the protective code |
| `@workflow` | Multi-step data flows across functions/packages | Yes — each step links to its implementation |

### Workflow source linking example

```
@workflow query_execution_pipeline
@purpose Execute a SyndrQL query from parse to results
@code_version git:d470c0f
@steps
  1. Parse: queryparser.Parse(sql) → SelectStatement [src: syndrQL/select_parser.go:45]
  2. Plan: planner.CreateExecutionPlan(stmt, bundle) → ExecutionNode tree [src: planner/complete_planner.go:89]
  3. Index selection: tryIndexOptimization(bundle, whereExpr) → IndexScanNode or nil [src: planner/query_router.go:550]
  4. Execute: rootNode.Execute() → map[docID]*Document [src: planner/nodes.go:35]
  5. Project: applyProjection(docs, selectFields) → results [src: planner/complete_planner.go:210]
@errors_at
  step 1: ParseError — invalid SyndrQL syntax [src: syndrQL/select_parser.go:52]
  step 3: Falls back to FullScanNode if no suitable index found [src: planner/query_router.go:890]
@antipatterns
  - Don't return BRINScanNode without FilterNode wrapping. BRIN is lossy. [src: planner/nodes.go:245]
  - Don't assume tryIndexOptimization filters results. It returns raw index nodes. [src: planner/query_router.go:554]
```

---

## 6. Incremental updates

When code changes, the pipeline doesn't regenerate everything:

### Change detection

```
1. git diff HEAD~1..HEAD — find changed files
2. For each AID file, check if any [src:] references point to changed files
3. If no references point to changed files → update @code_version only
4. If references point to changed files:
   a. Extract the specific claims with stale references
   b. Re-run generator on ONLY those claims (not the whole file)
   c. Re-run reviewer on ONLY the updated claims
   d. Merge back into the AID file
   e. Update @code_version
```

### Merge strategy

Layer 2 fields that were human-reviewed (`@aid_status approved`) are not overwritten by re-generation. The pipeline:
- Flags them as `stale` if their source references changed
- Presents the diff to a human for review
- Only overwrites on explicit human approval

---

## 7. Selective loading

BM2 showed that loading all AID files hurts token efficiency. Layer 2 should include **package dependency hints** that enable selective loading:

```
@module query/planner
@depends [syndrQL, domain/index, domain/models, storage/buffer]
```

An agent working on the planner loads:
1. `planner.aid` (the target package)
2. AID files for packages listed in `@depends`
3. Nothing else

This reduces the 69-file AID set to ~5-6 files for a typical task.

---

## 8. Implementation plan

### Phase 1: Spec additions
- Add `[src:]` reference syntax to the AID spec
- Add `@code_version`, `@aid_status`, `@aid_generated_by`, `@aid_reviewed_by` fields
- Add `@depends` field for selective loading
- Document in `spec/format.md` and `spec/fields.md`

### Phase 2: Generator agent prompt
- Build the generator prompt template
- Test on SyndrDB's planner package
- Evaluate output quality

### Phase 3: Reviewer agent prompt
- Build the reviewer prompt template
- Test on generator output
- Measure verification accuracy

### Phase 4: Pipeline tooling
- Build `aid-gen-l2` command that orchestrates generator → reviewer
- Integrate with git for `@code_version` and staleness detection
- Support incremental updates

### Phase 5: Re-run BM2
- Generate Layer 2 AID for SyndrDB's planner + related packages
- Re-run the BRIN integration benchmark
- Compare: Layer 1 only vs Layer 1+2 vs no AID
