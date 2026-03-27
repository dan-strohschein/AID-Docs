# Using AID on a Greenfield Project

**How to use AID from day one on a new codebase.**

---

## The question

On an existing codebase, AID extracts knowledge that already exists in the code. But on a greenfield project, there IS no code yet. So when does AID enter the picture, and what does the development cycle look like?

The answer: **AID starts BEFORE the code does.** Write skeleton AID as the design artifact, then the AI builds to match the contracts. As the codebase grows, L1 extraction and L2 generation take over — the skeleton evolves into living documentation.

---

## Best practice: Skeleton AID as the plan

**Write skeleton AID files before writing code. Use them as the design spec the AI implements against.**

This is the single most impactful practice for greenfield projects. Here's why:

When an AI agent is planning or doing a first implementation, it makes hundreds of micro-decisions: What should this function return? What error types should it use? What's the relationship between these types? Should this method mutate or return a copy?

Without a skeleton AID, those decisions live in prose (a README or plan doc) that the AI has to interpret. Prose is ambiguous. "The client should handle errors gracefully" means nothing concrete. The AI fills in the gaps with assumptions, and some will be wrong. Every wrong assumption costs a cycle: write code → test fails → understand what was assumed wrong → rewrite.

With a skeleton AID, those decisions are already crystallized in a format the AI consumes directly:

```
@module myapp/client
@lang go
@version 0.1.0
@purpose HTTP client with retry support and connection pooling

---

@fn Get
@sig (key: str, timeout?: Duration) -> Value ! NotFoundError | TimeoutError
@post Connection remains open after error. Caller does not need to reconnect.
@related Post, Delete, Connection

---

@type Connection
@kind struct
@purpose Pooled connection with automatic reconnection
@implements [Closeable]
@fields
  host: str
  port: int
  maxRetries: int — Default 3. Range [0, 10].
  pool: ConnectionPool — Managed internally. Caller must call Close() when done.
@methods Get, Post, Delete, Close

---

@workflow request_lifecycle
@purpose Send a request, handle errors, maintain pool health
@steps
  1. Acquire: pool.Get() — get connection from pool
  2. Send: conn.Send(request) — send over the wire
  3. Handle: check response status, retry on 5xx up to maxRetries
  4. Return: release connection back to pool (automatic on success)
  5. Error: on persistent failure, mark connection dead, pool creates new one
@antipatterns
  - Don't create new connections per request. Use the pool.
  - Don't retry on 4xx errors. Only 5xx and timeouts are retryable.
```

That skeleton is three decisions per function, crystallized in a few lines: the parameter types, the exact error variants, and the post-condition contracts. The workflow tells the AI how the pieces fit together BEFORE any of them exist. The antipatterns prevent the most common architectural mistakes.

**The AI builds to match these contracts.** It doesn't guess. It doesn't assume. It implements what the skeleton specifies. When the implementation is done, L1 extraction replaces the skeleton with actual extracted data, and the skeleton's semantic content (workflows, antipatterns, invariants) becomes the seed for L2.

### The skeleton lifecycle

```
Day 1:  Human + AI write skeleton AID (sigs, types, workflows) → THE PLAN
Day 2:  AI implements code to match the AID contracts
Day 3:  L1 extractor runs → replaces skeleton sigs with actual extracted data
Week 2: L2 generator runs → adds source-linked invariants from real code
        Skeleton's manually-written workflows/antipatterns are preserved
        and enhanced with [src:] references
```

The skeleton AID IS the plan — in the most efficient format for an AI agent to consume.

---

## Day 1: Project setup

Before writing any code, create the AID infrastructure and your first skeleton:

```bash
mkdir -p .aidocs
```

Write your skeleton AID files — one per planned package. Focus on:
- `@sig` — function signatures (types, params, return values, error types)
- `@fields` — struct/class field definitions with constraints
- `@workflow` — how the major data flows should work
- `@antipatterns` — mistakes you already know to avoid from experience

Add to your `CLAUDE.md` (or `.cursorrules`, etc.):

```markdown
## AID Documentation

This project uses AID files in .aidocs/ for AI-optimized documentation.

- .aidocs/ contains skeleton AID files that define the planned API contracts
- IMPLEMENT CODE TO MATCH THE AID CONTRACTS — they are the spec
- After implementing a package, re-run the L1 extractor to update the AID
- After significant features, request L2 generation for the package
- AID files are committed alongside code — treat them as project artifacts
```

---

## The greenfield development cycle

Here's how a typical feature cycle works with AID, from empty repo to mature codebase:

### Sprint 1: Skeleton AID drives the first implementation

```
1. Design  → Human + AI write skeleton AID files (contracts, workflows)
2. Plan    → AI reads skeleton AID — knows the target architecture
3. Code    → AI implements to match the contracts
4. Test    → AI writes tests informed by @errors, @pre/@post from skeleton
5. Extract → L1 extractor runs, replaces skeleton with actual data
6. Commit  → Code + .aidocs/ committed together
```

The skeleton AID eliminates the most expensive part of sprint 1: the AI guessing at design decisions. Every signature, every error type, every relationship is specified upfront. The AI writes code to match, not code to guess.

**When to generate L1:** After the first implementation is done. The L1 extractor replaces the hand-written skeleton signatures with actual extracted data. Any skeleton workflows and antipatterns that aren't captured by L1 are preserved — they become the seed for L2 later.

### Sprint 2-3: Codebase grows — generate L1

The project now has real structure: multiple packages, cross-package dependencies, non-obvious patterns emerging. This is when the AI starts wasting tokens exploring.

```
1. Plan    → AI reads source (getting slower as codebase grows)
2. Code    → AI writes new package
3. Test    → Tests pass
4. GENERATE L1 AID for new and changed packages
   └─ aid-gen-go -output .aidocs ./src/...
5. GENERATE MANIFEST
   └─ aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
6. Commit  → Code + .aidocs/ committed together
```

The new step (4-5) takes seconds. Now the next task benefits — the AI reads the manifest and L1 AID instead of opening every file.

### Sprint 4+: Architecture stabilizes — generate L2

The project now has established patterns: how data flows, what invariants matter, where the sharp edges are. This is when L2 becomes valuable.

```
1. Plan    → AI reads manifest + relevant AID files
   └─ Cross-package task? Load full L2 + @depends chain
   └─ Single-file fix? Load --summary only
2. Code    → AI reuses existing components (knows they exist from AID)
3. Test    → Tests pass
4. UPDATE L1 for changed packages
   └─ aid-gen-go -output .aidocs ./changed/package/...
5. CHECK L2 STALENESS
   └─ aid-gen-l2 stale --aid .aidocs/pkg-l2.aid --project-root .
6. IF STALE: regenerate only stale claims
   └─ aid-gen-l2 update --aid .aidocs/pkg-l2.aid --project-root .
   └─ Send prompt to AI, apply corrections
7. VALIDATE
   └─ aid-validate .aidocs/*.aid
8. Commit  → Code + updated .aidocs/ committed together
```

### Mature project: the full cycle

Once L2 exists for core packages, this is the steady-state cycle:

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  1. PLAN                                                │
│     AI reads manifest → loads relevant AID              │
│     Knows: what exists, what the invariants are,        │
│     what NOT to do, WHY things are structured this way  │
│                                                         │
│  2. REFINE                                              │
│     AI references @decision blocks:                     │
│     "This was designed as X because Y — should we       │
│     change that decision or work within it?"            │
│                                                         │
│  3. CODE                                                │
│     AI writes implementation reusing existing           │
│     components from @methods, @related, @workflow       │
│     Respects @invariants and @antipatterns              │
│                                                         │
│  4. TEST                                                │
│     AI writes tests. AID's @errors and @pre/@post       │
│     inform what edge cases to test.                     │
│                                                         │
│  5. FIX                                                 │
│     AI adjusts until tests pass                         │
│                                                         │
│  6. UPDATE AID                                          │
│     a. Re-run L1 extractor on changed packages          │
│     b. Check L2 staleness                               │
│     c. If stale: re-verify stale claims only            │
│     d. If new package: generate L1, optionally L2       │
│     e. Update manifest                                  │
│     f. Validate all AID files                           │
│                                                         │
│  7. COMMIT                                              │
│     Code + tests + .aidocs/ committed together          │
│     PR includes AID diff alongside code diff            │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## When to generate L2 for a package

Not every package needs L2. L1 (type signatures and API surface) is sufficient for simple utility packages. L2 (workflows, invariants, antipatterns, decisions) is worth the investment when:

| Signal | Why L2 helps |
|--------|-------------|
| Package has 3+ consumers | Multiple callers need to understand the contract |
| Cross-package data flow | Workflows document how data moves between packages |
| Non-obvious invariants | "BRIN is lossy" isn't visible from the type signature |
| Historical design decisions | @decision blocks prevent undoing intentional tradeoffs |
| Complex state machines | State transitions, lifecycle management, ordering constraints |
| Error handling patterns | Which errors propagate, which are caught, which are fatal |

For a greenfield project, the first L2 candidates are usually:
1. The **core domain model** — the types everything else depends on
2. The **main data pipeline** — how requests flow from input to output
3. The **storage/persistence layer** — lifecycle, transactions, cleanup

---

## Automating the cycle

### Git hook (pre-commit)

```bash
#!/bin/bash
# .git/hooks/pre-commit — auto-generate L1 AID for changed Go packages

changed_dirs=$(git diff --cached --name-only --diff-filter=ACM -- '*.go' |
  xargs -I{} dirname {} | sort -u)

for dir in $changed_dirs; do
  aid-gen-go -output .aidocs "$dir"
done

aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
aid-validate .aidocs/*.aid

git add .aidocs/
```

### CI check

```yaml
# .github/workflows/aid.yml
- name: Validate AID docs
  run: |
    aid-validate .aidocs/*.aid
    aid-gen-l2 stale --aid .aidocs/*.aid --project-root . || echo "Stale AID detected"
```

### Claude Code hook

```json
{
  "hooks": {
    "post-commit": "aid-gen-go -output .aidocs ./src/... && aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid"
  }
}
```

---

## What NOT to do

1. **Don't skip skeleton AID on a greenfield project.** The skeleton is the most efficient plan format for an AI agent. Writing prose plans instead means the AI has to interpret ambiguous language — and some interpretations will be wrong.

2. **Don't run L1 extraction before code exists.** L1 is mechanical — it extracts from source. Run it after the first implementation, not before. Skeleton AID is hand-written; L1 AID is extracted.

3. **Don't generate L2 for every package.** L2 costs tokens to generate and tokens to load. Only generate it for packages where the architectural knowledge is genuinely non-obvious. A `utils/strings` package doesn't need L2.

4. **Don't treat AID as a substitute for reading code.** AID is a map, not the territory. It tells the agent where to look and what to expect. The agent still reads source code — it just reads the right files first.

5. **Don't let AID go stale.** Stale AID is worse than no AID — it gives the agent false confidence. Use `aid-gen-l2 stale` in CI. If claims are stale, either update them or delete them.

---

## The minimal viable AID setup

### Greenfield (no code yet)

```bash
# Create .aidocs/ and write skeleton AID for your first package
mkdir .aidocs
cat > .aidocs/myapp-client.aid << 'EOF'
@module myapp/client
@lang go
@version 0.1.0
@purpose HTTP client with retry support

---

@fn Get
@sig (url: str) -> Response ! HttpError
@errors
  HttpError.Timeout — request exceeded deadline
  HttpError.NotFound — 404 response

---

@workflow request_lifecycle
@steps
  1. Build request from URL + config
  2. Send with retry (max 3 attempts on 5xx)
  3. Parse response or return error
@antipatterns
  - Don't retry on 4xx. Only 5xx and timeouts.
EOF

# Tell your AI assistant to implement against these contracts
echo "Implement code to match the AID contracts in .aidocs/" >> CLAUDE.md
```

### Existing code (post-implementation)

```bash
# Generate L1 from actual source (replaces skeleton with real data)
aid-gen-go -output .aidocs ./src/...
aid-manifest-gen --dir .aidocs > .aidocs/manifest.aid
git add .aidocs/
```

Either way, even the minimal setup gives the AI:
- Contracts to implement against (skeleton) or API surface to reference (L1)
- A manifest showing all packages and their purposes
- Method lists showing what infrastructure exists

Add L2, validation, and staleness checks as the project grows and the payoff justifies the investment.
